#!/bin/bash
echo "第1行执行"

build() {
    echo "函数内部第1行"
    echo "函数内部第2行"
}

echo "第2行执行"
build
echo "第3行执行"





