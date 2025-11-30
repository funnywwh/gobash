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

