package lexer

import (
	"fmt"
)

// LexerErrorType 词法分析器错误类型
type LexerErrorType int

const (
	LexerErrorTypeInvalidChar LexerErrorType = iota // 无效字符
	LexerErrorTypeUnclosedQuote                      // 未闭合的引号
	LexerErrorTypeUnclosedString                     // 未闭合的字符串
	LexerErrorTypeInvalidUTF8                        // 无效的 UTF-8 序列
	LexerErrorTypeUnexpectedEOF                      // 意外的文件结束
	LexerErrorTypeInvalidEscape                      // 无效的转义序列
)

// LexerError 表示词法分析器错误
type LexerError struct {
	Type    LexerErrorType
	Message string
	Line    int
	Column  int
	Char    string // 导致错误的字符
}

// Error 实现 error 接口
func (e *LexerError) Error() string {
	if e.Line > 0 {
		switch e.Type {
		case LexerErrorTypeInvalidChar:
			return fmt.Sprintf("第%d行第%d列: 词法错误：无效字符 `%s'", 
				e.Line, e.Column, e.Char)
		case LexerErrorTypeUnclosedQuote:
			return fmt.Sprintf("第%d行第%d列: 词法错误：未闭合的引号", 
				e.Line, e.Column)
		case LexerErrorTypeUnclosedString:
			return fmt.Sprintf("第%d行第%d列: 词法错误：未闭合的字符串", 
				e.Line, e.Column)
		case LexerErrorTypeInvalidUTF8:
			return fmt.Sprintf("第%d行第%d列: 词法错误：无效的 UTF-8 序列", 
				e.Line, e.Column)
		case LexerErrorTypeUnexpectedEOF:
			return fmt.Sprintf("第%d行第%d列: 词法错误：意外的文件结束", 
				e.Line, e.Column)
		case LexerErrorTypeInvalidEscape:
			return fmt.Sprintf("第%d行第%d列: 词法错误：无效的转义序列 `%s'", 
				e.Line, e.Column, e.Char)
		default:
			return fmt.Sprintf("第%d行第%d列: 词法错误：%s", 
				e.Line, e.Column, e.Message)
		}
	}
	return fmt.Sprintf("词法错误：%s", e.Message)
}

// String 返回错误的字符串表示
func (e *LexerError) String() string {
	return e.Error()
}




