package cache

// config 存储缓存的可选配置
type config struct {
	// 后续阶段扩展：onEvict、ttl、shards 等
}

// Option 函数式配置项
type Option func(*config)

// applyOptions 将配置项应用到 config 上
func applyOptions(opts ...Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
