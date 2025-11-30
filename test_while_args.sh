#!/bin/bash
echo "参数个数: $#"
while [[ $# -gt 0 ]]; do
    echo "循环中: $1"
    shift
done
echo "循环结束"

