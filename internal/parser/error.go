package parser

import (
	"fmt"
	"gobash/internal/lexer"
)

// ParseError 表示解析错误
type ParseError struct {
	Type     ErrorType
	Message  string
	Token    lexer.Token
	Expected string // 期望的 token 类型或值
}

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeSyntax ErrorType = iota // 语法错误
	ErrorTypeUnexpectedToken          // 意外的 token
	ErrorTypeUnclosedQuote            // 未闭合的引号
	ErrorTypeUnclosedParen            // 未闭合的括号
	ErrorTypeUnclosedBrace            // 未闭合的大括号
	ErrorTypeUnclosedBracket          // 未闭合的方括号
	ErrorTypeUnclosedControlFlow      // 未闭合的控制流（if/fi, case/esac等）
	ErrorTypeInvalidExpression        // 无效的表达式
	ErrorTypeMissingToken             // 缺少 token
)

// Error 实现 error 接口
func (e *ParseError) Error() string {
	if e.Token.Line > 0 {
		if e.Expected != "" {
			return fmt.Sprintf("第%d行第%d列: %s，期望 %s，得到 %s", 
				e.Token.Line, e.Token.Column, e.Message, e.Expected, e.Token.Literal)
		}
		return fmt.Sprintf("第%d行第%d列: %s，得到 %s", 
			e.Token.Line, e.Token.Column, e.Message, e.Token.Literal)
	}
	return fmt.Sprintf("%s: %s", e.Message, e.Token.Literal)
}

// String 返回错误的字符串表示
func (e *ParseError) String() string {
	return e.Error()
}

// addError 添加解析错误
func (p *Parser) addError(errType ErrorType, message string, token lexer.Token, expected string) {
	err := &ParseError{
		Type:     errType,
		Message:  message,
		Token:    token,
		Expected: expected,
	}
	p.errors = append(p.errors, err.Error())
	p.parseErrors = append(p.parseErrors, err)
}

// addErrorf 添加格式化的解析错误
func (p *Parser) addErrorf(errType ErrorType, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	err := &ParseError{
		Type:    errType,
		Message: message,
		Token:   p.curToken,
	}
	p.errors = append(p.errors, err.Error())
	p.parseErrors = append(p.parseErrors, err)
}

