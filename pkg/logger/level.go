package logger

import "fmt"

// Level 日志级别，用 int 类型方便比较大小
// iota 从 0 开始递增，DEBUG=0 意味着零值就是最低级别，不设置也能看到所有日志
type Level int

const (
	DEBUG Level = iota // 调试信息，开发时用
	INFO               // 普通信息
	WARN               // 警告，不影响运行但需要注意
	ERROR              // 错误，功能可能受损
	FATAL              // 致命错误，程序即将退出
)

// String 返回级别名称，用于日志输出中的 [INFO] 这种标签
// 越界值返回 UNKNOWN，避免 panic
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	}
	return "UNKNOWN"
}

// ParseLevel 从字符串解析级别，方便从配置文件读取
func ParseLevel(s string) (Level, error) {
	switch s {
	case "DEBUG":
		return DEBUG, nil
	case "INFO":
		return INFO, nil
	case "WARN":
		return WARN, nil
	case "ERROR":
		return ERROR, nil
	case "FATAL":
		return FATAL, nil
	}
	return DEBUG, fmt.Errorf("unknown level: %s", s)
}
