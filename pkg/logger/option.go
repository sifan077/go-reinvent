package logger

import "io"

// Option 函数式选项模式
// 每个 Option 是一个闭包，接收 Logger 指针并修改其字段
// 好处：可选参数任意组合，不需要关心顺序，也不会出现参数列表很长的情况
type Option func(*Logger)

// WithLevel 设置最低日志级别，低于该级别的日志不输出
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// WithColorful 开关彩色输出，写文件时应设为 false
func WithColorful(enable bool) Option {
	return func(l *Logger) {
		l.colorful = enable
	}
}

// WithOutput 设置输出目标，默认是 os.Stdout
// 可以传入文件、bytes.Buffer（测试用）、网络连接等任何 io.Writer
func WithOutput(w io.Writer) Option {
	return func(l *Logger) {
		l.out = w
	}
}

// WithCaller 开关调用者信息（文件名:行号）
// 开启后每条日志会多一次 runtime.Caller 调用，有微小性能开销
func WithCaller(enable bool) Option {
	return func(l *Logger) {
		l.showCaller = enable
	}
}

// WithFormatter 替换默认格式化器
// 如果设置了自定义 Formatter，WithColorful 只影响 Logger 的字段，不影响自定义 Formatter
func WithFormatter(f Formatter) Option {
	return func(l *Logger) {
		l.formatter = f
	}
}
