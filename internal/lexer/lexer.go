// Package lexer 提供词法分析功能，将输入字符串分解为token序列
package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer 词法分析器
// 负责将输入的shell命令字符串分解为一系列token
type Lexer struct {
	input        string
	position     int  // 当前位置
	readPosition int  // 读取位置
	ch           byte // 当前字符
	line         int  // 当前行号
	column       int  // 当前列号
}

// New 创建新的词法分析器
func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 1,
	}
	l.readChar()
	return l
}

// readChar 读取下一个字符
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	if l.ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
}

// peekChar 查看下一个字符但不移动位置
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// peekChar2 查看下下个字符
func (l *Lexer) peekChar2() byte {
	if l.readPosition+1 >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition+1]
}

// NextToken 读取下一个token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: OR, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(PIPE, l.ch, tok.Line, tok.Column)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: AND, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(AMPERSAND, l.ch, tok.Line, tok.Column)
		}
	case '>':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: REDIRECT_APPEND, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '(' {
			// 进程替换 >(command)
			startLine := l.line
			startColumn := l.column
			l.readChar() // 跳过 >
			l.readChar() // 跳过 (
			tok = l.readProcessSubstitution()
			tok.Type = PROCESS_SUBSTITUTION_OUT
			tok.Line = startLine
			tok.Column = startColumn
		} else if isDigit(l.peekChar()) {
			// 处理文件描述符重定向，如 2>
			tok = l.readRedirectFD()
		} else {
			tok = newToken(REDIRECT_OUT, l.ch, tok.Line, tok.Column)
		}
	case '<':
		if l.peekChar() == '(' {
			// 进程替换 <(command)
			startLine := l.line
			startColumn := l.column
			l.readChar() // 跳过 <
			l.readChar() // 跳过 (
			tok = l.readProcessSubstitution()
			tok.Type = PROCESS_SUBSTITUTION_IN
			tok.Line = startLine
			tok.Column = startColumn
		} else {
			tok = newToken(REDIRECT_IN, l.ch, tok.Line, tok.Column)
		}
	case ';':
		tok = newToken(SEMICOLON, l.ch, tok.Line, tok.Column)
	case '(':
		tok = newToken(LPAREN, l.ch, tok.Line, tok.Column)
	case ')':
		tok = newToken(RPAREN, l.ch, tok.Line, tok.Column)
	case '{':
		tok = newToken(LBRACE, l.ch, tok.Line, tok.Column)
	case '}':
		tok = newToken(RBRACE, l.ch, tok.Line, tok.Column)
	case '[':
		if l.peekChar() == '[' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: DBL_LBRACKET, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(LBRACKET, l.ch, tok.Line, tok.Column)
		}
	case ']':
		if l.peekChar() == ']' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: DBL_RBRACKET, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(RBRACKET, l.ch, tok.Line, tok.Column)
		}
	case '=':
		// = 字符应该已经在标识符读取时处理了（数组赋值 arr=(...)）
		// 如果单独出现，作为非法字符
		tok = newToken(ILLEGAL, l.ch, tok.Line, tok.Column)
	case '\'':
		tok = l.readString('\'')
		tok.Type = STRING_SINGLE
	case '"':
		tok = l.readString('"')
		tok.Type = STRING_DOUBLE
	case '`':
		tok = l.readCommandSubstitution()
	case '\\':
		tok = newToken(ESCAPE, l.ch, tok.Line, tok.Column)
	case '$':
		// 检查是否是 $((expr)) 格式的算术展开
		if l.peekChar() == '(' {
			peek2 := l.peekChar2()
			if peek2 == '(' {
				// $(( 算术展开
				startLine := l.line
				startColumn := l.column
				l.readChar() // 跳过 $
				l.readChar() // 跳过第一个 (
				l.readChar() // 跳过第二个 (
				tok = l.readArithmeticExpansion()
				tok.Line = startLine
				tok.Column = startColumn
			} else {
				// $(command) 命令替换
				startLine := l.line
				startColumn := l.column
				l.readChar() // 跳过 $
				l.readChar() // 跳过 (
				tok = l.readCommandSubstitutionParen()
				tok.Line = startLine
				tok.Column = startColumn
			}
		} else {
			tok = l.readVariable()
		}
	case 0:
		tok.Literal = ""
		tok.Type = EOF
		tok.Line = l.line
		tok.Column = l.column
	case '\n':
		tok = newToken(NEWLINE, l.ch, tok.Line, tok.Column)
	default:
		if isLetter(l.ch) || l.ch == '_' {
			// 先尝试读取标识符，但检查是否包含点号（文件名）
			ident := l.readIdentifier()
			// 如果下一个字符是点号，继续读取（可能是文件名）
			if l.ch == '.' {
				tok.Literal = ident + l.readIdentifierOrPath()
				tok.Type = IDENTIFIER
				tok.Line = l.line
				tok.Column = l.column
				return tok
			}
			// 检查是否是数组元素赋值 arr[key]=value 或 arr[0]=value
			// 如果下一个字符是 [，读取直到 ]，然后检查是否是 =
			if l.ch == '[' {
				// 读取 [key] 或 [0]
				bracketPart := "["
				l.readChar() // 跳过 [
				for l.ch != ']' && l.ch != 0 && l.ch != '\n' {
					bracketPart += string(l.ch)
					l.readChar()
				}
				if l.ch == ']' {
					bracketPart += "]"
					l.readChar() // 跳过 ]
					// 检查下一个字符是否是 =
					if l.ch == '=' {
						// 这是数组元素赋值 arr[key]= 或 arr[0]=
						tok.Literal = ident + bracketPart + "="
						tok.Type = IDENTIFIER
						tok.Line = l.line
						tok.Column = l.column
						l.readChar() // 跳过 =
						return tok
					}
					// 不是赋值，只是数组访问，将 [key] 作为标识符的一部分
					tok.Literal = ident + bracketPart
					tok.Type = IDENTIFIER
					tok.Line = l.line
					tok.Column = l.column
					return tok
				}
			}
			// 检查是否是数组赋值 arr=(...)
			// 如果下一个字符是 = 且再下一个字符是 (，将 = 包含在标识符中
			if l.ch == '=' && l.peekChar() == '(' {
				tok.Literal = ident + "="
				tok.Type = IDENTIFIER
				tok.Line = l.line
				tok.Column = l.column
				l.readChar() // 跳过 =
				return tok
			}
			tok.Literal = ident
			tok.Type = LookupIdent(ident)
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else if isDigit(l.ch) {
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else {
			// 其他字符作为标识符的一部分（如路径中的/或.）
			if l.ch != 0 {
				tok.Literal = l.readIdentifierOrPath()
				tok.Type = IDENTIFIER
				tok.Line = l.line
				tok.Column = l.column
				return tok
			}
			tok = newToken(ILLEGAL, l.ch, tok.Line, tok.Column)
		}
	}

	l.readChar()
	return tok
}

// newToken 创建新token
func newToken(tokenType TokenType, ch byte, line, column int) Token {
	return Token{
		Type:    tokenType,
		Literal: string(ch),
		Line:    line,
		Column:  column,
	}
}

// readIdentifier 读取标识符
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readIdentifierOrPath 读取标识符或路径（包含特殊字符如点号、斜杠）
func (l *Lexer) readIdentifierOrPath() string {
	position := l.position
	for l.ch != 0 && 
		l.ch != ' ' && 
		l.ch != '\t' && 
		l.ch != '\n' &&
		l.ch != '\r' &&
		l.ch != '|' &&
		l.ch != '>' &&
		l.ch != '<' &&
		l.ch != '&' &&
		l.ch != ';' &&
		l.ch != '(' &&
		l.ch != ')' &&
		l.ch != '{' &&
		l.ch != '}' &&
		l.ch != '[' &&
		l.ch != ']' &&
		l.ch != '$' &&
		l.ch != '\'' &&
		l.ch != '"' &&
		l.ch != '`' &&
		l.ch != '=' { // 停止在 = 处，以便识别数组赋值
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber 读取数字
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readVariable 读取变量
func (l *Lexer) readVariable() Token {
	startLine := l.line
	startColumn := l.column
	l.readChar() // 跳过 $

	if l.ch == '{' {
		// ${VAR} 格式
		l.readChar() // 跳过 {
		position := l.position
		for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
			l.readChar()
		}
		if l.ch == '}' {
			varName := l.input[position:l.position]
			l.readChar() // 跳过 }
			return Token{
				Type:    VAR,
				Literal: varName,
				Line:    startLine,
				Column:  startColumn,
			}
		}
		return Token{Type: ILLEGAL, Literal: "${", Line: startLine, Column: startColumn}
	} else if isLetter(l.ch) || l.ch == '_' {
		// $VAR 格式
		position := l.position
		for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
			l.readChar()
		}
		return Token{
			Type:    VAR,
			Literal: l.input[position:l.position],
			Line:    startLine,
			Column:  startColumn,
		}
	} else {
		// 单独的 $，可能是 $? 等特殊变量
		return Token{
			Type:    DOLLAR,
			Literal: "$",
			Line:    startLine,
			Column:  startColumn,
		}
	}
}

// readRedirectFD 读取文件描述符重定向
func (l *Lexer) readRedirectFD() Token {
	startLine := l.line
	startColumn := l.column
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	fd := l.input[position:l.position]
	if l.ch == '>' {
		l.readChar()
		if l.ch == '>' {
			l.readChar()
			return Token{
				Type:    REDIRECT_APPEND,
				Literal: fd + ">>",
				Line:    startLine,
				Column:  startColumn,
			}
		}
		return Token{
			Type:    REDIRECT_OUT,
			Literal: fd + ">",
			Line:    startLine,
			Column:  startColumn,
		}
	}
	return Token{Type: ILLEGAL, Literal: fd, Line: startLine, Column: startColumn}
}

// readString 读取字符串（单引号、双引号或反引号）
func (l *Lexer) readString(quote byte) Token {
	startLine := l.line
	startColumn := l.column
	l.readChar() // 跳过开始的引号

	var literal strings.Builder
	for l.ch != quote && l.ch != 0 {
		if quote == '"' && l.ch == '\\' {
			// 双引号内允许转义
			l.readChar()
			if l.ch != 0 {
				literal.WriteByte(l.ch)
				l.readChar()
			}
		} else if quote == '"' && l.ch == '$' {
			// 双引号内需要保留 $ 以便后续展开变量
			literal.WriteByte(l.ch)
			l.readChar()
		} else {
			literal.WriteByte(l.ch)
			l.readChar()
		}
	}

	var result string
	if l.ch == quote {
		result = literal.String()
		l.readChar() // 跳过结束引号
	} else {
		// 未闭合的引号
		result = literal.String()
	}

	return Token{
		Type:    STRING,
		Literal: result,
		Line:    startLine,
		Column:  startColumn,
	}
}

// readCommandSubstitution 读取命令替换（反引号）
func (l *Lexer) readCommandSubstitution() Token {
	startLine := l.line
	startColumn := l.column
	l.readChar() // 跳过开始的反引号
	
	var literal strings.Builder
	for l.ch != '`' && l.ch != 0 {
		if l.ch == '\\' {
			// 转义字符
			l.readChar()
			if l.ch != 0 {
				literal.WriteByte(l.ch)
				l.readChar()
			}
		} else {
			literal.WriteByte(l.ch)
			l.readChar()
		}
	}
	
	var result string
	if l.ch == '`' {
		result = literal.String()
		l.readChar() // 跳过结束反引号
	} else {
		// 未闭合的反引号
		result = literal.String()
	}
	
	return Token{
		Type:    COMMAND_SUBSTITUTION,
		Literal: result,
		Line:    startLine,
		Column:  startColumn,
	}
}

// readArithmeticExpansion 读取算术展开（$((expr))格式）
func (l *Lexer) readArithmeticExpansion() Token {
	var literal strings.Builder
	depth := 2 // 已经有两个开括号
	
	for depth > 0 && l.ch != 0 {
		if l.ch == '(' {
			depth++
			literal.WriteByte(l.ch)
			l.readChar()
		} else if l.ch == ')' {
			depth--
			if depth > 0 {
				literal.WriteByte(l.ch)
			}
			if depth == 0 {
				l.readChar() // 跳过结束括号
				break
			}
			l.readChar()
		} else {
			literal.WriteByte(l.ch)
			l.readChar()
		}
	}
	
	return Token{
		Type:    ARITHMETIC_EXPANSION,
		Literal: literal.String(),
		Line:    l.line,
		Column:  l.column,
	}
}

// readCommandSubstitutionParen 读取命令替换（$(command)格式）
func (l *Lexer) readCommandSubstitutionParen() Token {
	var literal strings.Builder
	depth := 1 // 已经有一个开括号
	
	for depth > 0 && l.ch != 0 {
		if l.ch == '(' {
			depth++
			literal.WriteByte(l.ch)
			l.readChar()
		} else if l.ch == ')' {
			depth--
			if depth > 0 {
				literal.WriteByte(l.ch)
			}
			if depth == 0 {
				l.readChar() // 跳过结束括号
				break
			}
			l.readChar()
		} else if l.ch == '\\' {
			// 转义字符
			literal.WriteByte(l.ch)
			l.readChar()
			if l.ch != 0 {
				literal.WriteByte(l.ch)
				l.readChar()
			}
		} else {
			literal.WriteByte(l.ch)
			l.readChar()
		}
	}
	
	return Token{
		Type:    COMMAND_SUBSTITUTION,
		Literal: literal.String(),
		Line:    l.line,
		Column:  l.column,
	}
}

// readProcessSubstitution 读取进程替换（<(command)或>(command)格式）
func (l *Lexer) readProcessSubstitution() Token {
	var literal strings.Builder
	depth := 1 // 已经有一个开括号
	
	for depth > 0 && l.ch != 0 {
		if l.ch == '(' {
			depth++
			literal.WriteByte(l.ch)
			l.readChar()
		} else if l.ch == ')' {
			depth--
			if depth > 0 {
				literal.WriteByte(l.ch)
			}
			if depth == 0 {
				l.readChar() // 跳过结束括号
				break
			}
			l.readChar()
		} else if l.ch == '\\' {
			// 转义字符
			literal.WriteByte(l.ch)
			l.readChar()
			if l.ch != 0 {
				literal.WriteByte(l.ch)
				l.readChar()
			}
		} else {
			literal.WriteByte(l.ch)
			l.readChar()
		}
	}
	
	return Token{
		Type:    PROCESS_SUBSTITUTION_IN, // 临时类型，实际类型由调用者设置
		Literal: literal.String(),
		Line:    l.line,
		Column:  l.column,
	}
}

// skipWhitespace 跳过空白字符
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

// isLetter 判断是否为字母
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch >= utf8.RuneSelf && unicode.IsLetter(rune(ch))
}

// isDigit 判断是否为数字
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

