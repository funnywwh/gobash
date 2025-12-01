#!/bin/bash
# 测试脚本

echo "=== 测试基础命令 ==="
echo "Hello, World!"
pwd

echo ""
echo "=== 测试环境变量 ==="
export TEST_VAR="test value"
echo "TEST_VAR=$TEST_VAR"

echo ""
echo "=== 测试重定向 ==="
echo "test content" > test_output.txt
echo "append content" >> test_output.txt
cat test_output.txt

echo ""
echo "=== 测试管道 ==="
echo "hello" | echo

