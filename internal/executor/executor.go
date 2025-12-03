// Package executor 提供命令执行功能，解释执行AST并处理命令、管道、重定向等
package executor

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"gobash/internal/builtin"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// BreakError 表示break语句
var BreakError = errors.New("break")

// ContinueError 表示continue语句
var ContinueError = errors.New("continue")

// BreakLevelError 表示带层级的break语句
type BreakLevelError struct {
	Level int
}

func (e *BreakLevelError) Error() string {
	return fmt.Sprintf("break %d", e.Level)
}

// ContinueLevelError 表示带层级的continue语句
type ContinueLevelError struct {
	Level int
}

func (e *ContinueLevelError) Error() string {
	return fmt.Sprintf("continue %d", e.Level)
}

// ScriptExitError 表示脚本退出错误，包含退出码
type ScriptExitError struct {
	Code int
}

func (e *ScriptExitError) Error() string {
	return fmt.Sprintf("script exit %d", e.Code)
}

// Executor 执行器
// 负责解释执行AST，处理命令执行、管道、重定向、环境变量展开等功能
type Executor struct {
	env            map[string]string
	arrays         map[string][]string            // 数组存储：数组名 -> 元素列表
	assocArrays    map[string]map[string]string   // 关联数组存储：数组名 -> (键 -> 值)
	arrayTypes     map[string]string              // 数组类型：数组名 -> "array" 或 "assoc"
	builtins       map[string]builtin.BuiltinFunc
	functions      map[string]*parser.FunctionStatement
	options        map[string]bool // shell选项状态
	jobs           *JobManager     // 作业管理器
}

// New 创建新的执行器
func New() *Executor {
	e := &Executor{
		env:         make(map[string]string),
		arrays:      make(map[string][]string),
		assocArrays: make(map[string]map[string]string),
		arrayTypes:  make(map[string]string),
		builtins:    builtin.GetBuiltins(),
		functions:   make(map[string]*parser.FunctionStatement),
		options:     make(map[string]bool),
		jobs:        NewJobManager(),
	}
	// 初始化环境变量
	for _, env := range os.Environ() {
		key, value := splitEnv(env)
		e.env[key] = value
	}
	// 初始化位置参数：如果没有参数，$# 为 0
	e.env["#"] = "0"
	e.env["@"] = ""
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
	if stmt == nil {
		return nil // 空语句，直接返回
	}
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
	case *parser.CaseStatement:
		return e.executeCaseStatement(s)
	case *parser.BreakStatement:
		return e.executeBreak(s)
	case *parser.ContinueStatement:
		return e.executeContinue(s)
	default:
		return fmt.Errorf("unknown statement type: %T", stmt)
	}
}

// executeCommand 执行命令
func (e *Executor) executeCommand(cmd *parser.CommandStatement) error {
	if cmd == nil || cmd.Command == nil {
		return nil // 空命令，直接返回
	}

	// 获取命令名
	cmdName := e.evaluateExpression(cmd.Command)
	if cmdName == "" {
		return fmt.Errorf("命令名为空")
	}

	// 检查是否是简单的变量赋值 VAR=value
	// 注意：需要检查第一个 = 号，因为值中可能也包含 =（虽然不常见）
	if strings.Contains(cmdName, "=") {
		// 找到第一个 = 号的位置
		eqIndex := strings.Index(cmdName, "=")
		if eqIndex > 0 {
			// 检查变量名部分是否包含 [（关联数组赋值 arr[key]=value）
			varNamePart := strings.TrimSpace(cmdName[:eqIndex])
			if !strings.Contains(varNamePart, "[") {
				// 这是简单的变量赋值
				varName := varNamePart
				varValue := strings.TrimSpace(cmdName[eqIndex+1:])
				
				// 检查变量名是否有效（只包含字母、数字和下划线，且不能以数字开头）
				if varName != "" {
					isValidVarName := true
					for i, ch := range varName {
						if i == 0 {
							if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
								isValidVarName = false
								break
							}
						} else {
							if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
								isValidVarName = false
								break
							}
						}
					}
					
					if isValidVarName {
						// 移除引号（如果有）
						if len(varValue) >= 2 {
							if (varValue[0] == '"' && varValue[len(varValue)-1] == '"') ||
							   (varValue[0] == '\'' && varValue[len(varValue)-1] == '\'') {
								varValue = varValue[1 : len(varValue)-1]
							}
						}
						// 展开变量值中的变量（单引号字符串中的变量不应该展开，但这里已经移除了引号）
						varValue = e.expandVariablesInString(varValue)
						// 设置环境变量
						e.SetEnv(varName, varValue)
						return nil
					}
				}
			}
		}
	}

	// 检查是否是关联数组赋值 arr[key]=value
	if strings.Contains(cmdName, "[") && strings.Contains(cmdName, "]") && strings.Contains(cmdName, "=") {
		return e.executeAssocArrayAssignment(cmdName, cmd.Args)
	}

	// 检查是否为内置命令或特殊命令（[ 或 [[）
	if cmdName == "[" || cmdName == "[[" {
		// 处理 [ 或 [[ 命令（test命令）
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
		
		// 移除结束括号（] 或 ]]）
		if len(args) > 0 {
			lastArg := args[len(args)-1]
			if lastArg == "]" || lastArg == "]]" {
				args = args[:len(args)-1]
			}
		}
		
		// 对于 [[ 命令，需要支持 && 和 || 运算符
		if cmdName == "[[" {
			result, err := e.evaluateDoubleBracketExpression(args)
			if err != nil {
				if e.options["e"] {
					fmt.Fprintf(os.Stderr, "gobash: [[: %v\n", err)
					os.Exit(1)
				}
				return err
			}
			if !result {
				// 条件为假，返回退出码错误（ExitCode=1），这样while循环可以正确处理
				if e.options["e"] {
					fmt.Fprintf(os.Stderr, "gobash: [[: 条件为假\n")
					os.Exit(1)
				}
				// 返回一个ExitError，退出码为1
				// 创建一个命令来获取ExitError
				cmd := exec.Command("cmd", "/c", "exit", "1")
				_ = cmd.Run()
				if cmd.ProcessState != nil {
					return &exec.ExitError{ProcessState: cmd.ProcessState}
				}
				// 如果无法创建ExitError，返回一个普通错误
				return fmt.Errorf("test failed")
			}
			return nil
		}
		
		// 对于 [ 命令，调用test命令
		testFunc := e.builtins["test"]
		if testFunc == nil {
			return fmt.Errorf("test命令未找到")
		}
		
		if err := testFunc(args, e.env); err != nil {
			// 如果设置了 -e 选项且命令失败，输出错误信息后退出
			if e.options["e"] {
				fmt.Fprintf(os.Stderr, "gobash: test: %v\n", err)
				os.Exit(1)
			}
			return err
		}
		
		return nil
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
			// 检查是否是 exit 命令，如果是，直接返回，不包装
			if _, ok := err.(*builtin.ExitError); ok {
				return err
			}
			// 如果设置了 -e 选项且命令失败，输出错误信息后退出
			if err != nil && e.options["e"] {
				fmt.Fprintf(os.Stderr, "gobash: %s: %v\n", cmdName, err)
				os.Exit(1)
			}
			return err
		}
		
		// 为需要访问JobManager的命令设置引用
		if cmdName == "jobs" || cmdName == "fg" || cmdName == "bg" {
			builtin.SetJobManager(e.jobs)
		}
		
		if err := builtinFunc(args, e.env); err != nil {
			// 检查是否是 exit 命令，如果是，直接返回，不包装
			if _, ok := err.(*builtin.ExitError); ok {
				return err
			}
			// 如果设置了 -e 选项且命令失败，输出错误信息后退出
			if e.options["e"] {
				fmt.Fprintf(os.Stderr, "gobash: %s: %v\n", cmdName, err)
				os.Exit(1)
			}
			return fmt.Errorf("%s: %v", cmdName, err)
		}
		
		// 处理declare命令的特殊情况
		if cmdName == "declare" {
			// 检查是否声明了关联数组
			if assocName, ok := e.env["__WBASH_DECLARE_ASSOC__"]; ok {
				// 初始化关联数组
				if e.assocArrays[assocName] == nil {
					e.assocArrays[assocName] = make(map[string]string)
				}
				e.arrayTypes[assocName] = "assoc"
				delete(e.env, "__WBASH_DECLARE_ASSOC__")
			}
			// 检查是否声明了普通变量
			if varName, ok := e.env["__WBASH_DECLARE_VAR__"]; ok {
				e.arrayTypes[varName] = "var"
				delete(e.env, "__WBASH_DECLARE_VAR__")
			}
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
	// 如果设置了 -e 选项且命令失败，输出错误信息后退出
	if err != nil && e.options["e"] {
		// 输出错误信息到 stderr（如果还没有输出）
		if err != nil {
			fmt.Fprintf(os.Stderr, "gobash: %v\n", err)
		}
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

	// 对于前台命令，使用 Start() + Wait() 而不是 Run()，以便处理信号
	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("无法启动命令 '%s': %v", cmdName, err)
	}

	// 设置信号处理，当收到 SIGINT (Ctrl+C) 时，向子进程发送信号
	// os.Interrupt 在所有平台都可用（Windows/Linux/macOS）
	// syscall.SIGTERM 在 Unix 系统上可用，Windows 上会被 signal.Notify 自动忽略
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// 使用 goroutine 等待命令完成
	done := make(chan error, 1)
	go func() {
		done <- execCmd.Wait()
	}()

	// 等待命令完成或收到信号
	select {
	case err := <-done:
		// 命令完成，停止信号监听
		signal.Stop(sigChan)
		if err != nil {
			// 检查是否是命令未找到
			if _, ok := err.(*exec.ExitError); !ok {
				return fmt.Errorf("命令 '%s' 未找到或无法执行: %v", cmdName, err)
			}
			// 命令执行失败，返回退出码
			return err
		}
		return nil
	case sig := <-sigChan:
		// 收到中断信号，向子进程发送相同的信号
		if execCmd.Process != nil {
			// 尝试优雅地终止进程
			// 注意：在 Windows 上，某些信号可能不被支持，Signal() 可能返回错误
			// 我们忽略这个错误，因为如果 Signal() 失败，我们会用 Kill() 作为后备
			_ = execCmd.Process.Signal(sig)
			// 等待一小段时间让进程有机会退出
			select {
			case <-done:
				// 进程已经退出
			default:
				// 如果进程没有退出，强制终止
				execCmd.Process.Kill()
				<-done
			}
		}
		signal.Stop(sigChan)
		// 返回中断错误
		return fmt.Errorf("命令被中断")
	}
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
	if err := leftCmd.Start(); err != nil {
		rightCmd.Process.Kill()
		return fmt.Errorf("启动左侧命令 '%s' 失败: %v", leftCmdName, err)
	}

	// 设置信号处理，当收到 SIGINT (Ctrl+C) 时，向子进程发送信号
	// os.Interrupt 在所有平台都可用（Windows/Linux/macOS）
	// syscall.SIGTERM 在 Unix 系统上可用，Windows 上会被 signal.Notify 自动忽略
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 使用 goroutine 等待两个命令完成
	done := make(chan error, 2)
	go func() {
		done <- leftCmd.Wait()
	}()
	go func() {
		done <- rightCmd.Wait()
	}()

	// 等待命令完成或收到信号
	select {
	case err := <-done:
		// 第一个命令完成，检查是否有错误
		if err != nil {
			// 如果左侧命令失败，终止右侧命令
			if rightCmd.Process != nil {
				rightCmd.Process.Kill()
			}
			signal.Stop(sigChan)
			return fmt.Errorf("执行左侧命令 '%s' 失败: %v", leftCmdName, err)
		}
		// 关闭管道，让右侧命令知道输入结束
		pipe.Close()
		// 等待右侧命令完成
		err = <-done
		signal.Stop(sigChan)
		if err != nil {
			return fmt.Errorf("等待右侧命令 '%s' 完成失败: %v", rightCmdName, err)
		}
		return nil
	case sig := <-sigChan:
		// 收到中断信号，向两个进程发送相同的信号
		// 注意：在 Windows 上，某些信号可能不被支持，Signal() 可能返回错误
		// 我们忽略这个错误，因为如果 Signal() 失败，我们会用 Kill() 作为后备
		if leftCmd.Process != nil {
			_ = leftCmd.Process.Signal(sig)
		}
		if rightCmd.Process != nil {
			_ = rightCmd.Process.Signal(sig)
		}
		// 等待一小段时间让进程有机会退出
		select {
		case <-done:
			<-done
		default:
			// 如果进程没有退出，强制终止
			if leftCmd.Process != nil {
				leftCmd.Process.Kill()
			}
			if rightCmd.Process != nil {
				rightCmd.Process.Kill()
			}
			<-done
			<-done
		}
		signal.Stop(sigChan)
		// 返回中断错误
		return fmt.Errorf("命令被中断")
	}
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
		case parser.REDIRECT_HEREDOC, parser.REDIRECT_HEREDOC_STRIP:
			// Here-document 处理
			if redirect.HereDoc != nil {
				content := redirect.HereDoc.Content
				if content == "" {
					// 如果内容为空，尝试从标准输入读取（这在交互模式下可能需要）
					// 对于脚本模式，内容应该在解析时已经填充
					content = e.readHereDocument(redirect.HereDoc.Delimiter, redirect.HereDoc.Quoted, redirect.HereDoc.StripTabs)
					redirect.HereDoc.Content = content
				}
				// 创建临时文件或使用字符串作为输入
				reader := strings.NewReader(content)
				cmd.Stdin = io.NopCloser(reader)
			}
		case parser.REDIRECT_HERESTRING:
			// Here-string (<<<) 处理
			if redirect.Target != nil {
				content := e.evaluateExpression(redirect.Target)
				reader := strings.NewReader(content)
				cmd.Stdin = io.NopCloser(reader)
			}
		case parser.REDIRECT_DUP_IN:
			// <& 复制文件描述符
			_, err := strconv.Atoi(target)
			if err != nil {
				return fmt.Errorf("无效的文件描述符: %s", target)
			}
			// 这里简化处理，实际应该复制文件描述符
			// 在 Go 中，这需要更复杂的处理
		case parser.REDIRECT_DUP_OUT:
			// >& 复制文件描述符
			_, err := strconv.Atoi(target)
			if err != nil {
				return fmt.Errorf("无效的文件描述符: %s", target)
			}
			// 这里简化处理，实际应该复制文件描述符
		case parser.REDIRECT_CLOBBER:
			// >| 强制覆盖（与 > 相同，但忽略 noclobber 选项）
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
		case parser.REDIRECT_RW:
			// <> 读写重定向
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				return err
			}
			cmd.Stdin = file
			cmd.Stdout = file
		}
	}
	return nil
}

// executeIf 执行if语句
func (e *Executor) executeIf(stmt *parser.IfStatement) error {
	// 执行条件命令，检查退出码
	if err := e.executeCommand(stmt.Condition); err == nil {
		// 条件成功，执行consequence
		if err := e.executeBlock(stmt.Consequence); err != nil {
			// 检查是否是 break/continue 错误，需要向上传播
			if err == BreakError || err == ContinueError {
				return err
			}
			if _, ok := err.(*BreakLevelError); ok {
				return err
			}
			if _, ok := err.(*ContinueLevelError); ok {
				return err
			}
			return err
		}
	}

	// 条件失败，检查elif
	for _, elif := range stmt.Elif {
		if err := e.executeCommand(elif.Condition); err == nil {
			if err := e.executeBlock(elif.Consequence); err != nil {
				// 检查是否是 break/continue 错误，需要向上传播
				if err == BreakError || err == ContinueError {
					return err
				}
				if _, ok := err.(*BreakLevelError); ok {
					return err
				}
				if _, ok := err.(*ContinueLevelError); ok {
					return err
				}
				return err
			}
		}
	}

	// 执行else
	if stmt.Alternative != nil {
		if err := e.executeBlock(stmt.Alternative); err != nil {
			// 检查是否是 break/continue 错误，需要向上传播
			if err == BreakError || err == ContinueError {
				return err
			}
			if _, ok := err.(*BreakLevelError); ok {
				return err
			}
			if _, ok := err.(*ContinueLevelError); ok {
				return err
			}
			return err
		}
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
					// 检查是否是 break 或 continue
					if err == BreakError {
						break
					}
					if err == ContinueError {
						continue
					}
					if breakErr, ok := err.(*BreakLevelError); ok {
						if breakErr.Level <= 1 {
							break
						} else {
							// 需要跳出更多层，向上传播
							return err
						}
					}
					if continueErr, ok := err.(*ContinueLevelError); ok {
						if continueErr.Level <= 1 {
							continue
						} else {
							// 需要继续更多层，向上传播
							return err
						}
					}
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
			// 检查是否是 break 或 continue
			if err == BreakError {
				break
			}
			if err == ContinueError {
				continue
			}
			if breakErr, ok := err.(*BreakLevelError); ok {
				if breakErr.Level <= 1 {
					break
				} else {
					// 需要跳出更多层，向上传播
					return err
				}
			}
			if continueErr, ok := err.(*ContinueLevelError); ok {
				if continueErr.Level <= 1 {
					continue
				} else {
					// 需要继续更多层，向上传播
					return err
				}
			}
			return err
		}
	}

	return nil
}

// executeWhile 执行while循环
func (e *Executor) executeWhile(stmt *parser.WhileStatement) error {
	// 保存原始的 set -e 状态
	originalSetE := e.options["e"]
	// 在 while 循环条件中，临时禁用 set -e（bash 的行为）
	e.options["e"] = false
	
	for {
		// 执行条件命令，检查退出码
		// 如果命令返回错误（非零退出码），条件为假，退出循环
		// 如果命令成功（零退出码），条件为真，继续执行循环体
		err := e.executeCommand(stmt.Condition)
		if err != nil {
			// 检查是否是退出码错误（ExitError）
			if exitErr, ok := err.(*exec.ExitError); ok {
				// 这是正常的退出码，非零表示条件为假
				exitCode := exitErr.ExitCode()
				if exitCode != 0 {
					break
				}
			} else {
				// 其他错误（如命令未找到），也视为条件为假
				break
			}
		}
		// 条件为真，执行循环体（恢复原始的 set -e 状态）
		e.options["e"] = originalSetE
		// 检查循环体是否为空
		if stmt.Body != nil && len(stmt.Body.Statements) > 0 {
			if err := e.executeBlock(stmt.Body); err != nil {
				// 检查是否是 break 或 continue
				if err == BreakError {
					e.options["e"] = originalSetE
					break
				}
				if err == ContinueError {
					e.options["e"] = false
					continue
				}
				if breakErr, ok := err.(*BreakLevelError); ok {
					if breakErr.Level <= 1 {
						e.options["e"] = originalSetE
						break
					} else {
						// 需要跳出更多层，向上传播
						e.options["e"] = originalSetE
						return err
					}
				}
				if continueErr, ok := err.(*ContinueLevelError); ok {
					if continueErr.Level <= 1 {
						e.options["e"] = false
						continue
					} else {
						// 需要继续更多层，向上传播
						e.options["e"] = false
						return err
					}
				}
				// 在循环体中，如果 set -e 启用且出错，应该退出
				e.options["e"] = originalSetE
				return err
			}
		}
		// 再次禁用 set -e 用于下一次条件检查
		e.options["e"] = false
	}
	// 恢复原始的 set -e 状态
	e.options["e"] = originalSetE
	return nil
}

// executeBreak 执行break语句
func (e *Executor) executeBreak(stmt *parser.BreakStatement) error {
	if stmt.Level > 1 {
		return &BreakLevelError{Level: stmt.Level}
	}
	return BreakError
}

// executeContinue 执行continue语句
func (e *Executor) executeContinue(stmt *parser.ContinueStatement) error {
	if stmt.Level > 1 {
		return &ContinueLevelError{Level: stmt.Level}
	}
	return ContinueError
}

// executeBlock 执行代码块
func (e *Executor) executeBlock(block *parser.BlockStatement) error {
	for _, stmt := range block.Statements {
		if err := e.executeStatement(stmt); err != nil {
			// 传播 break/continue 错误
			if err == BreakError || err == ContinueError {
				return err
			}
			if _, ok := err.(*BreakLevelError); ok {
				return err
			}
			if _, ok := err.(*ContinueLevelError); ok {
				return err
			}
			return err
		}
	}
	return nil
}

// executeArrayAssignment 执行数组赋值
// 例如：arr=(1 2 3) 或 arr=([0]=a [1]=b [2]=c)
func (e *Executor) executeArrayAssignment(stmt *parser.ArrayAssignmentStatement) error {
	// 检查是否是带索引的数组赋值
	if len(stmt.IndexedValues) > 0 {
		// 带索引的数组赋值 arr=([0]=a [1]=b [2]=c)
		// 首先确定数组的最大索引
		maxIndex := -1
		indexedMap := make(map[int]string)
		hasStringKeys := false
		
		for indexStr, valueExpr := range stmt.IndexedValues {
			// 索引字符串可能是数字字符串或变量名
			// 先尝试直接解析为数字
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				// 不是数字索引，可能是关联数组
				hasStringKeys = true
				// 检查数组类型
				if arrayType, ok := e.arrayTypes[stmt.Name]; ok && arrayType == "assoc" {
					// 关联数组
					if e.assocArrays[stmt.Name] == nil {
						e.assocArrays[stmt.Name] = make(map[string]string)
					}
					// 展开索引中的变量
					key := e.expandVariablesInString(indexStr)
					value := e.evaluateExpression(valueExpr)
					e.assocArrays[stmt.Name][key] = value
				} else {
					// 创建关联数组
					if e.assocArrays[stmt.Name] == nil {
						e.assocArrays[stmt.Name] = make(map[string]string)
					}
					e.arrayTypes[stmt.Name] = "assoc"
					// 展开索引中的变量
					key := e.expandVariablesInString(indexStr)
					value := e.evaluateExpression(valueExpr)
					e.assocArrays[stmt.Name][key] = value
				}
				continue
			}
			
			// 数字索引
			if index > maxIndex {
				maxIndex = index
			}
			value := e.evaluateExpression(valueExpr)
			indexedMap[index] = value
		}
		
		// 如果是数字索引，创建普通数组
		if !hasStringKeys && maxIndex >= 0 {
			values := make([]string, maxIndex+1)
			for i, val := range indexedMap {
				values[i] = val
			}
			e.arrays[stmt.Name] = values
			e.arrayTypes[stmt.Name] = "array"
			// 设置环境变量
			if len(values) > 0 {
				e.env[stmt.Name] = values[0]
			}
			e.env[stmt.Name+"_LENGTH"] = fmt.Sprintf("%d", len(values))
		} else if hasStringKeys {
			// 有字符串键，已经处理为关联数组
			// 设置环境变量（关联数组的第一个值）
			if assocArr, ok := e.assocArrays[stmt.Name]; ok && len(assocArr) > 0 {
				// 获取第一个值（map 的顺序不确定，但至少设置一个值）
				for _, val := range assocArr {
					e.env[stmt.Name] = val
					break
				}
			}
		}
		return nil
	}
	
	// 普通数组赋值 arr=(1 2 3)
	values := make([]string, 0, len(stmt.Values))
	for _, expr := range stmt.Values {
		value := e.evaluateExpression(expr)
		values = append(values, value)
	}
	e.arrays[stmt.Name] = values
	e.arrayTypes[stmt.Name] = "array"
	// 同时设置环境变量，使用特殊格式存储数组长度
	e.env[stmt.Name+"_LENGTH"] = fmt.Sprintf("%d", len(values))
	// 设置第一个元素为默认值（Bash行为）
	if len(values) > 0 {
		e.env[stmt.Name] = values[0]
	}
	return nil
}

// executeCaseStatement 执行case语句
func (e *Executor) executeCaseStatement(stmt *parser.CaseStatement) error {
	// 求值case的值
	value := e.evaluateExpression(stmt.Value)
	
	// 遍历所有case子句
	for _, caseClause := range stmt.Cases {
		// 检查是否匹配
		matched := false
		for _, pattern := range caseClause.Patterns {
			// 对于完全匹配，直接比较字符串（移除空格）
			valueTrimmed := strings.TrimSpace(value)
			patternTrimmed := strings.TrimSpace(pattern)
			if valueTrimmed == patternTrimmed {
				matched = true
				break
			}
			// 如果直接匹配失败，尝试通配符匹配
			if matchPattern(valueTrimmed, patternTrimmed) {
				matched = true
				break
			}
		}
		
		if matched {
			// 执行匹配的case体
			return e.executeBlock(caseClause.Body)
		}
	}
	
	// 没有匹配的case，不执行任何操作
	return nil
}

// matchPattern 简单的模式匹配（支持 * 和 ? 通配符）
func matchPattern(value, pattern string) bool {
	// 如果模式是 *，匹配所有
	if pattern == "*" {
		return true
	}
	
	// 简单的通配符匹配
	patternIdx := 0
	valueIdx := 0
	
	for patternIdx < len(pattern) && valueIdx < len(value) {
		if pattern[patternIdx] == '*' {
			// * 匹配任意字符序列
			if patternIdx == len(pattern)-1 {
				return true // * 在末尾，匹配剩余所有
			}
			// 递归匹配
			for valueIdx <= len(value) {
				if matchPattern(value[valueIdx:], pattern[patternIdx+1:]) {
					return true
				}
				valueIdx++
			}
			return false
		} else if pattern[patternIdx] == '?' {
			// ? 匹配单个字符
			patternIdx++
			valueIdx++
		} else if pattern[patternIdx] == value[valueIdx] {
			patternIdx++
			valueIdx++
		} else {
			return false
		}
	}
	
	// 如果都匹配完了，返回true
	return patternIdx == len(pattern) && valueIdx == len(value)
}

// getArrayElement 获取数组元素
// 支持 ${arr[0]} 和 $arr[0] 格式（普通数组）
// 支持 ${arr[key]} 和 $arr[key] 格式（关联数组）
func (e *Executor) getArrayElement(varExpr string) string {
	// 解析数组名和索引
	// 格式：arr[0] 或 arr[key]
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
	
	// 检查是否是关联数组
	if arrayType, ok := e.arrayTypes[arrName]; ok && arrayType == "assoc" {
		// 关联数组：使用字符串键
		assocArr, ok := e.assocArrays[arrName]
		if !ok {
			if e.options["u"] {
				return "__UNDEFINED_VAR__" + arrName
			}
			return ""
		}
		// 展开键中的变量
		key := e.expandVariablesInString(indexStr)
		return assocArr[key]
	}
	
	// 普通数组：尝试解析为数字索引
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		// 如果不是数字，可能是关联数组但未声明类型，尝试作为字符串键
		if assocArr, ok := e.assocArrays[arrName]; ok {
			key := e.expandVariablesInString(indexStr)
			return assocArr[key]
		}
		return ""
	}
	
	// 获取普通数组
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

// expandArray 展开数组
// 如果 quoted 为 true，返回每个元素作为单独的词（用空格分隔）
// 如果 quoted 为 false，返回所有元素作为一个词（用 IFS 的第一个字符分隔）
func (e *Executor) expandArray(arrName string, quoted bool) string {
	// 检查是否是关联数组
	if arrayType, ok := e.arrayTypes[arrName]; ok && arrayType == "assoc" {
		assocArr, ok := e.assocArrays[arrName]
		if !ok {
			return ""
		}
		// 关联数组展开：返回所有值
		values := make([]string, 0, len(assocArr))
		for _, val := range assocArr {
			values = append(values, val)
		}
		if quoted {
			// ${arr[@]} - 每个元素作为单独的词
			return strings.Join(values, " ")
		}
		// ${arr[*]} - 所有元素作为一个词
		ifs := e.env["IFS"]
		if ifs == "" {
			ifs = " \t\n"
		}
		separator := ""
		if len(ifs) > 0 {
			separator = string(ifs[0])
		}
		if separator == "" {
			separator = " "
		}
		return strings.Join(values, separator)
	}
	
	// 普通数组
	arr, ok := e.arrays[arrName]
	if !ok {
		return ""
	}
	
	if quoted {
		// ${arr[@]} - 每个元素作为单独的词
		return strings.Join(arr, " ")
	}
	// ${arr[*]} - 所有元素作为一个词
	ifs := e.env["IFS"]
	if ifs == "" {
		ifs = " \t\n"
	}
	separator := ""
	if len(ifs) > 0 {
		separator = string(ifs[0])
	}
	if separator == "" {
		separator = " "
	}
	return strings.Join(arr, separator)
}

// executeAssocArrayAssignment 执行关联数组单个元素赋值
// 例如：arr[key]=value
func (e *Executor) executeAssocArrayAssignment(assignment string, args []parser.Expression) error {
	// 解析 arr[key]=value 格式
	eqIdx := strings.Index(assignment, "=")
	if eqIdx == -1 {
		return fmt.Errorf("无效的赋值语句: %s", assignment)
	}
	
	leftSide := assignment[:eqIdx]
	rightSide := assignment[eqIdx+1:]
	
	// 解析 arr[key]
	idx := strings.Index(leftSide, "[")
	if idx == -1 {
		return fmt.Errorf("无效的数组赋值: %s", assignment)
	}
	arrName := leftSide[:idx]
	idxEnd := strings.Index(leftSide, "]")
	if idxEnd == -1 {
		return fmt.Errorf("无效的数组赋值: %s", assignment)
	}
	keyStr := leftSide[idx+1 : idxEnd]
	
	// 获取值（如果有参数，使用第一个参数；否则使用rightSide）
	value := rightSide
	if len(args) > 0 {
		value = e.evaluateExpression(args[0])
	}
	
	// 检查是否是关联数组
	if arrayType, ok := e.arrayTypes[arrName]; ok && arrayType == "assoc" {
		// 确保关联数组已初始化
		if e.assocArrays[arrName] == nil {
			e.assocArrays[arrName] = make(map[string]string)
		}
		// 展开键中的变量
		key := e.expandVariablesInString(keyStr)
		e.assocArrays[arrName][key] = value
		return nil
	}
	
	// 如果不是关联数组，尝试作为普通数组处理（数字索引）
	index, err := strconv.Atoi(keyStr)
	if err == nil {
		// 数字索引，作为普通数组处理
		if e.arrays[arrName] == nil {
			e.arrays[arrName] = make([]string, 0)
		}
		// 扩展数组以容纳索引
		if index >= len(e.arrays[arrName]) {
			newArr := make([]string, index+1)
			copy(newArr, e.arrays[arrName])
			e.arrays[arrName] = newArr
		}
		e.arrays[arrName][index] = value
		e.arrayTypes[arrName] = "array"
		return nil
	}
	
	// 既不是关联数组也不是数字索引，创建关联数组
	if e.assocArrays[arrName] == nil {
		e.assocArrays[arrName] = make(map[string]string)
	}
	e.arrayTypes[arrName] = "assoc"
	key := e.expandVariablesInString(keyStr)
	e.assocArrays[arrName][key] = value
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
	case *parser.ParamExpandExpression:
		// 参数展开表达式 ${VAR...}
		result, err := e.expandParamExpression(ex)
		if err != nil {
			// 如果展开失败，返回错误信息（简化处理）
			return ""
		}
		return result
	case *parser.Variable:
		// 检查是否是数组访问 ${arr[0]} 或 $arr[0]
		if strings.Contains(ex.Name, "[") && strings.Contains(ex.Name, "]") {
			return e.getArrayElement(ex.Name)
		}
		// 检查是否是数组变量（返回所有元素，空格分隔）
		if arr, ok := e.arrays[ex.Name]; ok {
			return strings.Join(arr, " ")
		}
		// 检查是否是特殊变量 $#, $@, $*, $?, $!, $$, $0
		if ex.Name == "#" {
			if value, ok := e.env["#"]; ok {
				return value
			}
			return "0"
		}
		if ex.Name == "@" {
			if value, ok := e.env["@"]; ok {
				return value
			}
			return ""
		}
		if ex.Name == "*" {
			if value, ok := e.env["@"]; ok {
				return value
			}
			return ""
		}
		if ex.Name == "?" {
			if value, ok := e.env["?"]; ok {
				return value
			}
			return "0"
		}
		if ex.Name == "!" {
			if value, ok := e.env["!"]; ok {
				return value
			}
			return "0"
		}
		if ex.Name == "$" {
			// $$ 当前进程的PID
			return fmt.Sprintf("%d", os.Getpid())
		}
		if ex.Name == "0" {
			if value, ok := e.env["0"]; ok {
				return value
			}
			return os.Args[0]
		}
		// 检查是否是位置参数 $1, $2, ...
		if len(ex.Name) > 0 {
			isAllDigits := true
			for _, ch := range ex.Name {
				if ch < '0' || ch > '9' {
					isAllDigits = false
					break
				}
			}
			if isAllDigits {
				if value, ok := e.env[ex.Name]; ok {
					return value
				}
				// 如果设置了 -u 选项，未定义的位置参数应该报错
				if e.options["u"] {
					return "__UNDEFINED_VAR__" + ex.Name
				}
				return ""
			}
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
	case *parser.ProcessSubstitution:
		// 执行进程替换
		return e.executeProcessSubstitution(ex.Command, ex.IsInput)
	default:
		return ""
	}
}

// expandVariablesInString 展开字符串中的变量（如 "TEST=$TEST"）
// 优先处理转义序列（如 \$），然后处理变量展开
func (e *Executor) expandVariablesInString(s string) string {
	// 如果字符串为空，直接返回
	if len(s) == 0 {
		return ""
	}
	
	var result strings.Builder
	i := 0
	for i < len(s) {
		// 处理转义序列（除了 \$，\$ 留给变量展开处理）
		if s[i] == '\\' && i+1 < len(s) {
			escaped := s[i+1]
			
			if escaped == '$' {
				// \$ 转义：保留 \，然后继续处理 $（会在下面的 $ 处理中检查前面是否有 \）
				result.WriteByte('\\')
				i++ // 只跳过 \，不跳过 $，让 $ 进入下面的处理
			} else if escaped == '"' {
				// \" 转义：只输出 "，不输出 \
				result.WriteByte('"')
				i += 2 // 跳过 \ 和 "
			} else {
				i += 2 // 跳过 \ 和转义字符
				switch escaped {
				case '\\':
					// \\ 转义：输出单个 \
					result.WriteByte('\\')
				default:
					// 其他转义序列（\n, \t等）保持字面量（两个字符）
					result.WriteByte('\\')
					result.WriteByte(escaped)
				}
			}
		} else if s[i] == '$' && i+1 < len(s) {
			// 检查是否是算术展开 $((...))
			if i+2 < len(s) && s[i+1] == '(' && s[i+2] == '(' {
				// 找到匹配的 ))
				i += 3 // 跳过 $(( 
				startPos := i
				depth := 1
				for i < len(s) && depth > 0 {
					if i+1 < len(s) && s[i] == ')' && s[i+1] == ')' {
						depth--
						if depth == 0 {
							// 提取算术表达式
							expr := s[startPos:i]
							// 计算算术表达式
							result.WriteString(e.evaluateArithmetic(expr))
							i += 2 // 跳过 ))
							continue
						} else {
							i += 2
						}
					} else if i+1 < len(s) && s[i] == '(' && s[i+1] == '(' {
						depth++
						i += 2
					} else {
						i++
					}
				}
				if depth > 0 {
					// 括号不匹配，保留原样
					result.WriteString("$((")
					i = startPos
				}
				continue
			}
			
			// 检查 result 中最后一个字符是否是转义的 \
			if result.Len() > 0 {
				resultStr := result.String()
				lastChar := resultStr[len(resultStr)-1]
				if lastChar == '\\' {
					// 前面有 \，这是转义的 $，删除 result 中的 \，输出字面量 $ 并跳过后面的变量名（不展开）
					resultWithoutBackslash := resultStr[:len(resultStr)-1]
					result.Reset()
					result.WriteString(resultWithoutBackslash)
					result.WriteByte('$')
					i++ // 跳过 $，现在 i 指向 $ 后面的字符（比如 '1'）
					// 跳过后面的变量名部分，但不展开
					if i < len(s) {
						// 处理特殊变量 $#, $@, $*, $?, $!, $$, $0
						if s[i] == '#' || s[i] == '@' || s[i] == '*' || s[i] == '?' || s[i] == '!' || s[i] == '$' || s[i] == '0' {
							result.WriteByte(s[i])
							i++
						} else if isDigit(s[i]) {
							// $1, $2, ... 位置参数
							for i < len(s) && isDigit(s[i]) {
								result.WriteByte(s[i])
								i++
							}
						} else if s[i] == '{' {
							// ${VAR} 格式
							result.WriteByte(s[i])
							i++
							for i < len(s) && s[i] != '}' {
								result.WriteByte(s[i])
								i++
							}
							if i < len(s) && s[i] == '}' {
								result.WriteByte(s[i])
								i++
							}
						} else if isLetter(s[i]) || s[i] == '_' {
							// $VAR 格式
							for i < len(s) && (isLetter(s[i]) || isDigit(s[i]) || s[i] == '_' || s[i] == '[' || s[i] == ']') {
								result.WriteByte(s[i])
								i++
							}
						}
					}
					continue
				}
			}
			
			// 前面没有 \，正常展开变量
			// 处理变量展开
			var varName strings.Builder
			
			// 处理特殊变量 $#, $@, $*, $?, $!, $$, $0, $1, $2, ...
			if i+1 < len(s) && s[i+1] == '#' {
				// $# 参数个数
				i += 2
				if value, ok := e.env["#"]; ok {
					result.WriteString(value)
				} else {
					result.WriteString("0")
				}
				continue
			} else if i+1 < len(s) && s[i+1] == '@' {
				// $@ 所有参数
				i += 2
				if value, ok := e.env["@"]; ok {
					result.WriteString(value)
				}
				continue
			} else if i+1 < len(s) && s[i+1] == '*' {
				// $* 所有参数（与$@类似）
				i += 2
				if value, ok := e.env["@"]; ok {
					result.WriteString(value)
				}
				continue
			} else if i+1 < len(s) && s[i+1] == '?' {
				// $? 上一个命令的退出码
				i += 2
				if value, ok := e.env["?"]; ok {
					result.WriteString(value)
				} else {
					result.WriteString("0")
				}
				continue
			} else if i+1 < len(s) && s[i+1] == '!' {
				// $! 最后一个后台进程的PID
				i += 2
				if value, ok := e.env["!"]; ok {
					result.WriteString(value)
				} else {
					result.WriteString("0")
				}
				continue
			} else if i+1 < len(s) && s[i+1] == '$' {
				// $$ 当前进程的PID
				i += 2
				result.WriteString(fmt.Sprintf("%d", os.Getpid()))
				continue
			} else if i+1 < len(s) && s[i+1] == '0' {
				// $0 脚本名
				i += 2
				if value, ok := e.env["0"]; ok {
					result.WriteString(value)
				} else {
					result.WriteString(os.Args[0])
				}
				continue
			} else if i+1 < len(s) && isDigit(s[i+1]) {
				// $1, $2, ... 位置参数
				i++
				for i < len(s) && isDigit(s[i]) {
					varName.WriteByte(s[i])
					i++
				}
				varNameStr := varName.String()
				if value, ok := e.env[varNameStr]; ok {
					result.WriteString(value)
				} else if e.options["u"] {
					result.WriteString("__UNDEFINED_VAR__" + varNameStr)
				}
				continue
			}
			
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

// GetEnvMap 获取环境变量映射（用于builtin命令）
func (e *Executor) GetEnvMap() map[string]string {
	return e.env
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
// 正确处理嵌套的命令替换、转义和退出码
func (e *Executor) executeCommandSubstitution(command string) string {
	// 先展开命令字符串中的变量和嵌套的命令替换
	// 注意：命令替换中的命令本身不应该进行单词分割和路径名展开
	expandedCommand := e.expandCommandSubstitutionCommand(command)
	
	// 解析和执行命令
	l := lexer.New(expandedCommand)
	p := parser.New(l)
	program := p.ParseProgram()
	
	if len(p.Errors()) > 0 {
		// 解析错误，返回空字符串
		return ""
	}
	
	// 保存当前的标准输出
	oldStdout := os.Stdout
	
	// 创建管道捕获输出
	r, w, err := os.Pipe()
	if err != nil {
		return ""
	}
	os.Stdout = w
	
	// 在goroutine中读取输出，避免阻塞
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
	
	// 执行命令（在子shell环境中）
	// 注意：命令替换在子shell中执行，不应该影响当前shell的状态
	execErr := e.Execute(program)
	
	// 恢复标准输出
	w.Close()
	os.Stdout = oldStdout
	
	// 等待读取完成
	<-done
	
	// 恢复退出码（命令替换不应该改变当前shell的退出码，除非命令替换本身失败）
	// 但我们需要保存命令替换的退出码，以便在需要时使用
	// 这里简化处理，不恢复退出码，因为命令替换的退出码通常不影响当前shell
	
	// 处理执行错误
	if execErr != nil {
		// 执行错误，返回空字符串
		return ""
	}
	
	// 返回输出（移除末尾的换行符，如果存在）
	result := output.String()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	
	return result
}

// expandCommandSubstitutionCommand 展开命令替换中的命令字符串
// 处理变量展开和嵌套的命令替换，但不进行单词分割和路径名展开
func (e *Executor) expandCommandSubstitutionCommand(command string) string {
	// 展开变量和嵌套的命令替换
	result := e.expandVariablesInString(command)
	return result
}

// getExitCode 获取当前退出码
func (e *Executor) getExitCode() int {
	if exitCode, ok := e.env["?"]; ok {
		if code, err := strconv.Atoi(exitCode); err == nil {
			return code
		}
	}
	return 0
}

// executeProcessSubstitution 执行进程替换
// IsInput: true表示<(command)，false表示>(command)
func (e *Executor) executeProcessSubstitution(command string, isInput bool) string {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "gobash_process_subst_*")
	if err != nil {
		return ""
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	
	// 解析和执行命令
	l := lexer.New(command)
	p := parser.New(l)
	program := p.ParseProgram()
	
	if len(p.Errors()) > 0 {
		os.Remove(tmpPath)
		return ""
	}
	
	if isInput {
		// <(command): 执行命令并将输出写入临时文件
		oldStdout := os.Stdout
		file, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			os.Remove(tmpPath)
			return ""
		}
		os.Stdout = file
		
		execErr := e.Execute(program)
		
		file.Close()
		os.Stdout = oldStdout
		
		if execErr != nil {
			os.Remove(tmpPath)
			return ""
		}
	} else {
		// >(command): 创建临时文件供命令读取
		// 注意：>(command)通常用于将输出重定向到命令的输入
		// 这里简化实现，创建空文件
		file, err := os.Create(tmpPath)
		if err != nil {
			os.Remove(tmpPath)
			return ""
		}
		file.Close()
	}
	
	// 返回临时文件路径
	return tmpPath
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

// evaluateArithmeticExpression 计算算术表达式
// 支持运算符: +, -, *, /, %, ** (幂), << (左移), >> (右移), & (按位与), | (按位或), ^ (按位异或), ~ (按位非)
// 支持比较运算符: <, <=, >, >=, ==, !=
// 支持逻辑运算符: &&, ||, ! (逻辑非)
// 支持括号和函数调用
func evaluateArithmeticExpression(expr string) (int64, error) {
	// 移除所有空白字符
	expr = strings.ReplaceAll(expr, " ", "")
	expr = strings.ReplaceAll(expr, "\t", "")
	
	if expr == "" {
		return 0, nil
	}
	
	// 使用递归下降解析器
	pos := 0
	result, err := parseArithmeticExpression(expr, &pos)
	if err != nil {
		return 0, err
	}
	
	// 确保解析完整个表达式
	if pos < len(expr) {
		return 0, fmt.Errorf("unexpected character at position %d: %c", pos, expr[pos])
	}
	
	return result, nil
}

// parseArithmeticExpression 解析算术表达式（处理逻辑或 ||）
func parseArithmeticExpression(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticAndExpression(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if *pos+1 < len(expr) && expr[*pos] == '|' && expr[*pos+1] == '|' {
			*pos += 2
			right, err := parseArithmeticAndExpression(expr, pos)
			if err != nil {
				return 0, err
			}
			// 逻辑或：如果左边非零，返回左边，否则返回右边
			if result != 0 {
				result = 1
			} else if right != 0 {
				result = 1
			} else {
				result = 0
			}
		} else {
			break
		}
	}
	
	return result, nil
}

// parseArithmeticAndExpression 解析逻辑与表达式（处理 &&）
func parseArithmeticAndExpression(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticComparison(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if *pos+1 < len(expr) && expr[*pos] == '&' && expr[*pos+1] == '&' {
			*pos += 2
			right, err := parseArithmeticComparison(expr, pos)
			if err != nil {
				return 0, err
			}
			// 逻辑与：如果两边都非零，返回1，否则返回0
			if result != 0 && right != 0 {
				result = 1
			} else {
				result = 0
			}
		} else {
			break
		}
	}
	
	return result, nil
}

// parseArithmeticComparison 解析比较表达式（处理 <, <=, >, >=, ==, !=）
func parseArithmeticComparison(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticBitwiseOr(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if *pos+1 < len(expr) && expr[*pos] == '<' && expr[*pos+1] == '=' {
			*pos += 2
			right, err := parseArithmeticBitwiseOr(expr, pos)
			if err != nil {
				return 0, err
			}
			if result <= right {
				result = 1
			} else {
				result = 0
			}
		} else if *pos+1 < len(expr) && expr[*pos] == '>' && expr[*pos+1] == '=' {
			*pos += 2
			right, err := parseArithmeticBitwiseOr(expr, pos)
			if err != nil {
				return 0, err
			}
			if result >= right {
				result = 1
			} else {
				result = 0
			}
		} else if *pos+1 < len(expr) && expr[*pos] == '=' && expr[*pos+1] == '=' {
			*pos += 2
			right, err := parseArithmeticBitwiseOr(expr, pos)
			if err != nil {
				return 0, err
			}
			if result == right {
				result = 1
			} else {
				result = 0
			}
		} else if *pos+1 < len(expr) && expr[*pos] == '!' && expr[*pos+1] == '=' {
			*pos += 2
			right, err := parseArithmeticBitwiseOr(expr, pos)
			if err != nil {
				return 0, err
			}
			if result != right {
				result = 1
			} else {
				result = 0
			}
		} else if expr[*pos] == '<' {
			*pos++
			right, err := parseArithmeticBitwiseOr(expr, pos)
			if err != nil {
				return 0, err
			}
			if result < right {
				result = 1
			} else {
				result = 0
			}
		} else if expr[*pos] == '>' {
			*pos++
			right, err := parseArithmeticBitwiseOr(expr, pos)
			if err != nil {
				return 0, err
			}
			if result > right {
				result = 1
			} else {
				result = 0
			}
		} else {
			break
		}
	}
	
	return result, nil
}

// parseArithmeticBitwiseOr 解析按位或表达式（处理 |）
func parseArithmeticBitwiseOr(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticBitwiseXor(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if expr[*pos] == '|' && (*pos+1 >= len(expr) || expr[*pos+1] != '|') {
			*pos++
			right, err := parseArithmeticBitwiseXor(expr, pos)
			if err != nil {
				return 0, err
			}
			result |= right
		} else {
			break
		}
	}
	
	return result, nil
}

// parseArithmeticBitwiseXor 解析按位异或表达式（处理 ^）
func parseArithmeticBitwiseXor(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticBitwiseAnd(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if expr[*pos] == '^' {
			*pos++
			right, err := parseArithmeticBitwiseAnd(expr, pos)
			if err != nil {
				return 0, err
			}
			result ^= right
		} else {
			break
		}
	}
	
	return result, nil
}

// parseArithmeticBitwiseAnd 解析按位与表达式（处理 &）
func parseArithmeticBitwiseAnd(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticShift(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if expr[*pos] == '&' && (*pos+1 >= len(expr) || expr[*pos+1] != '&') {
			*pos++
			right, err := parseArithmeticShift(expr, pos)
			if err != nil {
				return 0, err
			}
			result &= right
		} else {
			break
		}
	}
	
	return result, nil
}

// parseArithmeticShift 解析位移表达式（处理 <<, >>）
func parseArithmeticShift(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticAddSub(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if *pos+1 < len(expr) && expr[*pos] == '<' && expr[*pos+1] == '<' {
			*pos += 2
			right, err := parseArithmeticAddSub(expr, pos)
			if err != nil {
				return 0, err
			}
			result <<= right
		} else if *pos+1 < len(expr) && expr[*pos] == '>' && expr[*pos+1] == '>' {
			*pos += 2
			right, err := parseArithmeticAddSub(expr, pos)
			if err != nil {
				return 0, err
			}
			result >>= right
		} else {
			break
		}
	}
	
	return result, nil
}

// parseArithmeticAddSub 解析加减表达式（处理 +, -）
func parseArithmeticAddSub(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticMulDivMod(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if expr[*pos] == '+' {
			*pos++
			term, err := parseArithmeticMulDivMod(expr, pos)
			if err != nil {
				return 0, err
			}
			result += term
		} else if expr[*pos] == '-' {
			*pos++
			term, err := parseArithmeticMulDivMod(expr, pos)
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

// parseArithmeticMulDivMod 解析乘除模表达式（处理 *, /, %）
func parseArithmeticMulDivMod(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticPower(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if expr[*pos] == '*' && (*pos+1 >= len(expr) || expr[*pos+1] != '*') {
			*pos++
			factor, err := parseArithmeticPower(expr, pos)
			if err != nil {
				return 0, err
			}
			result *= factor
		} else if expr[*pos] == '/' {
			*pos++
			factor, err := parseArithmeticPower(expr, pos)
			if err != nil {
				return 0, err
			}
			if factor == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			result /= factor
		} else if expr[*pos] == '%' {
			*pos++
			factor, err := parseArithmeticPower(expr, pos)
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

// parseArithmeticPower 解析幂表达式（处理 **）
func parseArithmeticPower(expr string, pos *int) (int64, error) {
	result, err := parseArithmeticUnary(expr, pos)
	if err != nil {
		return 0, err
	}
	
	for *pos < len(expr) {
		if *pos+1 < len(expr) && expr[*pos] == '*' && expr[*pos+1] == '*' {
			*pos += 2
			exponent, err := parseArithmeticUnary(expr, pos)
			if err != nil {
				return 0, err
			}
			// 计算幂
			power := int64(1)
			for i := int64(0); i < exponent; i++ {
				power *= result
			}
			result = power
		} else {
			break
		}
	}
	
	return result, nil
}

// parseArithmeticUnary 解析一元表达式（处理 +, -, ~, !）
func parseArithmeticUnary(expr string, pos *int) (int64, error) {
	if *pos >= len(expr) {
		return 0, fmt.Errorf("unexpected end of expression")
	}
	
	// 处理一元运算符
	if expr[*pos] == '+' {
		*pos++
		return parseArithmeticUnary(expr, pos)
	} else if expr[*pos] == '-' {
		*pos++
		result, err := parseArithmeticUnary(expr, pos)
		if err != nil {
			return 0, err
		}
		return -result, nil
	} else if expr[*pos] == '~' {
		*pos++
		result, err := parseArithmeticUnary(expr, pos)
		if err != nil {
			return 0, err
		}
		return ^result, nil
	} else if expr[*pos] == '!' {
		*pos++
		result, err := parseArithmeticUnary(expr, pos)
		if err != nil {
			return 0, err
		}
		// 逻辑非：如果非零，返回0，否则返回1
		if result != 0 {
			return 0, nil
		}
		return 1, nil
	}
	
	return parseArithmeticFactor(expr, pos)
}

// parseArithmeticFactor 解析因子（处理数字、括号、函数调用）
func parseArithmeticFactor(expr string, pos *int) (int64, error) {
	if *pos >= len(expr) {
		return 0, fmt.Errorf("unexpected end of expression")
	}
	
	// 处理括号
	if expr[*pos] == '(' {
		*pos++
		result, err := parseArithmeticExpression(expr, pos)
		if err != nil {
			return 0, err
		}
		if *pos >= len(expr) || expr[*pos] != ')' {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		*pos++
		return result, nil
	}
	
	// 处理函数调用（必须在解析数字之前）
	// 检查是否是函数调用，如 abs(, min(, max( 等
	funcName := ""
	funcStart := *pos
	
	// 先尝试读取函数名（字母、数字、下划线）
	for *pos < len(expr) {
		ch := expr[*pos]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
			funcName += string(ch)
			*pos++
		} else if ch >= '0' && ch <= '9' {
			// 数字可以作为函数名的一部分，但如果是第一个字符，则不是函数名
			if funcName == "" {
				break
			}
			funcName += string(ch)
			*pos++
		} else if ch == '(' {
			// 找到函数名和开括号，这是一个函数调用
			if funcName != "" {
				*pos++ // 跳过 (
				args, err := parseArithmeticFunctionArgs(expr, pos)
				if err != nil {
					return 0, err
				}
				// 调用算术函数
				result, err := evaluateArithmeticFunction(funcName, args)
				if err != nil {
					return 0, fmt.Errorf("arithmetic function %s: %v", funcName, err)
				}
				return result, nil
			}
			// 如果没有函数名，这是括号表达式，不是函数调用
			break
		} else {
			// 不是函数调用，恢复位置
			*pos = funcStart
			break
		}
	}
	
	// 如果不是函数调用，恢复位置
	if funcName != "" {
		*pos = funcStart
	}
	
	// 解析数字
	start := *pos
	if expr[*pos] == '-' || expr[*pos] == '+' {
		*pos++
	}
	
	if *pos >= len(expr) || !isDigitArith(expr[*pos]) {
		return 0, fmt.Errorf("expected number at position %d: %c", *pos, expr[*pos])
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

// parseArithmeticFunctionArgs 解析算术函数参数
func parseArithmeticFunctionArgs(expr string, pos *int) ([]int64, error) {
	var args []int64
	
	// 跳过空白字符
	for *pos < len(expr) && (expr[*pos] == ' ' || expr[*pos] == '\t') {
		*pos++
	}
	
	// 如果下一个字符是 )，没有参数
	if *pos < len(expr) && expr[*pos] == ')' {
		*pos++ // 跳过 )
		return args, nil
	}
	
	// 解析参数列表
	for {
		// 解析一个参数（算术表达式）
		arg, err := parseArithmeticExpression(expr, pos)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		
		// 跳过空白字符
		for *pos < len(expr) && (expr[*pos] == ' ' || expr[*pos] == '\t') {
			*pos++
		}
		
		// 检查是否有更多参数
		if *pos >= len(expr) {
			return nil, fmt.Errorf("missing closing parenthesis in function call")
		}
		
		if expr[*pos] == ')' {
			*pos++ // 跳过 )
			break
		} else if expr[*pos] == ',' {
			*pos++ // 跳过 ,
			// 继续解析下一个参数
		} else {
			return nil, fmt.Errorf("unexpected character in function arguments: %c", expr[*pos])
		}
	}
	
	return args, nil
}

// evaluateArithmeticFunction 计算算术函数
func evaluateArithmeticFunction(name string, args []int64) (int64, error) {
	switch name {
	case "abs":
		if len(args) != 1 {
			return 0, fmt.Errorf("abs requires 1 argument, got %d", len(args))
		}
		if args[0] < 0 {
			return -args[0], nil
		}
		return args[0], nil
		
	case "min":
		if len(args) < 1 {
			return 0, fmt.Errorf("min requires at least 1 argument, got %d", len(args))
		}
		min := args[0]
		for i := 1; i < len(args); i++ {
			if args[i] < min {
				min = args[i]
			}
		}
		return min, nil
		
	case "max":
		if len(args) < 1 {
			return 0, fmt.Errorf("max requires at least 1 argument, got %d", len(args))
		}
		max := args[0]
		for i := 1; i < len(args); i++ {
			if args[i] > max {
				max = args[i]
			}
		}
		return max, nil
		
	case "length":
		// length 函数需要字符串参数，但算术表达式中只能处理数字
		// 这里简化处理，将数字转换为字符串再计算长度
		if len(args) != 1 {
			return 0, fmt.Errorf("length requires 1 argument, got %d", len(args))
		}
		str := fmt.Sprintf("%d", args[0])
		if args[0] < 0 {
			// 负数，去掉负号
			str = str[1:]
		}
		return int64(len(str)), nil
		
	case "int":
		// int 函数向下取整（对于整数，直接返回）
		if len(args) != 1 {
			return 0, fmt.Errorf("int requires 1 argument, got %d", len(args))
		}
		return args[0], nil
		
	case "rand":
		// rand 函数返回 0 到 32767 之间的随机数
		if len(args) > 0 {
			return 0, fmt.Errorf("rand takes no arguments, got %d", len(args))
		}
		// 使用简单的线性同余生成器
		// 注意：这不是线程安全的，但对于单线程 shell 足够了
		return int64(rand.Intn(32768)), nil
		
	case "srand":
		// srand 函数设置随机数种子
		if len(args) > 1 {
			return 0, fmt.Errorf("srand requires 0 or 1 argument, got %d", len(args))
		}
		if len(args) == 1 {
			rand.Seed(args[0])
		} else {
			rand.Seed(time.Now().UnixNano())
		}
		return 0, nil
		
	default:
		return 0, fmt.Errorf("unknown arithmetic function: %s", name)
	}
}

// evaluateDoubleBracketExpression 计算 [[ 表达式（支持 && 和 ||）
func (e *Executor) evaluateDoubleBracketExpression(args []string) (bool, error) {
	if len(args) == 0 {
		return false, fmt.Errorf("[[: 缺少参数")
	}
	
	// 移除结束括号 ]]
	if len(args) > 0 && args[len(args)-1] == "]]" {
		args = args[:len(args)-1]
	}
	
	// 使用递归下降解析器处理 && 和 ||
	return e.parseDoubleBracketExpression(args, 0)
}

// parseDoubleBracketExpression 解析 [[ 表达式（处理 || 运算符，优先级最低）
func (e *Executor) parseDoubleBracketExpression(args []string, pos int) (bool, error) {
	left, newPos, err := e.parseDoubleBracketAndExpression(args, pos)
	if err != nil {
		return false, err
	}
	pos = newPos
	
	for pos < len(args) && args[pos] == "||" {
		pos++ // 跳过 ||
		right, newPos, err := e.parseDoubleBracketAndExpression(args, pos)
		if err != nil {
			return false, err
		}
		pos = newPos
		left = left || right
	}
	
	return left, nil
}

// parseDoubleBracketAndExpression 解析 && 表达式（优先级高于 ||）
func (e *Executor) parseDoubleBracketAndExpression(args []string, pos int) (bool, int, error) {
	left, newPos, err := e.parseDoubleBracketPrimaryExpression(args, pos)
	if err != nil {
		return false, pos, err
	}
	pos = newPos
	
	for pos < len(args) && args[pos] == "&&" {
		pos++ // 跳过 &&
		right, newPos, err := e.parseDoubleBracketPrimaryExpression(args, pos)
		if err != nil {
			return false, pos, err
		}
		pos = newPos
		left = left && right
	}
	
	return left, pos, nil
}

// parseDoubleBracketPrimaryExpression 解析基本表达式（单个测试或括号表达式）
func (e *Executor) parseDoubleBracketPrimaryExpression(args []string, pos int) (bool, int, error) {
	if pos >= len(args) {
		return false, pos, fmt.Errorf("[[: 表达式不完整")
	}
	
	// 处理括号表达式
	if args[pos] == "(" {
		pos++
		// 找到匹配的右括号
		depth := 1
		endPos := pos
		for endPos < len(args) && depth > 0 {
			if args[endPos] == "(" {
				depth++
			} else if args[endPos] == ")" {
				depth--
			}
			if depth > 0 {
				endPos++
			}
		}
		if depth != 0 {
			return false, pos, fmt.Errorf("[[: 括号不匹配")
		}
		
		// 递归解析括号内的表达式
		result, err := e.parseDoubleBracketExpression(args[pos:endPos], 0)
		if err != nil {
			return false, pos, err
		}
		return result, endPos + 1, nil
	}
	
	// 处理 ! 运算符
	if args[pos] == "!" {
		pos++
		result, newPos, err := e.parseDoubleBracketPrimaryExpression(args, pos)
		if err != nil {
			return false, pos, err
		}
		return !result, newPos, nil
	}
	
	// 处理单个测试表达式
	// 找到测试表达式的结束位置（遇到 &&, ||, ), 或到达末尾）
	endPos := pos
	for endPos < len(args) {
		if args[endPos] == "&&" || args[endPos] == "||" || args[endPos] == ")" {
			break
		}
		endPos++
	}
	
	// 提取测试表达式
	testArgs := args[pos:endPos]
	if len(testArgs) == 0 {
		return false, pos, fmt.Errorf("[[: 空表达式")
	}
	
	// 调用 test 命令来求值
	testFunc := e.builtins["test"]
	if testFunc == nil {
		return false, pos, fmt.Errorf("test命令未找到")
	}
	
	// 临时修改环境变量，调用 test 命令
	// 注意：test 命令返回 error 表示失败，nil 表示成功
	err := testFunc(testArgs, e.env)
	result := err == nil
	
	return result, endPos, nil
}

// readHereDocument 读取 Here-document 内容
// 从标准输入读取，直到找到分隔符
func (e *Executor) readHereDocument(delimiter string, quoted bool, stripTabs bool) string {
	var content strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// 检查是否是分隔符
		if strings.TrimSpace(line) == delimiter {
			break
		}
		
		// 如果 stripTabs 为 true，剥离前导制表符
		if stripTabs {
			line = strings.TrimLeft(line, "\t")
		}
		
		// 如果 quoted 为 false，展开变量
		if !quoted {
			line = e.expandVariablesInString(line)
		}
		
		content.WriteString(line)
		content.WriteString("\n")
	}
	
	return content.String()
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

