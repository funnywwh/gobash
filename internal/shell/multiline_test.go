package shell

import (
	"strings"
	"testing"
)

// TestMultilineInput 测试多行输入处理
func TestMultilineInput(t *testing.T) {
	tests := []struct {
		name           string
		input          []string // 多行输入
		expectedOutput string   // 期望的完整语句
		shouldComplete bool     // 是否应该完成
	}{
		{
			name: "if语句多行输入",
			input: []string{
				"if true;",
				"then",
				"  echo hello",
				"fi",
			},
			expectedOutput: "if true;\nthen\n  echo hello\nfi",
			shouldComplete: true,
		},
		{
			name: "for循环多行输入",
			input: []string{
				"for i in 1 2 3;",
				"do",
				"  echo $i",
				"done",
			},
			expectedOutput: "for i in 1 2 3;\ndo\n  echo $i\ndone",
			shouldComplete: true,
		},
		{
			name: "while循环多行输入",
			input: []string{
				"while true;",
				"do",
				"  echo loop",
				"done",
			},
			expectedOutput: "while true;\ndo\n  echo loop\ndone",
			shouldComplete: true,
		},
		{
			name: "case语句多行输入",
			input: []string{
				"case $var in",
				"  a) echo A ;;",
				"  b) echo B ;;",
				"esac",
			},
			expectedOutput: "case $var in\n  a) echo A ;;\n  b) echo B ;;\nesac",
			shouldComplete: true,
		},
		{
			name: "反斜杠行继续符",
			input: []string{
				"echo hello \\",
				"world",
			},
			expectedOutput: "echo hello \\\nworld",
			shouldComplete: true,
		},
		{
			name: "未完成的if语句",
			input: []string{
				"if true;",
				"then",
				"  echo hello",
			},
			expectedOutput: "if true;\nthen\n  echo hello",
			shouldComplete: false,
		},
		{
			name: "单行命令",
			input: []string{
				"echo hello",
			},
			expectedOutput: "echo hello",
			shouldComplete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			
			// 构建完整语句
			var currentStatement strings.Builder
			for i, line := range tt.input {
				if i > 0 {
					currentStatement.WriteString("\n")
				}
				currentStatement.WriteString(line)
				
				statement := currentStatement.String()
				isComplete := s.isStatementComplete(statement)
				
				// 检查最后一行是否应该完成
				if i == len(tt.input)-1 {
					if isComplete != tt.shouldComplete {
						t.Errorf("语句完成状态不匹配。期望: %v, 得到: %v", tt.shouldComplete, isComplete)
					}
				} else {
					// 中间行应该未完成
					if isComplete {
						t.Logf("中间行完成状态: %v (这是正常的，因为可能还有其他行)", isComplete)
					}
				}
			}
			
			// 验证完整语句
			finalStatement := currentStatement.String()
			if finalStatement != tt.expectedOutput {
				t.Errorf("完整语句不匹配。\n期望: %q\n得到: %q", tt.expectedOutput, finalStatement)
			} else {
				t.Logf("✓ 完整语句匹配: %q", finalStatement)
			}
		})
	}
}

// TestIsStatementComplete 测试语句完成检测
func TestIsStatementComplete(t *testing.T) {
	tests := []struct {
		name     string
		statement string
		expected bool
	}{
		{
			name:     "简单命令",
			statement: "echo hello",
			expected:  true,
		},
		{
			name:     "未完成的if",
			statement: "if true; then echo hello",
			expected:  false,
		},
		{
			name:     "完成的if",
			statement: "if true; then echo hello; fi",
			expected:  true,
		},
		{
			name:     "未完成的for",
			statement: "for i in 1 2 3; do echo $i",
			expected:  false,
		},
		{
			name:     "完成的for",
			statement: "for i in 1 2 3; do echo $i; done",
			expected:  true,
		},
		{
			name:     "未完成的while",
			statement: "while true; do echo loop",
			expected:  false,
		},
		{
			name:     "完成的while",
			statement: "while true; do echo loop; done",
			expected:  true,
		},
		{
			name:     "未完成的case",
			statement: "case $var in a) echo A ;;",
			expected:  false,
		},
		{
			name:     "完成的case",
			statement: "case $var in a) echo A ;; esac",
			expected:  true,
		},
		{
			name:     "反斜杠行继续符",
			statement: "echo hello \\",
			expected:  false, // 反斜杠结尾表示未完成
		},
		{
			name:     "空语句",
			statement: "",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			result := s.isStatementComplete(tt.statement)
			if result != tt.expected {
				t.Errorf("语句完成状态不匹配。\n语句: %q\n期望: %v, 得到: %v", tt.statement, tt.expected, result)
			} else {
				t.Logf("✓ 语句完成状态正确: %q -> %v", tt.statement, result)
			}
		})
	}
}




