package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			input: "echo hello",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "echo"},
				{Type: IDENTIFIER, Literal: "hello"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "echo 'hello world'",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "echo"},
				{Type: STRING_SINGLE, Literal: "hello world"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "echo \"hello $VAR\"",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "echo"},
				{Type: STRING_DOUBLE, Literal: "hello $VAR"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "ls | grep test",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "ls"},
				{Type: PIPE, Literal: "|"},
				{Type: IDENTIFIER, Literal: "grep"},
				{Type: IDENTIFIER, Literal: "test"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "echo $VAR",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "echo"},
				{Type: VAR, Literal: "VAR"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "echo $((1 + 2))",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "echo"},
				{Type: ARITHMETIC_EXPANSION, Literal: "1 + 2"}, // 表达式部分不包含括号
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "echo $(pwd)",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "echo"},
				{Type: COMMAND_SUBSTITUTION, Literal: "pwd"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "cd /tmp && ls",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "cd"},
				{Type: IDENTIFIER, Literal: "/tmp"},
				{Type: AND, Literal: "&&"},
				{Type: IDENTIFIER, Literal: "ls"},
				{Type: EOF, Literal: ""},
			},
		},
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, expectedToken := range tt.expected {
			tok := l.NextToken()
			if tok.Type != expectedToken.Type {
				t.Errorf("测试 '%s' [%d]: token类型错误，期望 %s，得到 %s",
					tt.input, i, expectedToken.Type, tok.Type)
			}
			if tok.Literal != expectedToken.Literal {
				t.Errorf("测试 '%s' [%d]: token字面量错误，期望 '%s'，得到 '%s'",
					tt.input, i, expectedToken.Literal, tok.Literal)
			}
		}
	}
}

func TestReadString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		quote    byte
	}{
		{"'hello world'", "hello world", '\''},
		{"\"hello world\"", "hello world", '"'},
		{"'test'", "test", '\''},
	}

	for _, tt := range tests {
		l := New(tt.input)
		// readString会自动跳过开头的引号
		tok := l.readString(tt.quote)
		if tok.Literal != tt.expected {
			t.Errorf("字符串读取错误，期望 '%s'，得到 '%s'", tt.expected, tok.Literal)
		}
	}
}

func TestReadVariable(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"$VAR", "VAR"},
		{"$HOME", "HOME"},
		{"${VAR}", "VAR"},
	}

	for _, tt := range tests {
		l := New(tt.input)
		// readVariable会自动跳过 $
		tok := l.readVariable()
		if tok.Literal != tt.expected {
			t.Errorf("变量读取错误，期望 '%s'，得到 '%s'", tt.expected, tok.Literal)
		}
	}
}

func TestReadNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123", "123"},
		{"42", "42"},
		{"0", "0"},
	}

	for _, tt := range tests {
		l := New(tt.input)
		result := l.readNumber()
		if result != tt.expected {
			t.Errorf("数字读取错误，期望 '%s'，得到 '%s'", tt.expected, result)
		}
	}
}

func TestReadIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"echo", "echo"},
		{"ls", "ls"},
		{"test_command", "test_command"},
	}

	for _, tt := range tests {
		l := New(tt.input)
		result := l.readIdentifier()
		if result != tt.expected {
			t.Errorf("标识符读取错误，期望 '%s'，得到 '%s'", tt.expected, result)
		}
	}
}

// TestNestedCommandSubstitution 测试嵌套的命令替换
func TestNestedCommandSubstitution(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"$(echo $(pwd))", "echo $(pwd)"},
		{"$(echo \"test\")", "echo \"test\""},
		{"$(echo 'test')", "echo 'test'"},
		{"$(echo $(echo test))", "echo $(echo test)"},
	}

	for _, tt := range tests {
		l := New(tt.input)
		// 跳过 $ 和 (
		l.readChar() // 跳过 $
		l.readChar() // 跳过 (
		tok := l.readCommandSubstitutionParen()
		if tok.Literal != tt.expected {
			t.Errorf("嵌套命令替换读取错误，期望 '%s'，得到 '%s'", tt.expected, tok.Literal)
		}
	}
}

// TestNestedArithmeticExpansion 测试嵌套的算术展开
// TestEscapedNewline 测试转义的换行符处理
func TestEscapedNewline(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			// 多行命令：行尾的反斜杠应该忽略换行符
			input: "echo hello \\\nworld",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "echo"},
				{Type: IDENTIFIER, Literal: "hello"},
				{Type: IDENTIFIER, Literal: "world"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			// 普通换行符应该被识别为 NEWLINE
			input: "echo hello\nworld",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "echo"},
				{Type: IDENTIFIER, Literal: "hello"},
				{Type: NEWLINE, Literal: "\n"},
				{Type: IDENTIFIER, Literal: "world"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			// 引号内的换行符应该被保留
			input: "echo \"hello\nworld\"",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "echo"},
				{Type: STRING_DOUBLE, Literal: "hello\nworld"},
				{Type: EOF, Literal: ""},
			},
		},
	}

	for _, tt := range tests {
		l := New(tt.input)
		for i, expected := range tt.expected {
			tok := l.NextToken()
			if tok.Type != expected.Type {
				t.Errorf("测试 '%s' token %d: 类型错误，期望 %v，得到 %v", tt.input, i, expected.Type, tok.Type)
			}
			if tok.Literal != expected.Literal {
				t.Errorf("测试 '%s' token %d: 字面量错误，期望 '%s'，得到 '%s'", tt.input, i, expected.Literal, tok.Literal)
			}
		}
	}
}

// TestUTF8Support 测试 UTF-8 多字节字符支持
// 注意：此测试当前被跳过，因为 UTF-8 支持尚未完全实现
// TODO: 实现完整的 UTF-8 支持后启用此测试
func TestUTF8Support(t *testing.T) {
	t.Skip("UTF-8 支持尚未完全实现，跳过测试")
	// 测试用例将在 UTF-8 支持完全实现后添加
}

func TestNestedArithmeticExpansion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"$((1 + 2))", "1 + 2"},
		{"$((1 + (2 + 3)))", "1 + (2 + 3)"},
		{"$((1 + $((2 + 3))))", "1 + $((2 + 3))"},
	}

	for _, tt := range tests {
		l := New(tt.input)
		// 跳过 $ 和两个 (
		l.readChar() // 跳过 $
		l.readChar() // 跳过第一个 (
		l.readChar() // 跳过第二个 (
		tok := l.readArithmeticExpansion()
		if tok.Literal != tt.expected {
			t.Errorf("嵌套算术展开读取错误，期望 '%s'，得到 '%s'", tt.expected, tok.Literal)
		}
	}
}

