#!/bin/bash
set -- a
echo "初始: \$#=$#, \$1=$1"
case $1 in
    a)
        echo "匹配 a"
        shift
        echo "shift后: \$#=$#, \$1=$1"
        ;;
    *)
        echo "匹配 *"
        ;;
esac
echo "case结束: \$#=$#, \$1=$1"

