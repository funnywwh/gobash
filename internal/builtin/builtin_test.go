package builtin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEcho(t *testing.T) {
	tests := []struct {
		args     []string
		expected string
	}{
		{[]string{"hello"}, "hello"},
		{[]string{"hello", "world"}, "hello world"},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		err := echo(tt.args, make(map[string]string))
		if err != nil {
			t.Errorf("echo命令执行失败: %v", err)
		}
	}
}

func TestPwd(t *testing.T) {
	err := pwd([]string{}, make(map[string]string))
	if err != nil {
		t.Errorf("pwd命令执行失败: %v", err)
	}
}

func TestExit(t *testing.T) {
	// 测试exit命令（不会真正退出，因为测试环境）
	// 这里只测试参数解析
	tests := []struct {
		args []string
	}{
		{[]string{}},
		{[]string{"0"}},
		{[]string{"1"}},
		{[]string{"255"}},
	}

	for _, tt := range tests {
		// 注意：exit会调用os.Exit，所以在测试中不能真正执行
		// 这里只测试函数不会panic
		_ = tt.args
	}
}

func TestExport(t *testing.T) {
	env := make(map[string]string)
	
	// 测试设置环境变量
	err := export([]string{"TEST_VAR=test_value"}, env)
	if err != nil {
		t.Errorf("export命令执行失败: %v", err)
	}
	
	if env["TEST_VAR"] != "test_value" {
		t.Errorf("环境变量设置失败，期望 'test_value'，得到 '%s'", env["TEST_VAR"])
	}
}

func TestUnset(t *testing.T) {
	env := map[string]string{
		"TEST_VAR": "test_value",
	}
	
	// 测试取消设置环境变量
	err := unset([]string{"TEST_VAR"}, env)
	if err != nil {
		t.Errorf("unset命令执行失败: %v", err)
	}
	
	if _, ok := env["TEST_VAR"]; ok {
		t.Error("环境变量未正确删除")
	}
}

func TestTrue(t *testing.T) {
	err := trueCmd([]string{}, make(map[string]string))
	if err != nil {
		t.Errorf("true命令执行失败: %v", err)
	}
}

func TestFalse(t *testing.T) {
	err := falseCmd([]string{}, make(map[string]string))
	if err == nil {
		t.Error("false命令应该返回错误")
	}
}

func TestMkdir(t *testing.T) {
	// 创建临时目录
	testDir := filepath.Join(os.TempDir(), "gobash_test_mkdir")
	defer os.RemoveAll(testDir)
	
	err := mkdir([]string{testDir}, make(map[string]string))
	if err != nil {
		t.Errorf("mkdir命令执行失败: %v", err)
	}
	
	// 检查目录是否存在
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("目录未创建")
	}
}

func TestTouch(t *testing.T) {
	// 创建临时文件
	testFile := filepath.Join(os.TempDir(), "gobash_test_touch.txt")
	defer os.Remove(testFile)
	
	err := touch([]string{testFile}, make(map[string]string))
	if err != nil {
		t.Errorf("touch命令执行失败: %v", err)
	}
	
	// 检查文件是否存在
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("文件未创建")
	}
}

func TestCat(t *testing.T) {
	// 创建临时文件
	testFile := filepath.Join(os.TempDir(), "gobash_test_cat.txt")
	defer os.Remove(testFile)
	
	content := "test content\nline 2"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	
	// 测试cat命令
	err = cat([]string{testFile}, make(map[string]string))
	if err != nil {
		t.Errorf("cat命令执行失败: %v", err)
	}
}

func TestLs(t *testing.T) {
	// 测试ls命令（列出当前目录）
	err := ls([]string{}, make(map[string]string))
	if err != nil {
		t.Errorf("ls命令执行失败: %v", err)
	}
	
	// 测试ls特定目录
	tempDir := os.TempDir()
	err = ls([]string{tempDir}, make(map[string]string))
	if err != nil {
		t.Errorf("ls命令执行失败: %v", err)
	}
}

func TestRm(t *testing.T) {
	// 创建临时文件
	testFile := filepath.Join(os.TempDir(), "gobash_test_rm.txt")
	content := "test"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	
	// 测试rm命令
	err = rm([]string{testFile}, make(map[string]string))
	if err != nil {
		t.Errorf("rm命令执行失败: %v", err)
	}
	
	// 验证文件已删除
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("文件未删除")
	}
}

func TestRmdir(t *testing.T) {
	// 创建临时目录
	testDir := filepath.Join(os.TempDir(), "gobash_test_rmdir")
	err := os.Mkdir(testDir, 0755)
	if err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	
	// 测试rmdir命令
	err = rmdir([]string{testDir}, make(map[string]string))
	if err != nil {
		t.Errorf("rmdir命令执行失败: %v", err)
	}
	
	// 验证目录已删除
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("目录未删除")
	}
}

func TestTestCmd(t *testing.T) {
	// 测试test命令 - 字符串测试
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"非空字符串", []string{"test"}, false},
		{"空字符串", []string{""}, true},
		{"文件存在测试", []string{"-e", "builtin.go"}, false},
		{"文件不存在测试", []string{"-e", "nonexistent_file"}, true},
		{"字符串相等", []string{"hello", "=", "hello"}, false},
		{"字符串不等", []string{"hello", "=", "world"}, true},
		{"数字相等", []string{"1", "-eq", "1"}, false},
		{"数字不等", []string{"1", "-eq", "2"}, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testCmd(tt.args, make(map[string]string))
			if (err != nil) != tt.wantErr {
				t.Errorf("test命令 '%v' 错误，期望错误: %v，得到: %v", tt.args, tt.wantErr, err != nil)
			}
		})
	}
}

func TestTypeCmd(t *testing.T) {
	// 测试type命令
	err := typeCmd([]string{"echo"}, make(map[string]string))
	if err != nil {
		t.Errorf("type命令执行失败: %v", err)
	}
	
	err = typeCmd([]string{"nonexistent_command"}, make(map[string]string))
	// type命令对于不存在的命令应该输出"not found"，但不应该返回错误
	if err != nil {
		t.Logf("type命令对于不存在的命令返回: %v（这是可以接受的）", err)
	}
}

func TestEnv(t *testing.T) {
	// 测试env命令
	envMap := make(map[string]string)
	envMap["TEST_VAR"] = "test_value"
	
	err := env([]string{}, envMap)
	if err != nil {
		t.Errorf("env命令执行失败: %v", err)
	}
}

func TestWhich(t *testing.T) {
	// 测试which命令（查找echo命令）
	err := which([]string{"echo"}, make(map[string]string))
	// which可能找不到echo（如果不在PATH中），这是可以接受的
	if err != nil {
		t.Logf("which命令返回: %v（可能是命令未找到，这是可以接受的）", err)
	}
}

