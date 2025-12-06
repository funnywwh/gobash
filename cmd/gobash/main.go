package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"gobash/internal/builtin"
	"gobash/internal/executor"
	"gobash/internal/shell"
)

func main() {
	var scriptPath = flag.String("c", "", "执行命令字符串")
	var scriptFile = flag.String("f", "", "执行脚本文件")
	flag.Parse()

	sh := shell.New()

	// 执行命令字符串
	if *scriptPath != "" {
		if err := sh.ExecuteReader(strings.NewReader(*scriptPath)); err != nil {
			// 检查是否是 exit 命令
			if exitErr, ok := err.(*builtin.ExitError); ok {
				os.Exit(exitErr.Code)
			}
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 执行脚本文件
	if *scriptFile != "" {
		// 获取 -f 之后的参数作为脚本参数
		scriptArgs := flag.Args()
		if err := sh.ExecuteScript(*scriptFile, scriptArgs...); err != nil {
			// 检查是否是 exit 命令
			if exitErr, ok := err.(*builtin.ExitError); ok {
				os.Exit(exitErr.Code)
			}
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 如果有命令行参数，作为脚本执行
	if len(os.Args) > 1 && os.Args[1][0] != '-' {
		// 收集所有脚本文件（支持通配符和多个文件）
		var scriptFiles []string
		var scriptArgs []string
		argsStartIndex := -1
		
		// 遍历所有非选项参数，区分脚本文件和脚本参数
		for i := 1; i < len(os.Args); i++ {
			arg := os.Args[i]
			
			// 检查是否包含通配符（如 *.sh）
			if strings.Contains(arg, "*") || strings.Contains(arg, "?") {
				// 使用通配符匹配文件
				matches, err := filepath.Glob(arg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "错误: 通配符匹配失败 %s: %v\n", arg, err)
					os.Exit(1)
				}
				if len(matches) == 0 {
					// 通配符没有匹配到文件，可能是脚本参数
					if argsStartIndex == -1 {
						argsStartIndex = i
					}
					scriptArgs = append(scriptArgs, arg)
					continue
				}
				scriptFiles = append(scriptFiles, matches...)
			} else {
				// 检查是否是实际存在的文件
				info, err := os.Stat(arg)
				if err == nil && !info.IsDir() {
					// 是文件，添加到脚本文件列表
					scriptFiles = append(scriptFiles, arg)
				} else {
					// 不是文件或不存在，作为脚本参数
					if argsStartIndex == -1 {
						argsStartIndex = i
					}
					scriptArgs = append(scriptArgs, arg)
				}
			}
		}
		
		// 如果没有找到任何文件，退出
		if len(scriptFiles) == 0 {
			fmt.Fprintf(os.Stderr, "错误: 没有找到要执行的脚本文件\n")
			os.Exit(1)
		}
		
		// 去重（防止重复执行）
		seen := make(map[string]bool)
		uniqueFiles := []string{}
		for _, file := range scriptFiles {
			if !seen[file] {
				seen[file] = true
				uniqueFiles = append(uniqueFiles, file)
			}
		}
		scriptFiles = uniqueFiles
		
		// 排序文件名，确保执行顺序一致
		sort.Strings(scriptFiles)
		
		// 调试输出：显示匹配到的文件数
		if len(scriptFiles) > 1 {
			fmt.Fprintf(os.Stderr, "找到 %d 个脚本文件，开始执行...\n", len(scriptFiles))
		}
		
		// 依次执行所有脚本文件
		hasError := false
		for i, scriptPath := range scriptFiles {
			// 检查是否是文件
			info, err := os.Stat(scriptPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "警告: 跳过 %s: %v\n", scriptPath, err)
				hasError = true
				continue
			}
			if info.IsDir() {
				fmt.Fprintf(os.Stderr, "警告: 跳过目录 %s\n", scriptPath)
				continue
			}
			
			// 如果是多个脚本，在执行前输出分隔线
			if len(scriptFiles) > 1 {
				// 获取脚本文件名（不包含路径）
				scriptName := filepath.Base(scriptPath)
				// 添加索引号（从1开始）
				index := i + 1
				separator := fmt.Sprintf("========================%d: %s=========================", index, scriptName)
				if i > 0 {
					// 在脚本之间添加空行
					fmt.Println()
				}
				fmt.Println(separator)
			}
			
			// 执行脚本，传递脚本参数（只有第一个脚本接收参数）
			// 使用 goroutine 来执行脚本，避免卡死
			scriptErr := make(chan error, 1)
			go func() {
				// 只有第一个脚本接收参数
				if i == 0 {
					scriptErr <- sh.ExecuteScript(scriptPath, scriptArgs...)
				} else {
					scriptErr <- sh.ExecuteScript(scriptPath)
				}
			}()
			
			// 设置超时（300秒），如果脚本卡死则跳过
			select {
			case err := <-scriptErr:
				if err != nil {
					// 检查是否是 exit 命令或脚本退出错误
					if exitErr, ok := err.(*builtin.ExitError); ok {
						// exit 命令是正常的脚本退出，记录退出码但继续执行下一个脚本
						if exitErr.Code != 0 {
							hasError = true
						}
						// 不输出错误信息，因为 exit 是正常的脚本退出
					} else if scriptExitErr, ok := err.(*executor.ScriptExitError); ok {
						// 脚本退出错误（由于 set -e），记录退出码但继续执行下一个脚本
						if scriptExitErr.Code != 0 {
							hasError = true
						}
						// 不输出错误信息，因为这是正常的脚本退出
					} else {
						fmt.Fprintf(os.Stderr, "错误: 执行脚本 %s 失败: %v\n", scriptPath, err)
						hasError = true
					}
				}
			case <-time.After(300 * time.Second):
				fmt.Fprintf(os.Stderr, "警告: 脚本 %s 执行超时（300秒），跳过\n", scriptPath)
				hasError = true
				// 注意：goroutine 可能仍在运行，但我们已经继续执行下一个脚本了
			}
		}
		
		// 所有脚本执行完成后，如果有错误则退出
		if hasError {
			os.Exit(1)
		}
		return
	}

	// 交互式模式
	sh.Run()
}

