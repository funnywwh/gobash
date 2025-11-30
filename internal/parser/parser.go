package parser

import (
	"gobash/internal/lexer"
)

// Parser 语法分析器
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
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
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
		default:
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

	// 解析命令（包括 [ 命令）
	if p.curToken.Type == lexer.IDENTIFIER || 
	   p.curToken.Type == lexer.STRING ||
	   p.curToken.Type == lexer.STRING_SINGLE ||
	   p.curToken.Type == lexer.STRING_DOUBLE ||
	   p.curToken.Type == lexer.LBRACKET {
		if p.curToken.Type == lexer.LBRACKET {
			// [ 命令，创建一个标识符表达式
			stmt.Command = &Identifier{Value: "["}
		} else {
			stmt.Command = p.parseExpression()
		}
		p.nextToken()
	}

	// 解析参数和重定向
	for p.curToken.Type != lexer.EOF && 
		p.curToken.Type != lexer.SEMICOLON &&
		p.curToken.Type != lexer.NEWLINE &&
		p.curToken.Type != lexer.PIPE &&
		p.curToken.Type != lexer.AND &&
		p.curToken.Type != lexer.OR &&
		p.curToken.Type != lexer.AMPERSAND &&
		p.curToken.Type != lexer.THEN &&
		p.curToken.Type != lexer.DO &&
		p.curToken.Type != lexer.DONE &&
		p.curToken.Type != lexer.FI &&
		p.curToken.Type != lexer.ELSE &&
		p.curToken.Type != lexer.ELIF {
		
		// 检查是否是 [ 命令的结束括号
		if p.curToken.Type == lexer.RBRACKET {
			// 将 ] 作为参数添加（test命令需要它）
			stmt.Args = append(stmt.Args, &Identifier{Value: "]"})
			p.nextToken()
			break
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
		if p.curToken.Type == lexer.IDENTIFIER || 
		   p.curToken.Type == lexer.STRING ||
		   p.curToken.Type == lexer.STRING_SINGLE ||
		   p.curToken.Type == lexer.STRING_DOUBLE ||
		   p.curToken.Type == lexer.VAR ||
		   p.curToken.Type == lexer.DOLLAR ||
		   p.curToken.Type == lexer.COMMAND_SUBSTITUTION ||
		   p.curToken.Type == lexer.ARITHMETIC_EXPANSION ||
		   p.curToken.Type == lexer.NUMBER {
			stmt.Args = append(stmt.Args, p.parseExpression())
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
	case lexer.STRING, lexer.STRING_SINGLE, lexer.STRING_DOUBLE:
		// 判断是单引号还是双引号字符串
		isQuote := p.curToken.Type == lexer.STRING_DOUBLE
		return &StringLiteral{Value: p.curToken.Literal, IsQuote: isQuote}
	case lexer.VAR:
		return &Variable{Name: p.curToken.Literal}
	case lexer.COMMAND_SUBSTITUTION:
		return &CommandSubstitution{Command: p.curToken.Literal}
	case lexer.ARITHMETIC_EXPANSION:
		return &ArithmeticExpansion{Expression: p.curToken.Literal}
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
		p.curToken.Type != lexer.ELIF {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

// Errors 返回解析错误
func (p *Parser) Errors() []string {
	return p.errors
}

