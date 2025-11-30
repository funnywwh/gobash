package executor

import (
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

func TestAssocArrayDeclare(t *testing.T) {
	e := New()
	
	// 测试声明关联数组
	input := "declare -A arr"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	err := e.Execute(program)
	if err != nil {
		t.Errorf("declare命令执行失败: %v", err)
	}
	
	// 验证关联数组已声明
	if arrayType, ok := e.arrayTypes["arr"]; !ok || arrayType != "assoc" {
		t.Errorf("关联数组未正确声明，类型: %v", e.arrayTypes["arr"])
	}
	
	if e.assocArrays["arr"] == nil {
		t.Error("关联数组未初始化")
	}
}

func TestAssocArrayAssignment(t *testing.T) {
	e := New()
	
	// 先声明关联数组
	e.assocArrays["arr"] = make(map[string]string)
	e.arrayTypes["arr"] = "assoc"
	
	// 测试关联数组赋值 arr[key]=value
	input := "arr[hello]=world"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	err := e.Execute(program)
	if err != nil {
		t.Errorf("关联数组赋值执行失败: %v", err)
	}
	
	// 验证赋值
	if value, ok := e.assocArrays["arr"]["hello"]; !ok || value != "world" {
		t.Errorf("关联数组赋值失败，期望 'world'，得到 '%s'", value)
	}
}

func TestAssocArrayAccess(t *testing.T) {
	e := New()
	
	// 先设置关联数组
	e.assocArrays["arr"] = map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	e.arrayTypes["arr"] = "assoc"
	
	// 测试关联数组访问 ${arr[key]}
	input := "echo ${arr[key1]}"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	err := e.Execute(program)
	if err != nil {
		t.Errorf("关联数组访问执行失败: %v", err)
	}
	
	// 验证访问
	value := e.getArrayElement("arr[key1]")
	if value != "value1" {
		t.Errorf("关联数组访问失败，期望 'value1'，得到 '%s'", value)
	}
}

