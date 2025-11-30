package executor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"gobash/internal/builtin"
	"gobash/internal/parser"
)

// Executor 执行器
type Executor struct {
	env       map[string]string
	builtins  map[string]builtin.BuiltinFunc
	functions map[string]*parser.FunctionStatement
	options   map[string]bool // shell选项状态
}

// New 创建新的执行器
func New() *Executor {
	e := &Executor{
		env:       make(map[string]string),
		builtins:  builtin.GetBuiltins(),
		functions: make(map[string]*parser.FunctionStatement),
		options:   make(map[string]bool),
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
		// 如果设置了 -x 选项，显示执行的命令
		if e.options["x"] {
			fmt.Fprintf(os.Stderr, "+ %s", cmdName)
			for _, arg := range cmd.Args {
				argValue := e.evaluateExpression(arg)
				fmt.Fprintf(os.Stderr, " %s", argValue)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
		
		args := make([]string, len(cmd.Args))
		for i, arg := range cmd.Args {
			args[i] = e.evaluateExpression(arg)
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
		args[i] = e.evaluateExpression(arg)
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

// evaluateExpression 求值表达式
func (e *Executor) evaluateExpression(expr parser.Expression) string {
	switch ex := expr.(type) {
	case *parser.Identifier:
		return ex.Value
	case *parser.StringLiteral:
		// 只有双引号字符串才展开变量，单引号字符串不展开
		if ex.IsQuote {
			return e.expandVariablesInString(ex.Value)
		}
		return ex.Value
	case *parser.Variable:
		if value, ok := e.env[ex.Name]; ok {
			return value
		}
		return ""
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
				// ${VAR} 格式
				i += 2
				for i < len(s) && s[i] != '}' {
					varName.WriteByte(s[i])
					i++
				}
				if i < len(s) && s[i] == '}' {
					i++
					// 展开变量
					if value, ok := e.env[varName.String()]; ok {
						result.WriteString(value)
					}
					continue
				}
			} else if i+1 < len(s) && (isLetter(s[i+1]) || s[i+1] == '_') {
				// $VAR 格式
				i++
				for i < len(s) && (isLetter(s[i]) || isDigit(s[i]) || s[i] == '_') {
					varName.WriteByte(s[i])
					i++
				}
				// 展开变量
				if value, ok := e.env[varName.String()]; ok {
					result.WriteString(value)
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

// splitEnv 分割环境变量字符串
func splitEnv(env string) (string, string) {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return env[:i], env[i+1:]
		}
	}
	return env, ""
}

