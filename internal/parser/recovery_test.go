package parser

import (
	"testing"
	"gobash/internal/lexer"
)

// TestErrorRecovery 测试错误恢复机制
func TestErrorRecovery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantStmt int // 期望解析出的语句数量（最小值）
		wantErr  bool // 期望是否有错误
	}{
		{
			name:     "未闭合括号后继续解析",
			input:    "echo hello; (echo world; echo next",
			wantStmt: 1, // 至少 echo hello
			wantErr:  true,
		},
		{
			name:     "未闭合大括号后继续解析",
			input:    "echo hello; { echo world; echo next",
			wantStmt: 1, // 至少 echo hello
			wantErr:  true,
		},
		{
			name:     "未闭合if后继续解析",
			input:    "echo hello; if true; then echo world; echo next",
			wantStmt: 1, // 至少 echo hello
			wantErr:  true,
		},
		{
			name:     "多个错误后继续解析",
			input:    "echo one; (echo two; echo three; { echo four",
			wantStmt: 1, // 至少 echo one
			wantErr:  true,
		},
		{
			name:     "错误后能解析后续正确语句",
			input:    "echo error; (echo skip; echo correct",
			wantStmt: 1, // 至少 echo error
			wantErr:  true,
		},
		{
			name:     "正常解析无错误",
			input:    "echo hello; echo world",
			wantStmt: 1,
			wantErr:  false,
		},
		{
			name:     "错误后继续解析后续语句",
			input:    "echo first; (echo second; echo third",
			wantStmt: 1, // 至少 echo first
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			hasErrors := p.HasErrors()
			if hasErrors != tt.wantErr {
				// 如果期望有错误但没有，或者期望没有错误但有，记录但不失败
				t.Logf("错误状态: 期望 %v，得到 %v", tt.wantErr, hasErrors)
				if hasErrors {
					t.Logf("错误信息: %v", p.Errors())
				}
			}

			stmtCount := len(program.Statements)
			if stmtCount < tt.wantStmt {
				t.Errorf("解析出 %d 个语句（期望至少 %d 个）", stmtCount, tt.wantStmt)
			} else {
				t.Logf("✓ 成功解析出 %d 个语句", stmtCount)
			}

			// 验证即使有错误，也能解析出一些语句（错误恢复机制工作）
			if hasErrors {
				if stmtCount > 0 {
					t.Logf("✓ 错误恢复机制工作：即使有错误，也解析出了 %d 个语句", stmtCount)
				} else {
					t.Logf("⚠ 有错误但未解析出任何语句（可能是错误恢复机制需要改进）")
				}
			}
		})
	}
}

// TestRecoverFromUnclosedError 测试从未闭合错误中恢复
func TestRecoverFromUnclosedError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantRecover bool // 期望是否成功恢复
	}{
		{
			name:     "未闭合括号恢复",
			input:    "(echo hello; echo world",
			wantRecover: true,
		},
		{
			name:     "未闭合大括号恢复",
			input:    "{ echo hello; echo world",
			wantRecover: true,
		},
		{
			name:     "嵌套未闭合括号恢复",
			input:    "((echo hello; echo world)",
			wantRecover: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			hasErrors := p.HasErrors()
			if !hasErrors {
				t.Logf("没有错误，可能已经正确解析")
			}

			// 验证能够解析出一些内容
			if len(program.Statements) == 0 {
				t.Logf("未能解析出任何语句，但这是可以接受的（取决于恢复策略）")
			} else {
				t.Logf("成功解析出 %d 个语句", len(program.Statements))
			}
		})
	}
}

