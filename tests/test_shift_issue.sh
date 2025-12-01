#!/bin/bash
set -- a
echo "初始: \$#=$#, \$1=$1"
shift
echo "shift后: \$#=$#, \$1=$1"
if [[ $# -gt 0 ]]; then
    echo "条件为真，\$#=$#"
else
    echo "条件为假，\$#=$#"
fi

