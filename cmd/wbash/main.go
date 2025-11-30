package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"gobash/internal/shell"
)

func main() {
	var scriptPath = flag.String("c", "", "执行命令字符串")
	var scriptFile = flag.String("f", "", "执行脚本文件")
	flag.Parse()

	sh := shell.New()

	// 执行命令字符串
	if *scriptPath != "" {
		if err := sh.ExecuteReader(strings.NewReader(*scriptPath)); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 执行脚本文件
	if *scriptFile != "" {
		if err := sh.ExecuteScript(*scriptFile); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 如果有命令行参数，作为脚本执行
	if len(os.Args) > 1 && os.Args[1][0] != '-' {
		if err := sh.ExecuteScript(os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 交互式模式
	sh.Run()
}

