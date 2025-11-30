package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// History 命令历史管理器
type History struct {
	commands []string
	maxSize  int
	index    int // 当前浏览位置
}

// NewHistory 创建新的历史管理器
func NewHistory(maxSize int) *History {
	return &History{
		commands: make([]string, 0, maxSize),
		maxSize:  maxSize,
		index:    0,
	}
}

// Add 添加命令到历史
func (h *History) Add(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	
	// 避免重复添加相同的命令
	if len(h.commands) > 0 && h.commands[len(h.commands)-1] == cmd {
		return
	}

	h.commands = append(h.commands, cmd)
	if len(h.commands) > h.maxSize {
		h.commands = h.commands[1:]
	}
	h.index = len(h.commands)
}

// Get 获取指定索引的历史命令
func (h *History) Get(index int) string {
	if index < 0 || index >= len(h.commands) {
		return ""
	}
	return h.commands[index]
}

// GetAll 获取所有历史命令
func (h *History) GetAll() []string {
	return h.commands
}

// Size 获取历史记录数量
func (h *History) Size() int {
	return len(h.commands)
}

// Prev 获取上一条命令
func (h *History) Prev() string {
	if h.index > 0 {
		h.index--
		return h.commands[h.index]
	}
	return ""
}

// Next 获取下一条命令
func (h *History) Next() string {
	if h.index < len(h.commands)-1 {
		h.index++
		return h.commands[h.index]
	}
	h.index = len(h.commands)
	return ""
}

// Reset 重置浏览位置
func (h *History) Reset() {
	h.index = len(h.commands)
}

// LoadFromFile 从文件加载历史记录
func (h *History) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在不算错误
		}
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			h.commands = append(h.commands, line)
			if len(h.commands) >= h.maxSize {
				break
			}
		}
	}
	h.index = len(h.commands)
	return nil
}

// SaveToFile 保存历史记录到文件
func (h *History) SaveToFile(filename string) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	content := strings.Join(h.commands, "\n")
	return os.WriteFile(filename, []byte(content), 0644)
}

// Print 打印历史记录
func (h *History) Print() {
	for i, cmd := range h.commands {
		fmt.Printf("%5d  %s\n", i+1, cmd)
	}
}

