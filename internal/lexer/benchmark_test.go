package lexer

import (
	"testing"
)

// BenchmarkNextToken 基准测试 token 读取性能
func BenchmarkNextToken(b *testing.B) {
	input := "echo hello world && ls -la | grep test"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}
	}
}

// BenchmarkReadIdentifier 基准测试标识符读取性能
func BenchmarkReadIdentifier(b *testing.B) {
	input := "variable_name_123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		_ = l.readIdentifier()
	}
}

// BenchmarkReadString 基准测试字符串读取性能
func BenchmarkReadString(b *testing.B) {
	input := `"hello world with spaces and \"quotes\""`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		_ = l.readString('"')
	}
}

// BenchmarkUTF8Support 基准测试 UTF-8 支持性能
func BenchmarkUTF8Support(b *testing.B) {
	input := "变量名=值 中文标识符"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(input)
		for {
			tok := l.NextToken()
			if tok.Type == EOF {
				break
			}
		}
	}
}

