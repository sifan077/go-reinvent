package cache

import "time"

// config 存储缓存的可选配置
type config struct {
	ttl time.Duration // 全局默认 TTL，0 表示永不过期
}

// Option 函数式配置项
type Option func(*config)

// WithTTL 设置全局默认过期时间。单个 Put 可通过传入 ttl 参数覆盖。
func WithTTL(d time.Duration) Option {
	return func(c *config) { c.ttl = d }
}

// applyOptions 将配置项应用到 config 上
func applyOptions(opts ...Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
