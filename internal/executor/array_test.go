package executor

import (
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

func TestArrayAssignment(t *testing.T) {
	e := New()
	
	// 测试数组赋值
	input := "arr=(1 2 3)"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	// 检查解析错误
	if len(p.Errors()) > 0 {
		t.Logf("解析错误: %v", p.Errors())
	}
	
	// 检查是否解析为数组赋值语句
	if len(program.Statements) == 0 {
		t.Fatal("没有解析到任何语句")
	}
	
	_, isArrayAssignment := program.Statements[0].(*parser.ArrayAssignmentStatement)
	if !isArrayAssignment {
		t.Logf("语句类型: %T，期望 *parser.ArrayAssignmentStatement", program.Statements[0])
		// 如果不是数组赋值，尝试执行看看会发生什么
	}
	
	err := e.Execute(program)
	if err != nil {
		t.Logf("执行错误（可能是正常的）: %v", err)
	}
	
	// 验证数组已创建
	if arr, ok := e.arrays["arr"]; !ok {
		t.Error("数组未创建")
	} else if len(arr) != 3 {
		t.Errorf("数组长度错误，期望 3，得到 %d", len(arr))
	} else if arr[0] != "1" || arr[1] != "2" || arr[2] != "3" {
		t.Errorf("数组元素错误，期望 [1 2 3]，得到 %v", arr)
	}
}

func TestArrayAccess(t *testing.T) {
	e := New()
	
	// 先创建数组
	e.arrays["arr"] = []string{"first", "second", "third"}
	
	// 测试数组访问 ${arr[0]}
	input := "echo ${arr[0]}"
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	err := e.Execute(program)
	if err != nil {
		t.Errorf("数组访问执行失败: %v", err)
	}
}

func TestArrayInString(t *testing.T) {
	e := New()
	
	// 先创建数组
	e.arrays["arr"] = []string{"hello", "world"}
	
	// 测试字符串中的数组访问
	input := "echo \"First: ${arr[0]}, Second: ${arr[1]}\""
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	
	err := e.Execute(program)
	if err != nil {
		t.Errorf("字符串中的数组访问执行失败: %v", err)
	}
}

