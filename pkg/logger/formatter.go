package logger

import (
	"fmt"
	"time"
)

// Formatter 格式化器接口
// 用接口而不是直接写死格式，是为了以后可以换 JSON 格式或其他自定义格式
type Formatter interface {
	Format(level Level, msg string, caller string, ts time.Time) string
}

// TextFormatter 默认的文本格式化器
// Colorful 控制是否输出带 ANSI 颜色的文本
type TextFormatter struct {
	Colorful bool
}

// NewTextFormatter 创建文本格式化器
func NewTextFormatter(colorful bool) *TextFormatter {
	return &TextFormatter{Colorful: colorful}
}

// Format 格式化一条日志
// 输出格式：2024-01-15 14:30:05.123 [INFO] main.go:42 - something happened
// caller 为空时省略文件信息部分（用户没开启 caller 时）
func (f *TextFormatter) Format(level Level, msg string, caller string, ts time.Time) string {
	// Go 的时间格式化很特别：必须用 2006-01-02 15:04:05 这个特定时间作为模板
	// .000 表示保留三位毫秒
	timestamp := ts.Format("2006-01-02 15:04:05.000")
	levelStr := fmt.Sprintf("[%s]", level.String())

	var line string
	if caller != "" {
		// 有 caller 信息时：时间 [级别] 文件:行号 - 消息
		line = fmt.Sprintf("%s %s %s - %s", timestamp, levelStr, caller, msg)
	} else {
		// 无 caller 信息时：时间 [级别] 消息
		line = fmt.Sprintf("%s %s %s", timestamp, levelStr, msg)
	}

	// 彩色模式：给整行包上颜色
	// 文件输出时 Colorful=false，就不会有 ANSI 转义码污染文件
	if f.Colorful {
		return wrapColor(colorForLevel(level), line) + "\n"
	}
	return line + "\n"
}
