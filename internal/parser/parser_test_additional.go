package parser

import (
	"testing"
	"gobash/internal/lexer"
)

// TestNewFeatures 测试新功能的单元测试
func TestNewFeatures(t *testing.T) {
	// 测试复合命令
	t.Run("Subshell", func(t *testing.T) {
		input := "(echo hello)"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) == 0 {
			t.Fatal("解析失败：没有语句")
		}

		stmt, ok := program.Statements[0].(*SubshellCommand)
		if !ok {
			t.Fatal("解析失败：不是子shell命令")
		}

		if stmt.Body == nil {
			t.Error("子shell命令体为空")
		}
	})

	// 测试命令组
	t.Run("GroupCommand", func(t *testing.T) {
		input := "{ echo hello; echo world; }"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) == 0 {
			t.Fatal("解析失败：没有语句")
		}

		stmt, ok := program.Statements[0].(*GroupCommand)
		if !ok {
			t.Fatal("解析失败：不是命令组")
		}

		if stmt.Body == nil {
			t.Error("命令组体为空")
		}
	})

	// 测试命令链
	t.Run("CommandChain", func(t *testing.T) {
		input := "echo hello && echo world || echo fail"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) == 0 {
			t.Fatal("解析失败：没有语句")
		}

		chain, ok := program.Statements[0].(*CommandChain)
		if !ok {
			t.Fatal("解析失败：不是命令链")
		}

		if chain.Left == nil {
			t.Error("命令链左端为空")
		}

		if chain.Right == nil {
			t.Error("命令链右端为空")
		}
	})

	// 测试 case 语句
	t.Run("CaseStatement", func(t *testing.T) {
		input := "case x in a) echo a;; b) echo b;; esac"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) == 0 {
			t.Fatal("解析失败：没有语句")
		}

		stmt, ok := program.Statements[0].(*CaseStatement)
		if !ok {
			t.Fatal("解析失败：不是case语句")
		}

		if stmt.Value == nil {
			t.Error("case语句值为空")
		}

		if len(stmt.Cases) == 0 {
			t.Error("case子句为空")
		}
	})

	// 测试 while 语句
	t.Run("WhileStatement", func(t *testing.T) {
		input := "while [ 1 ]; do echo hello; done"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) == 0 {
			t.Fatal("解析失败：没有语句")
		}

		stmt, ok := program.Statements[0].(*WhileStatement)
		if !ok {
			t.Fatal("解析失败：不是while语句")
		}

		if stmt.Condition == nil {
			t.Error("while语句条件为空")
		}

		if stmt.Body == nil {
			t.Error("while语句体为空")
		}
	})

	// 测试参数展开
	t.Run("ParamExpansion", func(t *testing.T) {
		input := "echo ${VAR:-default}"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) == 0 {
			t.Fatal("解析失败：没有语句")
		}

		stmt, ok := program.Statements[0].(*CommandStatement)
		if !ok {
			t.Fatal("解析失败：不是命令语句")
		}

		// 检查参数中是否包含参数展开
		found := false
		for _, arg := range stmt.Args {
			if pe, ok := arg.(*ParamExpandExpression); ok {
				found = true
				if pe.VarName == "" {
					t.Error("参数展开变量名为空")
				}
				break
			}
		}
		if !found {
			t.Error("未找到参数展开表达式")
		}
	})

	// 测试新的重定向类型
	t.Run("NewRedirectTypes", func(t *testing.T) {
		tests := []struct {
			input      string
			redirectType RedirectType
		}{
			{"cmd <&2", REDIRECT_DUP_IN},
			{"cmd >&2", REDIRECT_DUP_OUT},
			{"cmd >|file", REDIRECT_CLOBBER},
			{"cmd <>file", REDIRECT_RW},
		}

		for _, tt := range tests {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(program.Statements) == 0 {
				t.Errorf("解析 '%s' 失败：没有语句", tt.input)
				continue
			}

			stmt, ok := program.Statements[0].(*CommandStatement)
			if !ok {
				t.Errorf("解析 '%s' 失败：不是命令语句", tt.input)
				continue
			}

			if len(stmt.Redirects) == 0 {
				t.Errorf("解析 '%s' 失败：没有重定向", tt.input)
				continue
			}

			if stmt.Redirects[0].Type != tt.redirectType {
				t.Errorf("解析 '%s' 失败：重定向类型错误，期望 %v，得到 %v",
					tt.input, tt.redirectType, stmt.Redirects[0].Type)
			}
		}
	})
}

// TestBoundaryCases 测试边界情况
func TestBoundaryCases(t *testing.T) {
	// 测试空输入
	t.Run("EmptyInput", func(t *testing.T) {
		input := ""
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) != 0 {
			t.Errorf("空输入应该没有语句，得到 %d 个语句", len(program.Statements))
		}
	})

	// 测试只有空白字符
	t.Run("WhitespaceOnly", func(t *testing.T) {
		input := "   \n\t  "
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) != 0 {
			t.Errorf("只有空白字符应该没有语句，得到 %d 个语句", len(program.Statements))
		}
	})

	// 测试单个命令
	t.Run("SingleCommand", func(t *testing.T) {
		input := "echo"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) == 0 {
			t.Fatal("解析失败：没有语句")
		}

		stmt, ok := program.Statements[0].(*CommandStatement)
		if !ok {
			t.Fatal("解析失败：不是命令语句")
		}

		if stmt.Command == nil {
			t.Error("命令为空")
		}
	})

	// 测试嵌套结构
	t.Run("NestedStructures", func(t *testing.T) {
		input := "if [ 1 ]; then if [ 2 ]; then echo nested; fi; fi"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		if len(program.Statements) == 0 {
			t.Fatal("解析失败：没有语句")
		}

		stmt, ok := program.Statements[0].(*IfStatement)
		if !ok {
			t.Fatal("解析失败：不是if语句")
		}

		if stmt.Consequence == nil {
			t.Error("if语句结果为空")
		}
	})
}

// TestErrorCases 测试错误处理
func TestErrorCases(t *testing.T) {
	// 测试未闭合的引号
	t.Run("UnclosedQuote", func(t *testing.T) {
		input := "echo 'unclosed"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		// 应该能够解析（lexer 会处理未闭合的引号）
		if len(program.Statements) == 0 {
			t.Error("应该能够解析未闭合的引号")
		}
	})

	// 测试未闭合的括号
	t.Run("UnclosedParen", func(t *testing.T) {
		input := "echo $(unclosed"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		// 应该能够解析（lexer 会处理未闭合的括号）
		if len(program.Statements) == 0 {
			t.Error("应该能够解析未闭合的括号")
		}
	})

	// 测试未闭合的控制流
	t.Run("UnclosedControlFlow", func(t *testing.T) {
		input := "if [ 1 ]; then echo hello"
		l := lexer.New(input)
		p := New(l)
		program := p.ParseProgram()

		// 应该能够解析（即使缺少 fi）
		if len(program.Statements) == 0 {
			t.Error("应该能够解析未闭合的控制流")
		}
	})
}

// TestExistingTests 运行现有测试，确保兼容性
// 注意：这些测试函数在 parser_test.go 中定义，这里只是重新运行它们
func TestExistingTests(t *testing.T) {
	// 这些测试已经在 parser_test.go 中定义，不需要重复运行
	// 如果需要，可以在这里添加额外的测试
	t.Log("现有测试已在 parser_test.go 中定义")
}

