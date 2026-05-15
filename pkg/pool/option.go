package pool

// config 存储协程池的可选配置。
// 通过 Option 函数设置，未设置的字段使用零值或由 New 使用默认值填充。
type config struct {
	queueSize    int         // 任务队列容量，0 表示使用默认值（size * 2）
	panicHandler func(r any) // 任务 panic 时的回调，nil 表示静默吞掉
}

// Option 是协程池的函数选项。
type Option func(*config)

// WithQueueSize 设置任务队列容量。
// 默认值为 size * 2。设为 0 表示无缓冲（任务必须立即被 worker 接收）。
func WithQueueSize(size int) Option {
	return func(c *config) {
		c.queueSize = size
	}
}

// WithPanicHandler 设置任务 panic 时的回调函数。
// 默认行为是静默吞掉 panic（与 go func() 行为一致）。
func WithPanicHandler(fn func(r any)) Option {
	return func(c *config) {
		c.panicHandler = fn
	}
}

// applyOptions 应用选项并返回配置。
func applyOptions(opts []Option) config {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
