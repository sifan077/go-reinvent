package timeutil

import (
	"strings"
	"time"
)

// 将自定义 layout 转换为 Go 的 layout
// 支持: YYYY -> 2006, MM -> 01, DD -> 02, HH -> 15, mm -> 04, ss -> 05
var layoutReplacer = strings.NewReplacer(
	"YYYY", "2006",
	"MM", "01",
	"DD", "02",
	"HH", "15",
	"mm", "04",
	"ss", "05",
)

// convertLayout 将自定义 layout 转换为 Go 标准 layout
func convertLayout(layout string) string {
	// 如果是 Go 标准 layout 直接返回（通过检测是否包含 "2006" 判断）
	if strings.Contains(layout, "2006") {
		return layout
	}
	return layoutReplacer.Replace(layout)
}

// Format 格式化时间，支持自定义 layout（YYYY-MM-DD HH:mm:ss）
func Format(t time.Time, layout string) string {
	return t.Format(convertLayout(layout))
}

// Parse 解析时间字符串，支持自定义 layout
func Parse(s string, layout string) (time.Time, error) {
	return time.ParseInLocation(convertLayout(layout), s, time.Local)
}

// StartOfDay 当天零点
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay 当天最后一刻（23:59:59.999999999）
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

// StartOfMonth 当月第一天零点
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth 当月最后一天最后一刻
func EndOfMonth(t time.Time) time.Time {
	return StartOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}
