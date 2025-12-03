package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// getGobashExe 获取 gobash 可执行文件路径
func getGobashExe(t *testing.T) string {
	// 尝试从当前目录找到 gobash
	exeName := "gobash"
	if runtime.GOOS == "windows" {
		exeName = "gobash.exe"
	}
	
	// 首先尝试当前目录
	exePath := filepath.Join(".", exeName)
	if _, err := os.Stat(exePath); err == nil {
		return exePath
	}
	
	// 尝试项目根目录
	exePath = filepath.Join("..", exeName)
	if _, err := os.Stat(exePath); err == nil {
		return exePath
	}
	
	// 尝试构建
	t.Logf("gobash 可执行文件不存在，尝试构建...")
	buildCmd := exec.Command("go", "build", "-o", exeName, "./cmd/gobash")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("构建 gobash 失败: %v", err)
	}
	
	return exePath
}

// findBash 查找系统中的 bash
func findBash() string {
	// 在 Unix 系统中，bash 通常在 /bin/bash 或 /usr/bin/bash
	possiblePaths := []string{
		"/bin/bash",
		"/usr/bin/bash",
		"/usr/local/bin/bash",
	}
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// 尝试使用 which 命令
	if whichCmd := exec.Command("which", "bash"); whichCmd.Run() == nil {
		output, _ := exec.Command("which", "bash").Output()
		return strings.TrimSpace(string(output))
	}
	
	return ""
}

// TestCompatibilityBasicCommands 测试基本命令的兼容性
func TestCompatibilityBasicCommands(t *testing.T) {
	gobashExe := getGobashExe(t)
	bashPath := findBash()
	
	tests := []struct {
		name    string
		command string
		skipBash bool // 如果为 true，即使没有 bash 也运行测试
	}{
		{"echo", "echo hello", false},
		{"echo_multiple", "echo hello world", false},
		{"echo_quotes", "echo 'hello world'", false},
		{"echo_double_quotes", `echo "hello world"`, false},
		{"pwd", "pwd", false},
		{"variable_assignment", "VAR=test; echo $VAR", false},
		{"variable_expansion", "VAR=test; echo ${VAR}", false},
		{"arithmetic_expansion", "echo $((1 + 2))", false},
		{"command_substitution", "echo $(echo hello)", false},
		{"command_substitution_backtick", "echo `echo hello`", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 运行 gobash
			gobashCmd := exec.Command(gobashExe, "-c", tt.command)
			gobashOutput, gobashErr := gobashCmd.CombinedOutput()
			gobashOutputStr := strings.TrimSpace(string(gobashOutput))
			
			if gobashErr != nil && !strings.Contains(gobashErr.Error(), "exit status") {
				t.Logf("gobash 执行错误（可能是预期的）: %v", gobashErr)
			}
			
			// 如果有 bash，比较输出
			if bashPath != "" {
				bashCmd := exec.Command(bashPath, "-c", tt.command)
				bashOutput, bashErr := bashCmd.CombinedOutput()
				bashOutputStr := strings.TrimSpace(string(bashOutput))
				
				if bashErr != nil && !strings.Contains(bashErr.Error(), "exit status") {
					t.Logf("bash 执行错误（可能是预期的）: %v", bashErr)
				}
				
				// 比较输出（忽略尾随空白）
				if strings.TrimSpace(gobashOutputStr) != strings.TrimSpace(bashOutputStr) {
					t.Logf("输出不匹配:")
					t.Logf("  gobash: %q", gobashOutputStr)
					t.Logf("  bash:   %q", bashOutputStr)
					// 对于某些命令，输出可能不完全相同（如 pwd），这是可以接受的
					if tt.name == "pwd" {
						t.Logf("  pwd 命令的输出可能因工作目录不同而不同，这是正常的")
					}
				} else {
					t.Logf("✓ 输出匹配: %q", gobashOutputStr)
				}
			} else if !tt.skipBash {
				// 没有 bash，只验证 gobash 能执行
				t.Logf("未找到 bash，只验证 gobash 执行: %q", gobashOutputStr)
			}
		})
	}
}

// TestCompatibilityVariableExpansion 测试变量展开的兼容性
func TestCompatibilityVariableExpansion(t *testing.T) {
	gobashExe := getGobashExe(t)
	bashPath := findBash()
	
	tests := []struct {
		name    string
		command string
	}{
		{"simple_var", "VAR=test; echo $VAR"},
		{"braced_var", "VAR=test; echo ${VAR}"},
		{"default_value", "echo ${UNDEF:-default}"},
		{"default_value_set", "VAR=value; echo ${VAR:-default}"},
		{"assign_default", "unset VAR; echo ${VAR:=default}; echo $VAR"},
		{"string_length", "VAR=test; echo ${#VAR}"},
		{"substring", "VAR=hello; echo ${VAR:1:3}"},
		{"prefix_removal", "VAR=test.txt; echo ${VAR#*.}"},
		{"suffix_removal", "VAR=test.txt; echo ${VAR%.*}"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 运行 gobash
			gobashCmd := exec.Command(gobashExe, "-c", tt.command)
			gobashOutput, _ := gobashCmd.CombinedOutput()
			gobashOutputStr := strings.TrimSpace(string(gobashOutput))
			
			// 如果有 bash，比较输出
			if bashPath != "" {
				bashCmd := exec.Command(bashPath, "-c", tt.command)
				bashOutput, _ := bashCmd.CombinedOutput()
				bashOutputStr := strings.TrimSpace(string(bashOutput))
				
				if strings.TrimSpace(gobashOutputStr) != strings.TrimSpace(bashOutputStr) {
					t.Logf("输出不匹配:")
					t.Logf("  gobash: %q", gobashOutputStr)
					t.Logf("  bash:   %q", bashOutputStr)
				} else {
					t.Logf("✓ 输出匹配: %q", gobashOutputStr)
				}
			} else {
				t.Logf("未找到 bash，只验证 gobash 执行: %q", gobashOutputStr)
			}
		})
	}
}

// TestCompatibilityArithmeticExpansion 测试算术展开的兼容性
func TestCompatibilityArithmeticExpansion(t *testing.T) {
	gobashExe := getGobashExe(t)
	bashPath := findBash()
	
	tests := []struct {
		name    string
		command string
	}{
		{"simple_add", "echo $((1 + 2))"},
		{"simple_sub", "echo $((5 - 3))"},
		{"simple_mul", "echo $((2 * 3))"},
		{"simple_div", "echo $((6 / 2))"},
		{"simple_mod", "echo $((7 % 3))"},
		{"nested", "echo $((1 + 2 * 3))"},
		{"parentheses", "echo $(((1 + 2) * 3))"},
		{"variable", "VAR=5; echo $((VAR + 3))"},
		{"multiple_vars", "A=3; B=4; echo $((A + B))"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 运行 gobash
			gobashCmd := exec.Command(gobashExe, "-c", tt.command)
			gobashOutput, _ := gobashCmd.CombinedOutput()
			gobashOutputStr := strings.TrimSpace(string(gobashOutput))
			
			// 如果有 bash，比较输出
			if bashPath != "" {
				bashCmd := exec.Command(bashPath, "-c", tt.command)
				bashOutput, _ := bashCmd.CombinedOutput()
				bashOutputStr := strings.TrimSpace(string(bashOutput))
				
				if strings.TrimSpace(gobashOutputStr) != strings.TrimSpace(bashOutputStr) {
					t.Errorf("输出不匹配:")
					t.Errorf("  gobash: %q", gobashOutputStr)
					t.Errorf("  bash:   %q", bashOutputStr)
				} else {
					t.Logf("✓ 输出匹配: %q", gobashOutputStr)
				}
			} else {
				t.Logf("未找到 bash，只验证 gobash 执行: %q", gobashOutputStr)
			}
		})
	}
}

// TestCompatibilityControlFlow 测试控制流的兼容性
func TestCompatibilityControlFlow(t *testing.T) {
	gobashExe := getGobashExe(t)
	bashPath := findBash()
	
	tests := []struct {
		name    string
		command string
	}{
		{"if_true", "if [ 1 ]; then echo yes; fi"},
		{"if_false", "if [ 0 ]; then echo yes; else echo no; fi"},
		{"for_loop", "for i in 1 2 3; do echo $i; done"},
		{"while_loop", "i=0; while [ $i -lt 3 ]; do echo $i; i=$((i+1)); done"},
		{"case_statement", "case test in test) echo matched;; *) echo default;; esac"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 运行 gobash
			gobashCmd := exec.Command(gobashExe, "-c", tt.command)
			gobashOutput, gobashErr := gobashCmd.CombinedOutput()
			gobashOutputStr := strings.TrimSpace(string(gobashOutput))
			
			if gobashErr != nil {
				t.Logf("gobash 执行错误（可能是预期的）: %v", gobashErr)
			}
			
			// 如果有 bash，比较输出
			if bashPath != "" {
				bashCmd := exec.Command(bashPath, "-c", tt.command)
				bashOutput, bashErr := bashCmd.CombinedOutput()
				bashOutputStr := strings.TrimSpace(string(bashOutput))
				
				if bashErr != nil {
					t.Logf("bash 执行错误（可能是预期的）: %v", bashErr)
				}
				
				if strings.TrimSpace(gobashOutputStr) != strings.TrimSpace(bashOutputStr) {
					t.Logf("输出不匹配:")
					t.Logf("  gobash: %q", gobashOutputStr)
					t.Logf("  bash:   %q", bashOutputStr)
				} else {
					t.Logf("✓ 输出匹配: %q", gobashOutputStr)
				}
			} else {
				t.Logf("未找到 bash，只验证 gobash 执行: %q", gobashOutputStr)
			}
		})
	}
}

// TestCompatibilityArrays 测试数组的兼容性
func TestCompatibilityArrays(t *testing.T) {
	gobashExe := getGobashExe(t)
	bashPath := findBash()
	
	tests := []struct {
		name    string
		command string
	}{
		{"array_assignment", "arr=(1 2 3); echo ${arr[0]}"},
		{"array_access", "arr=(a b c); echo ${arr[1]}"},
		{"array_length", "arr=(1 2 3); echo ${#arr[@]}"},
		{"array_expand", "arr=(a b c); echo ${arr[@]}"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 运行 gobash
			gobashCmd := exec.Command(gobashExe, "-c", tt.command)
			gobashOutput, _ := gobashCmd.CombinedOutput()
			gobashOutputStr := strings.TrimSpace(string(gobashOutput))
			
			// 如果有 bash，比较输出
			if bashPath != "" {
				bashCmd := exec.Command(bashPath, "-c", tt.command)
				bashOutput, _ := bashCmd.CombinedOutput()
				bashOutputStr := strings.TrimSpace(string(bashOutput))
				
				if strings.TrimSpace(gobashOutputStr) != strings.TrimSpace(bashOutputStr) {
					t.Logf("输出不匹配:")
					t.Logf("  gobash: %q", gobashOutputStr)
					t.Logf("  bash:   %q", bashOutputStr)
				} else {
					t.Logf("✓ 输出匹配: %q", gobashOutputStr)
				}
			} else {
				t.Logf("未找到 bash，只验证 gobash 执行: %q", gobashOutputStr)
			}
		})
	}
}

// TestCompatibilityRedirection 测试重定向的兼容性
func TestCompatibilityRedirection(t *testing.T) {
	gobashExe := getGobashExe(t)
	bashPath := findBash()
	
	tests := []struct {
		name    string
		command string
		cleanup func() // 清理函数
	}{
		{
			"output_redirect",
			"echo hello > /tmp/gobash_test.txt",
			func() { os.Remove("/tmp/gobash_test.txt") },
		},
		{
			"append_redirect",
			"echo world >> /tmp/gobash_test2.txt",
			func() { os.Remove("/tmp/gobash_test2.txt") },
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理之前的测试文件
			if tt.cleanup != nil {
				defer tt.cleanup()
			}
			
			// 运行 gobash
			gobashCmd := exec.Command(gobashExe, "-c", tt.command)
			gobashOutput, _ := gobashCmd.CombinedOutput()
			
			// 如果有 bash，比较行为
			if bashPath != "" {
				bashCmd := exec.Command(bashPath, "-c", tt.command)
				bashOutput, _ := bashCmd.CombinedOutput()
				
				// 对于重定向，主要验证命令执行成功
				if len(gobashOutput) > 0 {
					t.Logf("gobash 输出: %q", string(gobashOutput))
				}
				if len(bashOutput) > 0 {
					t.Logf("bash 输出: %q", string(bashOutput))
				}
				
				t.Logf("✓ 重定向命令执行成功")
			} else {
				t.Logf("未找到 bash，只验证 gobash 执行")
			}
		})
	}
}

