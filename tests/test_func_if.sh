#!/bin/bash
echo "开始"

build() {
    local x=1
    if [[ $x -eq 1 ]]; then
        echo "x=1"
    fi
}

echo "函数定义完成"
build
echo "完成"



