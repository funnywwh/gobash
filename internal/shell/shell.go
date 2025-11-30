package shell

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"gobash/internal/executor"
	"gobash/internal/lexer"
	"gobash/internal/parser"
	"github.com/chzyer/readline"
)

// Shell Shell主结构
type Shell struct {
	executor *executor.Executor
	prompt   string
	running  bool
	aliases  map[string]string
	history  *History
	options  map[string]bool // shell选项状态
}

// New 创建新的Shell实例
func New() *Shell {
	history := NewHistory(1000)
	
	// 尝试加载历史记录
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home != "" {
		historyFile := filepath.Join(home, ".gobash_history")
		history.LoadFromFile(historyFile)
	}

	sh := &Shell{
		executor: executor.New(),
		prompt:   getPrompt(),
		running:  true,
		aliases:  make(map[string]string),
		history:  history,
		options:  make(map[string]bool),
	}
	
	// 将选项状态传递给执行器
	sh.executor.SetOptions(sh.options)
	
	return sh
}

// Run 运行交互式Shell
func (s *Shell) Run() {
	// 配置readline
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	
	historyFile := ""
	if home != "" {
		historyFile = filepath.Join(home, ".gobash_history")
	}
	
	// 创建readline配置
	config := &readline.Config{
		Prompt:          s.prompt,
		HistoryFile:     historyFile,
		HistoryLimit:    1000,
		AutoComplete:    nil, // 暂时不实现自动补全
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	}
	
	rl, err := readline.NewEx(config)
	if err != nil {
		// 如果readline初始化失败，回退到简单的bufio.Scanner
		s.runSimple()
		return
	}
	defer rl.Close()

	// readline会自动从HistoryFile加载历史记录，无需手动添加

	for s.running {
		// 更新提示符
		rl.SetPrompt(s.prompt)
		
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				// Ctrl+C，继续
				fmt.Println()
				continue
			}
			// EOF或其他错误，退出
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 处理多行输入（以\结尾）
		for strings.HasSuffix(line, "\\") {
			line = strings.TrimSuffix(line, "\\")
			rl.SetPrompt("> ")
			nextLine, err := rl.Readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					fmt.Println()
					break
				}
				break
			}
			line += " " + strings.TrimSpace(nextLine)
		}

		if err := s.executeLine(line); err != nil {
			fmt.Fprintf(os.Stderr, "gobash: %v\n", err)
		} else {
			// 成功执行的命令添加到历史记录
			s.history.Add(line)
			// 保存到readline历史记录
			rl.SaveHistory(line)
		}
		
		// 更新提示符（工作目录可能已改变）
		s.prompt = getPrompt()
	}
	
	// 保存历史记录
	s.saveHistory()
}

// runSimple 简单的运行模式（当readline不可用时回退）
func (s *Shell) runSimple() {
	scanner := bufio.NewScanner(os.Stdin)

	for s.running {
		fmt.Print(s.prompt)

		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 处理多行输入（以\结尾）
		for strings.HasSuffix(strings.TrimSpace(line), "\\") {
			line = strings.TrimSuffix(strings.TrimSpace(line), "\\")
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}
			line += " " + scanner.Text()
		}

		if err := s.executeLine(line); err != nil {
			fmt.Fprintf(os.Stderr, "gobash: %v\n", err)
		} else {
			// 成功执行的命令添加到历史记录
			s.history.Add(line)
		}
		
		// 更新提示符（工作目录可能已改变）
		s.prompt = getPrompt()
	}
	
	// 保存历史记录
	s.saveHistory()
}

// saveHistory 保存历史记录
func (s *Shell) saveHistory() {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home != "" {
		historyFile := filepath.Join(home, ".gobash_history")
		s.history.SaveToFile(historyFile)
	}
}

// ExecuteScript 执行脚本文件
func (s *Shell) ExecuteScript(scriptPath string) error {
	file, err := os.Open(scriptPath)
	if err != nil {
		return fmt.Errorf("无法打开脚本文件: %v", err)
	}
	defer file.Close()

	return s.ExecuteReader(file)
}

// ExecuteReader 从Reader执行命令
func (s *Shell) ExecuteReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	firstLine := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// 跳过空行
		if line == "" {
			continue
		}
		
		// 跳过shebang行（#!/bin/bash, #!/usr/bin/env bash等）
		if firstLine && strings.HasPrefix(line, "#!") {
			firstLine = false
			continue
		}
		firstLine = false
		
		// 跳过注释行（以#开头，但不是shebang）
		if strings.HasPrefix(line, "#") {
			continue
		}
		
		// 执行每一行
		if err := s.executeLine(scanner.Text()); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// executeLine 执行一行命令
func (s *Shell) executeLine(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	// 分割多个命令（分号分隔）
	commands := splitCommands(line)
	for _, cmd := range commands {
		if err := s.executeCommand(cmd); err != nil {
			return err
		}
	}

	return nil
}

// executeCommand 执行单个命令
func (s *Shell) executeCommand(input string) error {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	// 检查是否为特殊命令，需要特殊处理
	parts := strings.Fields(input)
	if len(parts) > 0 {
		cmd := parts[0]
		if cmd == "alias" {
			return s.handleAliasCommand(parts[1:])
		} else if cmd == "unalias" {
			return s.handleUnaliasCommand(parts[1:])
		} else if cmd == "history" {
			return s.handleHistoryCommand(parts[1:])
		} else if cmd == "set" {
			return s.handleSetCommand(parts[1:])
		}
	}

	// 展开别名
	input = s.expandAlias(input)

	// 词法分析
	l := lexer.New(input)
	
	// 语法分析
	p := parser.New(l)
	program := p.ParseProgram()

	// 检查解析错误
	if len(p.Errors()) > 0 {
		return fmt.Errorf("语法错误: %v", p.Errors())
	}

	// 执行
	if err := s.executor.Execute(program); err != nil {
		return fmt.Errorf("执行错误: %v", err)
	}

	return nil
}

// expandAlias 展开别名
func (s *Shell) expandAlias(input string) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return input
	}

	cmdName := parts[0]
	if alias, ok := s.aliases[cmdName]; ok {
		// 替换命令名，保留参数
		if len(parts) > 1 {
			return alias + " " + strings.Join(parts[1:], " ")
		}
		return alias
	}

	return input
}

// handleAliasCommand 处理alias命令
func (s *Shell) handleAliasCommand(args []string) error {
	if len(args) == 0 {
		// 显示所有别名
		for name, value := range s.aliases {
			fmt.Printf("alias %s='%s'\n", name, value)
		}
		return nil
	}

	// 设置别名
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			name := parts[0]
			value := strings.Trim(parts[1], "\"'")
			s.aliases[name] = value
		} else {
			// 显示特定别名
			if value, ok := s.aliases[arg]; ok {
				fmt.Printf("alias %s='%s'\n", arg, value)
			}
		}
	}

	return nil
}

// handleSetCommand 处理set命令
func (s *Shell) handleSetCommand(args []string) error {
	if len(args) == 0 {
		// 显示所有变量
		env := s.executor.GetOptions()
		_ = env // 暂时不使用，显示变量需要访问executor的env
		// 显示当前选项状态
		fmt.Println("--- Shell Options ---")
		for opt, enabled := range s.options {
			if enabled {
				fmt.Printf("set -%s\n", opt)
			} else {
				fmt.Printf("set +%s\n", opt)
			}
		}
		return nil
	}
	
	// 处理选项
	for _, arg := range args {
		if arg == "--" {
			// set -- 重置位置参数（这里暂时忽略）
			continue
		}
		
		if strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "+") {
			// 解析选项，如 -x, -e, +x, +e
			enable := arg[0] == '-'
			optionStr := arg[1:]
			
			// 处理多个选项，如 -xe
			for _, opt := range optionStr {
				optStr := string(opt)
				s.options[optStr] = enable
				
				// 同步到执行器
				s.executor.SetOptions(s.options)
			}
		} else {
			// 设置变量（set VAR=value）
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				s.executor.SetEnv(parts[0], parts[1])
			}
		}
	}
	
	return nil
}

// handleUnaliasCommand 处理unalias命令
func (s *Shell) handleUnaliasCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("unalias: 缺少操作数")
	}

	for _, name := range args {
		if name == "-a" {
			// 清除所有别名
			s.aliases = make(map[string]string)
		} else {
			delete(s.aliases, name)
		}
	}

	return nil
}

// splitCommands 分割命令（按分号）
func splitCommands(line string) []string {
	var commands []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if ch == '\\' && i+1 < len(line) {
			// 转义字符
			current.WriteByte(line[i+1])
			i++
			continue
		}

		if (ch == '"' || ch == '\'') && !inQuotes {
			inQuotes = true
			quoteChar = ch
			current.WriteByte(ch)
		} else if ch == quoteChar && inQuotes {
			inQuotes = false
			quoteChar = 0
			current.WriteByte(ch)
		} else if ch == ';' && !inQuotes {
			// 分号且不在引号内，分割命令
			cmd := strings.TrimSpace(current.String())
			if cmd != "" {
				commands = append(commands, cmd)
			}
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}

	// 添加最后一个命令
	cmd := strings.TrimSpace(current.String())
	if cmd != "" {
		commands = append(commands, cmd)
	}

	return commands
}

// getPrompt 获取提示符
func getPrompt() string {
	// 尝试获取用户名和主机名
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "user"
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "host"
	}

	wd, _ := os.Getwd()
	if wd == "" {
		wd = "~"
	}

	// 简化路径显示
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home != "" && strings.HasPrefix(wd, home) {
		// Windows路径处理
		wd = strings.Replace(wd, home, "~", 1)
		wd = strings.ReplaceAll(wd, "\\", "/")
	} else {
		// 统一使用正斜杠显示
		wd = strings.ReplaceAll(wd, "\\", "/")
	}

	return fmt.Sprintf("%s@%s:%s$ ", username, hostname, wd)
}

// handleHistoryCommand 处理history命令
func (s *Shell) handleHistoryCommand(args []string) error {
	if len(args) == 0 {
		// 显示所有历史
		s.history.Print()
		return nil
	}

	// 处理参数，如 history -c (清除历史)
	if len(args) > 0 && args[0] == "-c" {
		s.history = NewHistory(1000)
		return nil
	}

	// 显示最后N条历史
	// 简化实现，只显示所有历史
	s.history.Print()
	return nil
}

