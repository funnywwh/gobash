package lexer

import (
	"testing"
)

// TestNewRedirectTypes 测试新的重定向类型
func TestNewRedirectTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			input: "cmd <<EOF",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "cmd"},
				{Type: REDIRECT_HEREDOC, Literal: "<<"},
				{Type: IDENTIFIER, Literal: "EOF"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "cmd <<-EOF",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "cmd"},
				{Type: REDIRECT_HEREDOC_STRIP, Literal: "<<-"},
				{Type: IDENTIFIER, Literal: "EOF"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "cmd <&2",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "cmd"},
				{Type: REDIRECT_DUP_IN, Literal: "<&"},
				{Type: NUMBER, Literal: "2"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "cmd >&2",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "cmd"},
				{Type: REDIRECT_DUP_OUT, Literal: ">&"},
				{Type: NUMBER, Literal: "2"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "cmd >|file",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "cmd"},
				{Type: REDIRECT_CLOBBER, Literal: ">|"},
				{Type: IDENTIFIER, Literal: "file"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "cmd <>file",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "cmd"},
				{Type: REDIRECT_RW, Literal: "<>"},
				{Type: IDENTIFIER, Literal: "file"},
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

// TestNewOperators 测试新的操作符
func TestNewOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			input: "case x in ;; esac",
			expected: []Token{
				{Type: CASE, Literal: "case"},
				{Type: IDENTIFIER, Literal: "x"},
				{Type: IN, Literal: "in"},
				{Type: SEMI_SEMI, Literal: ";;"},
				{Type: ESAC, Literal: "esac"},
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "cmd |& cmd2",
			expected: []Token{
				{Type: IDENTIFIER, Literal: "cmd"},
				{Type: BAR_AND, Literal: "|&"},
				{Type: IDENTIFIER, Literal: "cmd2"},
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

// TestBoundaryCases 测试边界情况
func TestBoundaryCases(t *testing.T) {
	tests := []struct {
		input    string
		expected []Token
	}{
		{
			input: "",
			expected: []Token{
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "   ",
			expected: []Token{
				{Type: EOF, Literal: ""},
			},
		},
		{
			input: "\n",
			expected: []Token{
				{Type: NEWLINE, Literal: "\n"},
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

