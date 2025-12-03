package parser

import (
	"testing"
	"gobash/internal/lexer"
)

// TestParseDoubleBracket 测试 [[ condition ]] 命令解析
func TestParseDoubleBracket(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkFunc func(t *testing.T, stmt *CommandStatement)
	}{
		{
			name:  "基本 [[ 命令",
			input: "[[ -f file.txt ]]",
			checkFunc: func(t *testing.T, stmt *CommandStatement) {
				if stmt.Command == nil {
					t.Fatal("命令为空")
				}
				ident, ok := stmt.Command.(*Identifier)
				if !ok {
					t.Fatal("命令不是标识符")
				}
				if ident.Value != "[[" {
					t.Errorf("命令名错误，期望 '[[', 得到 '%s'", ident.Value)
				}
				if len(stmt.Args) == 0 {
					t.Fatal("没有参数")
				}
				// 检查最后一个参数是否是 ]]
				lastArg, ok := stmt.Args[len(stmt.Args)-1].(*Identifier)
				if !ok || lastArg.Value != "]]" {
					t.Errorf("最后一个参数应该是 ']]', 得到 '%v'", stmt.Args[len(stmt.Args)-1])
				}
			},
		},
		{
			name:  "[[ 命令带 &&",
			input: "[[ -f file.txt && -r file.txt ]]",
			checkFunc: func(t *testing.T, stmt *CommandStatement) {
				if stmt.Command == nil {
					t.Fatal("命令为空")
				}
				// 检查参数中是否包含 &&
				hasAnd := false
				for _, arg := range stmt.Args {
					if ident, ok := arg.(*Identifier); ok && ident.Value == "&&" {
						hasAnd = true
						break
					}
				}
				if !hasAnd {
					t.Error("参数中应该包含 '&&'")
				}
			},
		},
		{
			name:  "[[ 命令带 ||",
			input: "[[ -f file.txt || -d dir ]]",
			checkFunc: func(t *testing.T, stmt *CommandStatement) {
				if stmt.Command == nil {
					t.Fatal("命令为空")
				}
				// 检查参数中是否包含 ||
				hasOr := false
				for _, arg := range stmt.Args {
					if ident, ok := arg.(*Identifier); ok && ident.Value == "||" {
						hasOr = true
						break
					}
				}
				if !hasOr {
					t.Error("参数中应该包含 '||'")
				}
			},
		},
		{
			name:  "[[ 命令带括号",
			input: "[[ (-f file.txt) ]]",
			checkFunc: func(t *testing.T, stmt *CommandStatement) {
				if stmt.Command == nil {
					t.Fatal("命令为空")
				}
				// 检查参数中是否包含括号
				hasParen := false
				for _, arg := range stmt.Args {
					if ident, ok := arg.(*Identifier); ok && (ident.Value == "(" || ident.Value == ")") {
						hasParen = true
						break
					}
				}
				if !hasParen {
					t.Error("参数中应该包含括号")
				}
			},
		},
		{
			name:  "[[ 命令带 !",
			input: "[[ ! -f file.txt ]]",
			checkFunc: func(t *testing.T, stmt *CommandStatement) {
				if stmt.Command == nil {
					t.Fatal("命令为空")
				}
				// 检查参数中是否包含 !
				hasNot := false
				for _, arg := range stmt.Args {
					if ident, ok := arg.(*Identifier); ok && ident.Value == "!" {
						hasNot = true
						break
					}
				}
				if !hasNot {
					t.Error("参数中应该包含 '!'")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(program.Statements) == 0 {
				t.Fatalf("解析 '%s' 失败：没有语句", tt.input)
			}

			stmt, ok := program.Statements[0].(*CommandStatement)
			if !ok {
				t.Fatalf("解析 '%s' 失败：不是命令语句", tt.input)
			}

			tt.checkFunc(t, stmt)
		})
	}
}

