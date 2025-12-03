package parser

import (
	"testing"
	"gobash/internal/lexer"
)

// BenchmarkParseProgram 基准测试程序解析性能
func BenchmarkParseProgram(b *testing.B) {
	input := "echo hello && ls -la | grep test"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		_ = p.ParseProgram()
	}
}

// BenchmarkParseCommand 基准测试命令解析性能
func BenchmarkParseCommand(b *testing.B) {
	input := "echo hello world"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		_ = p.parseCommandStatement()
	}
}

// BenchmarkParseIfStatement 基准测试 if 语句解析性能
func BenchmarkParseIfStatement(b *testing.B) {
	input := "if [ 1 ]; then echo yes; else echo no; fi"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		_ = p.parseIfStatement()
	}
}

// BenchmarkParseForLoop 基准测试 for 循环解析性能
func BenchmarkParseForLoop(b *testing.B) {
	input := "for i in 1 2 3; do echo $i; done"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		_ = p.parseForStatement()
	}
}

// BenchmarkParseVariableAssignment 基准测试变量赋值解析性能
func BenchmarkParseVariableAssignment(b *testing.B) {
	input := "VAR=value"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(input)
		p := New(l)
		_ = p.parseStatement()
	}
}

