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
				// 在 ;; 后，跳过所有空白和换行，为后续的跳过逻辑做准备
				for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
					p.nextToken()
				}
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
				// 在移动后，立即检查是否是 ;;
				if p.curToken.Type == lexer.SEMICOLON && p.peekToken.Type == lexer.SEMICOLON {
					p.nextToken() // 跳过第一个 ;
					p.nextToken() // 跳过第二个 ;
					break
				}
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
			
			// 检查是否是单独的 * 模式（在 case 语句中，* 后面应该跟 )）
			// 如果当前是 IDENTIFIER 且值是 "*"，且下一个token是 )，这是下一个case模式
			if p.curToken.Type == lexer.IDENTIFIER && 
			   p.curToken.Literal == "*" && 
			   p.peekToken.Type == lexer.RPAREN {
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
			
			// 再次检查是否是单独的 * 模式（在解析语句之前）
			if p.curToken.Type == lexer.IDENTIFIER && 
			   p.curToken.Literal == "*" && 
			   p.peekToken.Type == lexer.RPAREN {
				break
			}
			
			// 在调用 parseStatement 之前，再次检查是否是下一个case模式
			// 这可以防止将模式（如 *）误解析为命令
			// 注意：必须在调用 parseStatement 之前检查，因为 parseStatement 会消耗 token
			if (p.curToken.Type == lexer.IDENTIFIER || 
				p.curToken.Type == lexer.STRING ||
				p.curToken.Type == lexer.STRING_SINGLE ||
				p.curToken.Type == lexer.STRING_DOUBLE) && 
				p.peekToken.Type == lexer.RPAREN {
				break
			}
			if p.curToken.Type == lexer.IDENTIFIER && 
			   p.curToken.Literal == "*" && 
			   p.peekToken.Type == lexer.RPAREN {
				break
			}
			
			// 解析语句
			stmt := p.parseStatement()
			if stmt != nil {
				body.Statements = append(body.Statements, stmt)
			} else {
				// 如果 parseStatement 返回 nil，可能是因为检测到了 case 模式
				// 检查是否是下一个case模式，如果是，立即停止解析当前case体
				if (p.curToken.Type == lexer.IDENTIFIER || 
					p.curToken.Type == lexer.STRING ||
					p.curToken.Type == lexer.STRING_SINGLE ||
					p.curToken.Type == lexer.STRING_DOUBLE) && 
					p.peekToken.Type == lexer.RPAREN {
					break
				}
				if p.curToken.Type == lexer.IDENTIFIER && 
				   p.curToken.Literal == "*" && 
				   p.peekToken.Type == lexer.RPAREN {
					break
				}
				// 如果 parseStatement 返回 nil 但不是 case 模式，可能是空语句或其他情况
				// 继续解析下一个语句
			}
			// 在 parseStatement 返回后，立即检查是否是 ;;（必须在检查其他内容之前）
			// 因为 ;; 是 case 分支的结束符，必须优先处理
			// 注意：parseCommandStatement 在遇到 NEWLINE 时会停止，但不会消耗 NEWLINE
			// 所以如果 p.curToken 是 NEWLINE，需要先移动到下一个 token，然后检查是否是 ;;
			// 如果 p.curToken 是 SEMICOLON，也需要检查是否是 ;;
			if p.curToken.Type == lexer.NEWLINE {
				// 检查下一个token是否是 ;;
				// 注意：我们只能看到下一个token（peekToken），无法直接看到第二个token
				// 所以需要先移动到下一个token，然后检查是否是 ;;
				p.nextToken()
				// 在移动后，立即检查是否是 ;;
				if p.curToken.Type == lexer.SEMICOLON && p.peekToken.Type == lexer.SEMICOLON {
					p.nextToken() // 跳过第一个 ;
					p.nextToken() // 跳过第二个 ;
					break
				}
				// 如果不是 ;;，检查是否是下一个case模式
				if (p.curToken.Type == lexer.IDENTIFIER || 
					p.curToken.Type == lexer.STRING ||
					p.curToken.Type == lexer.STRING_SINGLE ||
					p.curToken.Type == lexer.STRING_DOUBLE) && 
					p.peekToken.Type == lexer.RPAREN {
					break
				}
				if p.curToken.Type == lexer.IDENTIFIER && 
				   p.curToken.Literal == "*" && 
				   p.peekToken.Type == lexer.RPAREN {
					break
				}
				// 如果既不是 ;; 也不是下一个case模式，说明 ;; 可能被跳过或者有其他问题
				// 为了安全起见，继续解析（但这种情况不应该发生）
			} else if p.curToken.Type == lexer.SEMICOLON && p.peekToken.Type == lexer.SEMICOLON {
				p.nextToken() // 跳过第一个 ;
				p.nextToken() // 跳过第二个 ;
				break
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
			// 再次检查是否是单独的 * 模式（在nextToken之前）
			if p.curToken.Type == lexer.IDENTIFIER && 
			   p.curToken.Literal == "*" && 
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
				// 在移动后，立即检查是否是 ;;
				if p.curToken.Type == lexer.SEMICOLON && p.peekToken.Type == lexer.SEMICOLON {
					p.nextToken() // 跳过第一个 ;
					p.nextToken() // 跳过第二个 ;
					break
				}
				continue
			}
			p.nextToken()
		}
		
		if len(patterns) > 0 {
			stmt.Cases = append(stmt.Cases, &CaseClause{
				Patterns: patterns,
				Body:     body,
			})
		}
		
		// 在 ;; 后，跳过所有内容，直到遇到下一个case模式或esac
		// 这样可以防止 ;; 后的语句被解析为下一个case分支的body
		// 注意：这里需要跳过所有不是case模式或esac的内容
		// 先跳过 ;; 后的空白和换行
		for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}
		// 然后跳过所有内容，直到遇到下一个case模式或esac
		for p.curToken.Type != lexer.ESAC && p.curToken.Type != lexer.EOF {
			// 如果遇到下一个case模式（以标识符或字符串开头，且下一个token是 )），停止跳过
			if (p.curToken.Type == lexer.IDENTIFIER || 
				p.curToken.Type == lexer.STRING ||
				p.curToken.Type == lexer.STRING_SINGLE ||
				p.curToken.Type == lexer.STRING_DOUBLE) && 
				p.peekToken.Type == lexer.RPAREN {
				// 这是下一个case的模式，停止跳过
				break
			}
			// 检查是否是单独的 * 模式
			if p.curToken.Type == lexer.IDENTIFIER && 
			   p.curToken.Literal == "*" && 
			   p.peekToken.Type == lexer.RPAREN {
				break
			}
			// 如果遇到esac，退出循环
			if p.curToken.Type == lexer.ESAC {
				break
			}
			// 跳过其他所有内容（;; 后的语句不应该被解析）
			// 包括所有命令、语句等，直到遇到下一个case模式或esac
			p.nextToken()
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

