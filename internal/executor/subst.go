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
	
	// 处理数组访问 ${arr[0]} 或 ${arr[key]}
	if strings.HasPrefix(word, "[") {
		// 数组访问，暂时简化处理
		// TODO: 实现完整的数组访问
		return varValue, nil
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

