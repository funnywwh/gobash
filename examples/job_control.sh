#!/usr/bin/env gobash
# gobash 作业控制功能示例

echo "=== gobash 作业控制功能演示 ==="

echo ""
echo "1. 后台任务执行："
echo "启动一个后台任务（sleep 2 &）："
sleep 2 &
echo "任务已在后台运行"

echo ""
echo "2. 查看作业列表："
jobs

echo ""
echo "3. 启动多个后台任务："
echo "启动任务1（sleep 1 &）："
sleep 1 &
echo "启动任务2（sleep 1 &）："
sleep 1 &
echo "启动任务3（sleep 2 &）："
sleep 2 &

echo ""
echo "4. 再次查看作业列表："
jobs

echo ""
echo "5. 将后台任务转到前台："
echo "注意：在实际使用中，可以使用 fg [作业ID] 将任务转到前台"
echo "例如：fg 1 或 fg %1"

echo ""
echo "6. 继续后台任务："
echo "注意：在实际使用中，可以使用 bg [作业ID] 继续后台任务"
echo "例如：bg 1 或 bg %1"

echo ""
echo "=== 演示完成 ==="
echo "提示：在交互式shell中，可以使用以下命令："
echo "  - 后台执行：command &"
echo "  - 查看作业：jobs"
echo "  - 转到前台：fg [作业ID]"
echo "  - 继续后台：bg [作业ID]"

