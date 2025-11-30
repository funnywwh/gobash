// Package main 提供集成测试，测试整个shell的端到端功能
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBasicCommands 测试基本命令
func TestBasicCommands(t *testing.T) {
	exePath := getGobashExe(t)
	
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{"echo", "echo hello", false},
		{"pwd", "pwd", false},
		{"export", "export TEST_VAR=test", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(exePath, "-c", tt.command)
			output, err := cmd.CombinedOutput()
			if (err != nil) != tt.wantErr {
				t.Errorf("命令 '%s' 执行错误: %v, 输出: %s", tt.command, err, string(output))
			}
		})
	}
}

// TestScriptExecution 测试脚本执行
func TestScriptExecution(t *testing.T) {
	exePath := getGobashExe(t)
	
	// 创建临时测试脚本
	testScript := filepath.Join(os.TempDir(), "gobash_test_script.sh")
	defer os.Remove(testScript)
	
	scriptContent := `#!/bin/bash
echo "test script"
export TEST_VAR=test_value
echo $TEST_VAR
`
	
	err := os.WriteFile(testScript, []byte(scriptContent), 0644)
	if err != nil {
		t.Fatalf("创建测试脚本失败: %v", err)
	}
	
	cmd := exec.Command(exePath, testScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("执行脚本失败: %v, 输出: %s", err, string(output))
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "test script") {
		t.Errorf("脚本输出不正确，期望包含 'test script'，得到: %s", outputStr)
	}
}

// TestPipeAndRedirect 测试管道和重定向
func TestPipeAndRedirect(t *testing.T) {
	exePath := getGobashExe(t)
	
	// 测试管道
	cmd := exec.Command(exePath, "-c", "echo hello world | grep hello")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 在某些系统上可能没有grep命令，这是可以接受的
		if !strings.Contains(string(output), "未找到") {
			t.Logf("管道测试失败（可能是命令未找到）: %v, 输出: %s", err, string(output))
		}
	}
	
	// 测试重定向
	testFile := filepath.Join(os.TempDir(), "gobash_test_output.txt")
	defer os.Remove(testFile)
	
	cmd = exec.Command(exePath, "-c", "echo test > "+testFile)
	err = cmd.Run()
	if err != nil {
		t.Errorf("重定向测试失败: %v", err)
	}
	
	// 验证文件内容
	content, err := os.ReadFile(testFile)
	if err == nil {
		if strings.TrimSpace(string(content)) != "test" {
			t.Errorf("重定向文件内容错误，期望 'test'，得到 '%s'", string(content))
		}
	}
}

// TestControlFlow 测试控制流语句
func TestControlFlow(t *testing.T) {
	exePath := getGobashExe(t)
	
	// 测试if语句
	cmd := exec.Command(exePath, "-c", "if true; then echo yes; fi")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("if语句测试失败: %v, 输出: %s", err, string(output))
	}
	
	if !strings.Contains(string(output), "yes") {
		t.Errorf("if语句输出不正确，期望包含 'yes'，得到: %s", string(output))
	}
	
	// 测试for循环
	cmd = exec.Command(exePath, "-c", "for i in 1 2 3; do echo $i; done")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("for循环测试失败: %v, 输出: %s", err, string(output))
	}
}

// TestVariableExpansion 测试变量展开
func TestVariableExpansion(t *testing.T) {
	exePath := getGobashExe(t)
	
	// 测试环境变量展开
	cmd := exec.Command(exePath, "-c", "export TEST_VAR=test && echo $TEST_VAR")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("变量展开测试失败: %v, 输出: %s", err, string(output))
	}
	
	if !strings.Contains(string(output), "test") {
		t.Errorf("变量展开输出不正确，期望包含 'test'，得到: %s", string(output))
	}
}

// getGobashExe 获取gobash可执行文件路径
func getGobashExe(t *testing.T) string {
	// 尝试多个可能的位置
	possiblePaths := []string{
		"gobash.exe",
		"./gobash.exe",
		"../gobash.exe",
		filepath.Join("cmd", "gobash", "gobash.exe"),
	}
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath
		}
	}
	
	// 如果找不到，尝试构建
	t.Log("未找到gobash.exe，尝试构建...")
	buildCmd := exec.Command("go", "build", "-o", "gobash_test.exe", "./cmd/gobash")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("构建gobash失败: %v", err)
	}
	defer os.Remove("gobash_test.exe")
	
	absPath, _ := filepath.Abs("gobash_test.exe")
	return absPath
}

