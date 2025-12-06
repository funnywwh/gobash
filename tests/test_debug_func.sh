#!/bin/bash
echo "1"

test() {
    echo "2"
    if [ 1 ]; then
        echo "3"
    fi
    echo "4"
}

echo "5"
test
echo "6"

