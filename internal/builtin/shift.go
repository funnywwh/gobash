package builtin

import (
	"fmt"
	"strconv"
	"strings"
)

// shift 移动位置参数
// shift [n] - 将位置参数向左移动 n 个位置（默认为 1）
// 例如：shift 2 会将 $3 变成 $1，$4 变成 $2，等等
func shift(args []string, env map[string]string) error {
	n := 1
	if len(args) > 0 {
		parsed, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("shift: %s: 需要数字参数", args[0])
		}
		if parsed < 0 {
			return fmt.Errorf("shift: %d: 参数必须是正数", parsed)
		}
		n = parsed
	}
	
	// 获取当前参数个数
	argCount := 0
	if countStr, ok := env["#"]; ok {
		if parsed, err := strconv.Atoi(countStr); err == nil {
			argCount = parsed
		}
	}
	
	if n > argCount {
		return fmt.Errorf("shift: %d: 不能移动超过参数个数 (%d)", n, argCount)
	}
	
	// 移动位置参数：将 $n+1 变成 $1，$n+2 变成 $2，等等
	for i := 1; i <= argCount-n; i++ {
		oldKey := fmt.Sprintf("%d", i+n)
		newKey := fmt.Sprintf("%d", i)
		if value, ok := env[oldKey]; ok {
			env[newKey] = value
		} else {
			delete(env, newKey)
		}
	}
	
	// 删除移动后的参数
	for i := argCount - n + 1; i <= argCount; i++ {
		delete(env, fmt.Sprintf("%d", i))
	}
	
	// 更新 $# 和 $@
	newCount := argCount - n
	env["#"] = fmt.Sprintf("%d", newCount)
	
	// 更新 $@
	var allArgs []string
	for i := 1; i <= newCount; i++ {
		if value, ok := env[fmt.Sprintf("%d", i)]; ok {
			allArgs = append(allArgs, value)
		}
	}
	env["@"] = strings.Join(allArgs, " ")
	
	return nil
}

