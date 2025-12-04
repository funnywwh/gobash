// Package shell 提供交互式Shell的核心功能
//
// 该包实现了Shell的主要功能，包括：
// - 交互式REPL循环
// - 命令历史管理
// - 别名管理
// - Shell选项管理
// - 自动补全功能
// - 脚本文件执行
package shell

import (
	"bufio"
	"fmt"
	"gobash/internal/builtin"
	"gobash/internal/executor"
	"gobash/internal/lexer"
	"gobash/internal/parser"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
)

// Shell Shell主结构
// 管理Shell的状态，包括执行器、提示符、别名、历史记录和选项
type Shell struct {
	executor      *executor.Executor
	prompt        string
	running       bool
	aliases       map[string]string
	history       *History
	options       map[string]bool // shell选项状态
	errorReporter *ErrorReporter  // 错误报告器
}

// New 创建新的Shell实例
// 初始化Shell结构，加载历史记录，创建执行器实例
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
		executor:      executor.New(),
		prompt:        getPrompt(),
		running:       true,
		aliases:       make(map[string]string),
		history:       history,
		options:       make(map[string]bool),
		errorReporter: NewErrorReporter("", true), // 交互式模式
	}

	// 将选项状态传递给执行器
	sh.executor.SetOptions(sh.options)

	return sh
}

// Run 运行交互式Shell
// 启动REPL循环，支持readline库的交互功能（历史记录、自动补全等）
// 如果readline不可用，会自动回退到简单的输入模式
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

	// 创建自动补全器
	completer := NewCompleter(s)

	// 创建readline配置
	config := &readline.Config{
		Prompt:          s.prompt,
		HistoryFile:     historyFile,
		HistoryLimit:    1000,
		AutoComplete:    completer,
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

		var currentStatement strings.Builder
		for {
			line, err := rl.Readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					// Ctrl+C，继续
					fmt.Println()
					currentStatement.Reset()
					break
				}
				// EOF或其他错误，退出
				return
			}

			lineTrimmed := strings.TrimSpace(line)

			// 如果有未完成的语句，追加当前行
			if currentStatement.Len() > 0 {
				currentStatement.WriteString("\n")
				currentStatement.WriteString(line)
			} else {
				currentStatement.WriteString(line)
			}

			// 检查语句是否完成
			statement := currentStatement.String()
			isComplete := s.isStatementComplete(statement)

			// 也检查是否以反斜杠结尾（行继续符）
			if !isComplete || strings.HasSuffix(lineTrimmed, "\\") {
				// 语句未完成，继续读取下一行
				rl.SetPrompt("> ")
				continue
			}

			// 语句完成，执行
			break
		}

		line := currentStatement.String()
		if strings.TrimSpace(line) == "" {
			continue
		}

		if err := s.executeLine(line); err != nil {
			// 检查是否是 exit 命令
			if exitErr, ok := err.(*builtin.ExitError); ok {
				// 在交互式模式下，exit 命令退出整个程序
				os.Exit(exitErr.Code)
			}
			// 使用统一的错误报告器
			s.errorReporter.ReportError(err)
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
// 使用bufio.Scanner进行基本的命令行输入，不支持历史记录和自动补全
func (s *Shell) runSimple() {
	scanner := bufio.NewScanner(os.Stdin)

	for s.running {
		fmt.Print(s.prompt)

		var currentStatement strings.Builder
		for {
			if !scanner.Scan() {
				return
			}

			line := scanner.Text()
			lineTrimmed := strings.TrimSpace(line)

			// 如果有未完成的语句，追加当前行
			if currentStatement.Len() > 0 {
				currentStatement.WriteString("\n")
				currentStatement.WriteString(line)
			} else {
				currentStatement.WriteString(line)
			}

			// 检查语句是否完成
			statement := currentStatement.String()
			isComplete := s.isStatementComplete(statement)

			// 也检查是否以反斜杠结尾（行继续符）
			if !isComplete || strings.HasSuffix(lineTrimmed, "\\") {
				// 语句未完成，继续读取下一行
				fmt.Print("> ")
				continue
			}

			// 语句完成，执行
			line = statement
			break
		}

		line := currentStatement.String()
		if strings.TrimSpace(line) == "" {
			continue
		}

		if err := s.executeLine(line); err != nil {
			// 检查是否是 exit 命令
			if exitErr, ok := err.(*builtin.ExitError); ok {
				// 在交互式模式下，exit 命令退出整个程序
				os.Exit(exitErr.Code)
			}
			// 使用统一的错误报告器
			s.errorReporter.ReportError(err)
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
func (s *Shell) ExecuteScript(scriptPath string, args ...string) error {
	// 设置位置参数（$1, $2, ...）和 $#、$@
	for i, arg := range args {
		s.executor.SetEnv(fmt.Sprintf("%d", i+1), arg)
	}
	s.executor.SetEnv("#", fmt.Sprintf("%d", len(args)))
	s.executor.SetEnv("@", strings.Join(args, " "))

	file, err := os.Open(scriptPath)
	if err != nil {
		return fmt.Errorf("无法打开脚本文件: %v", err)
	}
	defer file.Close()

	// 设置错误报告器的脚本路径（非交互式模式）
	s.errorReporter = NewErrorReporter(scriptPath, false)
	return s.ExecuteReader(file)
}

// ExecuteReader 从Reader执行命令
// 用于执行脚本文件，自动跳过shebang行和注释行
// 支持多行语句（case、if、for等）
func (s *Shell) ExecuteReader(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	firstLine := true
	lineNum := 0
	var currentStatement strings.Builder

	for scanner.Scan() {
		lineNum++
		originalLine := scanner.Text()
		line := strings.TrimSpace(originalLine)

		// 跳过空行（但如果当前有未完成的语句，保留空行）
		if line == "" {
			if currentStatement.Len() > 0 {
				currentStatement.WriteString("\n")
			}
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

		// 如果有未完成的语句，追加当前行
		if currentStatement.Len() > 0 {
			currentStatement.WriteString("\n")
			currentStatement.WriteString(originalLine)
		} else {
			currentStatement.WriteString(originalLine)
		}

		// 检查当前行是否包含 heredoc 标记，如果有则立即跳过内容
		if strings.Contains(originalLine, "<<") {
			// 提取 heredoc 分隔符
			delim := extractHeredocDelimiterFromLine(originalLine)
			if delim != "" {
				// 跳过 heredoc 内容直到找到分隔符
				heredocContent := ""
				foundDelimiter := false
				for scanner.Scan() {
					lineNum++
					contentLine := scanner.Text()
					if strings.TrimSpace(contentLine) == delim {
						// 找到分隔符
						foundDelimiter = true
						break
					}
					// heredoc 内容保存（稍后可能需要用于解析）
					heredocContent += contentLine + "\n"
				}
				
				// heredoc 处理完成，不将内容添加到 statement
				// 但需要确保 parser 知道 heredoc 结束了
				_ = heredocContent // heredoc 内容不添加到 statement
				_ = foundDelimiter
			}
		}

		// 检查语句是否完成
		statement := currentStatement.String()
		isComplete := s.isStatementComplete(statement)
		if isComplete {
			// 执行完整的语句
			if err := s.executeLine(statement); err != nil {
				// 检查是否是 exit 命令或脚本退出错误
				if exitErr, ok := err.(*builtin.ExitError); ok {
					// 返回 ExitError，让调用者决定如何处理（不输出错误信息）
					return exitErr
				}
				if scriptExitErr, ok := err.(*executor.ScriptExitError); ok {
					// 返回 ScriptExitError，让调用者决定如何处理（不输出错误信息）
					return scriptExitErr
				}
				// 使用统一的错误报告器
				s.errorReporter.SetLineNum(lineNum)
				s.errorReporter.ReportError(err)
				// 输出语句内容（用于调试）
				fmt.Fprintf(os.Stderr, "  %s\n", statement)
				// 如果设置了set -e，遇到错误应该退出
				// 但是 ScriptExitError 和 ExitError 已经表示脚本退出，不需要再次包装
				if s.options["e"] {
					// 检查是否已经是退出错误
					if _, ok := err.(*builtin.ExitError); ok {
						return err
					}
					if _, ok := err.(*executor.ScriptExitError); ok {
						return err
					}
					return fmt.Errorf("脚本执行失败（第%d行）: %v", lineNum, err)
				}
			}
			// 重置当前语句
			currentStatement.Reset()
		}
	}

	// 如果还有未完成的语句，尝试执行
	if currentStatement.Len() > 0 {
		statement := currentStatement.String()
		if err := s.executeLine(statement); err != nil {
			// 检查是否是 exit 命令或脚本退出错误
			if exitErr, ok := err.(*builtin.ExitError); ok {
				// 返回 ExitError，让调用者决定如何处理（不输出错误信息）
				return exitErr
			}
			if scriptExitErr, ok := err.(*executor.ScriptExitError); ok {
				// 返回 ScriptExitError，让调用者决定如何处理（不输出错误信息）
				return scriptExitErr
			}
			// 使用统一的错误报告器
			s.errorReporter.SetLineNum(lineNum)
			s.errorReporter.ReportError(err)
			// 输出语句内容（用于调试）
			fmt.Fprintf(os.Stderr, "  %s\n", statement)
			if s.options["e"] {
				// 检查是否已经是退出错误
				if _, ok := err.(*builtin.ExitError); ok {
					return err
				}
				if _, ok := err.(*executor.ScriptExitError); ok {
					return err
				}
				return fmt.Errorf("脚本执行失败（第%d行）: %v", lineNum, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// isStatementComplete 检查语句是否完成
// 检查是否有关键字未闭合（case需要esac，if需要fi，for/while需要done等）
// 也检查是否以反斜杠结尾（行继续符）
func (s *Shell) isStatementComplete(statement string) bool {
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return true
	}

	// 检查是否以反斜杠结尾（行继续符）
	// 注意：需要检查去除尾部空白后的最后一个字符
	trimmed := strings.TrimRight(statement, " \t")
	if strings.HasSuffix(trimmed, "\\") {
		return false
	}

	// 使用更精确的匹配来统计关键字（必须是独立的单词）
	words := strings.Fields(statement)

	caseCount := 0
	for _, word := range words {
		if word == "case" {
			caseCount++
		}
	}
	esacCount := strings.Count(statement, "esac")

	ifCount := 0
	for _, word := range words {
		if word == "if" {
			ifCount++
		}
	}
	fiCount := strings.Count(statement, "fi")

	forCount := 0
	for _, word := range words {
		if word == "for" {
			forCount++
		}
	}

	whileCount := 0
	for _, word := range words {
		if word == "while" {
			whileCount++
		}
	}

	doneCount := 0
	for _, word := range words {
		if word == "done" {
			doneCount++
		}
	}

	// 检查case语句
	if caseCount > esacCount {
		return false
	}

	// 检查if语句
	if ifCount > fiCount {
		return false
	}

	// 检查for/while语句
	// 注意：while循环需要do关键字，所以如果whileCount > 0但没有done，且没有do，语句未完成
	if (forCount + whileCount) > doneCount {
		// 检查是否有do关键字（while循环需要do）
		// 使用更精确的匹配：do必须是独立的单词
		doCount := 0
		// 检查 " do "、" do\n"、";do "、";do\n"、"\ndo "、"\ndo\n" 等
		doPatterns := []string{" do ", " do\n", ";do ", ";do\n", "\ndo ", "\ndo\n", " do;", "\ndo;"}
		for _, pattern := range doPatterns {
			doCount += strings.Count(statement, pattern)
		}
		// 也检查行首的do（do在行首）
		if strings.HasPrefix(strings.TrimSpace(statement), "do ") ||
			strings.HasPrefix(strings.TrimSpace(statement), "do\n") {
			doCount++
		}
		// 如果while循环没有do关键字，语句未完成
		if whileCount > 0 && doCount == 0 && doneCount == 0 {
			return false
		}
		// 如果while循环有do但没有done，语句未完成
		if whileCount > 0 && doCount > 0 && doneCount == 0 {
			return false
		}
		// 如果for循环没有done，语句未完成
		if forCount > 0 && doneCount == 0 {
			return false
		}
	}

	// 检查函数定义 name() { ... }
	// 函数定义格式：name() { ... } 或 function name() { ... }
	// 需要检查是否有未闭合的大括号
	braceCount := 0
	inQuotes := false
	quoteChar := byte(0)
	for i := 0; i < len(statement); i++ {
		ch := statement[i]
		
		// 处理转义字符
		if ch == '\\' && i+1 < len(statement) {
			if !inQuotes {
				// 在引号外，转义字符用于转义下一个字符
				i++ // 跳过转义字符和下一个字符
				continue
			} else {
				// 在引号内，转义字符应该保留
				if i+1 < len(statement) && statement[i+1] == quoteChar {
					// 转义的引号，不改变引号状态
					i++
					continue
				}
				// 其他转义字符，保留
				continue
			}
		}
		
		// 处理引号
		if (ch == '"' || ch == '\'') && !inQuotes {
			inQuotes = true
			quoteChar = ch
		} else if ch == quoteChar && inQuotes {
			inQuotes = false
			quoteChar = 0
		}
		
		// 只统计引号外的大括号
		if !inQuotes {
			if ch == '{' {
				braceCount++
			} else if ch == '}' {
				braceCount--
			}
		}
	}
	
	// 如果有未闭合的大括号，语句未完成
	if braceCount > 0 {
		return false
	}

	return true
}

// extractHeredocDelimiterFromLine 从一行中提取 heredoc 分隔符
func extractHeredocDelimiterFromLine(line string) string {
	// 查找 << 或 <<-
	idx := strings.Index(line, "<<")
	if idx < 0 {
		return ""
	}
	
	// 检查是否是 <<- 或 <<<
	afterHeredoc := line[idx:]
	var suffix string
	if strings.HasPrefix(afterHeredoc, "<<-") {
		suffix = line[idx+3:]
	} else if strings.HasPrefix(afterHeredoc, "<<<") {
		// <<< here-string，不需要处理
		return ""
	} else if strings.HasPrefix(afterHeredoc, "<<") {
		suffix = line[idx+2:]
	} else {
		return ""
	}
	
	// 跳过空白字符
	suffix = strings.TrimLeft(suffix, " \t")
	if len(suffix) == 0 {
		return ""
	}
	
	// 检查分隔符是否带引号
	if suffix[0] == '\'' || suffix[0] == '"' {
		// 带引号的分隔符
		quoteChar := suffix[0]
		endQuote := strings.IndexByte(suffix[1:], quoteChar)
		if endQuote >= 0 {
			return suffix[1 : endQuote+1]
		}
		// 引号未闭合，尝试提取第一个字段
		parts := strings.Fields(suffix)
		if len(parts) > 0 {
			delim := parts[0]
			if len(delim) > 1 && (delim[0] == '\'' || delim[0] == '"') {
				return delim[1 : len(delim)-1]
			}
			return delim
		}
	} else {
		// 不带引号的分隔符
		parts := strings.Fields(suffix)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	
	return ""
}

// executeLine 执行一行命令
// 支持分号分隔的多个命令
func (s *Shell) executeLine(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	// 分割多个命令（分号分隔）
	commands := splitCommands(line)
	for _, cmd := range commands {
		if err := s.executeCommand(cmd); err != nil {
			// 检查是否是 exit 命令或脚本退出错误，如果是，直接返回
			if _, ok := err.(*builtin.ExitError); ok {
				return err
			}
			if _, ok := err.(*executor.ScriptExitError); ok {
				return err
			}
			return err
		}
	}

	return nil
}

// executeCommand 执行单个命令
// 处理别名展开、词法分析、语法分析和命令执行
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
		// 返回第一个解析错误（ParseError）
		if len(p.ParseErrors()) > 0 {
			return p.ParseErrors()[0]
		}
		// 如果没有 ParseError，返回通用错误
		return fmt.Errorf("语法错误: %v", p.Errors())
	}

	// 执行
	if err := s.executor.Execute(program); err != nil {
		// 检查是否是 exit 命令或脚本退出错误，如果是，直接返回，不包装
		if _, ok := err.(*builtin.ExitError); ok {
			return err
		}
		if _, ok := err.(*executor.ScriptExitError); ok {
			return err
		}
		// 直接返回错误，让调用者使用错误报告器格式化
		return err
	}

	return nil
}

// expandAlias 展开别名
// 如果命令名是已定义的别名，则替换为别名值
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
// 支持设置别名、显示所有别名或显示特定别名
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
// 支持设置/取消Shell选项（-x, -e, -u等）和设置变量
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
	// 检查是否有 -- 参数，如果有，则 -- 后面的所有参数都是位置参数
	argIndex := 0
	positionalArgsStart := -1
	for i, arg := range args {
		if arg == "--" {
			positionalArgsStart = i + 1
			break
		}
		argIndex++
	}

	// 如果找到了 --，处理位置参数
	if positionalArgsStart >= 0 {
		// 先清空所有现有的位置参数
		// 获取当前参数个数，然后删除所有位置参数
		envMap := s.executor.GetEnvMap()
		if countStr, ok := envMap["#"]; ok {
			if count, err := strconv.Atoi(countStr); err == nil && count > 0 {
				// 删除所有位置参数
				for i := 1; i <= count; i++ {
					delete(envMap, fmt.Sprintf("%d", i))
				}
			}
		}

		// 设置新的位置参数（-- 后面的所有参数）
		positionalArgs := args[positionalArgsStart:]
		for i, arg := range positionalArgs {
			s.executor.SetEnv(fmt.Sprintf("%d", i+1), arg)
		}
		s.executor.SetEnv("#", fmt.Sprintf("%d", len(positionalArgs)))
		s.executor.SetEnv("@", strings.Join(positionalArgs, " "))
		// 跳过处理 -- 和位置参数
		args = args[:positionalArgsStart-1]
	}

	// 处理选项（跳过已经处理过的 -- 和位置参数）
	for _, arg := range args {

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
// 支持删除特定别名或清除所有别名（-a选项）
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
	braceDepth := 0 // 大括号深度，用于跟踪函数定义和代码块

	for i := 0; i < len(line); i++ {
		ch := line[i]

		// 处理转义字符
		if ch == '\\' && i+1 < len(line) {
			if !inQuotes {
				// 在引号外，转义字符用于转义分号等，需要处理
				current.WriteByte(line[i+1])
				i++
				continue
			} else {
				// 在引号内，转义字符应该保留（由 lexer 处理）
				// 但需要检查是否是转义的引号（如 \"），如果是，跳过不当作引号结束
				if line[i+1] == quoteChar {
					// 转义的引号，保留 \ 和引号，继续
					current.WriteByte(ch)
					current.WriteByte(line[i+1])
					i++
					continue
				}
				// 其他转义字符，保留
				current.WriteByte(ch)
				continue
			}
		}

		if (ch == '"' || ch == '\'') && !inQuotes {
			inQuotes = true
			quoteChar = ch
			current.WriteByte(ch)
		} else if ch == quoteChar && inQuotes {
			inQuotes = false
			quoteChar = 0
			current.WriteByte(ch)
		} else if !inQuotes {
			// 跟踪大括号深度（只在引号外）
			if ch == '{' {
				braceDepth++
				current.WriteByte(ch)
			} else if ch == '}' {
				braceDepth--
				current.WriteByte(ch)
			} else if ch == ';' && braceDepth == 0 {
				// 检查是否是双分号 ;;（case语句的结束符）
				if i+1 < len(line) && line[i+1] == ';' {
					// 双分号，不分割命令，将 ;; 作为当前命令的一部分
					current.WriteByte(ch)
					current.WriteByte(line[i+1])
					i++ // 跳过第二个分号
					continue
				}

				// 检查分号后的单词是否是控制流关键字（do、then等）
				// 如果是，这个分号不应该分割命令
				remaining := line[i+1:]
				remaining = strings.TrimSpace(remaining)
				// 获取分号后的第一个单词
				nextWord := ""
				if len(remaining) > 0 {
					parts := strings.Fields(remaining)
					if len(parts) > 0 {
						nextWord = strings.ToLower(parts[0])
					}
				}

				// 控制流关键字列表（分号后可能出现的）
				controlFlowAfterSemicolon := []string{"do", "then", "else", "elif"}
				shouldSplit := true
				for _, keyword := range controlFlowAfterSemicolon {
					if nextWord == keyword {
						shouldSplit = false
						break
					}
				}

				if shouldSplit {
					// 分号且不在引号内，分割命令
					cmd := strings.TrimSpace(current.String())
					if cmd != "" {
						commands = append(commands, cmd)
					}
					current.Reset()
				} else {
					// 分号后是控制流关键字，不分割，将分号作为当前命令的一部分
					current.WriteByte(ch)
				}
			} else {
				current.WriteByte(ch)
			}
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
