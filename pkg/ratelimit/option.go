package ratelimit

import "time"

// config 保存限流器的配置参数。
type config struct {
	rate     float64
	burst    int
	interval time.Duration // 滑动窗口专用
}

// Option 是限流器的函数选项。
type Option func(*config)

// WithRate 设置每秒允许的请求数（令牌桶/漏桶）或每个窗口的最大请求数（滑动窗口）。
func WithRate(rate float64) Option {
	return func(c *config) {
		c.rate = rate
	}
}

// WithBurst 设置桶的最大容量（允许的最大突发数）。
func WithBurst(burst int) Option {
	return func(c *config) {
		c.burst = burst
	}
}

// WithInterval 设置滑动窗口的时间窗口大小。
func WithInterval(d time.Duration) Option {
	return func(c *config) {
		c.interval = d
	}
}

// applyOptions 应用选项并返回配置。
func applyOptions(opts []Option) config {
	cfg := config{
		rate:     100,
		burst:    100,
		interval: time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
