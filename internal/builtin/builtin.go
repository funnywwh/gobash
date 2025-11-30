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
}

// GetBuiltins 获取所有内置命令
func GetBuiltins() map[string]BuiltinFunc {
	return builtins
}

// cd 改变目录
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

// pwd 打印当前工作目录
func pwd(args []string, env map[string]string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Println(dir)
	return nil
}

// echo 打印参数
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

// set 设置shell选项（简化版）
func set(args []string, env map[string]string) error {
	if len(args) == 0 {
		// 显示所有变量
		for k, v := range env {
			fmt.Printf("%s=%s\n", k, v)
		}
		return nil
	}
	// TODO: 实现set选项
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
	for i, arg := range args {
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
	for i, arg := range args {
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

