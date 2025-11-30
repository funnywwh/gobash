package shell

import (
	"os"
	"path/filepath"
	"strings"
)

// Completer 实现readline的自动补全接口
type Completer struct {
	shell *Shell
}

// NewCompleter 创建新的补全器
func NewCompleter(s *Shell) *Completer {
	return &Completer{shell: s}
}

// Do 执行自动补全
func (c *Completer) Do(line []rune, pos int) (newLine [][]rune, length int) {
	// 将rune数组转换为字符串
	lineStr := string(line[:pos])
	
	// 分割命令行
	parts := strings.Fields(lineStr)
	if len(parts) == 0 {
		// 空行，补全命令
		return c.completeCommands("")
	}
	
	// 获取当前正在输入的部分
	current := parts[len(parts)-1]
	
	// 检查是否在输入命令（第一个词）
	if len(parts) == 1 {
		// 补全命令（内置命令、别名、外部命令）
		return c.completeCommands(current)
	}
	
	// 检查是否是变量（以$开头）
	if strings.HasPrefix(current, "$") {
		return c.completeVariables(current)
	}
	
	// 否则补全文件名
	return c.completeFiles(current)
}

// completeCommands 补全命令
func (c *Completer) completeCommands(prefix string) ([][]rune, int) {
	var matches [][]rune
	
	// 1. 内置命令
	builtins := []string{
		"cd", "pwd", "echo", "exit", "export", "unset", "env", "set",
		"ls", "cat", "mkdir", "rmdir", "rm", "touch", "clear",
		"alias", "unalias", "history", "which", "type", "true", "false",
		"test", "[", "head", "tail", "wc", "grep", "sort", "uniq", "cut",
	}
	
	for _, cmd := range builtins {
		if strings.HasPrefix(cmd, prefix) {
			// 只返回需要补全的部分（去掉已输入的前缀）
			suffix := cmd[len(prefix):]
			matches = append(matches, []rune(suffix))
		}
	}
	
	// 2. 别名
	for alias := range c.shell.aliases {
		if strings.HasPrefix(alias, prefix) {
			suffix := alias[len(prefix):]
			matches = append(matches, []rune(suffix))
		}
	}
	
	// 3. PATH中的外部命令（简化版，只检查常见命令）
	pathEnv := os.Getenv("PATH")
	if pathEnv != "" {
		paths := strings.Split(pathEnv, ":")
		if len(paths) == 0 {
			paths = strings.Split(pathEnv, ";")
		}
		
		seen := make(map[string]bool)
		for _, path := range paths {
			if path == "" {
				continue
			}
			entries, err := os.ReadDir(path)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				// 移除.exe扩展名（Windows）
				if strings.HasSuffix(name, ".exe") {
					name = name[:len(name)-4]
				}
				if strings.HasPrefix(name, prefix) && !seen[name] {
					seen[name] = true
					suffix := name[len(prefix):]
					matches = append(matches, []rune(suffix))
				}
			}
		}
	}
	
	return matches, len(prefix)
}

// completeVariables 补全环境变量
func (c *Completer) completeVariables(prefix string) ([][]rune, int) {
	var matches [][]rune
	
	// 移除$前缀
	varName := strings.TrimPrefix(prefix, "$")
	varName = strings.TrimPrefix(varName, "{")
	
	// 检查系统环境变量
	seen := make(map[string]bool)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) > 0 {
			key := parts[0]
			if strings.HasPrefix(key, varName) && !seen[key] {
				seen[key] = true
				// 只返回需要补全的部分（去掉已输入的变量名前缀）
				suffix := key[len(varName):]
				if strings.HasPrefix(prefix, "${") {
					// 如果原始前缀是 ${VAR，返回 VAR的剩余部分}
					matches = append(matches, []rune(suffix+"}"))
				} else {
					// 如果原始前缀是 $VAR，返回 VAR的剩余部分
					matches = append(matches, []rune(suffix))
				}
			}
		}
	}
	
	return matches, len(prefix)
}

// completeFiles 补全文件名
func (c *Completer) completeFiles(prefix string) ([][]rune, int) {
	var matches [][]rune
	
	// 处理路径
	dir := "."
	pattern := prefix
	
	// 如果包含路径分隔符，分离目录和文件名
	if strings.Contains(prefix, "/") || strings.Contains(prefix, "\\") {
		dir = filepath.Dir(prefix)
		pattern = filepath.Base(prefix)
		if dir == "." {
			dir = ""
		}
	}
	
	if dir == "" {
		dir = "."
	}
	
	// 读取目录
	entries, err := os.ReadDir(dir)
	if err != nil {
		return matches, len(prefix)
	}
	
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, pattern) {
			// 只返回需要补全的部分（去掉已输入的文件名前缀）
			suffix := name[len(pattern):]
			
			// 如果是目录，添加路径分隔符
			if entry.IsDir() {
				if strings.Contains(prefix, "\\") {
					suffix += "\\"
				} else {
					suffix += "/"
				}
			}
			matches = append(matches, []rune(suffix))
		}
	}
	
	// 返回需要替换的字符数（整个前缀）
	return matches, len(prefix)
}

