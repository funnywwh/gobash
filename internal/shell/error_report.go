package shell

import (
	"fmt"
	"os"
	"strings"
	"gobash/internal/executor"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// ErrorReporter 错误报告器
type ErrorReporter struct {
	scriptPath string // 脚本文件路径（如果是在执行脚本）
	lineNum    int    // 当前行号
	isInteractive bool // 是否是交互式模式
}

// NewErrorReporter 创建新的错误报告器
func NewErrorReporter(scriptPath string, isInteractive bool) *ErrorReporter {
	return &ErrorReporter{
		scriptPath:    scriptPath,
		isInteractive: isInteractive,
	}
}

// SetLineNum 设置当前行号
func (er *ErrorReporter) SetLineNum(lineNum int) {
	er.lineNum = lineNum
}

// ReportError 报告错误
// 根据错误类型格式化错误消息，参考 bash 的错误格式
func (er *ErrorReporter) ReportError(err error) {
	if err == nil {
		return
	}

	var errorMsg string
	var exitCode int

	// 根据错误类型格式化错误消息
	switch e := err.(type) {
	case *executor.ExecutionError:
		// 执行器错误
		errorMsg = er.formatExecutionError(e)
		exitCode = e.ExitCode()
	case *parser.ParseError:
		// 解析错误
		errorMsg = er.formatParseError(e)
		exitCode = 1
	case *lexer.LexerError:
		// 词法错误
		errorMsg = er.formatLexerError(e)
		exitCode = 1
	case error:
		// 其他错误
		errorMsg = er.formatGenericError(e)
		exitCode = 1
	}

	// 输出错误消息到 stderr
	fmt.Fprintf(os.Stderr, "%s\n", errorMsg)

	// 在非交互式模式下，如果设置了 set -e，应该退出
	// 但这里只负责报告错误，退出逻辑由调用者处理
	_ = exitCode
}

// formatExecutionError 格式化执行器错误
func (er *ErrorReporter) formatExecutionError(e *executor.ExecutionError) string {
	// 参考 bash 的错误格式：gobash: 文件名:行号: 错误消息
	var prefix string
	if er.scriptPath != "" {
		if er.lineNum > 0 {
			prefix = fmt.Sprintf("gobash: %s: 第%d行", er.scriptPath, er.lineNum)
		} else {
			prefix = fmt.Sprintf("gobash: %s", er.scriptPath)
		}
	} else {
		if er.isInteractive {
			// 交互式模式：gobash: 错误消息
			prefix = "gobash"
		} else {
			// 非交互式模式：gobash: 行号: 错误消息
			if er.lineNum > 0 {
				prefix = fmt.Sprintf("gobash: 第%d行", er.lineNum)
			} else {
				prefix = "gobash"
			}
		}
	}

	// 获取错误消息
	errorMsg := e.Error()

	// 如果错误消息已经包含前缀，不再重复添加
	if prefix != "" {
		return fmt.Sprintf("%s: %s", prefix, errorMsg)
	}
	return errorMsg
}

// formatParseError 格式化解析错误
func (er *ErrorReporter) formatParseError(e *parser.ParseError) string {
	// 解析错误已经包含了行号和列号信息
	// 格式：gobash: 第%d行第%d列: 语法错误：...
	errorMsg := e.Error()

	// 如果是在执行脚本，添加文件名
	if er.scriptPath != "" {
		// 检查错误消息是否已经包含文件名
		if !containsScriptPath(errorMsg, er.scriptPath) {
			return fmt.Sprintf("gobash: %s: %s", er.scriptPath, errorMsg)
		}
	}

	return errorMsg
}

// formatLexerError 格式化词法错误
func (er *ErrorReporter) formatLexerError(e *lexer.LexerError) string {
	// 词法错误已经包含了行号和列号信息
	// 格式：第%d行第%d列: 词法错误：...
	errorMsg := e.Error()

	// 如果是在执行脚本，添加文件名
	if er.scriptPath != "" {
		// 检查错误消息是否已经包含文件名
		if !containsScriptPath(errorMsg, er.scriptPath) {
			return fmt.Sprintf("gobash: %s: %s", er.scriptPath, errorMsg)
		}
	} else {
		// 添加 gobash 前缀
		if !containsPrefix(errorMsg, "gobash") {
			return fmt.Sprintf("gobash: %s", errorMsg)
		}
	}

	return errorMsg
}

// formatGenericError 格式化通用错误
func (er *ErrorReporter) formatGenericError(e error) string {
	var prefix string
	if er.scriptPath != "" {
		if er.lineNum > 0 {
			prefix = fmt.Sprintf("gobash: %s: 第%d行", er.scriptPath, er.lineNum)
		} else {
			prefix = fmt.Sprintf("gobash: %s", er.scriptPath)
		}
	} else {
		if er.isInteractive {
			prefix = "gobash"
		} else {
			if er.lineNum > 0 {
				prefix = fmt.Sprintf("gobash: 第%d行", er.lineNum)
			} else {
				prefix = "gobash"
			}
		}
	}

	return fmt.Sprintf("%s: %v", prefix, e)
}

// containsScriptPath 检查错误消息是否已经包含脚本路径
func containsScriptPath(msg, scriptPath string) bool {
	return contains(msg, scriptPath)
}

// containsPrefix 检查错误消息是否已经包含前缀
func containsPrefix(msg, prefix string) bool {
	return contains(msg, prefix)
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

