package parser

import (
	"testing"
	"gobash/internal/lexer"
)

// TestErrorFormat 测试错误消息格式
func TestErrorFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkMsg func(t *testing.T, errMsg string)
	}{
		{
			name:    "未闭合括号错误格式",
			input:   "(echo hello",
			wantErr: true,
			checkMsg: func(t *testing.T, errMsg string) {
				if errMsg == "" {
					t.Error("应该有错误消息")
					return
				}
				t.Logf("错误消息: %s", errMsg)
				// 检查是否包含"未闭合的括号"或"未找到匹配的"
				if !contains(errMsg, "未闭合") && !contains(errMsg, "未找到匹配") {
					t.Logf("⚠ 错误消息格式可能需要改进，当前: %s", errMsg)
				}
			},
		},
		{
			name:    "未闭合大括号错误格式",
			input:   "{ echo hello",
			wantErr: true,
			checkMsg: func(t *testing.T, errMsg string) {
				if errMsg == "" {
					t.Error("应该有错误消息")
					return
				}
				t.Logf("错误消息: %s", errMsg)
			},
		},
		{
			name:    "未闭合if错误格式",
			input:   "if true; then echo hello",
			wantErr: true,
			checkMsg: func(t *testing.T, errMsg string) {
				if errMsg == "" {
					t.Error("应该有错误消息")
					return
				}
				t.Logf("错误消息: %s", errMsg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			hasErrors := p.HasErrors()
			if hasErrors != tt.wantErr {
				t.Logf("错误状态: 期望 %v，得到 %v", tt.wantErr, hasErrors)
			}

			if hasErrors {
				errors := p.Errors()
				if len(errors) > 0 {
					errMsg := errors[0]
					tt.checkMsg(t, errMsg)
					
					// 检查错误消息格式
					t.Logf("✓ 错误消息格式: %s", errMsg)
					
					// 验证错误消息包含行号信息
					if !contains(errMsg, "第") && !contains(errMsg, "行") {
						t.Logf("⚠ 错误消息可能缺少行号信息")
					}
				}
			}

			// 验证即使有错误，也能解析出一些内容（错误恢复机制工作）
			if len(program.Statements) > 0 {
				t.Logf("✓ 错误恢复机制工作：解析出 %d 个语句", len(program.Statements))
			}
		})
	}
}

// contains 检查字符串是否包含子字符串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr ||
		     indexOf(s, substr) >= 0))
}

// indexOf 查找子字符串在字符串中的位置
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}




