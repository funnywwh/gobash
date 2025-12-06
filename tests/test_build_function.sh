#!/bin/bash
echo "开始测试 build 函数"

build() {
    local platform=$1
    local output_name="gobash"
    local output_path="./$output_name"
    
    echo "build 函数开始: platform=$platform"
    echo "准备执行: go build -o $output_path ./cmd/gobash"
    
    go build -o "$output_path" ./cmd/gobash
    
    echo "go build 命令执行完成"
    
    if [[ -f "$output_path" ]]; then
        echo "构建成功: $output_path"
    else
        echo "构建失败: $output_path"
        return 1
    fi
}

echo "调用 build 函数"
build "linux"
echo "build 函数调用完成"




