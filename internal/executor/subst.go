// Package executor 提供变量展开功能
package executor

import (
	"fmt"
	"os"
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

