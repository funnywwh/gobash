#!/bin/bash
# 综合测试脚本 - 运行所有测试

echo "=========================================="
echo "gobash 脚本执行问题综合测试"
echo "=========================================="
echo ""

# 测试1: 算术展开
echo ">>> 测试1: 算术展开在变量赋值中"
echo "----------------------------------------"
../gobash test_arithmetic_assignment.sh 2>&1
echo ""

# 测试2: while 循环
echo ">>> 测试2: while 循环"
echo "----------------------------------------"
timeout 5 ../gobash test_while_loop.sh 2>&1 || echo "测试超时或失败 (退出码: $?)"
echo ""

# 测试3: 变量展开
echo ">>> 测试3: 变量展开"
echo "----------------------------------------"
../gobash test_variable_expansion.sh 2>&1
echo ""

# 测试4: case 语句
echo ">>> 测试4: case 语句"
echo "----------------------------------------"
timeout 5 ../gobash test_case_statement.sh 2>&1 || echo "测试超时或失败 (退出码: $?)"
echo ""

# 测试5: 模拟 build.sh
echo ">>> 测试5: 模拟 build.sh 脚本"
echo "----------------------------------------"
timeout 5 ../gobash test_build_script.sh 2>&1 || echo "测试超时或失败 (退出码: $?)"
echo ""

echo "=========================================="
echo "所有测试完成"
echo "=========================================="

