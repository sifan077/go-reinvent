package timeutil

import (
	"fmt"
	"time"
)

// FriendlyTime 将时间转换为友好显示
// 刚刚（1分钟内）、X分钟前、X小时前、昨天 HH:mm、前天 HH:mm、更早显示日期
func FriendlyTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		return Format(t, "YYYY-MM-DD HH:mm:ss")
	}

	switch {
	case diff < time.Minute:
		return "刚刚"
	case diff < time.Hour:
		return fmt.Sprintf("%d分钟前", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d小时前", int(diff.Hours()))
	case diff < 48*time.Hour:
		return "昨天 " + Format(t, "HH:mm")
	case diff < 72*time.Hour:
		return "前天 " + Format(t, "HH:mm")
	default:
		return Format(t, "YYYY-MM-DD HH:mm")
	}
}

// Duration 将时间间隔转换为友好中文显示
// 例如: 1小时30分钟5秒
func Duration(d time.Duration) string {
	if d < time.Second {
		return "0秒"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	var result string
	if hours > 0 {
		result += fmt.Sprintf("%d小时", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%d分钟", minutes)
	}
	if seconds > 0 {
		result += fmt.Sprintf("%d秒", seconds)
	}
	return result
}
