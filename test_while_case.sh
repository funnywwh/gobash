#!/bin/bash
while [[ $# -gt 0 ]]; do
    echo "循环中: $1"
    case $1 in
        --win)
            echo "找到--win参数"
            shift
            ;;
        *)
            echo "未知参数: $1"
            shift
            ;;
    esac
done
echo "循环结束"

