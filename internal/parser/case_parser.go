package parser

import (
	"gobash/internal/lexer"
)

// parseCaseStatement 解析case语句
func (p *Parser) parseCaseStatement() *CaseStatement {
	stmt := &CaseStatement{}
	
	p.nextToken() // 跳过 case
	
	// 解析case的值
	stmt.Value = p.parseExpression()
	
	// 跳过 in
	if p.peekToken.Type == lexer.IN {
		p.nextToken() // 跳过 in
	}
	p.nextToken()
	
	// 解析case子句
	for p.curToken.Type != lexer.ESAC && p.curToken.Type != lexer.EOF {
		// 跳过换行和空白
		if p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.WHITESPACE {
			p.nextToken()
			continue
		}
		
		// 解析模式（直到遇到 )）
		patterns := []string{}
		patternStart := true
		for p.curToken.Type != lexer.RPAREN && 
			p.curToken.Type != lexer.ESAC &&
			p.curToken.Type != lexer.EOF {
			
			if p.curToken.Type == lexer.IDENTIFIER || 
			   p.curToken.Type == lexer.STRING ||
			   p.curToken.Type == lexer.STRING_SINGLE ||
			   p.curToken.Type == lexer.STRING_DOUBLE {
				pattern := p.curToken.Literal
				// 移除引号（如果有）
				if (p.curToken.Type == lexer.STRING_SINGLE || p.curToken.Type == lexer.STRING_DOUBLE) && len(pattern) >= 2 {
					if (pattern[0] == '\'' && pattern[len(pattern)-1] == '\'') ||
					   (pattern[0] == '"' && pattern[len(pattern)-1] == '"') {
						pattern = pattern[1 : len(pattern)-1]
					}
				}
				patterns = append(patterns, pattern)
				patternStart = false
			} else if p.curToken.Type == lexer.PIPE {
				// 模式分隔符 |
				p.nextToken()
				continue
			} else if p.curToken.Type == lexer.RPAREN {
				break
			} else if !patternStart {
				// 如果已经开始解析模式但遇到其他token，可能模式已结束
				break
			}
			p.nextToken()
		}
		
		// 跳过 )
		if p.curToken.Type == lexer.RPAREN {
			p.nextToken()
		}
		
		// 解析case体（直到遇到 ;; 或 ;& 或 ;;&）
		body := &BlockStatement{Statements: []Statement{}}
		for p.curToken.Type != lexer.ESAC && p.curToken.Type != lexer.EOF {
			// 检查是否是结束符 ;;
			if p.curToken.Type == lexer.SEMICOLON && p.peekToken.Type == lexer.SEMICOLON {
				p.nextToken() // 跳过第一个 ;
				p.nextToken() // 跳过第二个 ;
				break
			}
			
			// 跳过空白（但保留换行符，因为它是语句分隔符）
			if p.curToken.Type == lexer.WHITESPACE {
				p.nextToken()
				continue
			}
			
			// 如果遇到换行符，移动到下一个token（换行符是语句分隔符）
			if p.curToken.Type == lexer.NEWLINE {
				p.nextToken()
				continue
			}
			
			// 如果遇到下一个case模式（以标识符或字符串开头，且下一个token是 )），说明当前case体已结束（可能缺少 ;;）
			if (p.curToken.Type == lexer.IDENTIFIER || 
				p.curToken.Type == lexer.STRING ||
				p.curToken.Type == lexer.STRING_SINGLE ||
				p.curToken.Type == lexer.STRING_DOUBLE) && 
				p.peekToken.Type == lexer.RPAREN {
				// 这是下一个case的模式，当前case体已结束（可能缺少 ;;）
				break
			}
			
			// 检查是否遇到esac（case语句结束）
			if p.curToken.Type == lexer.ESAC {
				break
			}
			
			// 在解析语句之前，再次检查是否是下一个case模式
			// 这可以防止将模式（如 *）误解析为命令
			if (p.curToken.Type == lexer.IDENTIFIER || 
				p.curToken.Type == lexer.STRING ||
				p.curToken.Type == lexer.STRING_SINGLE ||
				p.curToken.Type == lexer.STRING_DOUBLE) && 
				p.peekToken.Type == lexer.RPAREN {
				break
			}
			
			// 解析语句
			stmt := p.parseStatement()
			if stmt != nil {
				body.Statements = append(body.Statements, stmt)
			}
			// 如果解析后遇到下一个case模式或esac，停止解析
			if p.curToken.Type == lexer.ESAC {
				break
			}
			// 检查是否是下一个case模式（在nextToken之前检查peekToken）
			if (p.curToken.Type == lexer.IDENTIFIER || 
				p.curToken.Type == lexer.STRING ||
				p.curToken.Type == lexer.STRING_SINGLE ||
				p.curToken.Type == lexer.STRING_DOUBLE) && 
				p.peekToken.Type == lexer.RPAREN {
				break
			}
			// 如果当前token已经是下一个case模式的一部分，不要nextToken
			if p.curToken.Type == lexer.ESAC {
				break
			}
			// 如果遇到换行符，移动到下一个token（换行符是语句分隔符）
			if p.curToken.Type == lexer.NEWLINE {
				p.nextToken()
			}
			p.nextToken()
		}
		
		if len(patterns) > 0 {
			stmt.Cases = append(stmt.Cases, &CaseClause{
				Patterns: patterns,
				Body:     body,
			})
		}
		
		// 如果遇到esac，退出循环
		if p.curToken.Type == lexer.ESAC {
			break
		}
	}
	
	if p.curToken.Type == lexer.ESAC {
		p.nextToken() // 跳过 esac
	}
	
	return stmt
}

