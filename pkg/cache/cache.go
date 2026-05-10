package cache

import "time"

// Cache 是缓存的通用接口，支持泛型键值对。
// 实现此接口可用于在不同缓存策略（LRU、LFU、FIFO 等）之间切换。
type Cache[K comparable, V any] interface {
	// Get 查找缓存，命中返回值和 true，未命中返回零值和 false。
	Get(key K) (V, bool)

	// Put 写入缓存。ttl 可选，传入则覆盖全局默认 TTL。
	Put(key K, value V, ttl ...time.Duration)

	// Remove 删除指定 key，返回是否删除成功。
	Remove(key K) bool

	// Len 返回当前缓存中的元素数量。
	Len() int

	// Stats 返回缓存访问统计快照。
	Stats() Stats

	// Close 关闭缓存，释放后台资源（如 janitor goroutine）。
	// 关闭后缓存仍可读写，但不再主动清理过期 key。
	Close() error
}
