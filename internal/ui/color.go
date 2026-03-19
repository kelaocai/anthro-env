package ui

import "os"

const (
	colorReset     = "\033[0m"
	colorBoldGreen = "\033[1;32m"
)

// IsColorSupported 检测是否支持 ANSI 颜色
// NO_COLOR 存在或 TERM=dumb/空 时返回 false
func IsColorSupported() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}

// BoldGreen 返回绿色加粗文字（不支持时原样返回）
func BoldGreen(s string) string {
	if !IsColorSupported() {
		return s
	}
	return colorBoldGreen + s + colorReset
}
