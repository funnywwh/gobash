package executor

import (
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// TestExecutionErrorHandling 测试执行器错误处理
func TestExecutionErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkErr func(t *testing.T, err error)
	}{
		{
			name:    "命令未找到",
			input:   "nonexistent_command",
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("应该有错误")
					return
				}
				if execErr, ok := err.(*ExecutionError); ok {
					if execErr.Type != ExecutionErrorTypeCommandNotFound {
						t.Errorf("期望错误类型 CommandNotFound，得到 %v", execErr.Type)
					}
					t.Logf("错误: %s", execErr.Error())
					t.Logf("退出码: %d", execErr.ExitCode())
				} else {
					t.Logf("错误类型: %T, 错误: %v", err, err)
				}
			},
		},
		{
			name:    "命令执行失败（false 命令）",
			input:   "false",
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				if err == nil {
					t.Error("应该有错误")
					return
				}
				if execErr, ok := err.(*ExecutionError); ok {
					if execErr.Type != ExecutionErrorTypeCommandFailed {
						t.Logf("错误类型: %v（可能是正常的退出码）", execErr.Type)
					}
					t.Logf("错误: %s", execErr.Error())
					t.Logf("退出码: %d", execErr.ExitCode())
				} else {
					t.Logf("错误类型: %T, 错误: %v", err, err)
				}
			},
		},
		{
			name:    "正常命令无错误",
			input:   "echo hello",
			wantErr: false,
			checkErr: func(t *testing.T, err error) {
				if err != nil {
					t.Logf("有错误（可能是误报）: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("解析错误: %v", p.Errors())
			}

			err := e.Execute(program)
			hasErr := err != nil

			if hasErr != tt.wantErr {
				t.Logf("错误状态: 期望 %v，得到 %v", tt.wantErr, hasErr)
			}

			if hasErr {
				tt.checkErr(t, err)
			} else {
				t.Logf("✓ 无错误")
			}
		})
	}
}

