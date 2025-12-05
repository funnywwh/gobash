package executor

import (
	"bytes"
	"io"
	"os"
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// TestNestedIfExecution 测试嵌套if语句的执行
func TestNestedIfExecution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "简单嵌套if - 外层true内层true",
			input:    "if [ 1 ]; then if [ 1 ]; then echo nested-true; fi; fi",
			expected: "nested-true\n",
			wantErr:  false,
		},
		{
			name:     "简单嵌套if - 外层true内层false",
			input:    "if [ 1 ]; then if [ 0 ]; then echo nested-true; fi; fi",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "简单嵌套if - 外层false",
			input:    "if [ 0 ]; then if [ 1 ]; then echo nested-true; fi; fi",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "嵌套if with else - 内层then",
			input:    "if [ 1 ]; then if [ 1 ]; then echo nested-then; else echo nested-else; fi; fi",
			expected: "nested-then\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if with else - 内层else",
			input:    "if [ 1 ]; then if [ 0 ]; then echo nested-then; else echo nested-else; fi; fi",
			expected: "nested-else\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if with elif - 内层then",
			input:    "if [ 1 ]; then if [ 1 ]; then echo nested-then; elif [ 1 ]; then echo nested-elif; fi; fi",
			expected: "nested-then\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if with elif - 内层elif",
			input:    "if [ 1 ]; then if [ 0 ]; then echo nested-then; elif [ 1 ]; then echo nested-elif; fi; fi",
			expected: "nested-elif\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if - else块中",
			input:    "if [ 0 ]; then echo then; else if [ 1 ]; then echo nested-then; fi; fi",
			expected: "nested-then\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if - elif块中",
			input:    "if [ 0 ]; then echo then; elif [ 1 ]; then if [ 1 ]; then echo nested-then; fi; fi",
			expected: "nested-then\n",
			wantErr:  false,
		},
		{
			name:     "三层嵌套if - 全部true",
			input:    "if [ 1 ]; then if [ 1 ]; then if [ 1 ]; then echo triple-nested; fi; fi; fi",
			expected: "triple-nested\n",
			wantErr:  false,
		},
		{
			name:     "三层嵌套if - 第二层false",
			input:    "if [ 1 ]; then if [ 0 ]; then if [ 1 ]; then echo triple-nested; fi; fi; fi",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "嵌套if with else和elif - then分支",
			input:    "if [ 1 ]; then if [ 1 ]; then echo nested-then; elif [ 1 ]; then echo nested-elif; else echo nested-else; fi; fi",
			expected: "nested-then\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if with else和elif - elif分支",
			input:    "if [ 1 ]; then if [ 0 ]; then echo nested-then; elif [ 1 ]; then echo nested-elif; else echo nested-else; fi; fi",
			expected: "nested-elif\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if with else和elif - else分支",
			input:    "if [ 1 ]; then if [ 0 ]; then echo nested-then; elif [ 0 ]; then echo nested-elif; else echo nested-else; fi; fi",
			expected: "nested-else\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()

			// 捕获输出
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				w.Close()
				os.Stdout = oldStdout
				t.Fatalf("解析错误: %v", p.Errors())
			}

			err := e.Execute(program)
			hasErr := err != nil

			w.Close()
			os.Stdout = oldStdout

			var output string
			if r != nil {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				output = buf.String()
				r.Close()
			}

			if hasErr != tt.wantErr {
				t.Errorf("错误状态不匹配，期望错误: %v, 得到错误: %v, 错误: %v", tt.wantErr, hasErr, err)
			}

			if output != tt.expected {
				t.Errorf("输出不匹配，期望: %q, 得到: %q", tt.expected, output)
			}
		})
	}
}

// TestNestedIfWithVariables 测试嵌套if语句中的变量使用
func TestNestedIfWithVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "嵌套if中使用变量",
			input:    "x=1; if [ $x ]; then if [ $x ]; then echo nested-$x; fi; fi",
			expected: "nested-1\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if中修改变量",
			input:    "x=1; if [ $x ]; then x=2; if [ $x ]; then echo nested-$x; fi; fi",
			expected: "nested-2\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if中算术展开",
			input:    "x=1; if [ $x ]; then if [ $((x+1)) ]; then echo nested-$((x+1)); fi; fi",
			expected: "nested-2\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()

			// 捕获输出
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				w.Close()
				os.Stdout = oldStdout
				t.Fatalf("解析错误: %v", p.Errors())
			}

			err := e.Execute(program)
			hasErr := err != nil

			w.Close()
			os.Stdout = oldStdout

			var output string
			if r != nil {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				output = buf.String()
				r.Close()
			}

			if hasErr != tt.wantErr {
				t.Errorf("错误状态不匹配，期望错误: %v, 得到错误: %v, 错误: %v", tt.wantErr, hasErr, err)
			}

			if output != tt.expected {
				t.Errorf("输出不匹配，期望: %q, 得到: %q", tt.expected, output)
			}
		})
	}
}

// TestNestedIfWithCommands 测试嵌套if语句中的命令执行
func TestNestedIfWithCommands(t *testing.T) {
	// 创建一个临时文件用于测试
	tmpFile := "/tmp/gobash_test_nested_if"
	os.WriteFile(tmpFile, []byte("test"), 0644)
	defer os.Remove(tmpFile)

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "嵌套if中使用test命令",
			input:    "if [ -f " + tmpFile + " ]; then if [ -r " + tmpFile + " ]; then echo nested-readable; fi; fi",
			expected: "nested-readable\n",
			wantErr:  false,
		},
		{
			name:     "嵌套if中使用[[命令",
			input:    "if [[ -f " + tmpFile + " ]]; then if [[ -r " + tmpFile + " ]]; then echo nested-readable; fi; fi",
			expected: "nested-readable\n",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()

			// 捕获输出
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				w.Close()
				os.Stdout = oldStdout
				t.Fatalf("解析错误: %v", p.Errors())
			}

			err := e.Execute(program)
			hasErr := err != nil

			w.Close()
			os.Stdout = oldStdout

			var output string
			if r != nil {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				output = buf.String()
				r.Close()
			}

			if hasErr != tt.wantErr {
				t.Errorf("错误状态不匹配，期望错误: %v, 得到错误: %v, 错误: %v", tt.wantErr, hasErr, err)
			}

			if output != tt.expected {
				t.Errorf("输出不匹配，期望: %q, 得到: %q", tt.expected, output)
			}
		})
