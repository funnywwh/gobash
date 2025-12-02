// Package lexer 提供词法分析功能，将输入字符串分解为token序列
package lexer

import (
	"strconv"
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
		} else if l.peekChar() == '&' {
			// |& 管道和stderr
			ch := l.ch
			l.readChar()
			tok = Token{Type: BAR_AND, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(PIPE, l.ch, tok.Line, tok.Column)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: AND, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '>' {
			// &> 或 &>>
			ch := l.ch
			l.readChar() // 跳过 &
			if l.peekChar() == '>' {
				// &>> 追加
				l.readChar()
				tok = Token{Type: AND_GREATER_GREATER, Literal: string(ch) + string(l.ch) + string(l.peekChar()), Line: tok.Line, Column: tok.Column}
			} else {
				// &> 覆盖
				tok = Token{Type: AND_GREATER, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
			}
		} else {
			tok = newToken(AMPERSAND, l.ch, tok.Line, tok.Column)
		}
	case '>':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: REDIRECT_APPEND, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '&' {
			// >& 重定向
			ch := l.ch
			l.readChar()
			tok = Token{Type: REDIRECT_DUP_OUT, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '|' {
			// >| 强制覆盖
			ch := l.ch
			l.readChar()
			tok = Token{Type: REDIRECT_CLOBBER, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
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
		if l.peekChar() == '<' {
			// << 或 <<- 或 <<<
			ch := l.ch
			l.readChar() // 跳过第一个 <
			peek2 := l.peekChar()
			if peek2 == '-' {
				// <<- Here-document with strip tabs
				l.readChar() // 跳过 -
				tok = Token{Type: REDIRECT_HEREDOC_STRIP, Literal: string(ch) + string(l.ch) + string(peek2), Line: tok.Line, Column: tok.Column}
			} else if peek2 == '<' {
				// <<< Here-string
				l.readChar() // 跳过第二个 <
				tok = Token{Type: REDIRECT_HEREDOC_TABS, Literal: string(ch) + string(l.ch) + string(peek2), Line: tok.Line, Column: tok.Column}
			} else {
				// << Here-document
				tok = Token{Type: REDIRECT_HEREDOC, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
			}
		} else if l.peekChar() == '&' {
			// <& 重定向
			ch := l.ch
			l.readChar()
			tok = Token{Type: REDIRECT_DUP_IN, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '>' {
			// <> 读写重定向
			ch := l.ch
			l.readChar()
			tok = Token{Type: REDIRECT_RW, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else if l.peekChar() == '(' {
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
		if l.peekChar() == ';' {
			ch := l.ch
			l.readChar()
			peek2 := l.peekChar()
			if peek2 == '&' {
				// ;;& case 语句
				l.readChar()
				tok = Token{Type: SEMI_SEMI_AND, Literal: string(ch) + string(l.ch) + string(peek2), Line: tok.Line, Column: tok.Column}
			} else {
				// ;; case 语句
				tok = Token{Type: SEMI_SEMI, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
			}
		} else if l.peekChar() == '&' {
			// ;& case 语句
			ch := l.ch
			l.readChar()
			tok = Token{Type: SEMI_AND, Literal: string(ch) + string(l.ch), Line: tok.Line, Column: tok.Column}
		} else {
			tok = newToken(SEMICOLON, l.ch, tok.Line, tok.Column)
		}
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
		// 检查是否是行尾的反斜杠（转义的换行符）
		peek := l.peekChar()
		if peek == '\n' {
			// 行尾的反斜杠，跳过反斜杠和换行符（多行命令）
			l.readChar() // 跳过反斜杠
			l.readChar() // 跳过换行符
			// 继续读取下一个 token
			return l.NextToken()
		}
		tok = newToken(ESCAPE, l.ch, tok.Line, tok.Column)
	case '$':
		// 检查是否是 $'...' 或 $"..." 格式
		peek1 := l.peekChar()
		if peek1 == '\'' {
			// $'...' ANSI-C 字符串
			startLine := l.line
			startColumn := l.column
			l.readChar() // 跳过 $
			l.readChar() // 跳过 '
			tok = l.readDollarSingleQuote()
			tok.Type = STRING_DOLLAR_SINGLE
			tok.Line = startLine
			tok.Column = startColumn
		} else if peek1 == '"' {
			// $"..." 国际化字符串
			startLine := l.line
			startColumn := l.column
			l.readChar() // 跳过 $
			l.readChar() // 跳过 "
			tok = l.readDollarDoubleQuote()
			tok.Type = STRING_DOLLAR_DOUBLE
			tok.Line = startLine
			tok.Column = startColumn
		} else if peek1 == '(' {
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
		// 普通换行符（转义的换行符已经在反斜杠处理中被处理）
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
		} else if l.ch == '-' {
			// 处理以 - 开头的标识符（如 --win, -a）
			// 使用 readIdentifierOrPath 来读取，因为它可以处理包含 - 的字符串
			tok.Literal = l.readIdentifierOrPath()
			tok.Type = IDENTIFIER
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

	// 处理特殊变量 $#, $@, $*, $?, $!, $$, $0
	if l.ch == '#' {
		l.readChar() // 跳过 #
		return Token{
			Type:    VAR,
			Literal: "#",
			Line:    startLine,
			Column:  startColumn,
		}
	}
	if l.ch == '@' {
		l.readChar() // 跳过 @
		return Token{
			Type:    VAR,
			Literal: "@",
			Line:    startLine,
			Column:  startColumn,
		}
	}
	if l.ch == '*' {
		l.readChar() // 跳过 *
		return Token{
			Type:    VAR,
			Literal: "*",
			Line:    startLine,
			Column:  startColumn,
		}
	}
	if l.ch == '?' {
		l.readChar() // 跳过 ?
		return Token{
			Type:    VAR,
			Literal: "?",
			Line:    startLine,
			Column:  startColumn,
		}
	}
	if l.ch == '!' {
		l.readChar() // 跳过 !
		return Token{
			Type:    VAR,
			Literal: "!",
			Line:    startLine,
			Column:  startColumn,
		}
	}
	if l.ch == '$' {
		l.readChar() // 跳过 $
		return Token{
			Type:    VAR,
			Literal: "$",
			Line:    startLine,
			Column:  startColumn,
		}
	}
	if l.ch == '0' {
		l.readChar() // 跳过 0
		return Token{
			Type:    VAR,
			Literal: "0",
			Line:    startLine,
			Column:  startColumn,
		}
	}
	if isDigit(l.ch) {
		// $1, $2, ... 位置参数
		position := l.position
		for isDigit(l.ch) {
			l.readChar()
		}
		return Token{
			Type:    VAR,
			Literal: l.input[position:l.position],
			Line:    startLine,
			Column:  startColumn,
		}
	}

	if l.ch == '{' {
		// ${VAR} 或 ${VAR...} 参数展开格式
		l.readChar() // 跳过 {
		position := l.position
		// 读取完整的参数展开表达式（包括所有操作符和值）
		// 例如：${VAR:-default}, ${VAR#pattern}, ${VAR:offset:length} 等
		depth := 1 // 括号深度
		for depth > 0 && l.ch != 0 {
			if l.ch == '{' {
				depth++
			} else if l.ch == '}' {
				depth--
				if depth == 0 {
					// 找到匹配的 }
					varExpr := l.input[position:l.position]
					l.readChar() // 跳过 }
					return Token{
						Type:    PARAM_EXPAND,
						Literal: varExpr,
						Line:    startLine,
						Column:  startColumn,
					}
				}
			} else if l.ch == '\'' || l.ch == '"' {
				// 跳过引号内的内容
				quote := l.ch
				l.readChar()
				for l.ch != quote && l.ch != 0 {
					if l.ch == '\\' && quote == '"' {
						l.readChar() // 跳过转义字符
					}
					l.readChar()
				}
				if l.ch == quote {
					l.readChar()
				}
				continue
			} else if l.ch == '`' {
				// 跳过命令替换
				l.readChar()
				for l.ch != '`' && l.ch != 0 {
					if l.ch == '\\' {
						l.readChar()
					}
					l.readChar()
				}
				if l.ch == '`' {
					l.readChar()
				}
				continue
			} else if l.ch == '$' && l.peekChar() == '(' {
				// 跳过 $(command) 命令替换
				l.readChar() // 跳过 $
				l.readChar() // 跳过 (
				cmdDepth := 1
				for cmdDepth > 0 && l.ch != 0 {
					if l.ch == '(' {
						cmdDepth++
					} else if l.ch == ')' {
						cmdDepth--
					} else if l.ch == '\\' {
						l.readChar() // 跳过转义字符
					}
					l.readChar()
				}
				continue
			}
			l.readChar()
		}
		// 如果没有找到匹配的 }，返回错误
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
	
	for l.ch != 0 {
		if quote == '"' && l.ch == '\\' {
			// 双引号内的转义处理
			nextPos := l.readPosition
			if nextPos < len(l.input) {
				nextCh := l.input[nextPos]
				if nextCh == '"' {
					// \" 转义为 "，只保存 " 而不保存 \
					literal.WriteByte('"')
					l.readChar() // 跳过 \
					l.readChar() // 跳过 "
					continue
				} else if nextCh == '\n' {
					// \n 转义为换行符，保留换行符但跳过反斜杠
					literal.WriteByte('\n')
					l.readChar() // 跳过 \
					l.readChar() // 跳过 \n（readChar 会自动更新行号）
					continue
				} else if nextCh == '\\' {
					// \\ 转义为单个反斜杠
					literal.WriteByte('\\')
					l.readChar() // 跳过第一个 \
					l.readChar() // 跳过第二个 \
					continue
				} else {
					// 其他转义序列保持原样（\$、\t等）
					literal.WriteByte(l.ch) // 写入 \
					l.readChar()
					if l.ch != 0 && l.ch != quote {
						literal.WriteByte(l.ch) // 写入转义字符
						l.readChar()
					}
					continue
				}
			}
		}
		
		if l.ch == quote {
			// 找到结束引号
			break
		}
		
		// 普通字符（包括空白字符，引号内的空白字符应该被保留）
		literal.WriteByte(l.ch)
		l.readChar()
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
// 正确处理嵌套的反引号、引号等
func (l *Lexer) readCommandSubstitution() Token {
	startLine := l.line
	startColumn := l.column
	l.readChar() // 跳过开始的反引号
	
	var literal strings.Builder
	backtickDepth := 1 // 反引号深度（支持嵌套的反引号，虽然 bash 不支持，但为了健壮性）
	
	for backtickDepth > 0 && l.ch != 0 {
		if l.ch == '`' {
			// 检查是否是转义的反引号
			if literal.Len() > 0 {
				lastChar := literal.String()[literal.Len()-1]
				if lastChar == '\\' {
					// 转义的反引号，写入字面量
					literal.WriteByte(l.ch)
					l.readChar()
					continue
				}
			}
			backtickDepth--
			if backtickDepth == 0 {
				l.readChar() // 跳过结束反引号
				break
			}
			literal.WriteByte(l.ch)
			l.readChar()
		} else if l.ch == '\'' || l.ch == '"' {
			// 处理引号内的内容（引号内的反引号不应该结束命令替换）
			quote := l.ch
			literal.WriteByte(l.ch)
			l.readChar()
			for l.ch != quote && l.ch != 0 {
				if l.ch == '\\' && quote == '"' {
					// 双引号内的转义
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
			if l.ch == quote {
				literal.WriteByte(l.ch)
				l.readChar()
			}
		} else if l.ch == '\\' {
			// 转义字符
			literal.WriteByte(l.ch)
			l.readChar()
			if l.ch != 0 {
				literal.WriteByte(l.ch)
				l.readChar()
			}
		} else if l.ch == '$' && l.peekChar() == '(' {
			// 嵌套的命令替换 $(...)，需要正确处理
			literal.WriteByte(l.ch)
			l.readChar() // 跳过 $
			literal.WriteByte(l.ch) // 写入 (
			l.readChar() // 跳过 (
			parenDepth := 1
			for parenDepth > 0 && l.ch != 0 {
				if l.ch == '(' {
					parenDepth++
					literal.WriteByte(l.ch)
					l.readChar()
				} else if l.ch == ')' {
					parenDepth--
					literal.WriteByte(l.ch)
					if parenDepth == 0 {
						l.readChar()
						break
					}
					l.readChar()
				} else if l.ch == '\'' || l.ch == '"' {
					// 处理引号
					quote := l.ch
					literal.WriteByte(l.ch)
					l.readChar()
					for l.ch != quote && l.ch != 0 {
						if l.ch == '\\' && quote == '"' {
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
					if l.ch == quote {
						literal.WriteByte(l.ch)
						l.readChar()
					}
				} else {
					literal.WriteByte(l.ch)
					l.readChar()
				}
			}
		} else {
			literal.WriteByte(l.ch)
			l.readChar()
		}
	}
	
	return Token{
		Type:    COMMAND_SUBSTITUTION,
		Literal: literal.String(),
		Line:    startLine,
		Column:  startColumn,
	}
}

// readArithmeticExpansion 读取算术展开（$((expr))格式）
// 正确处理嵌套的括号、引号、变量展开等
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
			if depth >= 2 {
				// depth >= 2 表示这是表达式内部的 )，应该写入 literal
				literal.WriteByte(l.ch)
				l.readChar()
			} else if depth == 0 {
				// depth == 0 表示这是结束的 ))，应该跳过
				l.readChar() // 跳过结束括号
				break
			} else {
				// depth == 1，这是结束的 )) 的第一个 )，不应该写入 literal
				// 但需要检查下一个字符是否是 )
				l.readChar()
				if l.ch == ')' {
					// 这是结束的 ))，跳过
					l.readChar()
					break
				} else {
					// 这不是结束的 ))，可能是其他情况，写入刚才的 )
					literal.WriteByte(')')
				}
			}
		} else if l.ch == '\'' || l.ch == '"' {
			// 处理引号内的内容（虽然算术表达式中引号不常见，但为了健壮性处理）
			quote := l.ch
			literal.WriteByte(l.ch)
			l.readChar()
			for l.ch != quote && l.ch != 0 {
				if l.ch == '\\' && quote == '"' {
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
			if l.ch == quote {
				literal.WriteByte(l.ch)
				l.readChar()
			}
		} else if l.ch == '$' && l.peekChar() == '(' {
			// 嵌套的命令替换或算术展开
			peek2 := l.peekChar2()
			if peek2 == '(' {
				// $((...)) 嵌套的算术展开，需要完整保留包括结束的 ))
				literal.WriteByte(l.ch)
				l.readChar() // 跳过 $
				literal.WriteByte(l.ch) // 写入第一个 (
				l.readChar() // 跳过第一个 (
				literal.WriteByte(l.ch) // 写入第二个 (
				l.readChar() // 跳过第二个 (
				nestedDepth := 2
				for nestedDepth > 0 && l.ch != 0 {
					if l.ch == '(' {
						nestedDepth++
						literal.WriteByte(l.ch)
						l.readChar()
					} else if l.ch == ')' {
						nestedDepth--
						if nestedDepth >= 2 {
							// 表达式内部的 )
							literal.WriteByte(l.ch)
							l.readChar()
						} else if nestedDepth == 0 {
							// 结束的 ))，需要写入两个 )
							literal.WriteByte(l.ch) // 写入第一个 )
							l.readChar()
							if l.ch == ')' {
								literal.WriteByte(l.ch) // 写入第二个 )
								l.readChar()
							}
							break
						} else {
							// depth == 1，这是结束的 )) 的第一个 )
							literal.WriteByte(l.ch) // 写入第一个 )
							l.readChar()
							if l.ch == ')' {
								literal.WriteByte(l.ch) // 写入第二个 )
								l.readChar()
								break
							}
						}
					} else {
						literal.WriteByte(l.ch)
						l.readChar()
					}
				}
			} else {
				// $(...) 命令替换（在算术展开中）
				literal.WriteByte(l.ch)
				l.readChar() // 跳过 $
				literal.WriteByte(l.ch) // 写入 (
				l.readChar() // 跳过 (
				nestedDepth := 1
				for nestedDepth > 0 && l.ch != 0 {
					if l.ch == '(' {
						nestedDepth++
						literal.WriteByte(l.ch)
						l.readChar()
					} else if l.ch == ')' {
						nestedDepth--
						literal.WriteByte(l.ch)
						if nestedDepth == 0 {
							l.readChar()
							break
						}
						l.readChar()
					} else if l.ch == '\'' || l.ch == '"' {
						quote := l.ch
						literal.WriteByte(l.ch)
						l.readChar()
						for l.ch != quote && l.ch != 0 {
							if l.ch == '\\' && quote == '"' {
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
						if l.ch == quote {
							literal.WriteByte(l.ch)
							l.readChar()
						}
					} else {
						literal.WriteByte(l.ch)
						l.readChar()
					}
				}
			}
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
// 正确处理嵌套的括号、引号、命令替换等
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
		} else if l.ch == '\'' || l.ch == '"' {
			// 处理引号内的内容（引号内的括号不应该影响深度计数）
			quote := l.ch
			literal.WriteByte(l.ch)
			l.readChar()
			for l.ch != quote && l.ch != 0 {
				if l.ch == '\\' && quote == '"' {
					// 双引号内的转义
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
			if l.ch == quote {
				literal.WriteByte(l.ch)
				l.readChar()
			}
		} else if l.ch == '`' {
			// 嵌套的反引号命令替换
			literal.WriteByte(l.ch)
			l.readChar()
			for l.ch != '`' && l.ch != 0 {
				if l.ch == '\\' {
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
			if l.ch == '`' {
				literal.WriteByte(l.ch)
				l.readChar()
			}
		} else if l.ch == '$' && l.peekChar() == '(' {
			// 嵌套的 $(...) 命令替换
			literal.WriteByte(l.ch)
			l.readChar() // 跳过 $
			literal.WriteByte(l.ch) // 写入 (
			l.readChar() // 跳过 (
			nestedDepth := 1
			for nestedDepth > 0 && l.ch != 0 {
				if l.ch == '(' {
					nestedDepth++
					literal.WriteByte(l.ch)
					l.readChar()
				} else if l.ch == ')' {
					nestedDepth--
					literal.WriteByte(l.ch)
					if nestedDepth == 0 {
						l.readChar()
						break
					}
					l.readChar()
				} else if l.ch == '\'' || l.ch == '"' {
					// 处理引号
					quote := l.ch
					literal.WriteByte(l.ch)
					l.readChar()
					for l.ch != quote && l.ch != 0 {
						if l.ch == '\\' && quote == '"' {
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
					if l.ch == quote {
						literal.WriteByte(l.ch)
						l.readChar()
					}
				} else {
					literal.WriteByte(l.ch)
					l.readChar()
				}
			}
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

// readDollarSingleQuote 读取 $'...' ANSI-C 字符串
func (l *Lexer) readDollarSingleQuote() Token {
	startLine := l.line
	startColumn := l.column
	var literal strings.Builder
	
	for l.ch != 0 {
		if l.ch == '\'' {
			// 找到结束引号
			l.readChar() // 跳过结束引号
			break
		}
		if l.ch == '\\' {
			// 处理转义序列
			l.readChar()
			if l.ch != 0 {
				// 处理 ANSI-C 转义序列
				switch l.ch {
				case 'a':
					literal.WriteByte('\a')
				case 'b':
					literal.WriteByte('\b')
				case 'f':
					literal.WriteByte('\f')
				case 'n':
					literal.WriteByte('\n')
				case 'r':
					literal.WriteByte('\r')
				case 't':
					literal.WriteByte('\t')
				case 'v':
					literal.WriteByte('\v')
				case '\\':
					literal.WriteByte('\\')
				case '\'':
					literal.WriteByte('\'')
				case 'x':
					// \xHH 十六进制
					l.readChar()
					hex := ""
					for isHexDigit(l.ch) && len(hex) < 2 {
						hex += string(l.ch)
						l.readChar()
					}
					if len(hex) > 0 {
						if val, err := strconv.ParseUint(hex, 16, 8); err == nil {
							literal.WriteByte(byte(val))
						}
					}
					continue
				case '0', '1', '2', '3', '4', '5', '6', '7':
					// \0NNN 八进制
					oct := string(l.ch)
					l.readChar()
					for isOctDigit(l.ch) && len(oct) < 3 {
						oct += string(l.ch)
						l.readChar()
					}
					if val, err := strconv.ParseUint(oct, 8, 8); err == nil {
						literal.WriteByte(byte(val))
					}
					continue
				default:
					literal.WriteByte('\\')
					literal.WriteByte(l.ch)
				}
				l.readChar()
			}
		} else {
			literal.WriteByte(l.ch)
			l.readChar()
		}
	}
	
	return Token{
		Type:    STRING_DOLLAR_SINGLE,
		Literal: literal.String(),
		Line:    startLine,
		Column:  startColumn,
	}
}

// readDollarDoubleQuote 读取 $"..." 国际化字符串
func (l *Lexer) readDollarDoubleQuote() Token {
	startLine := l.line
	startColumn := l.column
	var literal strings.Builder
	
	for l.ch != 0 {
		if l.ch == '"' {
			// 找到结束引号
			l.readChar() // 跳过结束引号
			break
		}
		if l.ch == '\\' {
			// 处理转义序列
			l.readChar()
			if l.ch != 0 {
				switch l.ch {
				case '"':
					literal.WriteByte('"')
				case '\\':
					literal.WriteByte('\\')
				case '$':
					literal.WriteByte('$')
				case '`':
					literal.WriteByte('`')
				default:
					// 其他转义序列保持原样
					literal.WriteByte('\\')
					literal.WriteByte(l.ch)
				}
				l.readChar()
			}
		} else {
			literal.WriteByte(l.ch)
			l.readChar()
		}
	}
	
	return Token{
		Type:    STRING_DOLLAR_DOUBLE,
		Literal: literal.String(),
		Line:    startLine,
		Column:  startColumn,
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
// 注意：不跳过换行符，因为换行符是重要的token（用于分隔命令）
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

// skipWhitespaceAndNewline 跳过空白字符和换行符
// 用于处理多行命令（行尾的反斜杠会忽略换行符）
func (l *Lexer) skipWhitespaceAndNewline() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
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

// isHexDigit 判断是否为十六进制数字
func isHexDigit(ch byte) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

// isOctDigit 判断是否为八进制数字
func isOctDigit(ch byte) bool {
	return '0' <= ch && ch <= '7'
}

