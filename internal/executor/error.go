package executor

import (
	"fmt"
	"strings"
)

// ExecutionErrorType 执行器错误类型
type ExecutionErrorType int

const (
	ExecutionErrorTypeCommandNotFound ExecutionErrorType = iota // 命令未找到
	ExecutionErrorTypeCommandFailed                              // 命令执行失败
	ExecutionErrorTypeRedirectError                              // 重定向错误
	ExecutionErrorTypePipeError                                  // 管道错误
	ExecutionErrorTypeVariableError                              // 变量错误
	ExecutionErrorTypeArithmeticError                            // 算术错误
	ExecutionErrorTypeInvalidExpression                          // 无效表达式
	ExecutionErrorTypeInterrupted                                // 命令被中断
	ExecutionErrorTypeUnknownStatement                           // 未知语句类型
)

// ExecutionError 表示执行器错误
type ExecutionError struct {
	Type        ExecutionErrorType
	Message     string
	Command     string   // 命令名
	Args        []string // 命令参数
	exitCode    int      // 退出码（如果可用）
	Context     string   // 错误上下文（如文件名、行号等）
	OriginalErr error    // 原始错误（如果可用）
}

// Error 实现 error 接口
func (e *ExecutionError) Error() string {
	var msg string
	switch e.Type {
	case ExecutionErrorTypeCommandNotFound:
		msg = fmt.Sprintf("命令未找到: %s", e.Command)
	case ExecutionErrorTypeCommandFailed:
		if e.exitCode != 0 {
			msg = fmt.Sprintf("命令执行失败: %s (退出码: %d)", e.Command, e.exitCode)
		} else {
			msg = fmt.Sprintf("命令执行失败: %s", e.Command)
		}
	case ExecutionErrorTypeRedirectError:
		msg = fmt.Sprintf("重定向错误: %s", e.Message)
	case ExecutionErrorTypePipeError:
		msg = fmt.Sprintf("管道错误: %s", e.Message)
	case ExecutionErrorTypeVariableError:
		msg = fmt.Sprintf("变量错误: %s", e.Message)
	case ExecutionErrorTypeArithmeticError:
		msg = fmt.Sprintf("算术错误: %s", e.Message)
	case ExecutionErrorTypeInvalidExpression:
		msg = fmt.Sprintf("无效表达式: %s", e.Message)
	case ExecutionErrorTypeInterrupted:
		msg = "命令被中断"
	case ExecutionErrorTypeUnknownStatement:
		msg = fmt.Sprintf("未知语句类型: %s", e.Message)
	default:
		msg = e.Message
	}

	// 添加上下文信息
	if e.Context != "" {
		msg = fmt.Sprintf("%s (%s)", msg, e.Context)
	}

	// 添加命令和参数信息
	if e.Command != "" {
		cmdStr := e.Command
		if len(e.Args) > 0 {
			cmdStr += " " + strings.Join(e.Args, " ")
		}
		msg = fmt.Sprintf("%s: %s", msg, cmdStr)
	}

	// 添加原始错误信息
	if e.OriginalErr != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.OriginalErr)
	}

	return msg
}

// ExitCode 返回退出码
func (e *ExecutionError) ExitCode() int {
	if e.exitCode != 0 {
		return e.exitCode
	}
	// 根据错误类型返回默认退出码
	switch e.Type {
	case ExecutionErrorTypeCommandNotFound:
		return 127 // bash 中命令未找到的退出码
	case ExecutionErrorTypeCommandFailed:
		return 1 // 命令执行失败
	case ExecutionErrorTypeRedirectError, ExecutionErrorTypePipeError:
		return 1
	case ExecutionErrorTypeVariableError:
		return 1
	case ExecutionErrorTypeArithmeticError, ExecutionErrorTypeInvalidExpression:
		return 1
	case ExecutionErrorTypeInterrupted:
		return 130 // bash 中被中断的退出码
	default:
		return 1
	}
}

// String 返回错误的字符串表示
func (e *ExecutionError) String() string {
	return e.Error()
}

// newExecutionError 创建新的执行器错误
func newExecutionError(errType ExecutionErrorType, message string, command string, args []string, exitCode int, context string, originalErr error) *ExecutionError {
	return &ExecutionError{
		Type:        errType,
		Message:     message,
		Command:     command,
		Args:        args,
		exitCode:    exitCode,
		Context:     context,
		OriginalErr: originalErr,
	}
}

