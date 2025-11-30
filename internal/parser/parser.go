// Package parser 提供语法分析功能，将token序列解析为抽象语法树（AST）
package parser

import (
	"strings"
	"gobash/internal/lexer"
)

// Parser 语法分析器
// 负责将token序列解析为抽象语法树（AST），支持shell的各种语法结构
type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  lexer.Token
	peekToken lexer.Token
	
	// 用于回退
	savedTokens []lexer.Token
}

// New 创建新的解析器
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
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
			// 这是数组赋值 arr=(1 2 3)
			stmt := &ArrayAssignmentStatement{Name: arrayName, Values: []Expression{}}
			p.nextToken() // 跳过 arr= token
			p.nextToken() // 跳过 (
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
				stmt.Values = append(stmt.Values, p.parseExpression())
				p.nextToken()
			}
			if p.curToken.Type == lexer.RPAREN {
				p.nextToken() // 跳过 )
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
			// 读取值（可能是字符串、标识符等）
			var value strings.Builder
			for p.curToken.Type != lexer.EOF && 
			    p.curToken.Type != lexer.NEWLINE && 
			    p.curToken.Type != lexer.SEMICOLON &&
			    p.curToken.Type != lexer.WHITESPACE {
				if p.curToken.Type == lexer.STRING || 
				   p.curToken.Type == lexer.STRING_SINGLE || 
				   p.curToken.Type == lexer.STRING_DOUBLE {
					value.WriteString(p.curToken.Literal)
				} else if p.curToken.Type == lexer.IDENTIFIER {
					value.WriteString(p.curToken.Literal)
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
			p.nextToken()
			break
		}
		if p.curToken.Type == lexer.DBL_RBRACKET {
			// 将 ]] 作为参数添加（[[命令需要它）
			stmt.Args = append(stmt.Args, &Identifier{Value: "]]"})
			p.nextToken()
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
		
		// 检查重定向
		if p.curToken.Type == lexer.REDIRECT_OUT ||
		   p.curToken.Type == lexer.REDIRECT_IN ||
		   p.curToken.Type == lexer.REDIRECT_APPEND {
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
		}
		
		// 在移动到下一个token之前，检查下一个token是否是换行符或语句结束标记
		// 如果是，停止解析参数（换行符是语句分隔符）
		if p.peekToken.Type == lexer.NEWLINE ||
		   p.peekToken.Type == lexer.SEMICOLON ||
		   p.peekToken.Type == lexer.FI ||
		   p.peekToken.Type == lexer.DONE ||
		   p.peekToken.Type == lexer.ELSE ||
		   p.peekToken.Type == lexer.ELIF ||
		   p.peekToken.Type == lexer.ESAC {
			break
		}
		
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
	default:
		return nil
	}

	// 读取目标文件
	p.nextToken()
	if p.curToken.Type == lexer.IDENTIFIER || 
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
	case lexer.VAR:
		return &Variable{Name: p.curToken.Literal}
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

	if p.peekToken.Type == lexer.FI {
		p.nextToken() // 跳过 fi
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
	}

	return stmt
}

// parseWhileStatement 解析while循环
func (p *Parser) parseWhileStatement() *WhileStatement {
	stmt := &WhileStatement{}

	p.nextToken() // 跳过 while

	stmt.Condition = p.parseCommandStatement()

	if p.peekToken.Type == lexer.DO {
		p.nextToken() // 跳过 do
	}

	p.nextToken()
	stmt.Body = p.parseBlockStatement()

	if p.peekToken.Type == lexer.DONE {
		p.nextToken() // 跳过 done
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

	for p.curToken.Type != lexer.EOF &&
		p.curToken.Type != lexer.FI &&
		p.curToken.Type != lexer.DONE &&
		p.curToken.Type != lexer.ELSE &&
		p.curToken.Type != lexer.ELIF &&
		p.curToken.Type != lexer.ESAC {
		// 跳过空白字符和换行（它们是语句分隔符）
		if p.curToken.Type == lexer.WHITESPACE || p.curToken.Type == lexer.NEWLINE {
			p.nextToken()
			continue
		}
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
		p.nextToken()
	}

	return block
}

// Errors 返回解析错误
func (p *Parser) Errors() []string {
	return p.errors
}

