// Package parser 提供语法分析功能，将token序列解析为抽象语法树（AST）
package parser

import (
	"fmt"
	"strconv"
	"strings"
	"gobash/internal/lexer"
)

// Parser 语法分析器
// 负责将token序列解析为抽象语法树（AST），支持shell的各种语法结构
type Parser struct {
	l      *lexer.Lexer
	errors []string // 保持向后兼容，存储错误消息字符串
	parseErrors []*ParseError // 新的结构化错误列表

	curToken  lexer.Token
	peekToken lexer.Token
	
	// 用于回退
	savedTokens []lexer.Token
}

// New 创建新的解析器
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:           l,
		errors:      []string{},
		parseErrors: []*ParseError{},
	}

	// 读取两个token，设置curToken和peekToken
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken 移动到下一个token
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// ParseProgram 解析程序
func (p *Parser) ParseProgram() *Program {
	program := &Program{}
	program.Statements = []Statement{}

	for p.curToken.Type != lexer.EOF {
		// 跳过空白字符和换行
		if p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			// 检查是否是命令链（; && ||）
			stmt = p.parseCommandChain(stmt)
			program.Statements = append(program.Statements, stmt)
		}
		// case语句已经处理了esac，curToken已经指向esac之后的位置
		// 如果curToken不是EOF，需要移动到下一个token
		if p.curToken.Type != lexer.EOF {
			p.nextToken()
		}
	}

	return program
}

// parseCommandChain 解析命令链（; && ||）
func (p *Parser) parseCommandChain(left Statement) Statement {
	for {
		// 跳过空白字符和换行
		for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}
		
		var op string
		if p.curToken.Type == lexer.SEMICOLON {
			op = ";"
			p.nextToken()
		} else if p.curToken.Type == lexer.AND {
			op = "&&"
			p.nextToken()
		} else if p.curToken.Type == lexer.OR {
			op = "||"
			p.nextToken()
		} else {
			// 没有操作符，返回
			return left
		}
		
		// 跳过空白字符和换行
		for p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
		}
		
		// 解析右侧命令
		right := p.parseStatement()
		if right == nil {
			return left
		}
		
		// 创建命令链
		left = &CommandChain{
			Left:     left,
			Right:    right,
			Operator: op,
		}
	}
}

// parseStatement 解析语句
func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.WHILE:
		return p.parseWhileStatement()
	case lexer.FUNCTION:
		return p.parseFunctionStatement()
	case lexer.CASE:
		return p.parseCaseStatement()
	case lexer.BREAK:
		return p.parseBreakStatement()
	case lexer.CONTINUE:
		return p.parseContinueStatement()
	case lexer.LPAREN:
		// 子shell (command)
		return p.parseSubshell()
	case lexer.LBRACE:
		// 命令组 { command; }
		return p.parseGroupCommand()
	case lexer.SEMICOLON:
		// 空语句，跳过
		p.nextToken()
		return p.parseStatement()
	default:
		// 检查是否是数组赋值 arr=(1 2 3)
		// 注意：lexer可能将 arr= 识别为一个IDENTIFIER，所以需要检查是否以 = 结尾
		var isArrayAssignment bool
		var arrayName string
		
		if p.curToken.Type == lexer.IDENTIFIER {
			// 检查是否是 arr=( 格式（arr= 被识别为一个token）
			if strings.HasSuffix(p.curToken.Literal, "=") && p.peekToken.Type == lexer.LPAREN {
				arrayName = strings.TrimSuffix(p.curToken.Literal, "=")
				isArrayAssignment = true
			} else if p.peekToken.Type == lexer.LPAREN {
				// 检查是否是 arr ( 格式（但这种情况应该是函数调用）
				// 先检查下一个token是否是 )
				savedCur := p.curToken
				savedPeek := p.peekToken
				p.nextToken() // 跳过 (
				if p.curToken.Type == lexer.RPAREN || 
				   p.curToken.Type == lexer.IDENTIFIER || 
				   p.curToken.Type == lexer.STRING ||
				   p.curToken.Type == lexer.STRING_SINGLE ||
				   p.curToken.Type == lexer.STRING_DOUBLE ||
				   p.curToken.Type == lexer.NUMBER {
					// 这可能是数组赋值 arr(1 2 3)，但Bash不支持这种语法
					// 恢复状态，继续处理
					p.curToken = savedCur
					p.peekToken = savedPeek
				} else {
					// 恢复状态
					p.curToken = savedCur
					p.peekToken = savedPeek
				}
			}
		}
		
		if isArrayAssignment {
			// 这是数组赋值 arr=(1 2 3) 或 arr=([0]=a [1]=b)
			stmt := &ArrayAssignmentStatement{
				Name:          arrayName,
				Values:        []Expression{},
				IndexedValues: make(map[string]Expression),
			}
			p.nextToken() // 跳过 arr= token
			p.nextToken() // 跳过 (
			
			// 检查是否是带索引的数组赋值 arr=([0]=a [1]=b)
			hasIndexedValues := false
			
			// 解析数组元素
			for p.curToken.Type != lexer.RPAREN && p.curToken.Type != lexer.EOF {
				if p.curToken.Type == lexer.RPAREN {
					break
				}
				// 跳过空白字符
				if p.curToken.Type == lexer.WHITESPACE {
					p.nextToken()
					continue
				}
				
				// 检查是否是带索引的元素 [index]=value
				if p.curToken.Type == lexer.LBRACKET {
					// 这是带索引的数组元素 [index]=value
					hasIndexedValues = true
					p.nextToken() // 跳过 [
					
					// 读取索引（可能是数字、字符串或变量）
					var indexExpr Expression
					if p.curToken.Type == lexer.NUMBER {
						indexExpr = p.parseExpression()
					} else if p.curToken.Type == lexer.IDENTIFIER || 
					          p.curToken.Type == lexer.STRING ||
					          p.curToken.Type == lexer.STRING_SINGLE ||
					          p.curToken.Type == lexer.STRING_DOUBLE ||
					          p.curToken.Type == lexer.VAR ||
					          p.curToken.Type == lexer.PARAM_EXPAND {
						indexExpr = p.parseExpression()
					} else {
						// 索引为空，使用下一个可用索引
						indexExpr = &Identifier{Value: ""}
					}
					
					// 检查是否有 ]
					if p.curToken.Type != lexer.RBRACKET {
						// 索引表达式可能包含多个 token，继续读取直到找到 ]
						for p.curToken.Type != lexer.RBRACKET && p.curToken.Type != lexer.EOF {
							p.nextToken()
						}
					}
					
					if p.curToken.Type == lexer.RBRACKET {
						p.nextToken() // 跳过 ]
					}
					
					// 检查是否有 =（在 lexer 中，单独的 = 会被识别为 ILLEGAL）
					// 但在数组赋值中，= 可能已经被包含在标识符中（如 arr[0]=value）
					// 或者下一个 token 是 ILLEGAL（单独的 =）
					if p.curToken.Type == lexer.ILLEGAL && p.curToken.Literal == "=" {
						// 单独的 = token
						p.nextToken() // 跳过 =
					} else if strings.HasSuffix(p.curToken.Literal, "=") {
						// token 包含 =（如 arr[0]=value 中的 =）
						// 这种情况已经在 lexer 中处理了，当前 token 应该是值
						// 但为了兼容，我们检查一下
					}
					
					// 读取值（如果当前 token 是 =，下一个 token 是值）
					// 如果当前 token 已经是值（因为 = 被包含在之前的 token 中），直接使用
					var valueExpr Expression
					if p.curToken.Type == lexer.ILLEGAL && p.curToken.Literal == "=" {
						// 刚刚跳过了 =，现在读取值
						valueExpr = p.parseExpression()
					} else {
						// 当前 token 可能就是值，或者需要解析表达式
						valueExpr = p.parseExpression()
					}
					
					// 将索引转换为字符串（用于 map 的 key）
					indexStr := ""
					if indexExpr != nil {
						// 这里先保存索引表达式，在执行时再求值
						// 暂时使用 Identifier 来存储索引的字符串表示
						if ident, ok := indexExpr.(*Identifier); ok {
							indexStr = ident.Value
						} else if str, ok := indexExpr.(*StringLiteral); ok {
							indexStr = str.Value
						} else if num, ok := indexExpr.(*Identifier); ok && num.Value != "" {
							// 尝试将数字字符串作为索引
							indexStr = num.Value
						} else {
							// 对于复杂表达式，需要求值
							// 这里我们创建一个特殊的表达式来标记需要求值
							indexStr = "__EXPR__"
						}
					}
					
					// 如果索引字符串为空，表示使用下一个可用索引
					if indexStr == "" {
						indexStr = fmt.Sprintf("%d", len(stmt.Values))
					}
					
					stmt.IndexedValues[indexStr] = valueExpr
				} else {
					// 普通数组元素（不带索引）
					stmt.Values = append(stmt.Values, p.parseExpression())
				}
				p.nextToken()
			}
			
			if p.curToken.Type == lexer.RPAREN {
				p.nextToken() // 跳过 )
			}
			
			// 如果使用了带索引的赋值，清空 Values（只使用 IndexedValues）
			if hasIndexedValues && len(stmt.IndexedValues) > 0 {
				stmt.Values = nil
			}
			
			return stmt
		}
		
		// 检查是否是函数定义格式 name() { ... }
		if p.curToken.Type == lexer.IDENTIFIER && p.peekToken.Type == lexer.LPAREN {
			// 保存当前状态
			name := p.curToken.Literal
			savedCur := p.curToken
			savedPeek := p.peekToken
			
			p.nextToken() // 跳过 (
			if p.curToken.Type == lexer.RPAREN {
				p.nextToken() // 跳过 )
				if p.curToken.Type == lexer.LBRACE {
					// 这是函数定义
					stmt := &FunctionStatement{Name: name}
					p.nextToken() // 跳过 {
					stmt.Body = p.parseBlockStatement()
					return stmt
				}
			}
			// 不是函数定义，恢复状态
			p.curToken = savedCur
			p.peekToken = savedPeek
		}
		return p.parseCommandStatement()
	}
}

// parseCommandStatement 解析命令语句
func (p *Parser) parseCommandStatement() *CommandStatement {
	stmt := &CommandStatement{}

	// 如果没有有效的命令 token，返回 nil
	if p.curToken.Type != lexer.IDENTIFIER && 
	   p.curToken.Type != lexer.STRING &&
	   p.curToken.Type != lexer.STRING_SINGLE &&
	   p.curToken.Type != lexer.STRING_DOUBLE &&
	   p.curToken.Type != lexer.LBRACKET &&
	   p.curToken.Type != lexer.DBL_LBRACKET &&
	   p.curToken.Type != lexer.VAR &&
	   p.curToken.Type != lexer.DOLLAR &&
	   p.curToken.Type != lexer.COMMAND_SUBSTITUTION &&
	   p.curToken.Type != lexer.ARITHMETIC_EXPANSION &&
	   p.curToken.Type != lexer.NUMBER {
		return nil
	}
	
	// 检查是否是 case 模式（如 *）后跟 )
	// 如果是，这不是命令，返回 nil
	if (p.curToken.Type == lexer.IDENTIFIER || 
	    p.curToken.Type == lexer.STRING ||
	    p.curToken.Type == lexer.STRING_SINGLE ||
	    p.curToken.Type == lexer.STRING_DOUBLE) && 
	    p.peekToken.Type == lexer.RPAREN {
		// 这可能是 case 模式，不是命令
		// 但我们需要在 case 解析上下文中才能确定
		// 这里先返回 nil，让调用者处理
		return nil
	}

	// 检查是否是变量赋值 VAR=value
	// 如果当前是标识符，下一个是 ILLEGAL（=），再下一个是值
	if p.curToken.Type == lexer.IDENTIFIER && p.peekToken.Type == lexer.ILLEGAL {
		// 检查 ILLEGAL token 是否是 =
		if p.peekToken.Literal == "=" {
			// 这是变量赋值，将 VAR=value 作为命令名
			varName := p.curToken.Literal
			p.nextToken() // 跳过 VAR
			p.nextToken() // 跳过 =
			// 读取值（可能是字符串、标识符、算术展开等）
			var value strings.Builder
			for p.curToken.Type != lexer.EOF && 
			    p.curToken.Type != lexer.NEWLINE && 
			    p.curToken.Type != lexer.SEMICOLON &&
			    p.curToken.Type != lexer.WHITESPACE &&
			    p.curToken.Type != lexer.RPAREN {
				if p.curToken.Type == lexer.STRING || 
				   p.curToken.Type == lexer.STRING_SINGLE || 
				   p.curToken.Type == lexer.STRING_DOUBLE {
					// 对于字符串 token，需要保留引号以便 executor 正确处理
					if p.curToken.Type == lexer.STRING_SINGLE {
						value.WriteString("'")
						value.WriteString(p.curToken.Literal)
						value.WriteString("'")
					} else if p.curToken.Type == lexer.STRING_DOUBLE {
						value.WriteString("\"")
						value.WriteString(p.curToken.Literal)
						value.WriteString("\"")
					} else {
						value.WriteString(p.curToken.Literal)
					}
				} else if p.curToken.Type == lexer.IDENTIFIER {
					value.WriteString(p.curToken.Literal)
				} else if p.curToken.Type == lexer.NUMBER {
					value.WriteString(p.curToken.Literal)
				} else if p.curToken.Type == lexer.ARITHMETIC_EXPANSION {
					// 处理算术展开 $((expr))
					// lexer 返回的 Literal 只是表达式部分，需要包装成 $((expr)) 格式
					value.WriteString("$((")
					value.WriteString(p.curToken.Literal)
					value.WriteString("))")
					p.nextToken() // 移动到下一个 token
					continue
				} else if p.curToken.Type == lexer.DOLLAR {
					// 处理 $VAR 或 $((expr)) 的开始
					// 先检查是否是算术展开 $((expr))
					if p.peekToken.Type == lexer.LPAREN {
						peek2 := p.peekToken
						p.nextToken() // 移动到 (
						if p.peekToken.Type == lexer.LPAREN {
							// $((expr)) 算术展开，读取完整的算术展开 token
							p.curToken = peek2 // 恢复，让 lexer 读取完整的算术展开
							p.nextToken() // 这会读取 $((expr)) 作为 ARITHMETIC_EXPANSION token
							if p.curToken.Type == lexer.ARITHMETIC_EXPANSION {
								value.WriteString("$(((")
								value.WriteString(p.curToken.Literal)
								value.WriteString("))")
								p.nextToken() // 移动到下一个 token
								continue
							}
						} else {
							// $(command) 命令替换，恢复
							p.curToken = peek2
						}
					}
					// 普通变量展开 $VAR
					value.WriteString("$")
				} else {
					break
				}
				p.nextToken()
			}
			// 将 VAR=value 作为命令名
			stmt.Command = &Identifier{Value: varName + "=" + value.String()}
			return stmt
		}
	}

	// 解析命令（包括 [ 和 [[ 命令）
	if p.curToken.Type == lexer.LBRACKET {
		// [ 命令，创建一个标识符表达式
		stmt.Command = &Identifier{Value: "["}
	} else if p.curToken.Type == lexer.DBL_LBRACKET {
		// [[ 命令，创建一个标识符表达式
		stmt.Command = &Identifier{Value: "[["}
	} else {
		stmt.Command = p.parseExpression()
	}
	p.nextToken()

	// 检查是否是 [[ 命令，需要特殊处理 && 和 ||
	isDoubleBracket := false
	if stmt.Command != nil {
		if ident, ok := stmt.Command.(*Identifier); ok && ident.Value == "[[" {
			isDoubleBracket = true
		}
	}
	
	// 解析参数和重定向
	for p.curToken.Type != lexer.EOF && 
		p.curToken.Type != lexer.SEMICOLON &&
		p.curToken.Type != lexer.NEWLINE &&
		p.curToken.Type != lexer.PIPE &&
		p.curToken.Type != lexer.AMPERSAND &&
		p.curToken.Type != lexer.THEN &&
		p.curToken.Type != lexer.DO &&
		p.curToken.Type != lexer.DONE &&
		p.curToken.Type != lexer.FI &&
		p.curToken.Type != lexer.ELSE &&
		p.curToken.Type != lexer.ELIF {
		
		// 如果遇到换行符，立即停止解析参数
		if p.curToken.Type == lexer.NEWLINE {
			break
		}
		
		// 对于非 [[ 命令，遇到 && 或 || 时停止（这些是命令分隔符）
		if !isDoubleBracket && (p.curToken.Type == lexer.AND || p.curToken.Type == lexer.OR) {
			break
		}
		
		// 检查是否是 [ 或 [[ 命令的结束括号
		if p.curToken.Type == lexer.RBRACKET {
			// 将 ] 作为参数添加（test命令需要它）
			stmt.Args = append(stmt.Args, &Identifier{Value: "]"})
			// 不调用 p.nextToken()，让调用者处理 ] 之后的 token
			break
		}
		if p.curToken.Type == lexer.DBL_RBRACKET {
			// 将 ]] 作为参数添加（[[命令需要它）
			stmt.Args = append(stmt.Args, &Identifier{Value: "]]"})
			// 不调用 p.nextToken()，让调用者处理 ]] 之后的 token
			break
		}
		
		// 对于 [[ 命令，将 && 和 || 作为参数
		if isDoubleBracket && (p.curToken.Type == lexer.AND || p.curToken.Type == lexer.OR) {
			op := "&&"
			if p.curToken.Type == lexer.OR {
				op = "||"
			}
			stmt.Args = append(stmt.Args, &Identifier{Value: op})
			p.nextToken()
			continue
		}
		
			// 检查重定向（包括所有重定向类型）
		if p.curToken.Type == lexer.REDIRECT_OUT ||
		   p.curToken.Type == lexer.REDIRECT_IN ||
		   p.curToken.Type == lexer.REDIRECT_APPEND ||
		   p.curToken.Type == lexer.REDIRECT_HEREDOC ||
		   p.curToken.Type == lexer.REDIRECT_HEREDOC_STRIP ||
		   p.curToken.Type == lexer.REDIRECT_HEREDOC_TABS ||
		   p.curToken.Type == lexer.REDIRECT_DUP_IN ||
		   p.curToken.Type == lexer.REDIRECT_DUP_OUT ||
		   p.curToken.Type == lexer.REDIRECT_CLOBBER ||
		   p.curToken.Type == lexer.REDIRECT_RW {
			redirect := p.parseRedirect()
			if redirect != nil {
				stmt.Redirects = append(stmt.Redirects, redirect)
			}
			p.nextToken()
			continue
		}
		
		// 解析参数
		// 注意：关键字（如 case、if、for 等）在命令参数位置时应该被当作普通标识符处理
		if p.curToken.Type == lexer.IDENTIFIER || 
		   p.curToken.Type == lexer.STRING ||
		   p.curToken.Type == lexer.STRING_SINGLE ||
		   p.curToken.Type == lexer.STRING_DOUBLE ||
		   p.curToken.Type == lexer.VAR ||
		   p.curToken.Type == lexer.DOLLAR ||
		   p.curToken.Type == lexer.COMMAND_SUBSTITUTION ||
		   p.curToken.Type == lexer.ARITHMETIC_EXPANSION ||
		   p.curToken.Type == lexer.NUMBER ||
		   p.curToken.Type == lexer.CASE ||
		   p.curToken.Type == lexer.IF ||
		   p.curToken.Type == lexer.THEN ||
		   p.curToken.Type == lexer.ELSE ||
		   p.curToken.Type == lexer.ELIF ||
		   p.curToken.Type == lexer.FI ||
		   p.curToken.Type == lexer.FOR ||
		   p.curToken.Type == lexer.WHILE ||
		   p.curToken.Type == lexer.DO ||
		   p.curToken.Type == lexer.DONE ||
		   p.curToken.Type == lexer.ESAC ||
		   p.curToken.Type == lexer.FUNCTION ||
		   p.curToken.Type == lexer.IN ||
		   p.curToken.Type == lexer.SELECT ||
		   p.curToken.Type == lexer.TIME {
			stmt.Args = append(stmt.Args, p.parseExpression())
			// parseExpression 不移动 token，所以 curToken 仍然是当前参数 token
			// 移动到下一个 token
			p.nextToken()
			// 检查当前 token 是否是换行符或语句结束标记
			if p.curToken.Type == lexer.NEWLINE ||
			   p.curToken.Type == lexer.SEMICOLON ||
			   p.curToken.Type == lexer.FI ||
			   p.curToken.Type == lexer.DONE ||
			   p.curToken.Type == lexer.ELSE ||
			   p.curToken.Type == lexer.ELIF ||
			   p.curToken.Type == lexer.ESAC ||
			   p.curToken.Type == lexer.EOF {
				// 遇到换行符或结束标记，停止解析
				break
			}
			// 继续解析下一个参数
			continue
		}
		
		// 如果不是参数类型的 token，移动到下一个
		p.nextToken()
	}

	// 解析管道
	if p.curToken.Type == lexer.PIPE {
		p.nextToken() // 跳过 |
		stmt.Pipe = p.parseCommandStatement()
		return stmt
	}

	// 检查后台执行
	if p.curToken.Type == lexer.AMPERSAND {
		stmt.Background = true
	}

	return stmt
}

// parseRedirect 解析重定向
func (p *Parser) parseRedirect() *Redirect {
	redirect := &Redirect{
		FD: 1, // 默认stdout
	}

	switch p.curToken.Type {
	case lexer.REDIRECT_OUT:
		redirect.Type = REDIRECT_OUTPUT
		// 检查是否有文件描述符，如 2>
		if len(p.curToken.Literal) > 1 {
			// 已经包含在token中，如 "2>"
			if p.curToken.Literal[0] >= '0' && p.curToken.Literal[0] <= '9' {
				redirect.FD = int(p.curToken.Literal[0] - '0')
			}
		}
	case lexer.REDIRECT_IN:
		redirect.Type = REDIRECT_INPUT
		redirect.FD = 0
	case lexer.REDIRECT_APPEND:
		redirect.Type = REDIRECT_APPEND
		if len(p.curToken.Literal) > 2 {
			if p.curToken.Literal[0] >= '0' && p.curToken.Literal[0] <= '9' {
				redirect.FD = int(p.curToken.Literal[0] - '0')
			}
		}
	case lexer.REDIRECT_HEREDOC:
		redirect.Type = REDIRECT_HEREDOC
		redirect.FD = 0
		redirect.HereDoc = &HereDocument{StripTabs: false}
	case lexer.REDIRECT_HEREDOC_STRIP:
		redirect.Type = REDIRECT_HEREDOC_STRIP
		redirect.FD = 0
		redirect.HereDoc = &HereDocument{StripTabs: true}
	case lexer.REDIRECT_HEREDOC_TABS:
		redirect.Type = REDIRECT_HERESTRING
		redirect.FD = 0
		// Here-string 不需要 HereDoc 结构
	case lexer.REDIRECT_DUP_IN:
		redirect.Type = REDIRECT_DUP_IN
		redirect.FD = 0
	case lexer.REDIRECT_DUP_OUT:
		redirect.Type = REDIRECT_DUP_OUT
		redirect.FD = 1
	case lexer.REDIRECT_CLOBBER:
		redirect.Type = REDIRECT_CLOBBER
		redirect.FD = 1
	case lexer.REDIRECT_RW:
		redirect.Type = REDIRECT_RW
		redirect.FD = 0
	default:
		return nil
	}

	// 读取目标文件或 Here-document 分隔符
	p.nextToken()
	
	// 对于 Here-document，分隔符可能是带引号的
	if redirect.Type == REDIRECT_HEREDOC || redirect.Type == REDIRECT_HEREDOC_STRIP {
		if redirect.HereDoc != nil {
			// 检查分隔符是否带引号
			if p.curToken.Type == lexer.STRING_SINGLE || p.curToken.Type == lexer.STRING_DOUBLE {
				redirect.HereDoc.Quoted = true
				// 提取分隔符（移除引号）
				expr := p.parseExpression()
				if str, ok := expr.(*StringLiteral); ok {
					redirect.HereDoc.Delimiter = str.Value
				} else if ident, ok := expr.(*Identifier); ok {
					redirect.HereDoc.Delimiter = ident.Value
				}
			} else if p.curToken.Type == lexer.IDENTIFIER {
				redirect.HereDoc.Quoted = false
				expr := p.parseExpression()
				if ident, ok := expr.(*Identifier); ok {
					redirect.HereDoc.Delimiter = ident.Value
				}
			}
			// Here-document 的内容将在执行时读取
			redirect.Target = nil
		}
	} else if p.curToken.Type == lexer.IDENTIFIER || 
	   p.curToken.Type == lexer.STRING ||
	   p.curToken.Type == lexer.STRING_SINGLE ||
	   p.curToken.Type == lexer.STRING_DOUBLE {
		redirect.Target = p.parseExpression()
	} else {
		// 重定向目标缺失
		return nil
	}

	return redirect
}

// parseExpression 解析表达式
func (p *Parser) parseExpression() Expression {
	switch p.curToken.Type {
	case lexer.IDENTIFIER:
		return &Identifier{Value: p.curToken.Literal}
	// 关键字在表达式上下文中应该被当作普通标识符处理
	case lexer.CASE, lexer.IF, lexer.THEN, lexer.ELSE, lexer.ELIF, lexer.FI,
		 lexer.FOR, lexer.WHILE, lexer.DO, lexer.DONE, lexer.ESAC,
		 lexer.FUNCTION, lexer.IN, lexer.SELECT, lexer.TIME:
		return &Identifier{Value: p.curToken.Literal}
	case lexer.STRING, lexer.STRING_SINGLE, lexer.STRING_DOUBLE:
		// 判断是单引号还是双引号字符串
		isQuote := p.curToken.Type == lexer.STRING_DOUBLE
		// 注意：parseExpression 不应该移动 token，所以这里不调用 nextToken
		return &StringLiteral{Value: p.curToken.Literal, IsQuote: isQuote}
	case lexer.STRING_DOLLAR_SINGLE:
		// $'...' ANSI-C 字符串
		return &StringLiteral{Value: p.curToken.Literal, IsQuote: false}
	case lexer.STRING_DOLLAR_DOUBLE:
		// $"..." 国际化字符串
		return &StringLiteral{Value: p.curToken.Literal, IsQuote: true}
	case lexer.VAR:
		return &Variable{Name: p.curToken.Literal}
	case lexer.PARAM_EXPAND:
		// 参数展开 ${VAR...}
		return p.parseParamExpand(p.curToken.Literal)
	case lexer.COMMAND_SUBSTITUTION:
		return &CommandSubstitution{Command: p.curToken.Literal}
	case lexer.ARITHMETIC_EXPANSION:
		return &ArithmeticExpansion{Expression: p.curToken.Literal}
	case lexer.PROCESS_SUBSTITUTION_IN:
		return &ProcessSubstitution{Command: p.curToken.Literal, IsInput: true}
	case lexer.PROCESS_SUBSTITUTION_OUT:
		return &ProcessSubstitution{Command: p.curToken.Literal, IsInput: false}
	case lexer.DOLLAR:
		// 处理特殊变量，如 $?
		p.nextToken()
		if p.curToken.Type == lexer.IDENTIFIER {
			return &Variable{Name: p.curToken.Literal}
		}
		// $? 等单字符特殊变量
		return &Variable{Name: "?"}
	case lexer.NUMBER:
		return &StringLiteral{Value: p.curToken.Literal}
	default:
		return &Identifier{Value: p.curToken.Literal}
	}
}

// parseParamExpand 解析参数展开表达式
// 例如：${VAR:-default}, ${VAR#pattern}, ${VAR:offset:length} 等
func (p *Parser) parseParamExpand(expr string) *ParamExpandExpression {
	pe := &ParamExpandExpression{}
	
	// 解析 ${VAR...} 格式
	// expr 已经是 VAR... 部分（不包含 ${ 和 }）
	
	// 查找操作符
	// 操作符可能是：:-, :=, :?, :+, #, ##, %, %%, :, #, !, /, //, ^, ^^, ,, ,,
	// 以及数组访问 [index]
	
	// 先检查是否是数组访问
	if idx := strings.Index(expr, "["); idx != -1 {
		// 数组访问，如 arr[0] 或 arr[key]
		pe.VarName = expr[:idx]
		// 数组索引部分将在变量展开时处理
		pe.Word = expr[idx:]
		return pe
	}
	
	// 检查操作符
	ops := []string{"##", "#", "%%", "%", ":=", ":-", ":?", ":+", "::", ":", "//", "/", "^^", "^", ",,", ","}
	for _, op := range ops {
		if idx := strings.Index(expr, op); idx != -1 {
			pe.VarName = expr[:idx]
			pe.Op = op
			pe.Word = expr[idx+len(op):]
			return pe
		}
	}
	
	// 检查是否是 ${#VAR} 格式（字符串长度）
	if len(expr) > 0 && expr[0] == '#' {
		pe.VarName = expr[1:]
		pe.Op = "#"
		return pe
	}
	
	// 检查是否是 ${!VAR} 格式（间接引用）
	if len(expr) > 0 && expr[0] == '!' {
		pe.VarName = expr[1:]
		pe.Op = "!"
		return pe
	}
	
	// 没有操作符，只是简单的变量
	pe.VarName = expr
	return pe
}

// parseIfStatement 解析if语句
func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{}

	p.nextToken() // 跳过 if

	// 解析条件（跳过可能的分号）
	if p.curToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}
	
	// 解析条件
	stmt.Condition = p.parseCommandStatement()
	
	// 如果条件后面有分号，跳过它
	if p.curToken.Type == lexer.SEMICOLON {
		p.nextToken()
	}

	if p.peekToken.Type == lexer.THEN {
		p.nextToken() // 跳过 then
	}

	p.nextToken()
	stmt.Consequence = p.parseBlockStatement()

	// 解析elif
	for p.peekToken.Type == lexer.ELIF {
		p.nextToken() // 跳过 elif
		condition := p.parseCommandStatement()
		if p.peekToken.Type == lexer.THEN {
			p.nextToken()
		}
		p.nextToken()
		consequence := p.parseBlockStatement()
		stmt.Elif = append(stmt.Elif, &ElifClause{
			Condition:   condition,
			Consequence: consequence,
		})
	}

	// 解析else
	if p.peekToken.Type == lexer.ELSE {
		p.nextToken() // 跳过 else
		p.nextToken()
		stmt.Alternative = p.parseBlockStatement()
	}

	// 检查并跳过 fi
	// 注意：parseBlockStatement 在遇到 FI 时会停止，所以 curToken 应该在 FI 上
	if p.curToken.Type == lexer.FI {
		p.nextToken() // 跳过 fi
	} else if p.peekToken.Type == lexer.FI {
		p.nextToken() // 跳过 fi
	} else if p.curToken.Type != lexer.EOF {
		// 未闭合的 if 语句
		p.addError(ErrorTypeUnclosedControlFlow, "if 语句未闭合，缺少 fi", p.curToken, "fi")
	}

	return stmt
}

// parseForStatement 解析for循环
func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{}

	p.nextToken() // 跳过 for

	if p.peekToken.Type == lexer.IDENTIFIER {
		p.nextToken()
		stmt.Variable = p.curToken.Literal
	}

	if p.peekToken.Type == lexer.IN {
		p.nextToken() // 跳过 in
		// 解析列表
		for p.peekToken.Type != lexer.DO && p.peekToken.Type != lexer.SEMICOLON {
			p.nextToken()
			if p.curToken.Type == lexer.IDENTIFIER || p.curToken.Type == lexer.STRING {
				stmt.In = append(stmt.In, p.parseExpression())
			}
		}
	}

	if p.peekToken.Type == lexer.DO {
		p.nextToken() // 跳过 do
	}

	p.nextToken()
	stmt.Body = p.parseBlockStatement()

	if p.peekToken.Type == lexer.DONE {
		p.nextToken() // 跳过 done
	} else if p.curToken.Type != lexer.EOF {
		// 未闭合的 for 循环
		p.addError(ErrorTypeUnclosedControlFlow, "for 循环未闭合，缺少 done", p.curToken, "done")
	}

	return stmt
}

// parseWhileStatement 解析while循环
func (p *Parser) parseWhileStatement() *WhileStatement {
	stmt := &WhileStatement{}

	p.nextToken() // 跳过 while

	stmt.Condition = p.parseCommandStatement()
	
	// 如果parseCommandStatement在遇到]]后break，curToken仍然停留在]]上
	// 需要移动到下一个token（可能是分号或换行符）
	if p.curToken.Type == lexer.DBL_RBRACKET {
		p.nextToken()
	}

	// 跳过可能的分号和换行
	// 注意：如果curToken是EOF，说明字符串已经结束，无法继续解析
	if p.curToken.Type == lexer.EOF {
		// 如果已经是EOF，说明传递给解析器的字符串不完整
		// 这种情况下，循环体将为空，但我们应该继续解析（让调用者处理）
	} else {
		for p.curToken.Type == lexer.SEMICOLON || p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.WHITESPACE {
			p.nextToken()
			if p.curToken.Type == lexer.EOF {
				break
			}
		}
	}

	// 检查是否有 do 关键字
	if p.curToken.Type == lexer.DO {
		p.nextToken() // 跳过 do
	} else if p.peekToken.Type == lexer.DO {
		p.nextToken() // 跳过 do
	}

	// 跳过可能的分号和换行
	for p.curToken.Type == lexer.SEMICOLON || p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.WHITESPACE {
		p.nextToken()
	}

	stmt.Body = p.parseBlockStatement()

	// 跳过可能的分号和换行
	for p.curToken.Type == lexer.SEMICOLON || p.curToken.Type == lexer.NEWLINE || p.curToken.Type == lexer.WHITESPACE {
		p.nextToken()
	}

	// 检查并跳过 done
	if p.curToken.Type == lexer.DONE {
		p.nextToken() // 跳过 done
	} else if p.peekToken.Type == lexer.DONE {
		p.nextToken() // 跳过 done
	} else if p.curToken.Type != lexer.EOF {
		// 未闭合的 while 循环
		p.addError(ErrorTypeUnclosedControlFlow, "while 循环未闭合，缺少 done", p.curToken, "done")
	}

	return stmt
}

// parseFunctionStatement 解析函数定义
func (p *Parser) parseFunctionStatement() *FunctionStatement {
	stmt := &FunctionStatement{}

	p.nextToken() // 跳过 function

	// 函数名可能是 function name 或 name()
	if p.peekToken.Type == lexer.IDENTIFIER {
		p.nextToken()
		stmt.Name = p.curToken.Literal
		// 检查是否有括号
		if p.peekToken.Type == lexer.LPAREN {
			p.nextToken() // 跳过 (
			if p.peekToken.Type == lexer.RPAREN {
				p.nextToken() // 跳过 )
			}
		}
	} else if p.curToken.Type == lexer.IDENTIFIER && p.peekToken.Type == lexer.LPAREN {
		// name() 格式
		stmt.Name = p.curToken.Literal
		p.nextToken() // 跳过 (
		if p.peekToken.Type == lexer.RPAREN {
			p.nextToken() // 跳过 )
		}
	}

	p.nextToken()
	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseBlockStatement 解析代码块
func (p *Parser) parseBlockStatement() *BlockStatement {
	block := &BlockStatement{}
	block.Statements = []Statement{}

	stmtCount := 0
	for p.curToken.Type != lexer.EOF &&
		p.curToken.Type != lexer.FI &&
		p.curToken.Type != lexer.DONE &&
		p.curToken.Type != lexer.ELSE &&
		p.curToken.Type != lexer.ELIF &&
		p.curToken.Type != lexer.ESAC {
		// 如果遇到结束标记，停止解析
		if p.curToken.Type == lexer.FI ||
		   p.curToken.Type == lexer.DONE ||
		   p.curToken.Type == lexer.ELSE ||
		   p.curToken.Type == lexer.ELIF ||
		   p.curToken.Type == lexer.ESAC {
			break
		}
		// 跳过空白字符和换行（它们是语句分隔符）
		if p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
			continue
		}
		stmtCount++
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		// 如果解析后遇到结束标记，停止解析
		if p.curToken.Type == lexer.FI ||
		   p.curToken.Type == lexer.DONE ||
		   p.curToken.Type == lexer.ELSE ||
		   p.curToken.Type == lexer.ELIF ||
		   p.curToken.Type == lexer.ESAC {
			break
		}
		// 跳过空白字符和换行，准备解析下一个语句
		// 注意：parseCommandStatement 在遇到 NEWLINE 时会停止，但 curToken 仍然是 NEWLINE
		// 所以我们需要跳过它
		if p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
			// 跳过 NEWLINE 后，检查是否是结束标记
			if p.curToken.Type == lexer.FI ||
			   p.curToken.Type == lexer.DONE ||
			   p.curToken.Type == lexer.ELSE ||
			   p.curToken.Type == lexer.ELIF ||
			   p.curToken.Type == lexer.ESAC {
				break
			}
			continue
		}
		// 如果 curToken 是下一个语句的开始（如 BREAK、CONTINUE 等），继续循环解析
		// 不要调用 p.nextToken()，让下一次循环处理
		if p.curToken.Type == lexer.BREAK || 
		   p.curToken.Type == lexer.CONTINUE ||
		   p.curToken.Type == lexer.IDENTIFIER ||
		   p.curToken.Type == lexer.IF ||
		   p.curToken.Type == lexer.FOR ||
		   p.curToken.Type == lexer.WHILE ||
		   p.curToken.Type == lexer.CASE {
			// 这是下一个语句的开始，继续循环
			continue
		}
		// 如果 curToken 不是 NEWLINE 或 WHITESPACE，也不是下一个语句的开始，移动到下一个 token
		p.nextToken()
	}

	return block
}

// parseBreakStatement 解析break语句
func (p *Parser) parseBreakStatement() *BreakStatement {
	stmt := &BreakStatement{Level: 1}
	
	p.nextToken() // 跳过 break
	
	// 检查是否有数字参数（break n）
	if p.curToken.Type == lexer.NUMBER {
		// 解析数字
		if level, err := strconv.Atoi(p.curToken.Literal); err == nil && level > 0 {
			stmt.Level = level
		}
		p.nextToken() // 跳过数字
	} else if p.curToken.Type == lexer.IDENTIFIER {
		// 也可能是标识符形式的数字（虽然不常见）
		if level, err := strconv.Atoi(p.curToken.Literal); err == nil && level > 0 {
			stmt.Level = level
			p.nextToken() // 跳过标识符
		}
	}
	
	return stmt
}

// parseContinueStatement 解析continue语句
func (p *Parser) parseContinueStatement() *ContinueStatement {
	stmt := &ContinueStatement{Level: 1}
	
	p.nextToken() // 跳过 continue
	
	// 检查是否有数字参数（continue n）
	if p.curToken.Type == lexer.NUMBER {
		// 解析数字
		if level, err := strconv.Atoi(p.curToken.Literal); err == nil && level > 0 {
			stmt.Level = level
		}
		p.nextToken() // 跳过数字
	} else if p.curToken.Type == lexer.IDENTIFIER {
		// 也可能是标识符形式的数字（虽然不常见）
		if level, err := strconv.Atoi(p.curToken.Literal); err == nil && level > 0 {
			stmt.Level = level
			p.nextToken() // 跳过标识符
		}
	}
	
	return stmt
}

// parseSubshell 解析子shell命令 (command)
func (p *Parser) parseSubshell() *SubshellCommand {
	stmt := &SubshellCommand{}
	
	p.nextToken() // 跳过 (
	
	// 解析命令列表
	stmt.Body = p.parseBlockStatement()
	
	// 检查并跳过 )
	if p.curToken.Type == lexer.RPAREN {
		p.nextToken()
	} else if p.curToken.Type != lexer.EOF {
		// 未闭合的括号
		p.addError(ErrorTypeUnclosedParen, "未闭合的括号", p.curToken, ")")
	}
	
	return stmt
}

// parseGroupCommand 解析命令组 { command; }
func (p *Parser) parseGroupCommand() *GroupCommand {
	stmt := &GroupCommand{}
	
	p.nextToken() // 跳过 {
	
	// 解析命令列表
	stmt.Body = p.parseBlockStatement()
	
	// 检查并跳过 }
	if p.curToken.Type == lexer.RBRACE {
		p.nextToken()
	} else if p.curToken.Type != lexer.EOF {
		// 未闭合的大括号
		p.addError(ErrorTypeUnclosedBrace, "未闭合的大括号", p.curToken, "}")
	}
	
	return stmt
}

// Errors 返回解析错误（字符串列表，保持向后兼容）
func (p *Parser) Errors() []string {
	return p.errors
}

// ParseErrors 返回结构化解析错误列表
func (p *Parser) ParseErrors() []*ParseError {
	return p.parseErrors
}

// HasErrors 检查是否有解析错误
func (p *Parser) HasErrors() bool {
	return len(p.errors) > 0 || len(p.parseErrors) > 0
}


