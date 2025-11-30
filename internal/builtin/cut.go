package builtin

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// cut 剪切字段（简化版）
func cut(args []string, env map[string]string) error {
	delimiter := "\t" // 默认制表符
	fields := ""
	files := []string{}
	
	// 解析参数
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// 解析选项
			if strings.HasPrefix(arg, "-d") {
				// -d 分隔符
				if len(arg) > 2 {
					delimiter = arg[2:]
				} else if i+1 < len(args) {
					delimiter = args[i+1]
					i++
				}
			} else if strings.HasPrefix(arg, "-f") {
				// -f 字段列表
				if len(arg) > 2 {
					fields = arg[2:]
				} else if i+1 < len(args) {
					fields = args[i+1]
					i++
				}
			}
		} else {
			files = append(files, arg)
		}
		i++
	}
	
	if fields == "" {
		return fmt.Errorf("cut: 必须指定字段列表 (-f)")
	}
	
	// 解析字段列表（支持 1,2,3 或 1-3 或 1-3,5 等格式）
	fieldList, err := parseFieldList(fields)
	if err != nil {
		return fmt.Errorf("cut: %v", err)
	}
	
	// 如果没有指定文件，从stdin读取
	if len(files) == 0 {
		return cutFromStdin(delimiter, fieldList)
	}
	
	// 处理多个文件
	for _, file := range files {
		if err := cutFromFile(file, delimiter, fieldList); err != nil {
			return err
		}
	}
	
	return nil
}

// parseFieldList 解析字段列表
func parseFieldList(fields string) ([]int, error) {
	var result []int
	parts := strings.Split(fields, ",")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			// 范围格式，如 1-3
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("无效的字段范围: %s", part)
			}
			start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("无效的字段范围: %s", part)
			}
			if start > end {
				return nil, fmt.Errorf("字段范围起始值不能大于结束值: %s", part)
			}
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
		} else {
			// 单个字段
			field, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("无效的字段号: %s", part)
			}
			result = append(result, field)
		}
	}
	
	// 去重并排序
	seen := make(map[int]bool)
	uniqueFields := []int{}
	for _, f := range result {
		if !seen[f] {
			seen[f] = true
			uniqueFields = append(uniqueFields, f)
		}
	}
	sort.Ints(uniqueFields)
	
	return uniqueFields, nil
}

// cutFromFile 从文件剪切
func cutFromFile(filename string, delimiter string, fieldList []int) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cut: %v", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		output := cutLine(line, delimiter, fieldList)
		fmt.Println(output)
	}
	
	return scanner.Err()
}

// cutFromStdin 从stdin剪切
func cutFromStdin(delimiter string, fieldList []int) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		output := cutLine(line, delimiter, fieldList)
		fmt.Println(output)
	}
	
	return scanner.Err()
}

// cutLine 剪切一行
func cutLine(line string, delimiter string, fieldList []int) string {
	parts := strings.Split(line, delimiter)
	var result []string
	
	for _, fieldNum := range fieldList {
		// 字段编号从1开始
		index := fieldNum - 1
		if index >= 0 && index < len(parts) {
			result = append(result, parts[index])
		}
	}
	
	return strings.Join(result, delimiter)
}

