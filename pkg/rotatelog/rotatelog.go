package rotatelog

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RotateWriter 实现 io.Writer 接口，支持按大小/日期自动轮转日志文件
type RotateWriter struct {
	mu           sync.Mutex
	filename     string   // 基础文件名，如 "logs/app.log"
	fp           *os.File // 当前打开的文件
	size         int64    // 当前文件已写字节数
	strategy     Strategy // 轮转策略
	maxSize      int64    // 单文件最大字节数（按大小轮转时使用）
	maxBackups   int      // 最多保留旧文件数，0=不限
	maxAge       int      // 最多保留天数，0=不限
	compress     bool     // 是否 gzip 压缩旧日志
	dateInterval string   // "daily" / "hourly"，空表示按大小轮转
	perm         os.FileMode
	lastRotate   time.Time // 上次轮转时间（按日期轮转时使用）
}

// New 创建 RotateWriter
// filename: 基础日志文件路径，如 "logs/app.log"
func New(filename string, opts ...Option) *RotateWriter {
	w := &RotateWriter{
		filename:   filename,
		maxSize:    100 * 1024 * 1024, // 默认 100MB
		maxBackups: 0,
		maxAge:     0,
		compress:   false,
		perm:       0644,
		lastRotate: time.Now(),
	}

	for _, opt := range opts {
		opt(w)
	}

	// 根据配置选择策略
	w.strategy = w.buildStrategy()

	return w
}

// buildStrategy 根据配置构建轮转策略
func (w *RotateWriter) buildStrategy() Strategy {
	if w.dateInterval != "" {
		if w.maxSize > 0 {
			return &CombinedStrategy{
				DateStrategy: DateStrategy{Interval: w.dateInterval},
				MaxSize:      w.maxSize,
			}
		}
		return &DateStrategy{Interval: w.dateInterval}
	}
	return &SizeStrategy{MaxSize: w.maxSize}
}

// Write 实现 io.Writer 接口
func (w *RotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 确保文件已打开
	if w.fp == nil {
		if err = w.openFile(); err != nil {
			return 0, err
		}
	}

	// 检查是否需要按日期轮转
	if w.dateInterval != "" && NeedsDateRotate(w.lastRotate, time.Now(), w.dateInterval) {
		if err = w.rotate(); err != nil {
			return 0, err
		}
	}

	// 写入数据
	n, err = w.fp.Write(p)
	w.size += int64(n)

	// 检查是否需要按大小轮转
	if w.strategy.ShouldRotate(&fileInfo{size: w.size}, time.Now()) {
		if err = w.rotate(); err != nil {
			return n, err
		}
	}

	return n, err
}

// Rotate 手动触发轮转
func (w *RotateWriter) Rotate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.rotate()
}

// rotate 内部轮转实现，调用前需持有锁
func (w *RotateWriter) rotate() error {
	if w.fp != nil {
		if err := w.fp.Close(); err != nil {
			return err
		}
	}

	now := time.Now()
	newName := w.strategy.NextFileName(w.filename, now)

	// 将当前文件重命名为归档名
	if err := os.Rename(w.filename, newName); err != nil && !os.IsNotExist(err) {
		return err
	}

	// 压缩刚轮转的文件（同步，在创建新文件之前）
	if w.compress {
		w.gzipFile(newName)
	}

	w.lastRotate = now
	w.size = 0

	// 创建新文件
	if err := w.openFile(); err != nil {
		return err
	}

	// 异步清理旧日志
	go w.cleanup()

	return nil
}

// openFile 打开或创建日志文件，调用前需持有锁
func (w *RotateWriter) openFile() error {
	dir := filepath.Dir(w.filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fp, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, w.perm)
	if err != nil {
		return err
	}

	w.fp = fp

	// 获取当前文件大小
	info, err := fp.Stat()
	if err != nil {
		w.size = 0
	} else {
		w.size = info.Size()
	}

	return nil
}

// Close 关闭当前文件
func (w *RotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.fp != nil {
		err := w.fp.Close()
		w.fp = nil
		return err
	}
	return nil
}

// fileInfo 用于在没有实际文件时传递大小信息给 Strategy
type fileInfo struct {
	size int64
}

func (f *fileInfo) Name() string       { return "" }
func (f *fileInfo) Size() int64        { return f.size }
func (f *fileInfo) Mode() os.FileMode  { return 0 }
func (f *fileInfo) ModTime() time.Time { return time.Time{} }
func (f *fileInfo) IsDir() bool        { return false }
func (f *fileInfo) Sys() any           { return nil }
