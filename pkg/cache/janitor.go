package cache

import (
	"math/rand"
	"time"
)

const (
	// defaultJanitorInterval 默认清理间隔
	defaultJanitorInterval = 1 * time.Minute

	// defaultSampleCount 每次采样检查的 key 数量
	defaultSampleCount = 20
)

// janitorFunc 是清理函数的类型，由 LRU 提供具体实现
type janitorFunc[K comparable, V any] func(cache *LRU[K, V])

// Janitor 后台清理 goroutine，定期采样检查并删除过期 key。
// 采用惰性删除 + 采样清理策略（Redis 同款）。
type Janitor[K comparable, V any] struct {
	interval    time.Duration
	sampleCount int
	stop        chan struct{}
	done        chan struct{}
}

// newJanitor 创建一个新的 Janitor。
func newJanitor[K comparable, V any](interval time.Duration, sampleCount int) *Janitor[K, V] {
	if interval <= 0 {
		interval = defaultJanitorInterval
	}
	if sampleCount <= 0 {
		sampleCount = defaultSampleCount
	}
	return &Janitor[K, V]{
		interval:    interval,
		sampleCount: sampleCount,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}
}

// start 启动后台清理 goroutine。cache 参数提供清理所需的缓存实例。
func (j *Janitor[K, V]) start(cache *LRU[K, V]) {
	go func() {
		defer close(j.done)
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				j.cleanup(cache)
			case <-j.stop:
				return
			}
		}
	}()
}

// stop 停止后台清理 goroutine。
func (j *Janitor[K, V]) stopCleanup() {
	close(j.stop)
	<-j.done // 等待 goroutine 退出
}

// cleanup 执行一次采样清理：随机采样 sampleCount 个 key，删除其中过期的。
func (j *Janitor[K, V]) cleanup(cache *LRU[K, V]) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if cache.size == 0 {
		return
	}

	// 采样数量不超过实际 key 数量
	count := j.sampleCount
	if count > cache.size {
		count = cache.size
	}

	// 随机采样：从 map 中随机选取 count 个 key
	keys := cache.keysForSampling(count)
	now := time.Now()

	for _, key := range keys {
		e, ok := cache.cache[key]
		if !ok {
			continue // 可能已被其他操作删除
		}
		// 检查是否过期
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			v := e.value
			cache.removeElement(e)
			delete(cache.cache, e.key)
			cache.size--
			cache.stats.expirations.Add(1)
			cache.fireEvict(key, v, EvictReasonExpired)
		}
	}
}

// keysForSampling 从 map 中随机选取 n 个 key（调用方需持有锁）。
func (c *LRU[K, V]) keysForSampling(n int) []K {
	keys := make([]K, 0, n)
	i := 0
	for k := range c.cache {
		if i >= n {
			break
		}
		keys = append(keys, k)
		i++
	}

	// Fisher-Yates 洗牌，确保随机性
	for i := len(keys) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		keys[i], keys[j] = keys[j], keys[i]
	}

	return keys
}
