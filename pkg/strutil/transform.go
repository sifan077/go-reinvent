package strutil

import "strings"

// Substr 安全截取字符串，支持中文等多字节字符
// start: 起始位置（支持负数，表示从末尾倒数）
// length: 截取长度（如果超出范围则截取到末尾）
func Substr(s string, start, length int) string {
	runes := []rune(s)
	runeLen := len(runes)

	// 处理负数 start
	if start < 0 {
		start = runeLen + start
	}
	// 边界检查
	if start < 0 {
		start = 0
	}
	if start >= runeLen {
		return ""
	}
	if start+length > runeLen {
		length = runeLen - start
	}
	return string(runes[start : start+length])
}

// PadLeft 在字符串左侧填充字符到指定长度
func PadLeft(s string, length int, padChar rune) string {
	runeLen := len([]rune(s))
	if runeLen >= length {
		return s
	}
	padding := strings.Repeat(string(padChar), length-runeLen)
	return padding + s
}

// PadRight 在字符串右侧填充字符到指定长度
func PadRight(s string, length int, padChar rune) string {
	runeLen := len([]rune(s))
	if runeLen >= length {
		return s
	}
	padding := strings.Repeat(string(padChar), length-runeLen)
	return s + padding
}

// Reverse 翻转字符串，支持中文等多字节字符
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// CamelToSnake 驼峰命名转下划线命名
// 例如: "helloWorld" -> "hello_world", "userID" -> "user_id"
func CamelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			// 非首字母且前一个字符不是大写，或者前一个字符是大写但下一个字符是小写（处理缩写如 "ID"）
			if i > 0 {
				prev := rune(s[i-1])
				next := rune(0)
				if i+1 < len(s) {
					next = rune(s[i+1])
				}
				if prev >= 'a' && prev <= 'z' || (prev >= 'A' && prev <= 'Z' && next >= 'a' && next <= 'z') {
					result.WriteRune('_')
				}
			}
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// SnakeToCamel 下划线命名转驼峰命名（首字母小写）
// 例如: "hello_world" -> "helloWorld"
func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			result.WriteString(strings.ToLower(part))
		} else {
			result.WriteString(Capitalize(strings.ToLower(part)))
		}
	}
	return result.String()
}

// Capitalize 首字母大写
func Capitalize(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return strings.ToUpper(string(runes[:1])) + string(runes[1:])
}
