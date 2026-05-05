package rotatelog

import "os"

// Option 函数式选项模式
type Option func(*RotateWriter)

// WithMaxSize 设置单个日志文件的最大大小（MB），默认 100
func WithMaxSize(mb int) Option {
	return func(w *RotateWriter) {
		w.maxSize = int64(mb) * 1024 * 1024
	}
}

// WithMaxBackups 设置最多保留的旧日志文件数量，默认 0（不限）
func WithMaxBackups(n int) Option {
	return func(w *RotateWriter) {
		w.maxBackups = n
	}
}

// WithMaxAge 设置最多保留的天数，默认 0（不限）
func WithMaxAge(days int) Option {
	return func(w *RotateWriter) {
		w.maxAge = days
	}
}

// WithCompress 是否 gzip 压缩旧日志，默认 false
func WithCompress(enable bool) Option {
	return func(w *RotateWriter) {
		w.compress = enable
	}
}

// WithRotateByDate 设置按日期轮转，interval 可选 "daily" 或 "hourly"
// 不调用此选项则默认按大小轮转
func WithRotateByDate(interval string) Option {
	return func(w *RotateWriter) {
		w.dateInterval = interval
	}
}

// WithPerm 设置日志文件权限，默认 0644
func WithPerm(perm os.FileMode) Option {
	return func(w *RotateWriter) {
		w.perm = perm
	}
}
