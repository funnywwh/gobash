package lexer

// TokenType 表示token的类型
type TokenType int

const (
	// 基础token
	ILLEGAL TokenType = iota
	EOF
	WHITESPACE
	NEWLINE

	// 标识符和字面量
	IDENTIFIER    // 命令名、变量名等
	STRING        // 字符串字面量
	STRING_SINGLE // 单引号字符串（不展开变量）
	STRING_DOUBLE // 双引号字符串（展开变量）
	NUMBER        // 数字

	// 操作符
	PIPE          // |
	REDIRECT_OUT  // >
	REDIRECT_IN   // <
	REDIRECT_APPEND // >>
	REDIRECT_HEREDOC // <<
	REDIRECT_HEREDOC_STRIP // <<-
	REDIRECT_HEREDOC_TABS // <<<
	REDIRECT_DUP_IN  // <&
	REDIRECT_DUP_OUT // >&
	REDIRECT_CLOBBER // >|
	REDIRECT_RW      // <>
	REDIRECT_FD      // 2>, 1>, etc.
	AND           // &&
	OR            // ||
	SEMICOLON     // ;
	SEMI_SEMI     // ;;
	SEMI_AND      // ;&
	SEMI_SEMI_AND // ;;&
	AMPERSAND     // &
	BAR_AND       // |&
	AND_GREATER   // &>
	AND_GREATER_GREATER // &>>

	// 引号和转义
	SINGLE_QUOTE // '
	DOUBLE_QUOTE // "
	BACKTICK     // `
	ESCAPE       // \

	// 变量
	DOLLAR // $
	VAR    // $VAR 或 ${VAR}
	PARAM_EXPAND // ${VAR...} 参数展开（包含所有形式）
	STRING_DOLLAR_SINGLE // $'...' ANSI-C 字符串
	STRING_DOLLAR_DOUBLE // $"..." 国际化字符串
	
	// 命令替换
	COMMAND_SUBSTITUTION // `command` 或 $(command)
	
	// 算术展开
	ARITHMETIC_EXPANSION // $((expr))
	
	// 进程替换
	PROCESS_SUBSTITUTION_IN  // <(command)
	PROCESS_SUBSTITUTION_OUT // >(command)

	// 括号和分组
	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	LBRACKET // [
	RBRACKET // ]
	DBL_LBRACKET // [[
	DBL_RBRACKET // ]]

	// 控制流关键字
	IF
	THEN
	ELSE
	ELIF
	FI
	FOR
	WHILE
	DO
	DONE
	CASE
	ESAC
	FUNCTION
	BREAK
	CONTINUE

	// 其他关键字
	IN
	SELECT
	TIME
	
	// Here-document
	HEREDOC_MARKER // Here-document 标记（如 EOF）
	HEREDOC_CONTENT // Here-document 内容
	
	// 赋值
	ASSIGNMENT_WORD // 赋值词（VAR=value）
	
	// 复合命令
	SUBSHELL_START // ( 子shell开始
	SUBSHELL_END   // ) 子shell结束
	GROUP_START    // { 命令组开始
	GROUP_END      // } 命令组结束
)

// Token 表示一个词法单元
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// String 返回token的字符串表示
func (t TokenType) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case WHITESPACE:
		return "WHITESPACE"
	case NEWLINE:
		return "NEWLINE"
	case IDENTIFIER:
		return "IDENTIFIER"
	case STRING:
		return "STRING"
	case STRING_SINGLE:
		return "STRING_SINGLE"
	case STRING_DOUBLE:
		return "STRING_DOUBLE"
	case NUMBER:
		return "NUMBER"
	case PIPE:
		return "PIPE"
	case REDIRECT_OUT:
		return "REDIRECT_OUT"
	case REDIRECT_IN:
		return "REDIRECT_IN"
	case REDIRECT_APPEND:
		return "REDIRECT_APPEND"
	case REDIRECT_HEREDOC:
		return "REDIRECT_HEREDOC"
	case REDIRECT_HEREDOC_STRIP:
		return "REDIRECT_HEREDOC_STRIP"
	case REDIRECT_HEREDOC_TABS:
		return "REDIRECT_HEREDOC_TABS"
	case REDIRECT_DUP_IN:
		return "REDIRECT_DUP_IN"
	case REDIRECT_DUP_OUT:
		return "REDIRECT_DUP_OUT"
	case REDIRECT_CLOBBER:
		return "REDIRECT_CLOBBER"
	case REDIRECT_RW:
		return "REDIRECT_RW"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case SEMICOLON:
		return "SEMICOLON"
	case SEMI_SEMI:
		return "SEMI_SEMI"
	case SEMI_AND:
		return "SEMI_AND"
	case SEMI_SEMI_AND:
		return "SEMI_SEMI_AND"
	case AMPERSAND:
		return "AMPERSAND"
	case BAR_AND:
		return "BAR_AND"
	case AND_GREATER:
		return "AND_GREATER"
	case AND_GREATER_GREATER:
		return "AND_GREATER_GREATER"
		return "AMPERSAND"
	case SINGLE_QUOTE:
		return "SINGLE_QUOTE"
	case DOUBLE_QUOTE:
		return "DOUBLE_QUOTE"
	case BACKTICK:
		return "BACKTICK"
	case ESCAPE:
		return "ESCAPE"
	case DOLLAR:
		return "DOLLAR"
	case VAR:
		return "VAR"
	case PARAM_EXPAND:
		return "PARAM_EXPAND"
	case STRING_DOLLAR_SINGLE:
		return "STRING_DOLLAR_SINGLE"
	case STRING_DOLLAR_DOUBLE:
		return "STRING_DOLLAR_DOUBLE"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case IF:
		return "IF"
	case THEN:
		return "THEN"
	case ELSE:
		return "ELSE"
	case ELIF:
		return "ELIF"
	case FI:
		return "FI"
	case FOR:
		return "FOR"
	case WHILE:
		return "WHILE"
	case DO:
		return "DO"
	case DONE:
		return "DONE"
	case FUNCTION:
		return "FUNCTION"
	case BREAK:
		return "BREAK"
	case CONTINUE:
		return "CONTINUE"
	case COMMAND_SUBSTITUTION:
		return "COMMAND_SUBSTITUTION"
	case ARITHMETIC_EXPANSION:
		return "ARITHMETIC_EXPANSION"
	case PROCESS_SUBSTITUTION_IN:
		return "PROCESS_SUBSTITUTION_IN"
	case PROCESS_SUBSTITUTION_OUT:
		return "PROCESS_SUBSTITUTION_OUT"
	case HEREDOC_MARKER:
		return "HEREDOC_MARKER"
	case HEREDOC_CONTENT:
		return "HEREDOC_CONTENT"
	case ASSIGNMENT_WORD:
		return "ASSIGNMENT_WORD"
	case SUBSHELL_START:
		return "SUBSHELL_START"
	case SUBSHELL_END:
		return "SUBSHELL_END"
	case GROUP_START:
		return "GROUP_START"
	case GROUP_END:
		return "GROUP_END"
	case LBRACKET:
		return "LBRACKET"
	case RBRACKET:
		return "RBRACKET"
	case DBL_LBRACKET:
		return "DBL_LBRACKET"
	case DBL_RBRACKET:
		return "DBL_RBRACKET"
	case CASE:
		return "CASE"
	case ESAC:
		return "ESAC"
	case IN:
		return "IN"
	case SELECT:
		return "SELECT"
	case TIME:
		return "TIME"
	default:
		return "UNKNOWN"
	}
}

// 关键字映射
var keywords = map[string]TokenType{
	"if":       IF,
	"then":     THEN,
	"else":     ELSE,
	"elif":     ELIF,
	"fi":       FI,
	"for":      FOR,
	"while":    WHILE,
	"do":       DO,
	"done":     DONE,
	"case":     CASE,
	"esac":     ESAC,
	"function": FUNCTION,
	"break":    BREAK,
	"continue": CONTINUE,
	"in":       IN,
	"select":   SELECT,
	"time":     TIME,
}

// LookupIdent 检查标识符是否为关键字
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENTIFIER
}

