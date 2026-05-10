package cache

import "time"

// config 存储缓存的可选配置
type config struct {
	ttl     time.Duration // 全局默认 TTL，0 表示永不过期
	onEvict any           // 淘汰回调，类型为 EvictCallback[K, V]，用 any 避免泛型参数泄漏到 config
	shards  int           // 分片数，0 表示使用默认值
	hasher  any           // 自定义哈希函数，类型为 func(K) uint64
}

// Option 函数式配置项
type Option func(*config)

// WithTTL 设置全局默认过期时间。单个 Put 可通过传入 ttl 参数覆盖。
func WithTTL(d time.Duration) Option {
	return func(c *config) { c.ttl = d }
}

// WithOnEvict 设置淘汰回调。回调在锁内同步执行，不得调用缓存方法以避免死锁。
func WithOnEvict[K comparable, V any](cb EvictCallback[K, V]) Option {
	return func(c *config) { c.onEvict = cb }
}

// WithShards 设置分片数量，会自动向上取整到 2 的幂。仅对 ShardedCache 生效。
func WithShards(n int) Option {
	return func(c *config) { c.shards = n }
}

// WithHasher 设置自定义哈希函数。仅对 ShardedCache 生效。
// hasher 类型应为 func(K) uint64。
func WithHasher(hasher any) Option {
	return func(c *config) { c.hasher = hasher }
}

// applyOptions 将配置项应用到 config 上
func applyOptions(opts ...Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
