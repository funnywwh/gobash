package lexer

import (
	"testing"
)

// TestLineContinuation 测试行尾反斜杠（行继续）
func TestLineContinuation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "基本行继续",
			input:    "echo hello \\\nworld",
			expected: []TokenType{IDENTIFIER, IDENTIFIER, IDENTIFIER},
		},
		{
			name:     "多行行继续",
			input:    "echo hello \\\n  world \\\n  test",
			expected: []TokenType{IDENTIFIER, IDENTIFIER, IDENTIFIER, IDENTIFIER},
		},
		{
			name:     "行继续后跟注释",
			input:    "echo hello \\\n  # comment\n  world",
			expected: []TokenType{IDENTIFIER, IDENTIFIER, IDENTIFIER},
		},
		{
			name:     "行继续后跟空白字符",
			input:    "echo hello \\\n    world",
			expected: []TokenType{IDENTIFIER, IDENTIFIER, IDENTIFIER},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			var tokens []TokenType
			for {
				tok := l.NextToken()
				if tok.Type == EOF {
					break
				}
				if tok.Type != WHITESPACE && tok.Type != NEWLINE {
					tokens = append(tokens, tok.Type)
				}
			}
			
			if len(tokens) != len(tt.expected) {
				t.Errorf("token 数量不匹配，期望 %d，得到 %d", len(tt.expected), len(tokens))
				return
			}
			
			for i, expectedType := range tt.expected {
				if i < len(tokens) && tokens[i] != expectedType {
					t.Errorf("token %d 类型不匹配，期望 %v，得到 %v", i, expectedType, tokens[i])
				}
			}
		})
	}
}




