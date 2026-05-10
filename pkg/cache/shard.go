package cache

import "time"

// ShardedCache 是一个基于分片锁的 LRU 缓存，将缓存分成多个独立分片以降低锁争用。
// 不同 key 大概率落在不同分片，可以并行读写。
type ShardedCache[K comparable, V any] struct {
	shards []*LRU[K, V]
	hasher func(K) uint64
	mask   uint64 // shardCnt - 1，用于位运算取模
}

// NewSharded 创建一个分片 LRU 缓存。
// shardCnt 会自动向上取整到最近的 2 的幂，每个分片的容量为 capacity/shardCnt。
func NewSharded[K comparable, V any](capacity int, shardCnt int, hasher func(K) uint64, opts ...Option) *ShardedCache[K, V] {
	if capacity <= 0 {
		panic("cache: capacity must be > 0")
	}
	if shardCnt <= 0 {
		shardCnt = defaultShardCount
	}
	if hasher == nil {
		hasher = FnvHash[K]
	}

	n := nextPowerOfTwo(shardCnt)
	perShard := max(capacity/n, 1)

	sc := &ShardedCache[K, V]{
		shards: make([]*LRU[K, V], n),
		hasher: hasher,
		mask:   uint64(n - 1),
	}
	for i := 0; i < n; i++ {
		sc.shards[i] = New[K, V](perShard, opts...)
	}
	return sc
}

// shardIndex 返回 key 对应的分片索引
func (sc *ShardedCache[K, V]) shardIndex(key K) uint64 {
	return sc.hasher(key) & sc.mask
}

// Get 从分片缓存中查找 key，命中后刷新访问顺序。
func (sc *ShardedCache[K, V]) Get(key K) (V, bool) {
	return sc.shards[sc.shardIndex(key)].Get(key)
}

// Peek 只读查找，不刷新访问顺序。
func (sc *ShardedCache[K, V]) Peek(key K) (V, bool) {
	return sc.shards[sc.shardIndex(key)].Peek(key)
}

// Put 写入分片缓存。
func (sc *ShardedCache[K, V]) Put(key K, value V, ttl ...time.Duration) {
	sc.shards[sc.shardIndex(key)].Put(key, value, ttl...)
}

// Remove 从分片缓存中删除 key。
func (sc *ShardedCache[K, V]) Remove(key K) bool {
	return sc.shards[sc.shardIndex(key)].Remove(key)
}

// Len 返回所有分片的元素总数。
func (sc *ShardedCache[K, V]) Len() int {
	total := 0
	for _, s := range sc.shards {
		total += s.Len()
	}
	return total
}

// Stats 合并所有分片的访问统计。
func (sc *ShardedCache[K, V]) Stats() Stats {
	var merged Stats
	for _, s := range sc.shards {
		ss := s.Stats()
		merged.Hits += ss.Hits
		merged.Misses += ss.Misses
		merged.Evictions += ss.Evictions
		merged.Expirations += ss.Expirations
		merged.Removals += ss.Removals
	}
	merged.Size = sc.Len()
	return merged
}

// ResetStats 重置所有分片的统计计数器。
func (sc *ShardedCache[K, V]) ResetStats() {
	for _, s := range sc.shards {
		s.ResetStats()
	}
}

// nextPowerOfTwo 返回大于等于 n 的最小 2 的幂。
func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

// Close 关闭分片缓存，停止所有分片的后台清理 goroutine。
func (sc *ShardedCache[K, V]) Close() error {
	for _, s := range sc.shards {
		if err := s.Close(); err != nil {
			return err
		}
	}
	return nil
}
