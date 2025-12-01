#!/bin/bash
echo "测试开始"
i=0
while [[ $i -lt 5 ]]; do
    echo "循环中: $i"
    i=$((i+1))
    if [[ $i -eq 2 ]]; then
        break
    fi
done
echo "测试结束"


