package strutil

import "strings"

// Mask 通用掩码函数
// s: 原始字符串
// start: 开始掩码的位置（从0开始）
// end: 结束掩码的位置（不包含）
// maskChar: 掩码字符
func Mask(s string, start, end int, maskChar rune) string {
	runes := []rune(s)
	runeLen := len(runes)

	if start < 0 {
		start = 0
	}
	if end > runeLen {
		end = runeLen
	}
	if start >= end {
		return s
	}

	var result strings.Builder
	result.WriteString(string(runes[:start]))
	for i := start; i < end; i++ {
		result.WriteRune(maskChar)
	}
	result.WriteString(string(runes[end:]))
	return result.String()
}

// MaskPhone 手机号掩码，保留前3位和后4位
// 例如: "13812345678" -> "138****5678"
func MaskPhone(phone string) string {
	runes := []rune(phone)
	if len(runes) < 7 {
		return phone
	}
	return string(runes[:3]) + strings.Repeat("*", len(runes)-7) + string(runes[len(runes)-4:])
}

// MaskEmail 邮箱掩码，保留首字符和域名
// 例如: "test@example.com" -> "t***@example.com"
func MaskEmail(email string) string {
	atIndex := strings.Index(email, "@")
	if atIndex <= 0 {
		return email
	}
	prefix := email[:atIndex]
	if len(prefix) <= 1 {
		return prefix + "***" + email[atIndex:]
	}
	return string(prefix[0]) + "***" + email[atIndex:]
}

// MaskIDCard 身份证号掩码，保留前3位和后4位
// 例如: "110101199001011234" -> "110***********1234"
func MaskIDCard(id string) string {
	runes := []rune(id)
	if len(runes) < 8 {
		return id
	}
	return string(runes[:3]) + strings.Repeat("*", len(runes)-7) + string(runes[len(runes)-4:])
}
