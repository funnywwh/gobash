package parser

// Node AST节点接口
type Node interface {
	String() string
}

// Statement 语句接口
type Statement interface {
	Node
	statementNode()
}

// Expression 表达式接口
type Expression interface {
	Node
	expressionNode()
}

// Program 程序根节点
type Program struct {
	Statements []Statement
}

func (p *Program) String() string {
	var out string
	for _, s := range p.Statements {
		out += s.String()
	}
	return out
}

// CommandStatement 命令语句
type CommandStatement struct {
	Command     Expression
	Args        []Expression
	Redirects   []*Redirect
	Background  bool
	Pipe        *CommandStatement
}

func (cs *CommandStatement) statementNode() {}
func (cs *CommandStatement) String() string {
	var out string
	if cs.Command != nil {
		out += cs.Command.String()
	}
	for _, arg := range cs.Args {
		out += " " + arg.String()
	}
	if cs.Pipe != nil {
		out += " | " + cs.Pipe.String()
	}
	return out
}

// Redirect 重定向
type Redirect struct {
	Type   RedirectType
	FD     int // 文件描述符，默认0=stdin, 1=stdout, 2=stderr
	Target Expression
}

type RedirectType int

const (
	REDIRECT_INPUT RedirectType = iota
	REDIRECT_OUTPUT
	REDIRECT_APPEND
	REDIRECT_HEREDOC
)

// IfStatement if语句
type IfStatement struct {
	Condition   *CommandStatement
	Consequence *BlockStatement
	Alternative *BlockStatement
	Elif        []*ElifClause
}

func (is *IfStatement) statementNode() {}
func (is *IfStatement) String() string {
	return "if statement"
}

// ElifClause elif子句
type ElifClause struct {
	Condition   *CommandStatement
	Consequence *BlockStatement
}

// ForStatement for循环
type ForStatement struct {
	Variable string
	In       []Expression
	Body     *BlockStatement
}

func (fs *ForStatement) statementNode() {}
func (fs *ForStatement) String() string {
	return "for statement"
}

// WhileStatement while循环
type WhileStatement struct {
	Condition *CommandStatement
	Body      *BlockStatement
}

func (ws *WhileStatement) statementNode() {}
func (ws *WhileStatement) String() string {
	return "while statement"
}

// BlockStatement 代码块
type BlockStatement struct {
	Statements []Statement
}

func (bs *BlockStatement) statementNode() {}
func (bs *BlockStatement) String() string {
	var out string
	for _, s := range bs.Statements {
		out += s.String() + "\n"
	}
	return out
}

// FunctionStatement 函数定义
type FunctionStatement struct {
	Name string
	Body *BlockStatement
}

func (fs *FunctionStatement) statementNode() {}
func (fs *FunctionStatement) String() string {
	return "function " + fs.Name
}

// Identifier 标识符
type Identifier struct {
	Value string
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) String() string {
	return i.Value
}

// StringLiteral 字符串字面量
type StringLiteral struct {
	Value   string
	IsQuote bool // 是否为引号字符串（双引号需要展开变量，单引号不需要）
}

func (sl *StringLiteral) expressionNode() {}
func (sl *StringLiteral) String() string {
	if sl.IsQuote {
		return "\"" + sl.Value + "\""
	}
	return "'" + sl.Value + "'"
}

// Variable 变量
type Variable struct {
	Name string
}

func (v *Variable) expressionNode() {}
func (v *Variable) String() string {
	return "$" + v.Name
}

