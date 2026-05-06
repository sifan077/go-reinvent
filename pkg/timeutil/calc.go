package timeutil

import "time"

// DaysBetween 计算两个日期之间的天数差（取绝对值）
// 先归一化到当天零点，再计算差值
func DaysBetween(a, b time.Time) int {
	a = StartOfDay(a)
	b = StartOfDay(b)
	diff := a.Sub(b)
	if diff < 0 {
		diff = -diff
	}
	return int(diff.Hours() / 24)
}
