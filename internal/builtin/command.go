package builtin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// command 执行命令或显示命令信息
// command [-pVv] command [arg ...]
// -v: 显示命令路径或类型
// -V: 显示详细描述
// -p: 使用标准PATH
func command(args []string, env map[string]string) error {
	if len(args) == 0 {
		return nil
	}

	// 解析选项
	verbose := false
	useStandardPath := false
	commandArgs := args

	// 检查选项
	for len(commandArgs) > 0 && strings.HasPrefix(commandArgs[0], "-") {
		opt := commandArgs[0]
		if opt == "-v" {
			verbose = true
			commandArgs = commandArgs[1:]
		} else if opt == "-V" {
			verbose = true // -V 也使用 verbose 模式，但输出更详细
			commandArgs = commandArgs[1:]
		} else if opt == "-p" {
			useStandardPath = true
			commandArgs = commandArgs[1:]
		} else {
			// 未知选项，可能是命令名
			break
		}
	}

	if len(commandArgs) == 0 {
		return nil
	}

	// 如果是 -v 或 -V 模式，显示命令信息
	if verbose {
		for _, cmdName := range commandArgs {
			// 检查是否为内置命令
			if _, ok := builtins[cmdName]; ok {
				if verbose {
					fmt.Printf("%s\n", cmdName)
				}
				continue
			}

			// 检查PATH环境变量
			pathEnv := os.Getenv("PATH")
			if useStandardPath {
				// 使用标准PATH（简化实现，使用当前PATH）
				pathEnv = os.Getenv("PATH")
			}

			if pathEnv != "" {
				paths := strings.Split(pathEnv, ":")
				if len(paths) == 0 || (len(paths) == 1 && paths[0] == "") {
					// Windows使用分号分隔
					paths = strings.Split(pathEnv, ";")
				}

				found := false
				for _, path := range paths {
					if path == "" {
						continue
					}
					fullPath := filepath.Join(path, cmdName)
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

				if found {
					continue
				}
			}

			// 命令未找到，不输出（command -v 的行为）
		}
		return nil
	}

	// 执行命令（忽略shell函数，直接执行外部命令或内置命令）
	if len(commandArgs) == 0 {
		return nil
	}

	cmdName := commandArgs[0]
	cmdArgs := commandArgs[1:]

	// 检查是否为内置命令
	if builtinFunc, ok := builtins[cmdName]; ok {
		return builtinFunc(cmdArgs, env)
	}

	// 执行外部命令
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Env = getEnvArray(env)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getEnvArray 将环境变量映射转换为数组
func getEnvArray(env map[string]string) []string {
	var result []string
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}




