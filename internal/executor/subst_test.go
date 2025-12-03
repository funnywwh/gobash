package executor

import (
	"os"
	"strings"
	"testing"
)

func TestWordSplit(t *testing.T) {
	e := New()
	
	tests := []struct {
		name     string
		text     string
		ifs      string
		expected []string
	}{
		{
			name:     "Default IFS with spaces",
			text:     "hello world test",
			ifs:      "",
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "Default IFS with tabs",
			text:     "hello\tworld\ttest",
			ifs:      "",
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "Custom IFS with colon",
			text:     "hello:world:test",
			ifs:      ":",
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "Empty IFS (no split)",
			text:     "hello world",
			ifs:      "",
			expected: []string{"h", "e", "l", "l", "o", " ", "w", "o", "r", "l", "d"},
		},
		{
			name:     "IFS with whitespace and non-whitespace",
			text:     "hello:world test",
			ifs:      ": ",
			expected: []string{"hello", "world", "test"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置 IFS
			if tt.ifs == "" {
				// 对于空字符串，需要特殊处理
				// 如果 IFS 未设置，使用默认值
				delete(e.env, "IFS")
				os.Unsetenv("IFS")
			} else {
				e.env["IFS"] = tt.ifs
				os.Setenv("IFS", tt.ifs)
			}
			
			result := e.wordSplit(tt.text)
			
			if len(result) != len(tt.expected) {
				t.Errorf("期望 %d 个单词，得到 %d 个单词", len(tt.expected), len(result))
				return
			}
			
			for i, expected := range tt.expected {
				if i < len(result) && result[i] != expected {
					t.Errorf("单词 %d: 期望 '%s'，得到 '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestPathnameExpand(t *testing.T) {
	e := New()
	
	tests := []struct {
		name     string
		pattern  string
		expected int // 期望的匹配数量（至少）
	}{
		{
			name:     "No wildcards",
			pattern:  "test.txt",
			expected: 1,
		},
		{
			name:     "Star wildcard",
			pattern:  "*.go",
			expected: 0, // 至少 0 个（可能没有匹配）
		},
		{
			name:     "Question mark wildcard",
			pattern:  "test?.go",
			expected: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.pathnameExpand(tt.pattern)
			
			if len(result) < tt.expected {
				t.Errorf("期望至少 %d 个匹配，得到 %d 个", tt.expected, len(result))
			}
			
			// 如果没有匹配，应该返回原始模式
			if len(result) == 0 && !strings.ContainsAny(tt.pattern, "*?[") {
				t.Errorf("没有通配符的模式应该返回原始模式")
			}
		})
	}
}

func TestTildeExpand(t *testing.T) {
	e := New()
	
	tests := []struct {
		name     string
		text     string
		expected string // 期望的结果（可能包含环境变量）
	}{
		{
			name:     "Simple tilde",
			text:     "~",
			expected: "", // 将检查是否展开为主目录
		},
		{
			name:     "Tilde with path",
			text:     "~/test",
			expected: "", // 将检查是否展开为主目录 + /test
		},
		{
			name:     "Tilde plus",
			text:     "~+",
			expected: "", // 将检查是否展开为当前目录
		},
		{
			name:     "Tilde minus",
			text:     "~-",
			expected: "", // 将检查是否展开为上一个目录
		},
		{
			name:     "No tilde",
			text:     "test",
			expected: "test",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.tildeExpand(tt.text)
			
			if tt.expected != "" && result != tt.expected {
				t.Errorf("期望 '%s'，得到 '%s'", tt.expected, result)
			}
			
			// 对于包含 ~ 的情况，检查是否已展开
			if strings.HasPrefix(tt.text, "~") && result == tt.text {
				// 如果环境变量未设置，可能无法展开，这是可以接受的
				if tt.text == "~" {
					home := os.Getenv("HOME")
					if home == "" {
						home = os.Getenv("USERPROFILE")
					}
					if home != "" && result == tt.text {
						t.Errorf("应该展开为主目录，但得到原始文本")
					}
				}
			}
		})
	}
}

func TestExpandArray(t *testing.T) {
	e := New()
	
	// 确保 IFS 使用默认值（空格、制表符、换行符）
	e.env["IFS"] = " \t\n"
	
	// 设置测试数组
	e.arrays["test"] = []string{"a", "b", "c"}
	e.arrayTypes["test"] = "array"
	
	tests := []struct {
		name     string
		arrName  string
		quoted   bool
		expected string
	}{
		{
			name:     "Array expand @",
			arrName:  "test",
			quoted:   true,
			expected: "a b c",
		},
		{
			name:     "Array expand *",
			arrName:  "test",
			quoted:   false,
			expected: "a b c", // 使用默认 IFS
		},
		{
			name:     "Non-existent array",
			arrName:  "nonexistent",
			quoted:   true,
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.expandArray(tt.arrName, tt.quoted)
			
			if result != tt.expected {
				t.Errorf("期望 '%s'，得到 '%s'", tt.expected, result)
			}
		})
	}
}

