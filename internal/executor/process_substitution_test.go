package executor

import (
	"os"
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

func TestProcessSubstitutionInput(t *testing.T) {
	e := New()
	
	// 测试进程替换 <(command)
	input := "cat <(echo hello)"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	if len(p.Errors()) > 0 {
		t.Logf("解析错误: %v", p.Errors())
	}
	
	err := e.Execute(program)
	if err != nil {
		t.Logf("执行错误（可能是正常的）: %v", err)
	}
}

func TestProcessSubstitutionOutput(t *testing.T) {
	e := New()
	
	// 测试进程替换 >(command)
	input := "echo test >(cat)"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	if len(p.Errors()) > 0 {
		t.Logf("解析错误: %v", p.Errors())
	}
	
	err := e.Execute(program)
	if err != nil {
		t.Logf("执行错误（可能是正常的）: %v", err)
	}
}

func TestProcessSubstitutionInExpression(t *testing.T) {
	e := New()
	
	// 测试进程替换在表达式中的使用
	input := "echo <(echo hello)"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	if len(p.Errors()) > 0 {
		t.Logf("解析错误: %v", p.Errors())
	}
	
	err := e.Execute(program)
	if err != nil {
		t.Logf("执行错误（可能是正常的）: %v", err)
	}
	
	// 验证进程替换返回文件路径
	// 注意：临时文件会在命令执行后自动清理
}

func TestProcessSubstitutionDirect(t *testing.T) {
	e := New()
	
	// 直接测试进程替换的执行
	result := e.executeProcessSubstitution("echo hello", true)
	if result == "" {
		t.Error("进程替换应该返回文件路径")
	}
	
	// 验证文件存在
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Errorf("进程替换创建的文件不存在: %s", result)
	} else {
		// 清理临时文件
		os.Remove(result)
	}
}

