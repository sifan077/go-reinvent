package logger

// ANSI 转义码常量
// 格式：\033[<数字>m，数字表示颜色/样式
// \033[0m 是重置，必须加在颜色文本后面，否则终端后面的文字也会变色
const (
	colorReset  = "\033[0m"  // 重置所有样式
	colorRed    = "\033[31m" // 红色
	colorGreen  = "\033[32m" // 绿色
	colorYellow = "\033[33m" // 黄色
	colorBlue   = "\033[34m" // 蓝色
	colorGray   = "\033[37m" // 灰色
	colorCyan   = "\033[36m" // 青色
	colorBold   = "\033[1m"  // 粗体，可以和颜色叠加
)

// colorForLevel 根据日志级别返回对应颜色
// FATAL 用粗体+红色，视觉上更醒目
func colorForLevel(level Level) string {
	switch level {
	case DEBUG:
		return colorCyan
	case INFO:
		return colorGreen
	case WARN:
		return colorYellow
	case ERROR:
		return colorRed
	case FATAL:
		return colorBold + colorRed // ANSI 粗体和颜色可以叠加
	}
	return colorReset
}

// wrapColor 给字符串加上颜色和重置码
// 避免在多处重复写 color + s + reset 的拼接逻辑
func wrapColor(color, s string) string {
	return color + s + colorReset
}
