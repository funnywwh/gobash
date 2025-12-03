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
		for p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.WHITESPACE {
			p.nextToken()
		}
		
		// 检查是否到达结束
		if p.curToken.Type == lexer.ESAC || p.curToken.Type == lexer.EOF {
			break
		}
		
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
		foundDoubleSemicolon := false
		for p.curToken.Type != lexer.ESAC && p.curToken.Type != lexer.EOF {
			// 跳过空白和换行
			for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
				p.nextToken()
			}
			
			// 检查是否到达结束
			if p.curToken.Type == lexer.ESAC || p.curToken.Type == lexer.EOF {
				break
			}
			
			// 检查是否是结束符 ;; 或 ;& 或 ;;&
			if p.curToken.Type == lexer.SEMI_SEMI {
				p.nextToken() // 跳过 ;;
				foundDoubleSemicolon = true
				// 跳过 ;; 后的空白和换行
				for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
					p.nextToken()
				}
				break
			} else if p.curToken.Type == lexer.SEMI_AND {
				// ;& 表示 fallthrough
				p.nextToken() // 跳过 ;&
				foundDoubleSemicolon = true
				// 跳过 ;& 后的空白和换行
				for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
					p.nextToken()
				}
				break
			} else if p.curToken.Type == lexer.SEMI_SEMI_AND {
				// ;;& 表示测试下一个模式
				p.nextToken() // 跳过 ;;&
				foundDoubleSemicolon = true
				// 跳过 ;;& 后的空白和换行
				for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
					p.nextToken()
				}
				break
			} else if p.curToken.Type == lexer.SEMICOLON && p.peekToken.Type == lexer.SEMICOLON {
				// 兼容旧的 ;; 格式（两个独立的 SEMICOLON token）
				p.nextToken() // 跳过第一个 ;
				p.nextToken() // 跳过第二个 ;
				foundDoubleSemicolon = true
				// 跳过 ;; 后的空白和换行
				for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
					p.nextToken()
				}
				break
			}
			
			// 检查是否是下一个case模式
			if (p.curToken.Type == lexer.IDENTIFIER || 
				p.curToken.Type == lexer.STRING ||
				p.curToken.Type == lexer.STRING_SINGLE ||
				p.curToken.Type == lexer.STRING_DOUBLE) && 
				p.peekToken.Type == lexer.RPAREN {
				// 这是下一个case的模式，当前case体已结束
				break
			}
			
			// 解析语句
			stmt := p.parseStatement()
			if stmt != nil {
				body.Statements = append(body.Statements, stmt)
				// parseStatement 后，如果 peekToken 是 NEWLINE，移动到 NEWLINE
				// 因为 parseCommandStatement 在遇到 NEWLINE 时会停止，但不会消耗 NEWLINE
				if p.peekToken.Type == lexer.NEWLINE {
					p.nextToken()
				}
			}
		}
		
		if len(patterns) > 0 {
			stmt.Cases = append(stmt.Cases, &CaseClause{
				Patterns: patterns,
				Body:     body,
			})
		}
		
		// 如果找到 ;;，需要跳过 ;; 后的所有内容，直到遇到下一个case模式或esac
		// 如果没有找到 ;;，说明遇到了下一个 case 模式或 esac，不需要跳过，直接继续循环
		if foundDoubleSemicolon {
			// 在 ;; 后，跳过所有内容，直到遇到下一个case模式或esac
			// 注意：在 body 解析循环中已经跳过了 ;; 和空白/换行，所以这里 curToken 应该指向第一个非空白/换行 token
			for p.curToken.Type != lexer.ESAC && p.curToken.Type != lexer.EOF {
				// 跳过空白和换行
				for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
					p.nextToken()
				}
				
				// 检查是否到达结束
				if p.curToken.Type == lexer.ESAC || p.curToken.Type == lexer.EOF {
					break
				}
				
				// 检查是否是下一个case模式（必须在跳过之前检查）
				if (p.curToken.Type == lexer.IDENTIFIER || 
					p.curToken.Type == lexer.STRING ||
					p.curToken.Type == lexer.STRING_SINGLE ||
					p.curToken.Type == lexer.STRING_DOUBLE) && 
					p.peekToken.Type == lexer.RPAREN {
					// 这是下一个case的模式，停止跳过，继续循环解析它
					break
				}
				// 检查是否是单独的 * 模式
				if p.curToken.Type == lexer.IDENTIFIER && 
				   p.curToken.Literal == "*" && 
				   p.peekToken.Type == lexer.RPAREN {
					// 这是下一个case的模式，停止跳过，继续循环解析它
					break
				}
				// 如果遇到esac，退出循环
				if p.curToken.Type == lexer.ESAC {
					break
				}
				// 直接跳过token，直到遇到下一个case模式或esac
				// 不要使用 parseStatement，因为它可能在遇到 SEMICOLON 时返回 nil，导致死循环
				p.nextToken()
			}
			
			// 如果遇到esac，退出case解析循环
			if p.curToken.Type == lexer.ESAC {
				break
			}
			// 如果遇到下一个case模式，继续循环解析它（不要break，让循环继续）
		}
	}
	
	if p.curToken.Type == lexer.ESAC {
		p.nextToken() // 跳过 esac
	} else if p.curToken.Type != lexer.EOF {
		// 未闭合的 case 语句
		p.addError(ErrorTypeUnclosedControlFlow, "case 语句未闭合，缺少 esac", p.curToken, "esac")
	}
	
	return stmt
}
