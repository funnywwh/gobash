package parser

import (
	"testing"
	"gobash/internal/lexer"
)

// TestNestedIfStatements 测试嵌套if语句的解析
func TestNestedIfStatements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkFunc func(t *testing.T, program *Program)
	}{
		{
			name:  "简单嵌套if - then块中",
			input: "if [ 1 ]; then if [ 2 ]; then echo nested; fi; fi",
			checkFunc: func(t *testing.T, program *Program) {
				if len(program.Statements) == 0 {
					t.Fatal("解析失败：没有语句")
				}

				stmt, ok := program.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("解析失败：不是if语句")
				}

				if stmt.Consequence == nil {
					t.Fatal("if语句结果为空")
				}

				// 检查consequence中是否有嵌套的if语句
				if len(stmt.Consequence.Statements) == 0 {
					t.Fatal("consequence中没有语句")
				}

				nestedIf, ok := stmt.Consequence.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("consequence中不是嵌套的if语句")
				}

				if nestedIf.Consequence == nil {
					t.Fatal("嵌套if语句结果为空")
				}
			},
		},
		{
			name:  "嵌套if with else - then块中",
			input: "if [ 1 ]; then if [ 2 ]; then echo nested-then; else echo nested-else; fi; fi",
			checkFunc: func(t *testing.T, program *Program) {
				if len(program.Statements) == 0 {
					t.Fatal("解析失败：没有语句")
				}

				stmt, ok := program.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("解析失败：不是if语句")
				}

				if stmt.Consequence == nil {
					t.Fatal("if语句结果为空")
				}

				// 检查consequence中是否有嵌套的if语句
				if len(stmt.Consequence.Statements) == 0 {
					t.Fatal("consequence中没有语句")
				}

				nestedIf, ok := stmt.Consequence.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("consequence中不是嵌套的if语句")
				}

				if nestedIf.Alternative == nil {
					t.Fatal("嵌套if语句的else块为空")
				}
			},
		},
		{
			name:  "嵌套if with elif - then块中",
			input: "if [ 1 ]; then if [ 2 ]; then echo nested-then; elif [ 3 ]; then echo nested-elif; fi; fi",
			checkFunc: func(t *testing.T, program *Program) {
				if len(program.Statements) == 0 {
					t.Fatal("解析失败：没有语句")
				}

				stmt, ok := program.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("解析失败：不是if语句")
				}

				if stmt.Consequence == nil {
					t.Fatal("if语句结果为空")
				}

				// 检查consequence中是否有嵌套的if语句
				if len(stmt.Consequence.Statements) == 0 {
					t.Fatal("consequence中没有语句")
				}

				nestedIf, ok := stmt.Consequence.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("consequence中不是嵌套的if语句")
				}

				if len(nestedIf.Elif) == 0 {
					t.Fatal("嵌套if语句的elif块为空")
				}
			},
		},
		{
			name:  "嵌套if - else块中",
			input: "if [ 1 ]; then echo then; else if [ 2 ]; then echo nested-then; fi; fi",
			checkFunc: func(t *testing.T, program *Program) {
				if len(program.Statements) == 0 {
					t.Fatal("解析失败：没有语句")
				}

				stmt, ok := program.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("解析失败：不是if语句")
				}

				if stmt.Alternative == nil {
					t.Fatal("if语句的else块为空")
				}

				// 检查alternative中是否有嵌套的if语句
				if len(stmt.Alternative.Statements) == 0 {
					t.Fatal("alternative中没有语句")
				}

				nestedIf, ok := stmt.Alternative.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("alternative中不是嵌套的if语句")
				}

				if nestedIf.Consequence == nil {
					t.Fatal("嵌套if语句结果为空")
				}
			},
		},
		{
			name:  "嵌套if - elif块中",
			input: "if [ 1 ]; then echo then; elif [ 2 ]; then if [ 3 ]; then echo nested-then; fi; fi",
			checkFunc: func(t *testing.T, program *Program) {
				if len(program.Statements) == 0 {
					t.Fatal("解析失败：没有语句")
				}

				stmt, ok := program.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("解析失败：不是if语句")
				}

				if len(stmt.Elif) == 0 {
					t.Fatal("if语句的elif块为空")
				}

				// 检查elif的consequence中是否有嵌套的if语句
				if len(stmt.Elif[0].Consequence.Statements) == 0 {
					t.Fatal("elif的consequence中没有语句")
				}

				nestedIf, ok := stmt.Elif[0].Consequence.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("elif的consequence中不是嵌套的if语句")
				}

				if nestedIf.Consequence == nil {
					t.Fatal("嵌套if语句结果为空")
				}
			},
		},
		{
			name:  "三层嵌套if",
			input: "if [ 1 ]; then if [ 2 ]; then if [ 3 ]; then echo triple-nested; fi; fi; fi",
			checkFunc: func(t *testing.T, program *Program) {
				if len(program.Statements) == 0 {
					t.Fatal("解析失败：没有语句")
				}

				stmt, ok := program.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("解析失败：不是if语句")
				}

				if stmt.Consequence == nil {
					t.Fatal("if语句结果为空")
				}

				// 第一层嵌套
				if len(stmt.Consequence.Statements) == 0 {
					t.Fatal("consequence中没有语句")
				}

				nestedIf1, ok := stmt.Consequence.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("consequence中不是嵌套的if语句")
				}

				// 第二层嵌套
				if len(nestedIf1.Consequence.Statements) == 0 {
					t.Fatal("嵌套if1的consequence中没有语句")
				}

				nestedIf2, ok := nestedIf1.Consequence.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("嵌套if1的consequence中不是嵌套的if语句")
				}

				if nestedIf2.Consequence == nil {
					t.Fatal("三层嵌套if语句结果为空")
				}
			},
		},
		{
			name:  "嵌套if with else和elif",
			input: "if [ 1 ]; then if [ 2 ]; then echo nested-then; elif [ 3 ]; then echo nested-elif; else echo nested-else; fi; fi",
			checkFunc: func(t *testing.T, program *Program) {
				if len(program.Statements) == 0 {
					t.Fatal("解析失败：没有语句")
				}

				stmt, ok := program.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("解析失败：不是if语句")
				}

				if stmt.Consequence == nil {
					t.Fatal("if语句结果为空")
				}

				// 检查consequence中是否有嵌套的if语句
				if len(stmt.Consequence.Statements) == 0 {
					t.Fatal("consequence中没有语句")
				}

				nestedIf, ok := stmt.Consequence.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("consequence中不是嵌套的if语句")
				}

				if len(nestedIf.Elif) == 0 {
					t.Fatal("嵌套if语句的elif块为空")
				}

				if nestedIf.Alternative == nil {
					t.Fatal("嵌套if语句的else块为空")
				}
			},
		},
		{
			name:  "多行嵌套if",
			input: "if [ 1 ]\nthen\n  if [ 2 ]\n  then\n    echo nested\n  fi\nfi",
			checkFunc: func(t *testing.T, program *Program) {
				if len(program.Statements) == 0 {
					t.Fatal("解析失败：没有语句")
				}

				stmt, ok := program.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("解析失败：不是if语句")
				}

				if stmt.Consequence == nil {
					t.Fatal("if语句结果为空")
				}

				// 检查consequence中是否有嵌套的if语句
				if len(stmt.Consequence.Statements) == 0 {
					t.Fatal("consequence中没有语句")
				}

				nestedIf, ok := stmt.Consequence.Statements[0].(*IfStatement)
				if !ok {
					t.Fatal("consequence中不是嵌套的if语句")
				}

				if nestedIf.Consequence == nil {
					t.Fatal("嵌套if语句结果为空")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Errorf("解析错误: %v", p.Errors())
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, program)
			}
		})
	}
}





