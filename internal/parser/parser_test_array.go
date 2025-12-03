package parser

import (
	"testing"
	"gobash/internal/lexer"
)

// TestParseArrayAssignment 测试数组赋值解析
func TestParseArrayAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkFunc func(t *testing.T, stmt *ArrayAssignmentStatement)
	}{
		{
			name:  "基本数组赋值",
			input: "arr=(1 2 3)",
			checkFunc: func(t *testing.T, stmt *ArrayAssignmentStatement) {
				if stmt.Name != "arr" {
					t.Errorf("数组名错误，期望 'arr', 得到 '%s'", stmt.Name)
				}
				if len(stmt.Values) != 3 {
					t.Errorf("数组元素数量错误，期望 3，得到 %d", len(stmt.Values))
				}
			},
		},
		{
			name:  "带索引的数组赋值",
			input: "arr=([0]=a [1]=b [2]=c)",
			checkFunc: func(t *testing.T, stmt *ArrayAssignmentStatement) {
				if stmt.Name != "arr" {
					t.Errorf("数组名错误，期望 'arr', 得到 '%s'", stmt.Name)
				}
				if len(stmt.IndexedValues) != 3 {
					t.Errorf("带索引的数组元素数量错误，期望 3，得到 %d", len(stmt.IndexedValues))
				}
				// 检查索引
				if _, ok := stmt.IndexedValues["0"]; !ok {
					t.Error("缺少索引 '0'")
				}
				if _, ok := stmt.IndexedValues["1"]; !ok {
					t.Error("缺少索引 '1'")
				}
				if _, ok := stmt.IndexedValues["2"]; !ok {
					t.Error("缺少索引 '2'")
				}
			},
		},
		{
			name:  "带索引的数组赋值（不连续索引）",
			input: "arr=([0]=a [2]=c)",
			checkFunc: func(t *testing.T, stmt *ArrayAssignmentStatement) {
				if stmt.Name != "arr" {
					t.Errorf("数组名错误，期望 'arr', 得到 '%s'", stmt.Name)
				}
				if len(stmt.IndexedValues) != 2 {
					t.Errorf("带索引的数组元素数量错误，期望 2，得到 %d", len(stmt.IndexedValues))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(program.Statements) == 0 {
				t.Fatalf("解析 '%s' 失败：没有语句", tt.input)
			}

			stmt, ok := program.Statements[0].(*ArrayAssignmentStatement)
			if !ok {
				t.Fatalf("解析 '%s' 失败：不是数组赋值语句，得到 %T", tt.input, program.Statements[0])
			}

			tt.checkFunc(t, stmt)
		})
	}
}

