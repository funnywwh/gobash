// Package executor 提供变量展开功能
package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"gobash/internal/parser"
)

// ExpandFlags 展开标志
type ExpandFlags int

const (
	ExpandFlagQuoted ExpandFlags = 1 << iota // 引号内展开
	ExpandFlagNoSplit                        // 不进行单词分割
	ExpandFlagNoGlob                         // 不进行路径名展开
	ExpandFlagNoTilde                        // 不进行波浪号展开
)

// ExpandContext 展开上下文
type ExpandContext struct {
	Env      map[string]string // 环境变量
	Flags    ExpandFlags       // 展开标志
	IFS      string            // 内部字段分隔符
	PositionalArgs []string    // 位置参数 ($1, $2, ...)
}

// expandParamExpression 展开参数表达式
// 例如：${VAR:-default}, ${VAR#pattern}, ${VAR:offset:length} 等
func (e *Executor) expandParamExpression(pe *parser.ParamExpandExpression) (string, error) {
	varName := pe.VarName
	op := pe.Op
	word := pe.Word
	
	// 获取变量值
	varValue := e.env[varName]
	if varValue == "" {
		varValue = os.Getenv(varName)
	}
	
	// 处理数组访问 ${arr[0]} 或 ${arr[key]} 或 ${arr[@]} 或 ${arr[*]}
	if strings.HasPrefix(word, "[") {
		// 解析数组索引或展开符号
		// 格式：[0], [key], [@], [*]
		idxEnd := strings.Index(word, "]")
		if idxEnd == -1 {
			return "", fmt.Errorf("未闭合的数组索引: %s", word)
		}
		indexStr := word[1:idxEnd] // 去掉 [ 和 ]
		
		// 处理数组展开 ${arr[@]} 或 ${arr[*]}
		if indexStr == "@" || indexStr == "*" {
			return e.expandArray(varName, indexStr == "@"), nil
		}
		
		// 处理数组元素访问 ${arr[0]} 或 ${arr[key]}
		return e.getArrayElement(varName + word), nil
	}
	
	// 根据操作符进行展开
	switch op {
	case "":
		// 简单的变量展开 ${VAR}
		return varValue, nil
		
	case ":-":
		// ${VAR:-word} - 如果 VAR 未设置或为空，使用 word
		if varValue == "" {
			return e.expandWord(word), nil
		}
		return varValue, nil
		
	case ":=":
		// ${VAR:=word} - 如果 VAR 未设置或为空，将 word 赋值给 VAR
		if varValue == "" {
			expandedWord := e.expandWord(word)
			e.env[varName] = expandedWord
			os.Setenv(varName, expandedWord)
			return expandedWord, nil
		}
		return varValue, nil
		
	case ":?":
		// ${VAR:?word} - 如果 VAR 未设置或为空，显示错误并退出
		if varValue == "" {
			errorMsg := word
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("%s: parameter null or not set", varName)
		} else {
			errorMsg = e.expandWord(errorMsg)
		}
		return "", fmt.Errorf("%s", errorMsg)
		}
		return varValue, nil
		
	case ":+":
		// ${VAR:+word} - 如果 VAR 已设置且非空，使用 word，否则为空
		if varValue != "" {
			return e.expandWord(word), nil
		}
		return "", nil
		
	case "#":
		// ${VAR#pattern} - 删除最短匹配前缀
		if varValue == "" {
			return "", nil
		}
		pattern := e.expandWord(word)
		if strings.HasPrefix(varValue, pattern) {
			return varValue[len(pattern):], nil
		}
		return varValue, nil
		
	case "##":
		// ${VAR##pattern} - 删除最长匹配前缀
		if varValue == "" {
			return "", nil
		}
		pattern := e.expandWord(word)
		// 使用正则表达式匹配最长前缀
		re := regexp.MustCompile("^" + regexp.QuoteMeta(pattern))
		return re.ReplaceAllString(varValue, ""), nil
		
	case "%":
		// ${VAR%pattern} - 删除最短匹配后缀
		if varValue == "" {
			return "", nil
		}
		pattern := e.expandWord(word)
		if strings.HasSuffix(varValue, pattern) {
			return varValue[:len(varValue)-len(pattern)], nil
		}
		return varValue, nil
		
	case "%%":
		// ${VAR%%pattern} - 删除最长匹配后缀
		if varValue == "" {
			return "", nil
		}
		pattern := e.expandWord(word)
		// 使用正则表达式匹配最长后缀
		re := regexp.MustCompile(regexp.QuoteMeta(pattern) + "$")
		return re.ReplaceAllString(varValue, ""), nil
		
	case ":":
		// ${VAR:offset} 或 ${VAR:offset:length} - 子字符串
		if varValue == "" {
			return "", nil
		}
		parts := strings.Split(word, ":")
		if len(parts) == 1 {
			// ${VAR:offset}
			offset, err := strconv.Atoi(parts[0])
			if err != nil {
				return "", fmt.Errorf("invalid offset: %s", parts[0])
			}
			if offset < 0 {
				offset = len(varValue) + offset
			}
			if offset < 0 || offset >= len(varValue) {
				return "", nil
			}
			return varValue[offset:], nil
		} else if len(parts) == 2 {
			// ${VAR:offset:length}
			offset, err := strconv.Atoi(parts[0])
			if err != nil {
				return "", fmt.Errorf("invalid offset: %s", parts[0])
			}
			length, err := strconv.Atoi(parts[1])
			if err != nil {
				return "", fmt.Errorf("invalid length: %s", parts[1])
			}
			if offset < 0 {
				offset = len(varValue) + offset
			}
			if offset < 0 || offset >= len(varValue) {
				return "", nil
			}
			if length < 0 {
				length = len(varValue) - offset + length
			}
			if length <= 0 {
				return "", nil
			}
			end := offset + length
			if end > len(varValue) {
				end = len(varValue)
			}
			return varValue[offset:end], nil
		}
		return varValue, nil
		
	case "!":
		// ${!VAR} - 间接引用
		indirectVarName := varValue
		if indirectVarName == "" {
			return "", nil
		}
		indirectValue := e.env[indirectVarName]
		if indirectValue == "" {
			indirectValue = os.Getenv(indirectVarName)
		}
		return indirectValue, nil
		
	default:
		// 未知操作符，返回原值
		return varValue, nil
	}
}

// wordSplit 根据 IFS 分割单词
// 根据 bash 的行为：
// 1. 如果 IFS 未设置或为空，不进行分割（返回单个单词）
// 2. 如果 IFS 包含空白字符（空格、制表符、换行符），连续的空白字符会被压缩为一个分隔符
// 3. 如果 IFS 包含非空白字符，每个字符都是分隔符
// 4. 如果 IFS 为空字符串（但已设置），不进行分割（每个字符都是独立的单词）
func (e *Executor) wordSplit(text string) []string {
	ifs := e.env["IFS"]
	
	// 如果 IFS 未设置，使用默认值
	if ifs == "" {
		// 检查是否是未设置（nil）还是空字符串
		// 在 Go 中，如果 env["IFS"] 不存在，返回空字符串
		// 我们需要检查 os.Getenv 来区分
		if os.Getenv("IFS") == "" {
			// IFS 未设置，使用默认值 " \t\n"
			ifs = " \t\n"
		} else {
			// IFS 设置为空字符串，不进行分割
			// 每个字符都是独立的单词
			words := make([]string, 0, len(text))
			for _, r := range text {
				words = append(words, string(r))
			}
			return words
		}
	}
	
	// 检查 IFS 是否只包含空白字符
	hasWhitespace := false
	hasNonWhitespace := false
	for _, r := range ifs {
		if r == ' ' || r == '\t' || r == '\n' {
			hasWhitespace = true
		} else {
			hasNonWhitespace = true
		}
	}
	
	if hasWhitespace && !hasNonWhitespace {
		// IFS 只包含空白字符，压缩连续的空白字符
		words := []string{}
		currentWord := strings.Builder{}
		inWhitespace := false
		
		for _, r := range text {
			isWhitespace := r == ' ' || r == '\t' || r == '\n'
			if isWhitespace {
				if !inWhitespace && currentWord.Len() > 0 {
					// 遇到空白字符，保存当前单词
					words = append(words, currentWord.String())
					currentWord.Reset()
				}
				inWhitespace = true
			} else {
				if inWhitespace {
					inWhitespace = false
				}
				currentWord.WriteRune(r)
			}
		}
		
		// 添加最后一个单词（如果有）
		if currentWord.Len() > 0 {
			words = append(words, currentWord.String())
		}
		
		return words
	} else if hasNonWhitespace {
		// IFS 包含非空白字符，每个字符都是分隔符
		// 同时压缩连续的空白字符（如果 IFS 中也包含空白字符）
		words := []string{}
		currentWord := strings.Builder{}
		
		for _, r := range text {
			isSeparator := false
			for _, ifsRune := range ifs {
				if r == ifsRune {
					isSeparator = true
					break
				}
			}
			
			if isSeparator {
				// 遇到分隔符，保存当前单词
				if currentWord.Len() > 0 {
					words = append(words, currentWord.String())
					currentWord.Reset()
				}
			} else {
				currentWord.WriteRune(r)
			}
		}
		
		// 添加最后一个单词（如果有）
		if currentWord.Len() > 0 {
			words = append(words, currentWord.String())
		}
		
		return words
	}
	
	// 默认情况：不分割
	return []string{text}
}

// pathnameExpand 路径名展开（通配符）
// 根据 bash 的行为：
// 1. `*` 匹配任意数量的字符（除了 `/`）
// 2. `?` 匹配单个字符（除了 `/`）
// 3. `[...]` 匹配字符类
// 4. `**` 递归匹配（如果启用 globstar 选项）
// 5. 隐藏文件（以 `.` 开头）需要特殊处理
func (e *Executor) pathnameExpand(pattern string) []string {
	// 如果模式不包含通配符，直接返回
	if !strings.ContainsAny(pattern, "*?[") {
		return []string{pattern}
	}
	
	// 检查是否启用 globstar 选项（支持 ** 递归匹配）
	// 注意：globstar 是 shopt 选项，不是 set 选项
	// 这里简化处理，检查环境变量 GLOBSTAR 或 options["globstar"]
	globstarEnabled := false
	if e != nil {
		// 检查环境变量
		if globstar, ok := e.env["GLOBSTAR"]; ok && globstar == "1" {
			globstarEnabled = true
		}
		// 检查 options（如果通过 SetOptions 设置）
		options := e.GetOptions()
		if options != nil {
			if val, ok := options["globstar"]; ok && val {
				globstarEnabled = true
			}
		}
	}
	
	// 如果启用 globstar 且模式包含 **，使用递归匹配
	if globstarEnabled && strings.Contains(pattern, "**") {
		return e.pathnameExpandRecursive(pattern)
	}
	
	// 使用 filepath.Glob 进行匹配
	matches, err := filepath.Glob(pattern)
	if err != nil {
		// 如果出错，返回原始模式
		return []string{pattern}
	}
	
	// 如果没有匹配，bash 的行为是返回原始模式
	if len(matches) == 0 {
		return []string{pattern}
	}
	
	// 处理隐藏文件：如果模式不以 `.` 开头，不匹配隐藏文件
	if !strings.HasPrefix(pattern, ".") {
		filtered := []string{}
		for _, match := range matches {
			// 检查是否是隐藏文件
			base := filepath.Base(match)
			if !strings.HasPrefix(base, ".") {
				filtered = append(filtered, match)
			}
		}
		if len(filtered) > 0 {
			return filtered
		}
		// 如果没有非隐藏文件匹配，返回原始模式
		return []string{pattern}
	}
	
	return matches
}

// pathnameExpandRecursive 递归路径名展开（支持 **）
// ** 匹配零个或多个目录和子目录
func (e *Executor) pathnameExpandRecursive(pattern string) []string {
	// 将 ** 替换为临时标记，然后递归匹配
	// 策略：将 ** 替换为 *，然后递归遍历目录
	
	// 如果模式是 **，匹配当前目录及其所有子目录
	if pattern == "**" {
		return e.matchRecursive(".", "*")
	}
	
	// 如果模式以 **/ 开头，匹配所有目录
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:] // 跳过 "**/"
		if suffix == "" {
			return e.matchRecursive(".", "*")
		}
		return e.matchRecursive(".", suffix)
	}
	
	// 如果模式以 /** 结尾，匹配所有子目录
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3] // 移除 "/**"
		if prefix == "" {
			return e.matchRecursive(".", "*")
		}
		// 先匹配前缀，然后在每个匹配的目录中递归匹配
		prefixMatches := e.pathnameExpand(prefix)
		result := []string{}
		for _, pm := range prefixMatches {
			info, err := os.Stat(pm)
			if err == nil && info.IsDir() {
				subMatches := e.matchRecursive(pm, "*")
				result = append(result, subMatches...)
			}
			result = append(result, pm)
		}
		return result
	}
	
	// 如果模式包含 /**/，分割并递归匹配
	if strings.Contains(pattern, "/**/") {
		parts := strings.SplitN(pattern, "/**/", 2)
		prefix := parts[0]
		suffix := parts[1]
		
		// 先匹配前缀
		prefixMatches := e.pathnameExpand(prefix)
		result := []string{}
		for _, pm := range prefixMatches {
			info, err := os.Stat(pm)
			if err == nil && info.IsDir() {
				// 递归匹配后缀
				subMatches := e.matchRecursive(pm, suffix)
				result = append(result, subMatches...)
			}
			// 也检查前缀本身是否匹配完整模式
			fullPath := filepath.Join(pm, suffix)
			if matches, err := filepath.Glob(fullPath); err == nil {
				result = append(result, matches...)
			}
		}
		if len(result) > 0 {
			return result
		}
		// 如果没有匹配，返回原始模式
		return []string{pattern}
	}
	
	// 其他情况，使用普通匹配
	return e.pathnameExpand(strings.ReplaceAll(pattern, "**", "*"))
}

// matchRecursive 递归匹配目录
func (e *Executor) matchRecursive(dir string, pattern string) []string {
	result := []string{}
	
	// 读取目录
	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}
	
	// 匹配当前目录中的文件
	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		
		// 检查是否匹配模式
		matched, err := filepath.Match(pattern, entry.Name())
		if err == nil && matched {
			result = append(result, entryPath)
		}
		
		// 如果是目录，递归匹配
		if entry.IsDir() {
			subMatches := e.matchRecursive(entryPath, pattern)
			result = append(result, subMatches...)
		}
	}
	
	return result
}

// expandPathnamePattern 展开路径名模式（支持字符类）
// 这是一个辅助函数，用于将 bash 的字符类语法转换为 Go 的 glob 模式
func expandPathnamePattern(pattern string) string {
	// 将 bash 的字符类 `[...]` 转换为 Go 的 glob 模式
	// Go 的 filepath.Match 支持 `[...]` 字符类，所以可以直接使用
	// 但需要处理一些特殊情况：
	// 1. `[!...]` 或 `[^...]` 表示否定字符类
	// 2. `[...]` 中的 `-` 表示范围
	
	result := strings.Builder{}
	i := 0
	for i < len(pattern) {
		if pattern[i] == '[' {
			// 处理字符类
			result.WriteByte('[')
			i++
			
			// 检查是否是否定字符类
			if i < len(pattern) && (pattern[i] == '!' || pattern[i] == '^') {
				// Go 的 filepath.Match 不支持 `[!...]`，需要转换为 `[^...]`
				result.WriteByte('^')
				i++
			}
			
			// 复制字符类内容
			for i < len(pattern) && pattern[i] != ']' {
				result.WriteByte(pattern[i])
				i++
			}
			
			if i < len(pattern) {
				result.WriteByte(']')
				i++
			}
		} else {
			result.WriteByte(pattern[i])
			i++
		}
	}
	
	return result.String()
}

// tildeExpand 波浪号展开
// 根据 bash 的行为：
// 1. `~` - 当前用户主目录
// 2. `~user` - 指定用户主目录
// 3. `~+` - 当前工作目录（PWD）
// 4. `~-` - 上一个工作目录（OLDPWD）
func (e *Executor) tildeExpand(text string) string {
	if !strings.HasPrefix(text, "~") {
		return text
	}
	
	// 处理 `~`
	if text == "~" {
		home := os.Getenv("HOME")
		if home == "" {
			// Windows 上使用 USERPROFILE
			home = os.Getenv("USERPROFILE")
		}
		if home == "" {
			// 如果都没有，返回原始文本
			return text
		}
		return home
	}
	
	// 处理 `~+` - 当前工作目录
	if strings.HasPrefix(text, "~+") {
		pwd := os.Getenv("PWD")
		if pwd == "" {
			pwd, _ = os.Getwd()
		}
		if pwd == "" {
			return text
		}
		if text == "~+" {
			return pwd
		}
		// `~+/path` 格式
		return pwd + text[2:]
	}
	
	// 处理 `~-` - 上一个工作目录
	if strings.HasPrefix(text, "~-") {
		oldpwd := os.Getenv("OLDPWD")
		if oldpwd == "" {
			return text
		}
		if text == "~-" {
			return oldpwd
		}
		// `~-/path` 格式
		return oldpwd + text[2:]
	}
	
	// 处理 `~user` - 指定用户主目录
	if len(text) > 1 {
		// 查找用户名结束位置（遇到 / 或字符串结束）
		usernameEnd := 1
		for usernameEnd < len(text) && text[usernameEnd] != '/' {
			usernameEnd++
		}
		
		username := text[1:usernameEnd]
		rest := text[usernameEnd:]
		
		// 获取用户主目录
		// 在 Unix 系统上，可以通过 os/user 包获取
		// 在 Windows 上，需要特殊处理
		home := e.getUserHomeDir(username)
		if home == "" {
			// 如果找不到用户，返回原始文本
			return text
		}
		
		return home + rest
	}
	
	return text
}

// getUserHomeDir 获取用户主目录
func (e *Executor) getUserHomeDir(username string) string {
	// 如果是当前用户
	if username == "" || username == os.Getenv("USER") || username == os.Getenv("USERNAME") {
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	
	// 对于其他用户，在 Unix 系统上可以通过 os/user 包获取
	// 在 Windows 上，可以尝试从环境变量或注册表获取
	// 这里先实现一个简化版本
	// TODO: 实现完整的用户主目录查找
	
	// 尝试从环境变量获取（某些系统可能设置）
	envKey := "HOME_" + username
	if home := os.Getenv(envKey); home != "" {
		return home
	}
	
	// 如果找不到，返回空字符串（bash 的行为是返回原始文本）
	return ""
}

// expandWord 展开 word（可能包含变量、命令替换等）
func (e *Executor) expandWord(word string) string {
	// 简单的实现：展开变量
	result := e.expandVariablesInString(word)
	return result
}

// expandStringLength 展开字符串长度 ${#VAR}
func (e *Executor) expandStringLength(varName string) string {
	varValue := e.env[varName]
	if varValue == "" {
		varValue = os.Getenv(varName)
	}
	return strconv.Itoa(len(varValue))
}

