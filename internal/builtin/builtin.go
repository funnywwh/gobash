// Package builtin 提供所有内置命令的实现
// 
// 内置命令是shell的核心功能，包括：
// - 目录操作：cd, pwd
// - 文件操作：ls, cat, mkdir, rmdir, rm, touch, clear
// - 文本处理：head, tail, wc, grep, sort, uniq, cut
// - 环境变量：export, unset, env, set
// - 控制命令：exit, alias, unalias, history, which, type, true, false, test
// - 作业控制：jobs, fg, bg
//
// 所有内置命令都遵循 BuiltinFunc 函数签名，接收参数列表和环境变量映射。
package builtin

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BuiltinFunc 内置命令函数类型
// 所有内置命令必须符合此函数签名
// args: 命令参数列表
// env: 环境变量映射，可以读取和修改环境变量
type BuiltinFunc func(args []string, env map[string]string) error

var builtins map[string]BuiltinFunc

func init() {
	builtins = make(map[string]BuiltinFunc)
	builtins["cd"] = cd
	builtins["pwd"] = pwd
	builtins["echo"] = echo
	builtins["exit"] = exit
	builtins["export"] = export
	builtins["unset"] = unset
	builtins["env"] = env
	builtins["set"] = set
	builtins["ls"] = ls
	builtins["cat"] = cat
	builtins["mkdir"] = mkdir
	builtins["rmdir"] = rmdir
	builtins["rm"] = rm
	builtins["touch"] = touch
	builtins["clear"] = clear
	builtins["alias"] = alias
	builtins["unalias"] = unalias
	builtins["history"] = history
	builtins["which"] = which
	builtins["type"] = typeCmd
	builtins["true"] = trueCmd
	builtins["false"] = falseCmd
	builtins["test"] = testCmd
	builtins["["] = testCmd // [ 是 test 的别名，但需要处理结尾的 ]
	builtins["head"] = head
	builtins["tail"] = tail
	builtins["wc"] = wc
	builtins["grep"] = grep
	builtins["sort"] = sortCmd
	builtins["uniq"] = uniq
	builtins["cut"] = cut
	builtins["jobs"] = jobs
	builtins["fg"] = fg
	builtins["bg"] = bg
	builtins["declare"] = declare
}

// GetBuiltins 获取所有内置命令
// 返回一个包含所有已注册内置命令的映射表
// 键为命令名，值为对应的命令函数
func GetBuiltins() map[string]BuiltinFunc {
	return builtins
}

// cd 改变当前工作目录
// 支持相对路径、绝对路径和~展开（用户主目录）
// 如果没有参数，切换到用户主目录
func cd(args []string, env map[string]string) error {
	var dir string
	if len(args) == 0 {
		// 没有参数，切换到home目录
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		if home == "" {
			usr, err := user.Current()
			if err != nil {
				return err
			}
			dir = usr.HomeDir
		} else {
			dir = home
		}
	} else {
		dir = args[0]
		// 展开 ~
		if strings.HasPrefix(dir, "~") {
			home := os.Getenv("HOME")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
			if home == "" {
				usr, err := user.Current()
				if err != nil {
					return err
				}
				home = usr.HomeDir
			}
			dir = strings.Replace(dir, "~", home, 1)
		}
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("cd: %v", err)
	}

	// 更新PWD环境变量
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	env["PWD"] = pwd
	os.Setenv("PWD", pwd)

	return nil
}

// pwd 显示当前工作目录的绝对路径
// 输出当前shell的工作目录
func pwd(args []string, env map[string]string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Println(dir)
	return nil
}

// echo 输出文本到标准输出
// 将所有参数用空格连接后输出，最后换行
func echo(args []string, env map[string]string) error {
	if len(args) == 0 {
		fmt.Println()
		return nil
	}
	output := strings.Join(args, " ")
	// 如果输出为空字符串，至少打印一个换行
	if output == "" {
		fmt.Println()
	} else {
		fmt.Println(output)
	}
	return nil
}

// exit 退出shell
func exit(args []string, env map[string]string) error {
	code := 0
	if len(args) > 0 {
		// 解析退出码
		if parsed, err := strconv.Atoi(args[0]); err == nil {
			code = parsed
		} else {
			// 如果无法解析，使用默认值0
			code = 0
		}
	}
	os.Exit(code)
	return nil
}

// export 导出环境变量
// 将变量设置到环境变量中，格式为 KEY=VALUE
// 支持多个变量同时设置
func export(args []string, env map[string]string) error {
	if len(args) == 0 {
		// 显示所有导出的环境变量
		for k, v := range env {
			fmt.Printf("export %s=%s\n", k, v)
		}
		return nil
	}

	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
			os.Setenv(parts[0], parts[1])
		} else {
			// 只设置变量名，使用现有值
			if value, ok := env[arg]; ok {
				os.Setenv(arg, value)
			}
		}
	}

	return nil
}

// unset 取消设置环境变量
// 从环境变量映射中删除指定的变量
// 支持同时删除多个变量
func unset(args []string, env map[string]string) error {
	for _, arg := range args {
		delete(env, arg)
		os.Unsetenv(arg)
	}
	return nil
}

// env 显示环境变量
func env(args []string, env map[string]string) error {
	for k, v := range env {
		fmt.Printf("%s=%s\n", k, v)
	}
	return nil
}

// set 设置shell选项
// 注意：set命令的实际处理在shell.go中的handleSetCommand函数中完成
// 这个函数作为占位符，主要用于非交互式执行场景
func set(args []string, env map[string]string) error {
	if len(args) == 0 {
		// 显示所有变量
		for k, v := range env {
			fmt.Printf("%s=%s\n", k, v)
		}
		return nil
	}
	// set命令的选项处理在shell层完成（shell.go中的handleSetCommand）
	// 这里只处理变量设置
	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
				os.Setenv(parts[0], parts[1])
			}
		}
	}
	return nil
}

// ls 列出目录内容
func ls(args []string, env map[string]string) error {
	var path string
	longFormat := false
	showAll := false

	// 解析参数
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			if strings.Contains(arg, "l") {
				longFormat = true
			}
			if strings.Contains(arg, "a") {
				showAll = true
			}
		} else if i == len(args)-1 {
			path = arg
		}
	}

	if path == "" {
		path = "."
	}

	// 展开 ~
	if strings.HasPrefix(path, "~") {
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		if home != "" {
			path = strings.Replace(path, "~", home, 1)
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("ls: %v", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("ls: %v", err)
	}

	if !info.IsDir() {
		// 单个文件
		if longFormat {
			printFileInfo(info, info.Name())
		} else {
			fmt.Println(info.Name())
		}
		return nil
	}

	// 目录
	entries, err := file.Readdir(-1)
	if err != nil {
		return fmt.Errorf("ls: %v", err)
	}

	// 排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}

		if longFormat {
			printFileInfo(entry, name)
		} else {
			fmt.Print(name + "  ")
		}
	}

	if !longFormat {
		fmt.Println()
	}

	return nil
}

// printFileInfo 打印文件详细信息
func printFileInfo(info os.FileInfo, name string) {
	mode := info.Mode().String()
	size := info.Size()
	modTime := info.ModTime().Format("Jan 02 15:04")
	dir := ""
	if info.IsDir() {
		dir = "d"
	} else {
		dir = "-"
	}
	fmt.Printf("%s%s %8d %s %s\n", dir, mode[1:10], size, modTime, name)
}

// cat 显示文件内容
// 将指定文件的内容输出到标准输出
// 支持多个文件，会依次显示
func cat(args []string, env map[string]string) error {
	if len(args) == 0 {
		// 从stdin读取
		_, err := io.Copy(os.Stdout, os.Stdin)
		return err
	}

	for _, filename := range args {
		// 展开 ~
		if strings.HasPrefix(filename, "~") {
			home := os.Getenv("HOME")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
			if home != "" {
				filename = strings.Replace(filename, "~", home, 1)
			}
		}

		file, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("cat: %v", err)
		}

		_, err = io.Copy(os.Stdout, file)
		file.Close()
		if err != nil {
			return fmt.Errorf("cat: %v", err)
		}
	}

	return nil
}

// mkdir 创建目录
// 支持 -p 选项创建父目录（如果不存在）
// 支持同时创建多个目录
func mkdir(args []string, env map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("mkdir: 缺少操作数")
	}

	parents := false
	paths := []string{}

	for _, arg := range args {
		if arg == "-p" || arg == "--parents" {
			parents = true
		} else {
			paths = append(paths, arg)
		}
	}

	for _, path := range paths {
		// 展开 ~
		if strings.HasPrefix(path, "~") {
			home := os.Getenv("HOME")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
			if home != "" {
				path = strings.Replace(path, "~", home, 1)
			}
		}

		if parents {
			err := os.MkdirAll(path, 0755)
			if err != nil {
				return fmt.Errorf("mkdir: %v", err)
			}
		} else {
			err := os.Mkdir(path, 0755)
			if err != nil {
				return fmt.Errorf("mkdir: %v", err)
			}
		}
	}

	return nil
}

// rmdir 删除空目录
// 只能删除空目录，如果目录不为空会返回错误
// 支持同时删除多个目录
func rmdir(args []string, env map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("rmdir: 缺少操作数")
	}

	for _, path := range args {
		// 展开 ~
		if strings.HasPrefix(path, "~") {
			home := os.Getenv("HOME")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
			if home != "" {
				path = strings.Replace(path, "~", home, 1)
			}
		}

		err := os.Remove(path)
		if err != nil {
			return fmt.Errorf("rmdir: %v", err)
		}
	}

	return nil
}

// rm 删除文件或目录
func rm(args []string, env map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("rm: 缺少操作数")
	}

	recursive := false
	force := false
	paths := []string{}

	for _, arg := range args {
		if arg == "-r" || arg == "-R" || arg == "--recursive" {
			recursive = true
		} else if arg == "-f" || arg == "--force" {
			force = true
		} else if arg == "-rf" || arg == "-rR" || arg == "-fr" {
			recursive = true
			force = true
		} else {
			paths = append(paths, arg)
		}
	}

	for _, path := range paths {
		// 展开 ~
		if strings.HasPrefix(path, "~") {
			home := os.Getenv("HOME")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
			if home != "" {
				path = strings.Replace(path, "~", home, 1)
			}
		}

		info, err := os.Stat(path)
		if err != nil {
			if !force {
				return fmt.Errorf("rm: %v", err)
			}
			continue
		}

		if info.IsDir() {
			if recursive {
				err = os.RemoveAll(path)
			} else {
				err = fmt.Errorf("rm: %s: 是一个目录", path)
			}
		} else {
			err = os.Remove(path)
		}

		if err != nil && !force {
			return fmt.Errorf("rm: %v", err)
		}
	}

	return nil
}

// touch 创建文件或更新文件时间戳
func touch(args []string, env map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("touch: 缺少操作数")
	}

	for _, filename := range args {
		// 展开 ~
		if strings.HasPrefix(filename, "~") {
			home := os.Getenv("HOME")
			if home == "" {
				home = os.Getenv("USERPROFILE")
			}
			if home != "" {
				filename = strings.Replace(filename, "~", home, 1)
			}
		}

		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("touch: %v", err)
		}
		file.Close()

		// 更新时间戳
		now := time.Now()
		os.Chtimes(filename, now, now)
	}

	return nil
}

// clear 清屏
func clear(args []string, env map[string]string) error {
	// Windows使用cls，Unix使用clear
	fmt.Print("\033[2J\033[H")
	return nil
}

// alias 设置或显示别名
// 注意：这个函数需要通过shell来访问别名map，使用环境变量作为通信机制
func alias(args []string, env map[string]string) error {
	if len(args) == 0 {
		// 显示所有别名 - 通过环境变量获取
		aliasesStr := env["__WBASH_ALIASES__"]
		if aliasesStr == "" {
			return nil
		}
		// 解析别名字符串（格式：name1=value1;name2=value2;...）
		parts := strings.Split(aliasesStr, ";")
		for _, part := range parts {
			if part != "" {
				fmt.Println("alias " + part)
			}
		}
		return nil
	}

	// 设置别名 - 通过环境变量传递
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			name := parts[0]
			value := strings.Trim(parts[1], "\"'")
			// 通过环境变量传递别名信息给shell
			env["__WBASH_SET_ALIAS__"] = name + "=" + value
		} else {
			// 显示特定别名
			name := arg
			aliasesStr := env["__WBASH_ALIASES__"]
			if aliasesStr != "" {
				parts := strings.Split(aliasesStr, ";")
				for _, part := range parts {
					if strings.HasPrefix(part, name+"=") {
						fmt.Println("alias " + part)
						return nil
					}
				}
			}
		}
	}

	return nil
}

// unalias 取消设置别名
func unalias(args []string, env map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("unalias: 缺少操作数")
	}

	for _, name := range args {
		if name == "-a" {
			// 清除所有别名
			env["__WBASH_UNSET_ALL_ALIASES__"] = "1"
		} else {
			// 清除特定别名
			env["__WBASH_UNSET_ALIAS__"] = name
		}
	}

	return nil
}

// history 显示命令历史（简化版，实际由shell处理）
func history(args []string, env map[string]string) error {
	// history命令由shell直接处理，这里只是占位
	return nil
}

// which 查找命令路径
func which(args []string, env map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("which: 缺少操作数")
	}

	for _, cmdName := range args {
		// 检查是否为内置命令
		if _, ok := builtins[cmdName]; ok {
			fmt.Printf("%s: shell builtin\n", cmdName)
			continue
		}

		// 检查PATH环境变量
		pathEnv := os.Getenv("PATH")
		if pathEnv == "" {
			continue
		}

		paths := strings.Split(pathEnv, ":")
		if len(paths) == 0 {
			// Windows使用分号分隔
			paths = strings.Split(pathEnv, ";")
		}

		found := false
		for _, path := range paths {
			if path == "" {
				continue
			}
			fullPath := filepath.Join(path, cmdName)
			// Windows需要添加.exe扩展名
			if _, err := os.Stat(fullPath); err == nil {
				fmt.Println(fullPath)
				found = true
				break
			}
			// 尝试添加.exe
			if _, err := os.Stat(fullPath + ".exe"); err == nil {
				fmt.Println(fullPath + ".exe")
				found = true
				break
			}
		}

		if !found {
			// 命令未找到，但不报错（与bash行为一致）
		}
	}

	return nil
}

// typeCmd 显示命令类型
func typeCmd(args []string, env map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("type: 缺少操作数")
	}

	for _, cmdName := range args {
		// 检查是否为内置命令
		if _, ok := builtins[cmdName]; ok {
			fmt.Printf("%s is a shell builtin\n", cmdName)
			continue
		}

		// 检查是否为别名（通过环境变量，实际由shell处理）
		// 这里简化处理，只检查内置命令和外部命令

		// 检查PATH环境变量
		pathEnv := os.Getenv("PATH")
		if pathEnv != "" {
			paths := strings.Split(pathEnv, ":")
			if len(paths) == 0 {
				paths = strings.Split(pathEnv, ";")
			}

			found := false
			for _, path := range paths {
				if path == "" {
					continue
				}
				fullPath := filepath.Join(path, cmdName)
				if _, err := os.Stat(fullPath); err == nil {
					fmt.Printf("%s is %s\n", cmdName, fullPath)
					found = true
					break
				}
				if _, err := os.Stat(fullPath + ".exe"); err == nil {
					fmt.Printf("%s is %s\n", cmdName, fullPath+".exe")
					found = true
					break
				}
			}

			if found {
				continue
			}
		}

		// 命令未找到
		fmt.Printf("type: %s: not found\n", cmdName)
	}

	return nil
}

// trueCmd 总是成功返回
func trueCmd(args []string, env map[string]string) error {
	return nil
}

// falseCmd 总是失败返回
func falseCmd(args []string, env map[string]string) error {
	return fmt.Errorf("false")
}

// declare 声明变量或数组
// 支持 -A 选项声明关联数组
// 例如：declare -A arr
func declare(args []string, env map[string]string) error {
	if len(args) == 0 {
		// 显示所有变量（简化实现）
		return nil
	}
	
	assocArray := false
	var varName string
	
	// 解析参数
	for _, arg := range args {
		if arg == "-A" {
			assocArray = true
		} else if !strings.HasPrefix(arg, "-") {
			varName = arg
		}
	}
	
	if varName != "" {
		// 通过环境变量传递关联数组声明信息
		if assocArray {
			env["__WBASH_DECLARE_ASSOC__"] = varName
		} else {
			env["__WBASH_DECLARE_VAR__"] = varName
		}
	}
	
	return nil
}

// testCmd 测试条件（test命令和[命令）
func testCmd(args []string, env map[string]string) error {
	// 处理 [ 命令，需要移除结尾的 ]
	if len(args) > 0 && args[len(args)-1] == "]" {
		args = args[:len(args)-1]
	}
	
	if len(args) == 0 {
		return fmt.Errorf("test: 缺少参数")
	}
	
	// 解析测试表达式
	result, err := evaluateTestExpression(args)
	if err != nil {
		return err
	}
	
	if !result {
		return fmt.Errorf("test failed")
	}
	
	return nil
}

// evaluateTestExpression 计算测试表达式
func evaluateTestExpression(args []string) (bool, error) {
	if len(args) == 0 {
		return false, fmt.Errorf("test: 缺少参数")
	}
	
	// 单参数：检查字符串是否非空
	if len(args) == 1 {
		return args[0] != "", nil
	}
	
	// 两参数：文件测试或字符串测试
	if len(args) == 2 {
		op := args[0]
		value := args[1]
		
		// 字符串测试
		if op == "-n" {
			return value != "", nil
		}
		if op == "-z" {
			return value == "", nil
		}
		
		// 文件测试
		switch op {
		case "-f":
			return testFile(value, func(info os.FileInfo) bool {
				return !info.IsDir()
			})
		case "-d":
			return testFile(value, func(info os.FileInfo) bool {
				return info.IsDir()
			})
		case "-e":
			return testFile(value, func(info os.FileInfo) bool {
				return true
			})
		case "-r":
			return testFile(value, func(info os.FileInfo) bool {
				// 简化：检查文件是否存在
				return true
			})
		case "-w":
			return testFile(value, func(info os.FileInfo) bool {
				// 简化：检查文件是否存在
				return true
			})
		case "-x":
			return testFile(value, func(info os.FileInfo) bool {
				// 简化：检查文件是否存在
				return true
			})
		}
		
		// 默认：检查第一个参数是否非空
		return args[0] != "", nil
	}
	
	// 三参数：二元操作
	if len(args) == 3 {
		left := args[0]
		op := args[1]
		right := args[2]
		
		switch op {
		case "=":
			return left == right, nil
		case "!=":
			return left != right, nil
		case "-eq":
			return compareNumbers(left, right, "==")
		case "-ne":
			return compareNumbers(left, right, "!=")
		case "-lt":
			return compareNumbers(left, right, "<")
		case "-le":
			return compareNumbers(left, right, "<=")
		case "-gt":
			return compareNumbers(left, right, ">")
		case "-ge":
			return compareNumbers(left, right, ">=")
		}
	}
	
	return false, fmt.Errorf("test: 不支持的表达式")
}

// testFile 测试文件
func testFile(path string, testFunc func(os.FileInfo) bool) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, nil // 文件不存在，返回false
	}
	return testFunc(info), nil
}

// compareNumbers 比较数字
func compareNumbers(left, right, op string) (bool, error) {
	leftNum, err1 := strconv.ParseInt(left, 10, 64)
	rightNum, err2 := strconv.ParseInt(right, 10, 64)
	
	if err1 != nil || err2 != nil {
		// 如果无法解析为数字，进行字符串比较
		switch op {
		case "==":
			return left == right, nil
		case "!=":
			return left != right, nil
		case "<":
			return left < right, nil
		case "<=":
			return left <= right, nil
		case ">":
			return left > right, nil
		case ">=":
			return left >= right, nil
		}
		return false, nil
	}
	
	switch op {
	case "==":
		return leftNum == rightNum, nil
	case "!=":
		return leftNum != rightNum, nil
	case "<":
		return leftNum < rightNum, nil
	case "<=":
		return leftNum <= rightNum, nil
	case ">":
		return leftNum > rightNum, nil
	case ">=":
		return leftNum >= rightNum, nil
	}
	
	return false, nil
}

// head 显示文件的前几行
func head(args []string, env map[string]string) error {
	n := 10 // 默认显示10行
	files := []string{}
	
	// 解析参数
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// 解析 -n 选项
			if strings.HasPrefix(arg, "-n") {
				if len(arg) > 2 {
					// -n5 格式
					if num, err := strconv.Atoi(arg[2:]); err == nil {
						n = num
					}
				} else if i+1 < len(args) {
					// -n 5 格式
					if num, err := strconv.Atoi(args[i+1]); err == nil {
						n = num
						i++ // 跳过下一个参数
					}
				}
			} else if num, err := strconv.Atoi(arg[1:]); err == nil {
				// -5 格式
				n = num
			}
		} else {
			files = append(files, arg)
		}
		i++
	}
	
	// 如果没有指定文件，从stdin读取
	if len(files) == 0 {
		return headFromStdin(n)
	}
	
	// 处理多个文件
	for i, file := range files {
		if len(files) > 1 {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("==> %s <==\n", file)
		}
		
		if err := headFromFile(file, n); err != nil {
			return err
		}
	}
	
	return nil
}

// headFromFile 从文件读取前n行
func headFromFile(filename string, n int) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("head: %v", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineCount := 0
	
	for scanner.Scan() && lineCount < n {
		fmt.Println(scanner.Text())
		lineCount++
	}
	
	return scanner.Err()
}

// headFromStdin 从stdin读取前n行
func headFromStdin(n int) error {
	scanner := bufio.NewScanner(os.Stdin)
	lineCount := 0
	
	for scanner.Scan() && lineCount < n {
		fmt.Println(scanner.Text())
		lineCount++
	}
	
	return scanner.Err()
}

// tail 显示文件的后几行
func tail(args []string, env map[string]string) error {
	n := 10 // 默认显示10行
	files := []string{}
	
	// 解析参数
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// 解析 -n 选项
			if strings.HasPrefix(arg, "-n") {
				if len(arg) > 2 {
					// -n5 格式
					if num, err := strconv.Atoi(arg[2:]); err == nil {
						n = num
					}
				} else if i+1 < len(args) {
					// -n 5 格式
					if num, err := strconv.Atoi(args[i+1]); err == nil {
						n = num
						i++ // 跳过下一个参数
					}
				}
			} else if num, err := strconv.Atoi(arg[1:]); err == nil {
				// -5 格式
				n = num
			}
		} else {
			files = append(files, arg)
		}
		i++
	}
	
	// 如果没有指定文件，从stdin读取
	if len(files) == 0 {
		return tailFromStdin(n)
	}
	
	// 处理多个文件
	for i, file := range files {
		if len(files) > 1 {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("==> %s <==\n", file)
		}
		
		if err := tailFromFile(file, n); err != nil {
			return err
		}
	}
	
	return nil
}

// tailFromFile 从文件读取后n行
func tailFromFile(filename string, n int) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("tail: %v", err)
	}
	defer file.Close()
	
	// 读取所有行
	scanner := bufio.NewScanner(file)
	lines := []string{}
	
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	if err := scanner.Err(); err != nil {
		return err
	}
	
	// 显示最后n行
	start := len(lines) - n
	if start < 0 {
		start = 0
	}
	
	for i := start; i < len(lines); i++ {
		fmt.Println(lines[i])
	}
	
	return nil
}

// tailFromStdin 从stdin读取后n行（简化实现，使用缓冲区）
func tailFromStdin(n int) error {
	scanner := bufio.NewScanner(os.Stdin)
	lines := []string{}
	
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		// 只保留最后n行
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	
	// 显示所有行
	for _, line := range lines {
		fmt.Println(line)
	}
	
	return scanner.Err()
}

// wc 统计行数、字数、字符数
func wc(args []string, env map[string]string) error {
	showLines := true
	showWords := true
	showChars := true
	showBytes := false
	files := []string{}
	
	// 解析参数
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// 解析选项
			for _, ch := range arg[1:] {
				switch ch {
				case 'l':
					showLines = true
					showWords = false
					showChars = false
					showBytes = false
				case 'w':
					showWords = true
					showLines = false
					showChars = false
					showBytes = false
				case 'c':
					showBytes = true
					showLines = false
					showWords = false
					showChars = false
				case 'm':
					showChars = true
					showLines = false
					showWords = false
					showBytes = false
				}
			}
		} else {
			files = append(files, arg)
		}
		i++
	}
	
	// 如果没有指定文件，从stdin读取
	if len(files) == 0 {
		return wcFromStdin(showLines, showWords, showChars, showBytes, "")
	}
	
	// 处理多个文件
	totalLines := int64(0)
	totalWords := int64(0)
	totalChars := int64(0)
	totalBytes := int64(0)
	
	for _, file := range files {
		lines, words, chars, bytes, err := wcFromFile(file, showLines, showWords, showChars, showBytes)
		if err != nil {
			return err
		}
		
		// 显示统计结果
		wcPrint(showLines, showWords, showChars, showBytes, lines, words, chars, bytes, file)
		
		totalLines += lines
		totalWords += words
		totalChars += chars
		totalBytes += bytes
	}
	
	// 如果有多个文件，显示总计
	if len(files) > 1 {
		wcPrint(showLines, showWords, showChars, showBytes, totalLines, totalWords, totalChars, totalBytes, "total")
	}
	
	return nil
}

// wcFromFile 统计文件
func wcFromFile(filename string, showLines, showWords, showChars, showBytes bool) (int64, int64, int64, int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("wc: %v", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lines := int64(0)
	words := int64(0)
	chars := int64(0)
	bytes := int64(0)
	
	for scanner.Scan() {
		line := scanner.Text()
		lines++
		words += int64(len(strings.Fields(line)))
		chars += int64(len(line)) + 1 // +1 for newline
		bytes += int64(len(line)) + 1
	}
	
	if err := scanner.Err(); err != nil {
		return 0, 0, 0, 0, err
	}
	
	return lines, words, chars, bytes, nil
}

// wcFromStdin 从stdin统计
func wcFromStdin(showLines, showWords, showChars, showBytes bool, filename string) error {
	scanner := bufio.NewScanner(os.Stdin)
	lines := int64(0)
	words := int64(0)
	chars := int64(0)
	bytes := int64(0)
	
	for scanner.Scan() {
		line := scanner.Text()
		lines++
		words += int64(len(strings.Fields(line)))
		chars += int64(len(line)) + 1 // +1 for newline
		bytes += int64(len(line)) + 1
	}
	
	if err := scanner.Err(); err != nil {
		return err
	}
	
	wcPrint(showLines, showWords, showChars, showBytes, lines, words, chars, bytes, filename)
	return nil
}

// wcPrint 打印统计结果
func wcPrint(showLines, showWords, showChars, showBytes bool, lines, words, chars, bytes int64, filename string) {
	parts := []string{}
	
	if showLines {
		parts = append(parts, fmt.Sprintf("%8d", lines))
	}
	if showWords {
		parts = append(parts, fmt.Sprintf("%8d", words))
	}
	if showBytes {
		parts = append(parts, fmt.Sprintf("%8d", bytes))
	}
	if showChars {
		parts = append(parts, fmt.Sprintf("%8d", chars))
	}
	
	// 如果所有选项都关闭，默认显示所有
	if !showLines && !showWords && !showChars && !showBytes {
		parts = []string{
			fmt.Sprintf("%8d", lines),
			fmt.Sprintf("%8d", words),
			fmt.Sprintf("%8d", bytes),
		}
	}
	
	result := strings.Join(parts, " ")
	if filename != "" {
		result += " " + filename
	}
	
	fmt.Println(result)
}

// grep 文本搜索（简化版）
func grep(args []string, env map[string]string) error {
	if len(args) == 0 {
		return fmt.Errorf("grep: 缺少操作数")
	}
	
	// 解析参数
	pattern := ""
	files := []string{}
	caseInsensitive := false
	showLineNumbers := false
	showOnlyMatches := false
	
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// 解析选项
			for _, ch := range arg[1:] {
				switch ch {
				case 'i':
					caseInsensitive = true
				case 'n':
					showLineNumbers = true
				case 'o':
					showOnlyMatches = true
				}
			}
		} else if pattern == "" {
			pattern = arg
		} else {
			files = append(files, arg)
		}
		i++
	}
	
	if pattern == "" {
		return fmt.Errorf("grep: 缺少模式")
	}
	
	// 如果没有指定文件，从stdin读取
	if len(files) == 0 {
		return grepFromStdin(pattern, caseInsensitive, showLineNumbers, showOnlyMatches, "")
	}
	
	// 处理多个文件
	for i, file := range files {
		if len(files) > 1 {
			if i > 0 {
				fmt.Println()
			}
		}
		
		if err := grepFromFile(file, pattern, caseInsensitive, showLineNumbers, showOnlyMatches, len(files) > 1); err != nil {
			return err
		}
	}
	
	return nil
}

// grepFromFile 从文件搜索
func grepFromFile(filename string, pattern string, caseInsensitive, showLineNumbers, showOnlyMatches, showFilename bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("grep: %v", err)
	}
	defer file.Close()
	
	searchPattern := pattern
	if caseInsensitive {
		searchPattern = strings.ToLower(pattern)
	}
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		searchLine := line
		if caseInsensitive {
			searchLine = strings.ToLower(line)
		}
		
		if strings.Contains(searchLine, searchPattern) {
			prefix := ""
			if showFilename {
				prefix = filename + ":"
			}
			if showLineNumbers {
				if prefix != "" {
					prefix += fmt.Sprintf("%d:", lineNum)
				} else {
					prefix = fmt.Sprintf("%d:", lineNum)
				}
			}
			
			if showOnlyMatches {
				// 只显示匹配的部分
				start := strings.Index(searchLine, searchPattern)
				if start >= 0 {
					match := line[start : start+len(pattern)]
					if prefix != "" {
						fmt.Printf("%s%s\n", prefix, match)
					} else {
						fmt.Println(match)
					}
				}
			} else {
				if prefix != "" {
					fmt.Printf("%s%s\n", prefix, line)
				} else {
					fmt.Println(line)
				}
			}
		}
	}
	
	return scanner.Err()
}

// grepFromStdin 从stdin搜索
func grepFromStdin(pattern string, caseInsensitive, showLineNumbers, showOnlyMatches bool, filename string) error {
	searchPattern := pattern
	if caseInsensitive {
		searchPattern = strings.ToLower(pattern)
	}
	
	scanner := bufio.NewScanner(os.Stdin)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		searchLine := line
		if caseInsensitive {
			searchLine = strings.ToLower(line)
		}
		
		if strings.Contains(searchLine, searchPattern) {
			prefix := ""
			if showLineNumbers {
				prefix = fmt.Sprintf("%d:", lineNum)
			}
			
			if showOnlyMatches {
				// 只显示匹配的部分
				start := strings.Index(searchLine, searchPattern)
				if start >= 0 {
					match := line[start : start+len(pattern)]
					if prefix != "" {
						fmt.Printf("%s%s\n", prefix, match)
					} else {
						fmt.Println(match)
					}
				}
			} else {
				if prefix != "" {
					fmt.Printf("%s%s\n", prefix, line)
				} else {
					fmt.Println(line)
				}
			}
		}
	}
	
	return scanner.Err()
}

// sortCmd 排序（简化版）
func sortCmd(args []string, env map[string]string) error {
	reverse := false
	numeric := false
	unique := false
	files := []string{}
	
	// 解析参数
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// 解析选项
			for _, ch := range arg[1:] {
				switch ch {
				case 'r':
					reverse = true
				case 'n':
					numeric = true
				case 'u':
					unique = true
				}
			}
		} else {
			files = append(files, arg)
		}
		i++
	}
	
	// 如果没有指定文件，从stdin读取
	if len(files) == 0 {
		return sortFromStdin(reverse, numeric, unique)
	}
	
	// 处理多个文件
	allLines := []string{}
	for _, file := range files {
		lines, err := readLinesFromFile(file)
		if err != nil {
			return fmt.Errorf("sort: %v", err)
		}
		allLines = append(allLines, lines...)
	}
	
	// 排序
	sortedLines := sortLines(allLines, reverse, numeric, unique)
	
	// 输出
	for _, line := range sortedLines {
		fmt.Println(line)
	}
	
	return nil
}

// sortFromStdin 从stdin排序
func sortFromStdin(reverse, numeric, unique bool) error {
	scanner := bufio.NewScanner(os.Stdin)
	lines := []string{}
	
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	if err := scanner.Err(); err != nil {
		return err
	}
	
	// 排序
	sortedLines := sortLines(lines, reverse, numeric, unique)
	
	// 输出
	for _, line := range sortedLines {
		fmt.Println(line)
	}
	
	return nil
}

// readLinesFromFile 从文件读取所有行
func readLinesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lines := []string{}
	
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	return lines, scanner.Err()
}

// sortLines 排序行
func sortLines(lines []string, reverse, numeric, unique bool) []string {
	// 去重
	if unique {
		seen := make(map[string]bool)
		uniqueLines := []string{}
		for _, line := range lines {
			if !seen[line] {
				seen[line] = true
				uniqueLines = append(uniqueLines, line)
			}
		}
		lines = uniqueLines
	}
	
	// 排序
	if numeric {
		sort.Slice(lines, func(i, j int) bool {
			numI, errI := strconv.ParseFloat(lines[i], 64)
			numJ, errJ := strconv.ParseFloat(lines[j], 64)
			
			if errI != nil && errJ != nil {
				// 都不是数字，按字符串比较
				if reverse {
					return lines[i] > lines[j]
				}
				return lines[i] < lines[j]
			}
			if errI != nil {
				// i不是数字，排在后面
				return false
			}
			if errJ != nil {
				// j不是数字，排在后面
				return true
			}
			
			// 都是数字，按数值比较
			if reverse {
				return numI > numJ
			}
			return numI < numJ
		})
	} else {
		sort.Slice(lines, func(i, j int) bool {
			if reverse {
				return lines[i] > lines[j]
			}
			return lines[i] < lines[j]
		})
	}
	
	return lines
}

// uniq 去重（简化版）
func uniq(args []string, env map[string]string) error {
	count := false
	showOnlyDuplicates := false
	ignoreCase := false
	files := []string{}
	
	// 解析参数
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// 解析选项
			for _, ch := range arg[1:] {
				switch ch {
				case 'c':
					count = true
				case 'd':
					showOnlyDuplicates = true
				case 'i':
					ignoreCase = true
				}
			}
		} else {
			files = append(files, arg)
		}
		i++
	}
	
	// 如果没有指定文件，从stdin读取
	if len(files) == 0 {
		return uniqFromStdin(count, showOnlyDuplicates, ignoreCase)
	}
	
	// 处理多个文件
	for _, file := range files {
		if err := uniqFromFile(file, count, showOnlyDuplicates, ignoreCase); err != nil {
			return err
		}
	}
	
	return nil
}

// uniqFromFile 从文件去重
func uniqFromFile(filename string, count, showOnlyDuplicates, ignoreCase bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("uniq: %v", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	prevLine := ""
	prevLineCount := 0
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		compareLine := line
		comparePrev := prevLine
		
		if ignoreCase {
			compareLine = strings.ToLower(line)
			comparePrev = strings.ToLower(prevLine)
		}
		
		if compareLine == comparePrev {
			// 重复行
			prevLineCount++
		} else {
			// 新行，先输出前一行（如果有）
			if prevLine != "" {
				if !showOnlyDuplicates || prevLineCount > 0 {
					output := ""
					if count {
						output = fmt.Sprintf("%7d ", prevLineCount+1)
					}
					output += prevLine
					fmt.Println(output)
				}
			}
			prevLine = line
			prevLineCount = 0
		}
	}
	
	// 输出最后一行
	if prevLine != "" {
		if !showOnlyDuplicates || prevLineCount > 0 {
			output := ""
			if count {
				output = fmt.Sprintf("%7d ", prevLineCount+1)
			}
			output += prevLine
			fmt.Println(output)
		}
	}
	
	return scanner.Err()
}

// uniqFromStdin 从stdin去重
func uniqFromStdin(count, showOnlyDuplicates, ignoreCase bool) error {
	scanner := bufio.NewScanner(os.Stdin)
	prevLine := ""
	prevLineCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		compareLine := line
		comparePrev := prevLine
		
		if ignoreCase {
			compareLine = strings.ToLower(line)
			comparePrev = strings.ToLower(prevLine)
		}
		
		if compareLine == comparePrev {
			// 重复行
			prevLineCount++
		} else {
			// 新行，先输出前一行（如果有）
			if prevLine != "" {
				if !showOnlyDuplicates || prevLineCount > 0 {
					output := ""
					if count {
						output = fmt.Sprintf("%7d ", prevLineCount+1)
					}
					output += prevLine
					fmt.Println(output)
				}
			}
			prevLine = line
			prevLineCount = 0
		}
	}
	
	// 输出最后一行
	if prevLine != "" {
		if !showOnlyDuplicates || prevLineCount > 0 {
			output := ""
			if count {
				output = fmt.Sprintf("%7d ", prevLineCount+1)
			}
			output += prevLine
			fmt.Println(output)
		}
	}
	
	return scanner.Err()
}

