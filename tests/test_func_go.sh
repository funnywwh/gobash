#!/bin/bash
echo "开始"

build() {
    echo "函数开始"
    go version
    echo "函数结束"
}

echo "函数定义完成"
build
echo "完成"





