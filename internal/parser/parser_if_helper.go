package parser

import (
	"gobash/internal/lexer"
)

// parseIfConsequence 解析 if 语句的 consequence 块
// 这个函数会手动解析，直到遇到属于当前 if 的 elif/else
func (p *Parser) parseIfConsequence() *BlockStatement {
	block := &BlockStatement{}
	block.Statements = []Statement{}

	// 跟踪嵌套的控制流语句层级
	nestingLevel := 0 // 0 表示当前 if 的层级

	for p.curToken.Type != lexer.EOF && p.curToken.Type != lexer.RBRACE {
		// 跳过空白字符和换行
		if p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
			continue
		}

		// 如果遇到属于当前 if 的 elif 或 else（且没有嵌套），停止解析
		if nestingLevel == 0 && (p.curToken.Type == lexer.ELIF || p.curToken.Type == lexer.ELSE) {
			break
		}

		// 检查是否是 if 语句的开始（会增加嵌套层级）
		wasIf := false
		if p.curToken.Type == lexer.IF {
			nestingLevel++
			wasIf = true
		}

		// 解析语句
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}

		// 如果刚才解析的是 if 语句，parseIfStatement 会完全解析整个 if 语句（包括 fi）
		// 所以 curToken 应该在 fi 之后的 token 上，嵌套层级应该减少
		if wasIf {
			// parseIfStatement 已经解析完整个 if 语句（包括 fi），
			// 所以 curToken 现在应该在 fi 之后的 token 上
			// 嵌套层级应该减少
			if nestingLevel > 0 {
				nestingLevel--
			}
		}

		// 跳过空白字符和换行
		if p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}
	}

	return block
}

// parseIfAlternative 解析 if 语句的 else 块
// 这个函数会手动解析，直到遇到属于当前 if 的 fi
func (p *Parser) parseIfAlternative() *BlockStatement {
	block := &BlockStatement{}
	block.Statements = []Statement{}

	// 跟踪嵌套的控制流语句层级
	nestingLevel := 0 // 0 表示当前 if 的层级

	for p.curToken.Type != lexer.EOF && p.curToken.Type != lexer.RBRACE {
		// 跳过空白字符和换行
		if p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
			continue
		}

		// 如果遇到属于当前 if 的 fi（且没有嵌套），停止解析
		if nestingLevel == 0 && p.curToken.Type == lexer.FI {
			break
		}

		// 检查是否是 if 语句的开始（会增加嵌套层级）
		wasIf := false
		if p.curToken.Type == lexer.IF {
			nestingLevel++
			wasIf = true
		}

		// 解析语句
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}

		// 如果刚才解析的是 if 语句，parseIfStatement 会完全解析整个 if 语句（包括 fi）
		// 所以 curToken 应该在 fi 之后的 token 上，嵌套层级应该减少
		if wasIf {
			// parseIfStatement 已经解析完整个 if 语句（包括 fi），
			// 所以 curToken 现在应该在 fi 之后的 token 上
			// 嵌套层级应该减少
			if nestingLevel > 0 {
				nestingLevel--
			}
		}

		// 跳过空白字符和换行
		if p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}
	}

	return block
}


