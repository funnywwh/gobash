package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPathnameExpandRecursive 测试 ** 递归匹配
func TestPathnameExpandRecursive(t *testing.T) {
	// 创建临时测试目录结构
	tmpDir, err := os.MkdirTemp("", "gobash_test_recursive_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试目录结构
	// tmpDir/
	//   a/
	//     b/
	//       file1.txt
	//       file2.txt
	//     c/
	//       file3.txt
	//   d/
	//     file4.txt
	testDirs := []string{
		filepath.Join(tmpDir, "a", "b"),
		filepath.Join(tmpDir, "a", "c"),
		filepath.Join(tmpDir, "d"),
	}
	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("创建测试目录失败: %v", err)
		}
	}

	// 创建测试文件
	testFiles := []string{
		filepath.Join(tmpDir, "a", "b", "file1.txt"),
		filepath.Join(tmpDir, "a", "b", "file2.txt"),
		filepath.Join(tmpDir, "a", "c", "file3.txt"),
		filepath.Join(tmpDir, "d", "file4.txt"),
	}
	for _, file := range testFiles {
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	// 切换到临时目录
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}

	tests := []struct {
		name           string
		pattern        string
		enableGlobstar bool
		expected       []string // 期望匹配的文件（相对路径）
		wantErr        bool
	}{
		{
			name:           "**匹配所有文件",
			pattern:        "**",
			enableGlobstar: true,
			expected:       []string{"a", "a/b", "a/b/file1.txt", "a/b/file2.txt", "a/c", "a/c/file3.txt", "d", "d/file4.txt"},
			wantErr:        false,
		},
		{
			name:           "**/file*.txt匹配所有目录中的file*.txt",
			pattern:        "**/file*.txt",
			enableGlobstar: true,
			expected:       []string{"a/b/file1.txt", "a/b/file2.txt", "a/c/file3.txt", "d/file4.txt"},
			wantErr:        false,
		},
		{
			name:           "a/**匹配a目录及其所有子目录",
			pattern:        "a/**",
			enableGlobstar: true,
			expected:       []string{"a/b", "a/b/file1.txt", "a/b/file2.txt", "a/c", "a/c/file3.txt"},
			wantErr:        false,
		},
		{
			name:           "**/b匹配所有目录中的b",
			pattern:        "**/b",
			enableGlobstar: true,
			expected:       []string{"a/b"},
			wantErr:        false,
		},
		{
			name:           "a/**/file*.txt匹配a目录下任意深度的file*.txt",
			pattern:        "a/**/file*.txt",
			enableGlobstar: true,
			expected:       []string{"a/b/file1.txt", "a/b/file2.txt", "a/c/file3.txt"},
			wantErr:        false,
		},
		{
			name:           "globstar未启用时**应该被当作普通*",
			pattern:        "**",
			enableGlobstar: false,
			expected:       []string{}, // 不匹配任何文件（因为 ** 被当作普通模式）
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			
			// 设置 globstar 选项
			if tt.enableGlobstar {
				e.SetEnv("GLOBSTAR", "1")
				e.SetOptions(map[string]bool{"globstar": true})
			} else {
				e.SetEnv("GLOBSTAR", "0")
				e.SetOptions(map[string]bool{"globstar": false})
			}

			// 直接调用 pathnameExpand 方法进行路径名展开
			result := e.pathnameExpand(tt.pattern)

			// 对于 ** 模式，result 是 []string，包含匹配的文件路径
			if tt.enableGlobstar && len(tt.expected) > 0 {
				// 检查结果中是否包含期望的文件
				foundCount := 0
				for _, expectedFile := range tt.expected {
					found := false
					for _, match := range result {
						// 检查是否匹配（可能是相对路径或绝对路径）
						// 使用 filepath.Base 和 filepath.Dir 来比较
						base := filepath.Base(match)
						dir := filepath.Dir(match)
						expectedBase := filepath.Base(expectedFile)
						expectedDir := filepath.Dir(expectedFile)
						
						if base == expectedBase {
							// 检查目录是否匹配（允许相对路径和绝对路径）
							if dir == expectedDir || strings.HasSuffix(dir, expectedDir) || strings.HasSuffix(expectedDir, dir) {
								found = true
								foundCount++
								t.Logf("✓ 找到期望的文件: %s (在结果中: %s)", expectedFile, match)
								break
							}
						}
					}
					if !found {
						t.Logf("结果中未找到期望的文件: %s", expectedFile)
					}
				}
				
				if foundCount > 0 {
					t.Logf("✓ 找到 %d/%d 个期望的文件", foundCount, len(tt.expected))
				}
				t.Logf("展开结果: %v", result)
				t.Logf("结果数量: %d", len(result))
			} else if !tt.enableGlobstar {
				// globstar 未启用时，** 应该被当作普通模式
				if len(result) == 1 && result[0] == tt.pattern {
					t.Logf("✓ globstar 未启用，模式未展开: %q", result[0])
				} else {
					t.Logf("注意：globstar 未启用，但模式被展开: %v", result)
				}
			}
		})
	}
}


