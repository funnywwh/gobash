package executor

import (
	"os"
	"path/filepath"
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// TestDoubleBracketCommand 测试 [[ condition ]] 命令执行
func TestDoubleBracketCommand(t *testing.T) {
	e := New()
	
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkFunc func(t *testing.T, err error)
	}{
		{
			name:    "基本 [[ 命令 - 文件存在",
			input:   "[[ -f /dev/null ]]",
			wantErr: false,
			checkFunc: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("期望成功，得到错误: %v", err)
				}
			},
		},
		{
			name:    "基本 [[ 命令 - 文件不存在",
			input:   "[[ -f /nonexistent/file ]]",
			wantErr: true, // 文件不存在，条件为假，应该返回错误
			checkFunc: func(t *testing.T, err error) {
				if err == nil {
					t.Error("期望错误（文件不存在），但没有错误")
				}
			},
		},
		{
			name:    "[[ 命令带 && - 两个条件都为真",
			input:   "[[ -f /dev/null && -r /dev/null ]]",
			wantErr: false,
			checkFunc: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("期望成功，得到错误: %v", err)
				}
			},
		},
		{
			name:    "[[ 命令带 && - 一个条件为假",
			input:   "[[ -f /dev/null && -f /nonexistent ]]",
			wantErr: true, // 第二个条件为假，整个表达式为假
			checkFunc: func(t *testing.T, err error) {
				if err == nil {
					t.Error("期望错误（条件为假），但没有错误")
				}
			},
		},
		{
			name:    "[[ 命令带 || - 至少一个条件为真",
			input:   "[[ -f /nonexistent || -f /dev/null ]]",
			wantErr: false, // 第二个条件为真，整个表达式为真
			checkFunc: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("期望成功，得到错误: %v", err)
				}
			},
		},
		{
			name:    "[[ 命令带 || - 两个条件都为假",
			input:   "[[ -f /nonexistent1 || -f /nonexistent2 ]]",
			wantErr: true, // 两个条件都为假，整个表达式为假
			checkFunc: func(t *testing.T, err error) {
				if err == nil {
					t.Error("期望错误（条件为假），但没有错误")
				}
			},
		},
		{
			name:    "[[ 命令带 !",
			input:   "[[ ! -f /nonexistent ]]",
			wantErr: false, // 文件不存在，! 后为真
			checkFunc: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("期望成功，得到错误: %v", err)
				}
			},
		},
		{
			name:    "[[ 命令带括号",
			input:   "[[ (-f /dev/null) ]]",
			wantErr: false,
			checkFunc: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("期望成功，得到错误: %v", err)
				}
			},
		},
		{
			name:    "[[ 命令复杂表达式",
			input:   "[[ (-f /dev/null && -r /dev/null) || -d /tmp ]]",
			wantErr: false,
			checkFunc: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("期望成功，得到错误: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(program.Statements) == 0 {
				t.Fatalf("解析 '%s' 失败：没有语句", tt.input)
			}

			err := e.Execute(program)
			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("错误状态不匹配，期望错误: %v, 得到错误: %v, 错误: %v", tt.wantErr, hasErr, err)
			}
			
			if tt.checkFunc != nil {
				tt.checkFunc(t, err)
			}
		})
	}
}

// TestDoubleBracketInIfStatement 测试在 if 语句中使用 [[
func TestDoubleBracketInIfStatement(t *testing.T) {
	e := New()
	
	// 创建临时文件用于测试
	testFile := filepath.Join(os.TempDir(), "gobash_test_double_bracket.txt")
	defer os.Remove(testFile)
	
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "if 语句中使用 [[",
			input:   "if [[ -f " + testFile + " ]]; then echo found; fi",
			wantErr: false,
		},
		{
			name:    "if 语句中使用 [[ 带 &&",
			input:   "if [[ -f " + testFile + " && -r " + testFile + " ]]; then echo found; fi",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(program.Statements) == 0 {
				t.Fatalf("解析 '%s' 失败：没有语句", tt.input)
			}

			err := e.Execute(program)
			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("错误状态不匹配，期望错误: %v, 得到错误: %v, 错误: %v", tt.wantErr, hasErr, err)
			}
		})
	}
}

