package parser

import "fmt"

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
	Type        RedirectType
	FD          int // 文件描述符，默认0=stdin, 1=stdout, 2=stderr
	Target      Expression
	HereDoc     *HereDocument // Here-document 信息（如果适用）
}

type RedirectType int

const (
	REDIRECT_INPUT RedirectType = iota
	REDIRECT_OUTPUT
	REDIRECT_APPEND
	REDIRECT_HEREDOC
	REDIRECT_HEREDOC_STRIP
	REDIRECT_HERESTRING
	REDIRECT_DUP_IN
	REDIRECT_DUP_OUT
	REDIRECT_CLOBBER
	REDIRECT_RW
)

// HereDocument Here-document 信息
type HereDocument struct {
	Delimiter   string // 分隔符
	Quoted      bool   // 分隔符是否带引号（带引号时不展开变量）
	StripTabs   bool   // 是否剥离前导制表符（<<-）
	Content     string // Here-document 内容（在执行时填充）
}

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

// ArrayAssignmentStatement 数组赋值语句
// 例如：arr=(1 2 3) 或 arr=([0]=a [1]=b [2]=c)
type ArrayAssignmentStatement struct {
	Name   string
	Values []Expression
	// IndexedValues 存储带索引的数组元素 [index]=value
	// 如果 IndexedValues 不为空，使用它；否则使用 Values
	IndexedValues map[string]Expression // key 是索引（字符串形式，支持数字和字符串键）
}

func (as *ArrayAssignmentStatement) statementNode() {}
func (as *ArrayAssignmentStatement) String() string {
	return "array assignment: " + as.Name
}

// CaseStatement case语句
type CaseStatement struct {
	Value  Expression
	Cases  []*CaseClause
}

func (cs *CaseStatement) statementNode() {}
func (cs *CaseStatement) String() string {
	return "case statement"
}

// CaseClause case子句
type CaseClause struct {
	Patterns []string // 匹配模式列表（用 | 分隔）
	Body     *BlockStatement
}

// BreakStatement break语句
type BreakStatement struct {
	Level int // break 的层级，默认为1（跳出1层循环）
}

func (bs *BreakStatement) statementNode() {}
func (bs *BreakStatement) String() string {
	if bs.Level > 1 {
		return fmt.Sprintf("break %d", bs.Level)
	}
	return "break"
}

// ContinueStatement continue语句
type ContinueStatement struct {
	Level int // continue 的层级，默认为1（继续1层循环）
}

func (cs *ContinueStatement) statementNode() {}
func (cs *ContinueStatement) String() string {
	if cs.Level > 1 {
		return fmt.Sprintf("continue %d", cs.Level)
	}
	return "continue"
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

// CommandSubstitution 命令替换
type CommandSubstitution struct {
	Command string
}

func (cs *CommandSubstitution) expressionNode() {}
func (cs *CommandSubstitution) String() string {
	return "$(" + cs.Command + ")"
}

// ArithmeticExpansion 算术展开
type ArithmeticExpansion struct {
	Expression string
}

func (ae *ArithmeticExpansion) expressionNode() {}
func (ae *ArithmeticExpansion) String() string {
	return "$((" + ae.Expression + "))"
}

// ProcessSubstitution 进程替换
// 例如：<(command) 或 >(command)
type ProcessSubstitution struct {
	Command string
	IsInput bool // true表示<(command)，false表示>(command)
}

func (ps *ProcessSubstitution) expressionNode() {}
func (ps *ProcessSubstitution) String() string {
	if ps.IsInput {
		return "<(" + ps.Command + ")"
	}
	return ">(" + ps.Command + ")"
}

// ParamExpandExpression 参数展开表达式
// 例如：${VAR:-default}, ${VAR#pattern}, ${VAR:offset:length} 等
type ParamExpandExpression struct {
	VarName string // 变量名
	Op      string // 操作符（:-, :=, :?, :+, #, ##, %, %%, :, #, ! 等）
	Word    string // 操作数（默认值、模式、偏移量等）
	Flags   int    // 标志位（用于存储额外的信息）
}

func (pe *ParamExpandExpression) expressionNode() {}
func (pe *ParamExpandExpression) String() string {
	if pe.Op != "" {
		return fmt.Sprintf("${%s%s%s}", pe.VarName, pe.Op, pe.Word)
	}
	return fmt.Sprintf("${%s}", pe.VarName)
}

// SubshellCommand 子shell 命令
// 例如：(command)
type SubshellCommand struct {
	Body *BlockStatement
}

func (sc *SubshellCommand) statementNode() {}
func (sc *SubshellCommand) String() string {
	return "(subshell)"
}

// GroupCommand 命令组
// 例如：{ command; }
type GroupCommand struct {
	Body *BlockStatement
}

func (gc *GroupCommand) statementNode() {}
func (gc *GroupCommand) String() string {
	return "{group}"
}

// CommandChain 命令链
// 例如：cmd1; cmd2, cmd1 && cmd2, cmd1 || cmd2
type CommandChain struct {
	Left     Statement
	Right    Statement
	Operator string // ";", "&&", "||"
}

func (cc *CommandChain) statementNode() {}
func (cc *CommandChain) String() string {
	return fmt.Sprintf("%s %s %s", cc.Left.String(), cc.Operator, cc.Right.String())
}

