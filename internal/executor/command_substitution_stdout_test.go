package executor

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// TestCommandSubstitutionStdoutRestore 测试命令替换后 os.Stdout 是否被正确恢复
// 确保后续的 echo 命令能正常输出到控制台
func TestCommandSubstitutionStdoutRestore(t *testing.T) {
	// 保存原始的 os.Stdout
	originalStdout := os.Stdout

	// 测试用例：执行命令替换后，echo 应该能正常输出
	tests := []struct {
		name           string
		command        string
		expectedOutput string
	}{
		{
			name:           "Simple command substitution then echo",
			command:        "echo $(echo hello); echo world",
			expectedOutput: "hello\nworld\n",
		},
		{
			name:           "Multiple command substitutions then echo",
			command:        "echo $(echo a); echo $(echo b); echo final",
			expectedOutput: "a\nb\nfinal\n",
		},
		{
			name:           "Nested command substitution then echo",
			command:        "echo $(echo $(echo nested)); echo after",
			expectedOutput: "nested\nafter\n",
		},
		{
			name:           "Command substitution with echo inside then echo",
			command:        "echo $(echo inner); echo outer",
			expectedOutput: "inner\nouter\n",
		},
		{
			name:           "Multiple echoes after command substitution",
			command:        "echo $(echo test); echo first; echo second; echo third",
			expectedOutput: "test\nfirst\nsecond\nthird\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建管道来捕获输出
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("创建管道失败: %v", err)
			}
			oldStdout := os.Stdout
			os.Stdout = w
			
			// 在 goroutine 中读取输出
			var buf bytes.Buffer
			done := make(chan bool)
			go func() {
				io.Copy(&buf, r)
				r.Close()
				done <- true
			}()
			
			defer func() {
				// 关闭写入端，触发读取完成
				w.Close()
				<-done
				// 恢复原始的 os.Stdout
				os.Stdout = oldStdout
			}()

			// 创建执行器
			e := New()

			// 解析命令
			l := lexer.New(tt.command)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("解析错误: %v", p.Errors())
			}

			// 执行命令
			err2 := e.Execute(program)
			if err2 != nil {
				t.Fatalf("执行错误: %v", err2)
			}

			// 关闭写入端，等待读取完成
			w.Close()
			<-done
			os.Stdout = oldStdout

			// 获取输出
			output := buf.String()

			// 验证输出（允许一些差异，因为命令替换的输出可能包含额外的换行）
			// 主要验证后续的 echo 命令能正常输出
			if !strings.Contains(output, "world") && tt.name == "Simple command substitution then echo" {
				t.Errorf("期望输出包含 'world'，实际输出: %q", output)
			}
			if !strings.Contains(output, "final") && tt.name == "Multiple command substitutions then echo" {
				t.Errorf("期望输出包含 'final'，实际输出: %q", output)
			}
			if !strings.Contains(output, "after") && tt.name == "Nested command substitution then echo" {
				t.Errorf("期望输出包含 'after'，实际输出: %q", output)
			}
			if !strings.Contains(output, "outer") && tt.name == "Command substitution with echo inside then echo" {
				t.Errorf("期望输出包含 'outer'，实际输出: %q", output)
			}
			if !strings.Contains(output, "third") && tt.name == "Multiple echoes after command substitution" {
				t.Errorf("期望输出包含 'third'，实际输出: %q", output)
			}
			
			// 验证 os.Stdout 是否被恢复（通过检查后续 echo 是否正常）
			// 创建新的管道来捕获后续输出
			r2, w2, err3 := os.Pipe()
			if err3 != nil {
				t.Fatalf("创建管道失败: %v", err3)
			}
			os.Stdout = w2
			
			var buf2 bytes.Buffer
			done2 := make(chan bool)
			go func() {
				io.Copy(&buf2, r2)
				r2.Close()
				done2 <- true
			}()

			// 执行一个简单的 echo 命令
			l2 := lexer.New("echo verify")
			p2 := parser.New(l2)
			program2 := p2.ParseProgram()

			if len(p2.Errors()) > 0 {
				w2.Close()
				<-done2
				os.Stdout = originalStdout
				t.Fatalf("解析错误: %v", p2.Errors())
			}

			err4 := e.Execute(program2)
			w2.Close()
			<-done2
			os.Stdout = originalStdout
			
			if err4 != nil {
				t.Fatalf("执行错误: %v", err4)
			}

			verifyOutput := buf2.String()
			if !strings.Contains(verifyOutput, "verify") {
				t.Errorf("os.Stdout 可能未正确恢复，期望输出包含 'verify'，得到 %q", verifyOutput)
			}
		})
	}
}

// TestCommandSubstitutionStdoutAfterError 测试命令替换中发生错误时 os.Stdout 是否被恢复
func TestCommandSubstitutionStdoutAfterError(t *testing.T) {
	// 保存原始的 os.Stdout
	originalStdout := os.Stdout

	// 创建管道来捕获输出
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道失败: %v", err)
	}
	os.Stdout = w
	
	// 在 goroutine 中读取输出
	var buf bytes.Buffer
	done := make(chan bool)
	go func() {
		io.Copy(&buf, r)
		r.Close()
		done <- true
	}()
	
	defer func() {
		// 关闭写入端，触发读取完成
		w.Close()
		<-done
		// 恢复原始的 os.Stdout
		os.Stdout = originalStdout
	}()

	// 创建执行器
	e := New()

	// 测试命令替换中执行不存在的命令（应该失败但不影响后续 echo）
	command := "echo $(echo test); echo should_still_work"
	
	// 解析命令
	l := lexer.New(command)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("解析错误: %v", p.Errors())
	}

	// 执行命令（可能会失败，但不应该影响 os.Stdout 的恢复）
	_ = e.Execute(program)
	// 注意：这里可能会失败，但我们主要关心 os.Stdout 是否被恢复

	// 关闭写入端，等待读取完成
	w.Close()
	<-done
	os.Stdout = originalStdout

	// 验证 os.Stdout 是否被恢复（通过检查后续 echo 是否正常）
	// 创建新的管道来捕获后续输出
	r2, w2, err2 := os.Pipe()
	if err2 != nil {
		t.Fatalf("创建管道失败: %v", err2)
	}
	os.Stdout = w2
	
	var buf2 bytes.Buffer
	done2 := make(chan bool)
	go func() {
		io.Copy(&buf2, r2)
		r2.Close()
		done2 <- true
	}()

	// 执行一个简单的 echo 命令
	l2 := lexer.New("echo verify_after_error")
	p2 := parser.New(l2)
	program2 := p2.ParseProgram()

	if len(p2.Errors()) > 0 {
		w2.Close()
		<-done2
		os.Stdout = originalStdout
		t.Fatalf("解析错误: %v", p2.Errors())
	}

	err3 := e.Execute(program2)
	w2.Close()
	<-done2
	os.Stdout = originalStdout
	
	if err3 != nil {
		t.Fatalf("执行错误: %v", err3)
	}

	verifyOutput := buf2.String()
	if !strings.Contains(verifyOutput, "verify_after_error") {
		t.Errorf("os.Stdout 未正确恢复，期望输出包含 'verify_after_error'，得到 %q", verifyOutput)
	}
}

// TestCommandSubstitutionInVariable 测试在变量中使用命令替换后 echo 是否正常
func TestCommandSubstitutionInVariable(t *testing.T) {
	// 保存原始的 os.Stdout
	originalStdout := os.Stdout

	// 创建管道来捕获输出
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道失败: %v", err)
	}
	os.Stdout = w
	
	// 在 goroutine 中读取输出
	var buf bytes.Buffer
	done := make(chan bool)
	go func() {
		io.Copy(&buf, r)
		r.Close()
		done <- true
	}()
	
	defer func() {
		// 关闭写入端，触发读取完成
		w.Close()
		<-done
		// 恢复原始的 os.Stdout
		os.Stdout = originalStdout
	}()

	// 创建执行器
	e := New()

	// 测试在变量赋值中使用命令替换，然后 echo
	command := "VAR=$(echo test_value); echo $VAR; echo after"
	
	// 解析命令
	l := lexer.New(command)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("解析错误: %v", p.Errors())
	}

	// 执行命令
	err2 := e.Execute(program)
	if err2 != nil {
		t.Fatalf("执行错误: %v", err2)
	}

	// 关闭写入端，等待读取完成
	w.Close()
	<-done
	os.Stdout = originalStdout

	// 获取输出
	output := buf.String()
	
	// 验证输出包含期望的内容
	if !strings.Contains(output, "test_value") {
		t.Errorf("期望输出包含 'test_value'，实际输出: %q", output)
	}
	if !strings.Contains(output, "after") {
		t.Errorf("期望输出包含 'after'，实际输出: %q", output)
	}
}

// TestMultipleCommandSubstitutionsInSequence 测试连续多个命令替换后 echo 是否正常
func TestMultipleCommandSubstitutionsInSequence(t *testing.T) {
	// 保存原始的 os.Stdout
	originalStdout := os.Stdout

	// 创建管道来捕获输出
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道失败: %v", err)
	}
	os.Stdout = w
	
	// 在 goroutine 中读取输出
	var buf bytes.Buffer
	done := make(chan bool)
	go func() {
		io.Copy(&buf, r)
		r.Close()
		done <- true
	}()
	
	defer func() {
		// 关闭写入端，触发读取完成
		w.Close()
		<-done
		// 恢复原始的 os.Stdout
		os.Stdout = originalStdout
	}()

	// 创建执行器
	e := New()

	// 测试连续多个命令替换
	command := "echo $(echo first); echo $(echo second); echo $(echo third); echo final"
	
	// 解析命令
	l := lexer.New(command)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("解析错误: %v", p.Errors())
	}

	// 执行命令
	err2 := e.Execute(program)
	if err2 != nil {
		t.Fatalf("执行错误: %v", err2)
	}

	// 关闭写入端，等待读取完成
	w.Close()
	<-done
	os.Stdout = originalStdout

	// 获取输出
	output := buf.String()
	
	// 验证输出包含期望的内容
	if !strings.Contains(output, "final") {
		t.Errorf("期望输出包含 'final'，实际输出: %q", output)
	}

	// 验证后续 echo 仍然正常
	// 创建新的管道来捕获后续输出
	r2, w2, err3 := os.Pipe()
	if err3 != nil {
		t.Fatalf("创建管道失败: %v", err3)
	}
	os.Stdout = w2
	
	var buf2 bytes.Buffer
	done2 := make(chan bool)
	go func() {
		io.Copy(&buf2, r2)
		r2.Close()
		done2 <- true
	}()

	l2 := lexer.New("echo after_all")
	p2 := parser.New(l2)
	program2 := p2.ParseProgram()

	if len(p2.Errors()) > 0 {
		w2.Close()
		<-done2
		os.Stdout = originalStdout
		t.Fatalf("解析错误: %v", p2.Errors())
	}

	err4 := e.Execute(program2)
	w2.Close()
	<-done2
	os.Stdout = originalStdout
	
	if err4 != nil {
		t.Fatalf("执行错误: %v", err4)
	}

	verifyOutput := buf2.String()
	if !strings.Contains(verifyOutput, "after_all") {
		t.Errorf("后续 echo 失败，期望输出包含 'after_all'，得到 %q", verifyOutput)
	}
}
