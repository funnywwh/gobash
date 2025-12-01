#!/bin/bash
set -- a b
echo "初始: \$#=$#"
while [[ $# -gt 0 ]]; do
    echo "循环中: \$#=$#, \$1=$1"
    shift
    echo "shift后: \$#=$#"
done
echo "循环结束"

