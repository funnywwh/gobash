#!/bin/bash
set -- a
echo "初始: $#, \$1=$1"
shift
echo "shift后: $#, \$1=$1"
[[ $# -gt 0 ]] && echo "条件为真" || echo "条件为假"

