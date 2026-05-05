package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Logger 核心结构体
// 通过 Option 模式配置，用 New() 创建
type Logger struct {
	mu         sync.Mutex // 保护 Write 操作，避免并发写入时日志交错混乱
	level      Level      // 最低输出级别
	out        io.Writer  // 输出目标
	colorful   bool       // 是否彩色输出
	showCaller bool       // 是否显示调用位置
	formatter  Formatter  // 格式化器
	callerSkip int        // runtime.Caller 的跳过层数，固定为 2
}

// New 创建 Logger 实例
// 先设置合理默认值，再应用用户传入的 Option
// 默认：DEBUG 级别、stdout 输出、彩色开启、不显示 caller
func New(opts ...Option) *Logger {
	l := &Logger{
		level:      DEBUG,
		out:        os.Stdout,
		colorful:   true,
		showCaller: false,
		callerSkip: 2, // 调用链：外部代码 -> Info() -> output() -> runtime.Caller(2) 拿到外部代码
	}

	// 按顺序应用所有选项
	for _, opt := range opts {
		opt(l)
	}

	// 如果用户没提供自定义 Formatter，根据 colorful 设置创建默认的
	if l.formatter == nil {
		l.formatter = NewTextFormatter(l.colorful)
	}

	return l
}

// output 所有日志方法的统一出口
// 流程：级别检查 -> 加锁 -> 获取 caller -> 格式化 -> 写入
func (l *Logger) output(level Level, msg string) {
	// 级别检查放在锁外面，被过滤的日志不会产生任何锁竞争
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取调用者信息：文件名:行号
	// runtime.Caller(skip) 返回 (pc, file, line, ok)
	// skip=2: output(0) -> Info(1) -> 外部调用者(2)
	var caller string
	if l.showCaller {
		_, file, line, ok := runtime.Caller(l.callerSkip)
		if ok {
			// filepath.Base 取文件名，兼容 Windows 的反斜杠路径
			caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
	}

	ts := time.Now()
	formatted := l.formatter.Format(level, msg, caller, ts)
	l.out.Write([]byte(formatted))
}

// 以下是各级别的便捷方法

func (l *Logger) Debug(msg string) {
	l.output(DEBUG, msg)
}

func (l *Logger) Info(msg string) {
	l.output(INFO, msg)
}

func (l *Logger) Warn(msg string) {
	l.output(WARN, msg)
}

func (l *Logger) Error(msg string) {
	l.output(ERROR, msg)
}

// Fatal 输出致命错误日志后立即退出程序
// os.Exit 不会执行 defer，但进程都要死了，mutex 解不解锁无所谓
func (l *Logger) Fatal(msg string) {
	l.output(FATAL, msg)
	os.Exit(1)
}

// 以下是带格式化的版本，用法类似 fmt.Sprintf

func (l *Logger) Debugf(format string, args ...any) {
	l.output(DEBUG, fmt.Sprintf(format, args...))
}

func (l *Logger) Infof(format string, args ...any) {
	l.output(INFO, fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...any) {
	l.output(WARN, fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...any) {
	l.output(ERROR, fmt.Sprintf(format, args...))
}

func (l *Logger) Fatalf(format string, args ...any) {
	l.output(FATAL, fmt.Sprintf(format, args...))
	os.Exit(1)
}
