package executor

import (
	"os"
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// TestCommandSubstitutionStdoutRestoreSimple 简单测试命令替换后 os.Stdout 是否被恢复
// 通过直接检查 os.Stdout 的值来验证
func TestCommandSubstitutionStdoutRestoreSimple(t *testing.T) {
	// 保存原始的 os.Stdout
	originalStdout := os.Stdout

	// 创建执行器
	e := New()

	// 测试命令：执行命令替换
	command := "echo $(echo test)"
	
	// 解析命令
	l := lexer.New(command)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("解析错误: %v", p.Errors())
	}

	// 执行命令
	err := e.Execute(program)
	if err != nil {
		t.Fatalf("执行错误: %v", err)
	}

	// 验证 os.Stdout 是否被恢复（应该等于原始的 os.Stdout）
	if os.Stdout != originalStdout {
		t.Errorf("os.Stdout 未正确恢复！原始: %v, 当前: %v", originalStdout, os.Stdout)
	}
}

// TestCommandSubstitutionStdoutRestoreMultiple 测试多次命令替换后 os.Stdout 是否被恢复
func TestCommandSubstitutionStdoutRestoreMultiple(t *testing.T) {
	// 保存原始的 os.Stdout
	originalStdout := os.Stdout

	// 创建执行器
	e := New()

	// 测试命令：执行多次命令替换
	command := "echo $(echo first); echo $(echo second); echo $(echo third)"
	
	// 解析命令
	l := lexer.New(command)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("解析错误: %v", p.Errors())
	}

	// 执行命令
	err := e.Execute(program)
	if err != nil {
		t.Fatalf("执行错误: %v", err)
	}

	// 验证 os.Stdout 是否被恢复
	if os.Stdout != originalStdout {
		t.Errorf("os.Stdout 未正确恢复！原始: %v, 当前: %v", originalStdout, os.Stdout)
	}
}

// TestCommandSubstitutionStdoutRestoreNested 测试嵌套命令替换后 os.Stdout 是否被恢复
func TestCommandSubstitutionStdoutRestoreNested(t *testing.T) {
	// 保存原始的 os.Stdout
	originalStdout := os.Stdout

	// 创建执行器
	e := New()

	// 测试命令：执行嵌套命令替换
	command := "echo $(echo $(echo nested))"
	
	// 解析命令
	l := lexer.New(command)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("解析错误: %v", p.Errors())
	}

	// 执行命令
	err := e.Execute(program)
	if err != nil {
		t.Fatalf("执行错误: %v", err)
	}

	// 验证 os.Stdout 是否被恢复
	if os.Stdout != originalStdout {
		t.Errorf("os.Stdout 未正确恢复！原始: %v, 当前: %v", originalStdout, os.Stdout)
	}
}

// TestCommandSubstitutionStdoutRestoreAfterError 测试命令替换执行失败后 os.Stdout 是否被恢复
func TestCommandSubstitutionStdoutRestoreAfterError(t *testing.T) {
	// 保存原始的 os.Stdout
	originalStdout := os.Stdout

	// 创建执行器
	e := New()

	// 测试命令：执行可能失败的命令替换
	command := "echo $(nonexistent_command 2>/dev/null || echo error)"
	
	// 解析命令
	l := lexer.New(command)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("解析错误: %v", p.Errors())
	}

	// 执行命令（可能会失败，但不应该影响 os.Stdout 的恢复）
	_ = e.Execute(program)

	// 验证 os.Stdout 是否被恢复（即使命令失败也应该恢复）
	if os.Stdout != originalStdout {
		t.Errorf("os.Stdout 未正确恢复（命令失败后）！原始: %v, 当前: %v", originalStdout, os.Stdout)
	}
}

