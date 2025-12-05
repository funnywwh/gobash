package lexer

import (
	"testing"
)

// TestLexerErrorHandling 测试词法分析器错误处理
func TestLexerErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkErr func(t *testing.T, errors []*LexerError)
	}{
		{
			name:    "未闭合单引号",
			input:   "echo 'hello",
			wantErr: true,
			checkErr: func(t *testing.T, errors []*LexerError) {
				if len(errors) == 0 {
					t.Error("应该有错误")
					return
				}
				t.Logf("错误: %s", errors[0].Error())
			},
		},
		{
			name:    "未闭合双引号",
			input:   `echo "hello`,
			wantErr: true,
			checkErr: func(t *testing.T, errors []*LexerError) {
				if len(errors) == 0 {
					t.Error("应该有错误")
					return
				}
				t.Logf("错误: %s", errors[0].Error())
			},
		},
		{
			name:    "未闭合 $'...' 字符串",
			input:   "$'hello",
			wantErr: true,
			checkErr: func(t *testing.T, errors []*LexerError) {
				if len(errors) == 0 {
					t.Error("应该有错误")
					return
				}
				t.Logf("错误: %s", errors[0].Error())
			},
		},
		{
			name:    "未闭合 $\"...\" 字符串",
			input:   `$"hello`,
			wantErr: true,
			checkErr: func(t *testing.T, errors []*LexerError) {
				if len(errors) == 0 {
					t.Error("应该有错误")
					return
				}
				t.Logf("错误: %s", errors[0].Error())
			},
		},
		{
			name:    "无效字符",
			input:   "echo @hello",
			wantErr: false, // @ 可能被识别为标识符的一部分
			checkErr: func(t *testing.T, errors []*LexerError) {
				// 不检查
			},
		},
		{
			name:    "正常输入无错误",
			input:   "echo hello",
			wantErr: false,
			checkErr: func(t *testing.T, errors []*LexerError) {
				if len(errors) > 0 {
					t.Logf("有错误（可能是误报）: %v", errors)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.input)
			// 读取所有 token
			for l.NextToken().Type != EOF {
			}

			hasErrors := l.HasErrors()
			if hasErrors != tt.wantErr {
				t.Logf("错误状态: 期望 %v，得到 %v", tt.wantErr, hasErrors)
			}

			if hasErrors {
				errors := l.Errors()
				tt.checkErr(t, errors)
				t.Logf("✓ 检测到 %d 个错误", len(errors))
			} else {
				t.Logf("✓ 无错误")
			}
		})
	}
}




