package parser

import (
	"gobash/internal/lexer"
)

// syncPoint 同步点 token 类型，用于错误恢复
var syncPointTokens = []lexer.TokenType{
	lexer.SEMICOLON,    // ;
	lexer.NEWLINE,      // 换行符
	lexer.AND,          // &&
	lexer.OR,           // ||
	lexer.PIPE,         // |
	lexer.EOF,          // 文件结束
	lexer.IF,           // if
	lexer.FOR,          // for
	lexer.WHILE,        // while
	lexer.CASE,         // case
	lexer.FUNCTION,     // function
	lexer.DO,           // do
	lexer.DONE,         // done
	lexer.THEN,         // then
	lexer.ELSE,         // else
	lexer.ELIF,         // elif
	lexer.FI,           // fi
	lexer.ESAC,         // esac
}

// isSyncPoint 检查当前 token 是否是同步点
func (p *Parser) isSyncPoint(tokenType lexer.TokenType) bool {
	for _, syncType := range syncPointTokens {
		if tokenType == syncType {
			return true
		}
	}
	return false
}

// recoverFromError 从错误中恢复，跳过到下一个同步点
// 返回 true 表示成功恢复，false 表示无法恢复（如遇到 EOF）
func (p *Parser) recoverFromError() bool {
	// 如果当前已经是同步点，直接返回
	if p.isSyncPoint(p.curToken.Type) {
		return true
	}

	// 跳过当前 token 和后续的非同步点 token
	for p.curToken.Type != lexer.EOF {
		// 如果遇到同步点，停止恢复
		if p.isSyncPoint(p.curToken.Type) {
			return true
		}

		// 跳过当前 token
		p.nextToken()
	}

	// 到达 EOF，无法继续恢复
	return false
}

// recoverFromUnclosedError 从未闭合错误中恢复
// 对于未闭合的括号、大括号、控制流等，尝试找到匹配的结束符号
func (p *Parser) recoverFromUnclosedError(errType ErrorType) bool {
	switch errType {
	case ErrorTypeUnclosedParen:
		// 尝试找到右括号
		return p.recoverToToken(lexer.RPAREN)
	case ErrorTypeUnclosedBrace:
		// 尝试找到右大括号
		return p.recoverToToken(lexer.RBRACE)
	case ErrorTypeUnclosedControlFlow:
		// 尝试找到控制流结束关键字（fi, done, esac）
		return p.recoverToControlFlowEnd()
	default:
		// 其他错误，使用通用恢复
		return p.recoverFromError()
	}
}

// recoverToToken 恢复到指定的 token 类型
func (p *Parser) recoverToToken(targetType lexer.TokenType) bool {
	depth := 0
	for p.curToken.Type != lexer.EOF {
		// 如果找到目标 token 且深度为 0，恢复成功
		if p.curToken.Type == targetType && depth == 0 {
			p.nextToken() // 跳过目标 token
			return true
		}

		// 处理嵌套结构
		switch p.curToken.Type {
		case lexer.LPAREN:
			depth++
		case lexer.RPAREN:
			if targetType == lexer.RPAREN {
				depth--
			}
		case lexer.LBRACE:
			depth++
		case lexer.RBRACE:
			if targetType == lexer.RBRACE {
				depth--
			}
		}

		p.nextToken()
	}

	return false
}

// recoverToControlFlowEnd 恢复到控制流结束关键字
func (p *Parser) recoverToControlFlowEnd() bool {
	for p.curToken.Type != lexer.EOF {
		// 检查是否是控制流结束关键字
		if p.curToken.Type == lexer.FI || 
		   p.curToken.Type == lexer.DONE || 
		   p.curToken.Type == lexer.ESAC {
			p.nextToken() // 跳过结束关键字
			return true
		}

		// 如果遇到新的控制流开始，可能需要递归处理
		// 但为了简化，这里只做简单的跳过
		p.nextToken()
	}

	return false
}

// shouldContinueAfterError 判断是否应该在错误后继续解析
// 根据错误类型决定是否继续
func (p *Parser) shouldContinueAfterError(errType ErrorType) bool {
	// 对于某些严重错误，可能不应该继续
	// 但为了最大程度地恢复，我们允许继续解析
	switch errType {
	case ErrorTypeUnclosedQuote:
		// 未闭合的引号是严重错误，但可以尝试继续
		return true
	case ErrorTypeUnclosedParen, ErrorTypeUnclosedBrace, ErrorTypeUnclosedControlFlow:
		// 未闭合的结构，尝试恢复
		return true
	case ErrorTypeSyntax, ErrorTypeUnexpectedToken:
		// 语法错误，尝试继续
		return true
	default:
		// 其他错误，尝试继续
		return true
	}
}

