package executor

import (
	"os"
	"strings"
	"testing"
	"gobash/internal/lexer"
	"gobash/internal/parser"
)

// TestArithmeticFunctions 测试算术函数
func TestArithmeticFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "abs正数",
			input:    "echo $((abs(5)))",
			expected: "5",
			wantErr:  false,
		},
		{
			name:     "abs负数",
			input:    "echo $((abs(-5)))",
			expected: "5",
			wantErr:  false,
		},
		{
			name:     "abs零",
			input:    "echo $((abs(0)))",
			expected: "0",
			wantErr:  false,
		},
		{
			name:     "min两个参数",
			input:    "echo $((min(5, 3)))",
			expected: "3",
			wantErr:  false,
		},
		{
			name:     "min多个参数",
			input:    "echo $((min(5, 3, 7, 2)))",
			expected: "2",
			wantErr:  false,
		},
		{
			name:     "min负数",
			input:    "echo $((min(-5, -3)))",
			expected: "-5",
			wantErr:  false,
		},
		{
			name:     "max两个参数",
			input:    "echo $((max(5, 3)))",
			expected: "5",
			wantErr:  false,
		},
		{
			name:     "max多个参数",
			input:    "echo $((max(5, 3, 7, 2)))",
			expected: "7",
			wantErr:  false,
		},
		{
			name:     "max负数",
			input:    "echo $((max(-5, -3)))",
			expected: "-3",
			wantErr:  false,
		},
		{
			name:     "length正数",
			input:    "echo $((length(12345)))",
			expected: "5",
			wantErr:  false,
		},
		{
			name:     "length负数",
			input:    "echo $((length(-123)))",
			expected: "3",
			wantErr:  false,
		},
		{
			name:     "length零",
			input:    "echo $((length(0)))",
			expected: "1",
			wantErr:  false,
		},
		{
			name:     "int正数",
			input:    "echo $((int(123)))",
			expected: "123",
			wantErr:  false,
		},
		{
			name:     "int负数",
			input:    "echo $((int(-123)))",
			expected: "-123",
			wantErr:  false,
		},
		{
			name:     "rand无参数",
			input:    "echo $((rand()))",
			expected: "", // rand 返回随机数，只检查不报错
			wantErr:  false,
		},
		{
			name:     "srand设置种子",
			input:    "echo $((srand(123)))",
			expected: "0", // srand 返回 0（设置种子，不返回种子值）
			wantErr:  false,
		},
		{
			name:     "srand无参数",
			input:    "echo $((srand()))",
			expected: "", // srand 无参数时返回随机种子，只检查不报错
			wantErr:  false,
		},
		{
			name:     "嵌套函数调用",
			input:    "echo $((min(abs(-5), abs(3))))",
			expected: "3",
			wantErr:  false,
		},
		{
			name:     "函数调用与运算符",
			input:    "echo $((abs(-5) + min(3, 2)))",
			expected: "7",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				if tt.wantErr {
					t.Logf("期望的解析错误: %v", p.Errors())
					return
				}
				t.Fatalf("解析错误: %v", p.Errors())
			}

			// 捕获输出
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := e.Execute(program)

			w.Close()
			os.Stdout = oldStdout

			if err != nil {
				if tt.wantErr {
					t.Logf("期望的执行错误: %v", err)
					return
				}
				t.Fatalf("执行错误: %v", err)
			}

			// 读取输出
			var output strings.Builder
			buf := make([]byte, 1024)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					output.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
			r.Close()

			result := strings.TrimSpace(output.String())

			// 对于 rand 和 srand 无参数的情况，只检查不报错
			if tt.expected == "" {
				if result == "" {
					t.Logf("✓ 函数执行成功（无输出或随机输出）")
				} else {
					t.Logf("✓ 函数执行成功，输出: %s", result)
				}
				return
			}

			if result != tt.expected {
				t.Errorf("输出不匹配。\n期望: %q\n得到: %q", tt.expected, result)
			} else {
				t.Logf("✓ 输出匹配: %q", result)
			}
		})
	}
}

// TestArithmeticFunctionErrors 测试算术函数错误处理
func TestArithmeticFunctionErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "abs参数过多",
			input:   "echo $((abs(1, 2)))",
			wantErr: true,
		},
		{
			name:    "min参数不足",
			input:   "echo $((min()))",
			wantErr: true,
		},
		{
			name:    "max参数不足",
			input:   "echo $((max()))",
			wantErr: true,
		},
		{
			name:    "length参数过多",
			input:   "echo $((length(1, 2)))",
			wantErr: true,
		},
		{
			name:    "int参数过多",
			input:   "echo $((int(1, 2)))",
			wantErr: true,
		},
		{
			name:    "rand参数过多",
			input:   "echo $((rand(1)))",
			wantErr: true,
		},
		{
			name:    "未知函数",
			input:   "echo $((unknown(1)))",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New()
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				if tt.wantErr {
					t.Logf("✓ 期望的解析错误: %v", p.Errors())
					return
				}
				t.Fatalf("解析错误: %v", p.Errors())
			}

			err := e.Execute(program)

			if err != nil {
				if tt.wantErr {
					t.Logf("✓ 期望的执行错误: %v", err)
					return
				}
				t.Fatalf("执行错误: %v", err)
			}

			if !tt.wantErr {
				t.Logf("✓ 执行成功")
			} else {
				t.Errorf("期望有错误，但执行成功")
			}
		})
	}
}

