// Package executor 提供命令执行功能，解释执行AST并处理命令、管道、重定向等
package executor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"gobash/internal/builtin"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// Executor 执行器
// 负责解释执行AST，处理命令执行、管道、重定向、环境变量展开等功能
type Executor struct {
	env       map[string]string
	arrays    map[string][]string // 数组存储：数组名 -> 元素列表
	builtins  map[string]builtin.BuiltinFunc
	functions map[string]*parser.FunctionStatement
	options   map[string]bool // shell选项状态
	jobs      *JobManager     // 作业管理器
}

// New 创建新的执行器
func New() *Executor {
	e := &Executor{
		env:       make(map[string]string),
		arrays:    make(map[string][]string),
		builtins:  builtin.GetBuiltins(),
		functions: make(map[string]*parser.FunctionStatement),
		options:   make(map[string]bool),
		jobs:      NewJobManager(),
	}
	// 初始化环境变量
	for _, env := range os.Environ() {
		key, value := splitEnv(env)
		e.env[key] = value
	}
	return e
}

// SetOptions 设置shell选项
func (e *Executor) SetOptions(options map[string]bool) {
	e.options = options
}

// GetOptions 获取shell选项
func (e *Executor) GetOptions() map[string]bool {
	return e.options
}

// GetJobManager 获取作业管理器
func (e *Executor) GetJobManager() *JobManager {
	return e.jobs
}

// Execute 执行程序
func (e *Executor) Execute(program *parser.Program) error {
	for _, stmt := range program.Statements {
		if err := e.executeStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// executeStatement 执行语句
func (e *Executor) executeStatement(stmt parser.Statement) error {
	switch s := stmt.(type) {
	case *parser.CommandStatement:
		return e.executeCommand(s)
	case *parser.IfStatement:
		return e.executeIf(s)
	case *parser.ForStatement:
		return e.executeFor(s)
	case *parser.WhileStatement:
		return e.executeWhile(s)
	case *parser.FunctionStatement:
		// 存储函数定义
		e.functions[s.Name] = s
		return nil
	case *parser.BlockStatement:
		return e.executeBlock(s)
	case *parser.ArrayAssignmentStatement:
		return e.executeArrayAssignment(s)
	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}
}

// executeCommand 执行命令
func (e *Executor) executeCommand(cmd *parser.CommandStatement) error {
	if cmd.Command == nil {
		return fmt.Errorf("命令为空")
	}

	// 获取命令名
	cmdName := e.evaluateExpression(cmd.Command)
	if cmdName == "" {
		return fmt.Errorf("命令名为空")
	}

	// 检查是否为内置命令
	if builtinFunc, ok := e.builtins[cmdName]; ok {
		args := make([]string, len(cmd.Args))
		for i, arg := range cmd.Args {
			argValue := e.evaluateExpression(arg)
			// 检查未定义的变量（set -u）
			if strings.HasPrefix(argValue, "__UNDEFINED_VAR__") {
				varName := strings.TrimPrefix(argValue, "__UNDEFINED_VAR__")
				return fmt.Errorf("未定义的变量: %s", varName)
			}
			args[i] = argValue
		}
		
		// 如果设置了 -x 选项，显示执行的命令
		if e.options["x"] {
			fmt.Fprintf(os.Stderr, "+ %s", cmdName)
			for _, arg := range args {
				fmt.Fprintf(os.Stderr, " %s", arg)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
		
		// 处理内置命令的重定向
		if len(cmd.Redirects) > 0 {
			err := e.executeBuiltinWithRedirect(cmdName, builtinFunc, args, cmd.Redirects)
			// 如果设置了 -e 选项且命令失败，立即退出
			if err != nil && e.options["e"] {
				os.Exit(1)
			}
			return err
		}
		
		// 为需要访问JobManager的命令设置引用
		if cmdName == "jobs" || cmdName == "fg" || cmdName == "bg" {
			builtin.SetJobManager(e.jobs)
		}
		
		if err := builtinFunc(args, e.env); err != nil {
			// 如果设置了 -e 选项且命令失败，立即退出
			if e.options["e"] {
				os.Exit(1)
			}
			return fmt.Errorf("%s: %v", cmdName, err)
		}
		return nil
	}

	// 检查是否为定义的函数
	if fn, ok := e.functions[cmdName]; ok {
		return e.executeFunction(fn, cmd.Args)
	}

	// 如果设置了 -x 选项，显示执行的命令
	if e.options["x"] {
		cmdName := e.evaluateExpression(cmd.Command)
		fmt.Fprintf(os.Stderr, "+ %s", cmdName)
		for _, arg := range cmd.Args {
			argValue := e.evaluateExpression(arg)
			fmt.Fprintf(os.Stderr, " %s", argValue)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}
	
	// 执行外部命令
	err := e.executeExternalCommand(cmd)
	// 如果设置了 -e 选项且命令失败，立即退出
	if err != nil && e.options["e"] {
		os.Exit(1)
	}
	return err
}

// executeBuiltinWithRedirect 执行带重定向的内置命令
func (e *Executor) executeBuiltinWithRedirect(cmdName string, builtinFunc builtin.BuiltinFunc, args []string, redirects []*parser.Redirect) error {
	// 保存原始的stdout和stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	
	// 处理重定向
	var files []*os.File
	defer func() {
		// 恢复原始的stdout和stderr
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		// 关闭所有打开的文件
		for _, f := range files {
			f.Close()
		}
	}()
	
	for _, redirect := range redirects {
		target := e.evaluateExpression(redirect.Target)
		if target == "" {
			return fmt.Errorf("redirect target is empty")
		}
		
		switch redirect.Type {
		case parser.REDIRECT_OUTPUT:
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("重定向错误: %v", err)
			}
			files = append(files, file)
			if redirect.FD == 1 {
				os.Stdout = file
			} else if redirect.FD == 2 {
				os.Stderr = file
			}
		case parser.REDIRECT_APPEND:
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return fmt.Errorf("重定向错误: %v", err)
			}
			files = append(files, file)
			if redirect.FD == 1 {
				os.Stdout = file
			} else if redirect.FD == 2 {
				os.Stderr = file
			}
		case parser.REDIRECT_INPUT:
			file, err := os.Open(target)
			if err != nil {
				return fmt.Errorf("重定向错误: %v", err)
			}
			files = append(files, file)
			os.Stdin = file
		}
	}
	
	// 执行内置命令
	if err := builtinFunc(args, e.env); err != nil {
		return fmt.Errorf("%s: %v", cmdName, err)
	}
	
	return nil
}

// executeExternalCommand 执行外部命令
func (e *Executor) executeExternalCommand(cmd *parser.CommandStatement) error {
	cmdName := e.evaluateExpression(cmd.Command)
	if cmdName == "" {
		return fmt.Errorf("命令名为空")
	}

	// 构建参数
	args := make([]string, len(cmd.Args))
	for i, arg := range cmd.Args {
		argValue := e.evaluateExpression(arg)
		// 检查未定义的变量（set -u）
		if strings.HasPrefix(argValue, "__UNDEFINED_VAR__") {
			varName := strings.TrimPrefix(argValue, "__UNDEFINED_VAR__")
			return fmt.Errorf("未定义的变量: %s", varName)
		}
		args[i] = argValue
	}

	// 创建命令
	execCmd := exec.Command(cmdName, args...)
	execCmd.Env = e.getEnvArray()

	// 处理重定向
	if err := e.setupRedirects(execCmd, cmd.Redirects); err != nil {
		return fmt.Errorf("重定向错误: %v", err)
	}

	// 处理管道
	if cmd.Pipe != nil {
		return e.executePipe(cmd, cmd.Pipe)
	}

	// 设置标准输入输出（如果没有重定向）
	if execCmd.Stdin == nil {
		execCmd.Stdin = os.Stdin
	}
	if execCmd.Stdout == nil {
		execCmd.Stdout = os.Stdout
	}
	if execCmd.Stderr == nil {
		execCmd.Stderr = os.Stderr
	}

	// 执行命令
	if cmd.Background {
		if err := execCmd.Start(); err != nil {
			return fmt.Errorf("无法启动命令 '%s': %v", cmdName, err)
		}
		// 构建命令字符串用于显示
		cmdStr := cmdName
		for _, arg := range args {
			cmdStr += " " + arg
		}
		// 添加到作业管理器
		jobID := e.jobs.AddJob(execCmd, cmdStr)
		fmt.Fprintf(os.Stderr, "[%d] %d\n", jobID, execCmd.Process.Pid)
		return nil
	}

	if err := execCmd.Run(); err != nil {
		// 检查是否是命令未找到
		if _, ok := err.(*exec.ExitError); !ok {
			return fmt.Errorf("命令 '%s' 未找到或无法执行: %v", cmdName, err)
		}
		// 命令执行失败，返回退出码
		return err
	}

	return nil
}

// executePipe 执行管道
func (e *Executor) executePipe(left, right *parser.CommandStatement) error {
	leftCmdName := e.evaluateExpression(left.Command)
	if leftCmdName == "" {
		return fmt.Errorf("管道左侧命令名为空")
	}

	leftArgs := make([]string, len(left.Args))
	for i, arg := range left.Args {
		leftArgs[i] = e.evaluateExpression(arg)
	}

	rightCmdName := e.evaluateExpression(right.Command)
	if rightCmdName == "" {
		return fmt.Errorf("管道右侧命令名为空")
	}

	rightArgs := make([]string, len(right.Args))
	for i, arg := range right.Args {
		rightArgs[i] = e.evaluateExpression(arg)
	}

	// 创建左侧命令
	leftCmd := exec.Command(leftCmdName, leftArgs...)
	leftCmd.Env = e.getEnvArray()

	// 创建右侧命令
	rightCmd := exec.Command(rightCmdName, rightArgs...)
	rightCmd.Env = e.getEnvArray()

	// 设置管道
	pipe, err := leftCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建管道失败: %v", err)
	}
	rightCmd.Stdin = pipe
	rightCmd.Stdout = os.Stdout
	rightCmd.Stderr = os.Stderr

	// 启动右侧命令
	if err := rightCmd.Start(); err != nil {
		return fmt.Errorf("启动右侧命令 '%s' 失败: %v", rightCmdName, err)
	}

	// 启动左侧命令
	if err := leftCmd.Run(); err != nil {
		rightCmd.Process.Kill()
		return fmt.Errorf("执行左侧命令 '%s' 失败: %v", leftCmdName, err)
	}

	// 等待右侧命令完成
	if err := rightCmd.Wait(); err != nil {
		return fmt.Errorf("等待右侧命令 '%s' 完成失败: %v", rightCmdName, err)
	}

	return nil
}

// setupRedirects 设置重定向
func (e *Executor) setupRedirects(cmd *exec.Cmd, redirects []*parser.Redirect) error {
	for _, redirect := range redirects {
		target := e.evaluateExpression(redirect.Target)
		if target == "" {
			return fmt.Errorf("redirect target is empty")
		}

		switch redirect.Type {
		case parser.REDIRECT_OUTPUT:
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			if redirect.FD == 1 {
				cmd.Stdout = file
			} else if redirect.FD == 2 {
				cmd.Stderr = file
			} else {
				file.Close()
			}
		case parser.REDIRECT_APPEND:
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return err
			}
			if redirect.FD == 1 {
				cmd.Stdout = file
			} else if redirect.FD == 2 {
				cmd.Stderr = file
			} else {
				file.Close()
			}
		case parser.REDIRECT_INPUT:
			file, err := os.Open(target)
			if err != nil {
				return err
			}
			cmd.Stdin = file
		}
	}
	return nil
}

// executeIf 执行if语句
func (e *Executor) executeIf(stmt *parser.IfStatement) error {
	// 执行条件命令，检查退出码
	if err := e.executeCommand(stmt.Condition); err == nil {
		// 条件成功，执行consequence
		return e.executeBlock(stmt.Consequence)
	}

	// 条件失败，检查elif
	for _, elif := range stmt.Elif {
		if err := e.executeCommand(elif.Condition); err == nil {
			return e.executeBlock(elif.Consequence)
		}
	}

	// 执行else
	if stmt.Alternative != nil {
		return e.executeBlock(stmt.Alternative)
	}

	return nil
}

// executeFor 执行for循环
func (e *Executor) executeFor(stmt *parser.ForStatement) error {
	// 如果没有in子句，使用位置参数（$1, $2, ...）
	if len(stmt.In) == 0 {
		// 从环境变量中获取位置参数
		// 位置参数存储在环境变量中，键为"1", "2", "3"等
		// 参数个数存储在"#"中
		argCount := 0
		if countStr, ok := e.env["#"]; ok {
			if count, err := fmt.Sscanf(countStr, "%d", &argCount); err == nil && count == 1 {
				// 成功解析参数个数
			}
		}
		
		// 如果没有参数个数信息，尝试从$@获取
		if argCount == 0 {
			if allArgs, ok := e.env["@"]; ok && allArgs != "" {
				// 从$@中提取参数（空格分隔）
				args := strings.Fields(allArgs)
				argCount = len(args)
			}
		}
		
		// 遍历位置参数
		for i := 1; i <= argCount; i++ {
			key := fmt.Sprintf("%d", i)
			if value, ok := e.env[key]; ok {
				e.env[stmt.Variable] = value
				if err := e.executeBlock(stmt.Body); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// 有in子句，使用指定的值列表
	for _, item := range stmt.In {
		value := e.evaluateExpression(item)
		e.env[stmt.Variable] = value
		if err := e.executeBlock(stmt.Body); err != nil {
			return err
		}
	}

	return nil
}

// executeWhile 执行while循环
func (e *Executor) executeWhile(stmt *parser.WhileStatement) error {
	for {
		if err := e.executeCommand(stmt.Condition); err != nil {
			// 条件失败，退出循环
			break
		}
		if err := e.executeBlock(stmt.Body); err != nil {
			return err
		}
	}
	return nil
}

// executeBlock 执行代码块
func (e *Executor) executeBlock(block *parser.BlockStatement) error {
	for _, stmt := range block.Statements {
		if err := e.executeStatement(stmt); err != nil {
			return err
		}
	}
	return nil
}

// executeArrayAssignment 执行数组赋值
// 例如：arr=(1 2 3)
func (e *Executor) executeArrayAssignment(stmt *parser.ArrayAssignmentStatement) error {
	values := make([]string, 0, len(stmt.Values))
	for _, expr := range stmt.Values {
		value := e.evaluateExpression(expr)
		values = append(values, value)
	}
	e.arrays[stmt.Name] = values
	// 同时设置环境变量，使用特殊格式存储数组长度
	e.env[stmt.Name+"_LENGTH"] = fmt.Sprintf("%d", len(values))
	// 设置第一个元素为默认值（Bash行为）
	if len(values) > 0 {
		e.env[stmt.Name] = values[0]
	}
	return nil
}

// getArrayElement 获取数组元素
// 支持 ${arr[0]} 和 $arr[0] 格式
func (e *Executor) getArrayElement(varExpr string) string {
	// 解析数组名和索引
	// 格式：arr[0] 或 arr[1]
	idx := strings.Index(varExpr, "[")
	if idx == -1 {
		return ""
	}
	arrName := varExpr[:idx]
	idxEnd := strings.Index(varExpr, "]")
	if idxEnd == -1 {
		return ""
	}
	indexStr := varExpr[idx+1 : idxEnd]
	
	// 解析索引
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return ""
	}
	
	// 获取数组
	arr, ok := e.arrays[arrName]
	if !ok {
		// 如果设置了 -u 选项，未定义的数组应该报错
		if e.options["u"] {
			return "__UNDEFINED_VAR__" + arrName
		}
		return ""
	}
	
	// 检查索引范围
	if index < 0 || index >= len(arr) {
		return ""
	}
	
	return arr[index]
}

// evaluateExpression 求值表达式
func (e *Executor) evaluateExpression(expr parser.Expression) string {
	switch ex := expr.(type) {
	case *parser.Identifier:
		return ex.Value
	case *parser.StringLiteral:
		// 只有双引号字符串才展开变量，单引号字符串不展开
		if ex.IsQuote {
			result := e.expandVariablesInString(ex.Value)
			// 检查未定义的变量（set -u）
			if strings.Contains(result, "__UNDEFINED_VAR__") {
				// 提取第一个未定义的变量名
				start := strings.Index(result, "__UNDEFINED_VAR__")
				if start != -1 {
					rest := result[start+len("__UNDEFINED_VAR__"):]
					varName := ""
					for _, ch := range rest {
						if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
							varName += string(ch)
						} else {
							break
						}
					}
					return "__UNDEFINED_VAR__" + varName
				}
			}
			return result
		}
		return ex.Value
	case *parser.Variable:
		// 检查是否是数组访问 ${arr[0]} 或 $arr[0]
		if strings.Contains(ex.Name, "[") && strings.Contains(ex.Name, "]") {
			return e.getArrayElement(ex.Name)
		}
		// 检查是否是数组变量（返回所有元素，空格分隔）
		if arr, ok := e.arrays[ex.Name]; ok {
			return strings.Join(arr, " ")
		}
		if value, ok := e.env[ex.Name]; ok {
			return value
		}
		// 如果设置了 -u 选项，未定义的变量应该报错
		if e.options["u"] {
			// 返回特殊标记，让调用者知道变量未定义
			return "__UNDEFINED_VAR__" + ex.Name
		}
		return ""
	case *parser.CommandSubstitution:
		// 执行命令替换
		return e.executeCommandSubstitution(ex.Command)
	case *parser.ArithmeticExpansion:
		// 执行算术展开
		return e.evaluateArithmetic(ex.Expression)
	default:
		return ""
	}
}

// expandVariablesInString 展开字符串中的变量（如 "TEST=$TEST"）
func (e *Executor) expandVariablesInString(s string) string {
	// 如果字符串为空，直接返回
	if len(s) == 0 {
		return ""
	}
	
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '$' {
			// 转义的 $，保留为 $
			result.WriteByte('$')
			i += 2
		} else if s[i] == '$' && i+1 < len(s) {
			// 处理变量展开
			var varName strings.Builder
			
			if i+1 < len(s) && s[i+1] == '{' {
				// ${VAR} 或 ${arr[0]} 格式
				i += 2
				for i < len(s) && s[i] != '}' {
					varName.WriteByte(s[i])
					i++
				}
				if i < len(s) && s[i] == '}' {
					i++
					varNameStr := varName.String()
					// 检查是否是数组访问
					if strings.Contains(varNameStr, "[") {
						value := e.getArrayElement(varNameStr)
						if value != "" {
							result.WriteString(value)
						} else if e.options["u"] && !strings.Contains(value, "__UNDEFINED_VAR__") {
							result.WriteString("__UNDEFINED_VAR__" + varNameStr)
						}
					} else {
						// 检查是否是数组变量（返回所有元素）
						if arr, ok := e.arrays[varNameStr]; ok {
							result.WriteString(strings.Join(arr, " "))
						} else if value, ok := e.env[varNameStr]; ok {
							result.WriteString(value)
						} else if e.options["u"] {
							// 如果设置了 -u 选项，未定义的变量返回特殊标记
							result.WriteString("__UNDEFINED_VAR__" + varNameStr)
						}
					}
					continue
				}
			} else if i+1 < len(s) && (isLetter(s[i+1]) || s[i+1] == '_') {
				// $VAR 格式，可能包含数组访问 $arr[0]
				i++
				for i < len(s) && (isLetter(s[i]) || isDigit(s[i]) || s[i] == '_' || s[i] == '[' || s[i] == ']') {
					varName.WriteByte(s[i])
					i++
				}
				varNameStr := varName.String()
				// 检查是否是数组访问
				if strings.Contains(varNameStr, "[") {
					value := e.getArrayElement(varNameStr)
					if value != "" {
						result.WriteString(value)
					} else if e.options["u"] && !strings.Contains(value, "__UNDEFINED_VAR__") {
						result.WriteString("__UNDEFINED_VAR__" + varNameStr)
					}
				} else {
					// 检查是否是数组变量（返回所有元素）
					if arr, ok := e.arrays[varNameStr]; ok {
						result.WriteString(strings.Join(arr, " "))
					} else if value, ok := e.env[varNameStr]; ok {
						result.WriteString(value)
					} else if e.options["u"] {
						// 如果设置了 -u 选项，未定义的变量返回特殊标记
						result.WriteString("__UNDEFINED_VAR__" + varNameStr)
					}
				}
				continue
			}
			// 不是变量，保留原字符
			result.WriteByte(s[i])
			i++
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// isLetter 判断是否为字母
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}

// isDigit 判断是否为数字
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// getEnvArray 获取环境变量数组
func (e *Executor) getEnvArray() []string {
	env := make([]string, 0, len(e.env))
	for k, v := range e.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

// SetEnv 设置环境变量
func (e *Executor) SetEnv(key, value string) {
	e.env[key] = value
	os.Setenv(key, value)
}

// GetEnv 获取环境变量
func (e *Executor) GetEnv(key string) (string, bool) {
	value, ok := e.env[key]
	return value, ok
}

// executeFunction 执行函数
func (e *Executor) executeFunction(fn *parser.FunctionStatement, args []parser.Expression) error {
	// 保存当前环境变量
	oldEnv := make(map[string]string)
	for k, v := range e.env {
		oldEnv[k] = v
	}

	// 设置函数参数为位置参数（$1, $2, ...）
	for i, arg := range args {
		argValue := e.evaluateExpression(arg)
		e.env[fmt.Sprintf("%d", i+1)] = argValue
	}
	e.env["#"] = fmt.Sprintf("%d", len(args)) // $# 参数个数
	e.env["@"] = strings.Join(func() []string {
		result := make([]string, len(args))
		for i, arg := range args {
			result[i] = e.evaluateExpression(arg)
		}
		return result
	}(), " ") // $@ 所有参数

	// 执行函数体
	err := e.executeBlock(fn.Body)

	// 恢复环境变量（但保留新设置的环境变量）
	for k, v := range oldEnv {
		if _, exists := e.env[k]; !exists {
			delete(e.env, k)
		} else {
			e.env[k] = v
		}
	}

	// 清理位置参数
	for i := 1; i <= len(args); i++ {
		delete(e.env, fmt.Sprintf("%d", i))
	}
	delete(e.env, "#")
	delete(e.env, "@")

	return err
}

// executeCommandSubstitution 执行命令替换
func (e *Executor) executeCommandSubstitution(command string) string {
	// 解析和执行命令
	l := lexer.New(command)
	p := parser.New(l)
	program := p.ParseProgram()
	
	if len(p.Errors()) > 0 {
		return ""
	}
	
	// 捕获输出
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return ""
	}
	os.Stdout = w
	
	// 在goroutine中执行，避免阻塞
	done := make(chan bool)
	var output strings.Builder
	
	go func() {
		// 读取输出
		buf := make([]byte, 1024)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				break
			}
		}
		r.Close()
		done <- true
	}()
	
	// 执行命令
	execErr := e.Execute(program)
	
	w.Close()
	os.Stdout = oldStdout
	
	// 等待读取完成
	<-done
	
	if execErr != nil {
		return ""
	}
	
	result := output.String()
	// 移除末尾的换行符（如果存在）
	result = strings.TrimSuffix(result, "\n")
	result = strings.TrimSuffix(result, "\r\n")
	
	return result
}

// evaluateArithmetic 计算算术表达式
func (e *Executor) evaluateArithmetic(expr string) string {
	// 移除空白字符
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "0"
	}
	
	// 展开变量
	expr = e.expandVariablesInString(expr)
	
	// 简单的算术表达式求值
	// 支持: +, -, *, /, %, (, )
	// 使用递归下降解析器
	
	result, err := evaluateArithmeticExpression(expr)
	if err != nil {
		return "0"
	}
	
	return fmt.Sprintf("%d", result)
}

// evaluateArithmeticExpression 计算算术表达式（支持 +, -, *, /, %, 括号）
func evaluateArithmeticExpression(expr string) (int64, error) {
	// 移除所有空白字符
	expr = strings.ReplaceAll(expr, " ", "")
	expr = strings.ReplaceAll(expr, "\t", "")
	
	if expr == "" {
		return 0, nil
	}
	
	// 使用简单的递归下降解析器
	pos := 0
	result, err := parseExpression(expr, &pos)
	if err != nil {
		return 0, err
	}
	
	return result, nil
}

// parseExpression 解析表达式（处理 +, -）
func parseExpression(expr string, pos *int) (int64, error) {
	result, err := parseTerm(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if expr[*pos] == '+' {
			*pos++
			term, err := parseTerm(expr, pos)
			if err != nil {
				return 0, err
			}
			result += term
		} else if expr[*pos] == '-' {
			*pos++
			term, err := parseTerm(expr, pos)
			if err != nil {
				return 0, err
			}
			result -= term
		} else {
			break
		}
	}
	
	return result, nil
}

// parseTerm 解析项（处理 *, /, %）
func parseTerm(expr string, pos *int) (int64, error) {
	result, err := parseFactor(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if expr[*pos] == '*' {
			*pos++
			factor, err := parseFactor(expr, pos)
			if err != nil {
				return 0, err
			}
			result *= factor
		} else if expr[*pos] == '/' {
			*pos++
			factor, err := parseFactor(expr, pos)
			if err != nil {
				return 0, err
			}
			if factor == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			result /= factor
		} else if expr[*pos] == '%' {
			*pos++
			factor, err := parseFactor(expr, pos)
			if err != nil {
				return 0, err
			}
			if factor == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			result %= factor
		} else {
			break
		}
	}
	
	return result, nil
}

// parseFactor 解析因子（处理数字和括号）
func parseFactor(expr string, pos *int) (int64, error) {
	if *pos >= len(expr) {
		return 0, fmt.Errorf("unexpected end of expression")
	}
	
	if expr[*pos] == '(' {
		*pos++
		result, err := parseExpression(expr, pos)
		if err != nil {
			return 0, err
		}
		if *pos >= len(expr) || expr[*pos] != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		*pos++
		return result, nil
	}
	
	// 解析数字
	start := *pos
	if expr[*pos] == '-' || expr[*pos] == '+' {
		*pos++
	}
	
	if *pos >= len(expr) || !isDigitArith(expr[*pos]) {
		return 0, fmt.Errorf("expected number")
	}
	
	for *pos < len(expr) && isDigitArith(expr[*pos]) {
		*pos++
	}
	
	numStr := expr[start:*pos]
	value, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, err
	}
	
	return value, nil
}

// isDigitArith 判断是否为数字（用于算术表达式）
func isDigitArith(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// splitEnv 分割环境变量字符串
func splitEnv(env string) (string, string) {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return env[:i], env[i+1:]
		}
	}
	return env, ""
}

