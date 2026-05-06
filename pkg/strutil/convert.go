package strutil

import (
	"fmt"
	"strconv"
	"strings"
)

// ToInt 字符串转 int，失败返回 error
func ToInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}

// MustInt 字符串转 int，失败返回默认值
func MustInt(s string, defaultVal int) int {
	v, err := ToInt(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// ToFloat64 字符串转 float64，失败返回 error
func ToFloat64(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

// MustFloat64 字符串转 float64，失败返回默认值
func MustFloat64(s string, defaultVal float64) float64 {
	v, err := ToFloat64(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// ToBool 字符串转 bool
// 支持: "1","true","yes","on" -> true; "0","false","no","off" -> false
func ToBool(s string) (bool, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off", "":
		return false, nil
	default:
		return false, fmt.Errorf("cannot convert %q to bool", s)
	}
}

// MustBool 字符串转 bool，失败返回默认值
func MustBool(s string, defaultVal bool) bool {
	v, err := ToBool(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// ToString 任意类型转 string
func ToString(v any) string {
	return fmt.Sprintf("%v", v)
}
