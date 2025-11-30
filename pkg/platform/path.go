package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// NormalizePath 规范化路径，处理Windows和Unix路径差异
func NormalizePath(path string) string {
	// 将反斜杠转换为正斜杠（Windows兼容）
	path = strings.ReplaceAll(path, "\\", "/")
	
	// 处理 ~ 展开
	if strings.HasPrefix(path, "~") {
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		if home != "" {
			path = strings.Replace(path, "~", home, 1)
		}
	}
	
	return filepath.Clean(path)
}

// IsAbsolute 判断是否为绝对路径
func IsAbsolute(path string) bool {
	return filepath.IsAbs(path)
}

// JoinPath 连接路径
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

