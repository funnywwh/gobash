package executor

import (
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// BenchmarkVariableExpansion 基准测试变量展开性能
func BenchmarkVariableExpansion(b *testing.B) {
	e := New()
	e.SetEnv("VAR", "value")
	e.SetEnv("NUM", "42")
	
	text := "${VAR:-default} $NUM $((1+2))"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.expandWord(text)
	}
}

// BenchmarkWordSplit 基准测试单词分割性能
func BenchmarkWordSplit(b *testing.B) {
	e := New()
	e.SetEnv("IFS", " ")
	text := "hello world test example"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.wordSplit(text)
	}
}

// BenchmarkPathnameExpand 基准测试路径名展开性能
func BenchmarkPathnameExpand(b *testing.B) {
	e := New()
	pattern := "*.go"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.pathnameExpand(pattern)
	}
}

// BenchmarkArithmeticExpansion 基准测试算术展开性能
func BenchmarkArithmeticExpansion(b *testing.B) {
	e := New()
	expr := "1 + 2 * 3 - 4 / 2"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.evaluateArithmetic(expr)
	}
}

// BenchmarkCommandExecution 基准测试命令执行性能
func BenchmarkCommandExecution(b *testing.B) {
	e := New()
	input := "echo hello"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Execute(program)
	}
}

