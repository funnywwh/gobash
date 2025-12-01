#!/bin/bash
case a in
    a)
        echo "match a"
        ;;
        echo "这行不应该执行"
    *)
        echo "match *"
        ;;
esac

