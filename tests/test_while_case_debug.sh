#!/bin/bash
set -- --win
echo "初始: \$#=$#, \$1=$1"
while [[ $# -gt 0 ]]; do
    echo "循环开始: \$#=$#, \$1=$1"
    case $1 in
        --win)
            echo "找到--win参数"
            shift
            echo "shift后: \$#=$#"
            ;;
        *)
            echo "未知参数: $1"
            shift
            echo "shift后: \$#=$#"
            ;;
    esac
    echo "循环结束: \$#=$#"
done
echo "循环结束"

