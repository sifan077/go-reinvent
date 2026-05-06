package timeutil

import "time"

// Convert 将时间转换到指定时区
// tz: 时区名称，如 "Asia/Shanghai", "America/New_York"
func Convert(t time.Time, tz string) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return t, err
	}
	return t.In(loc), nil
}

// ToCST 将时间转换为北京时间（Asia/Shanghai）
func ToCST(t time.Time) time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return t.In(loc)
}

// LocalNow 获取指定时区的当前时间
func LocalNow(tz string) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().In(loc), nil
}
