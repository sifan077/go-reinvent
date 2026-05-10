package cache

import "sync/atomic"

// EvictReason 淘汰原因
type EvictReason int

const (
	EvictReasonCapacity EvictReason = iota // 容量满触发淘汰
	EvictReasonExpired                     // TTL 过期
	EvictReasonRemoved                     // 手动删除
)

func (r EvictReason) String() string {
	switch r {
	case EvictReasonCapacity:
		return "capacity"
	case EvictReasonExpired:
		return "expired"
	case EvictReasonRemoved:
		return "removed"
	default:
		return "unknown"
	}
}

// EvictCallback 淘汰回调函数。注意：回调在锁内同步执行，不得调用缓存方法。
type EvictCallback[K comparable, V any] func(key K, value V, reason EvictReason)

// Stats 缓存访问统计
type Stats struct {
	Hits        int64 // 命中次数
	Misses      int64 // 未命中次数
	Evictions   int64 // 容量淘汰次数
	Expirations int64 // 过期淘汰次数
	Removals    int64 // 手动删除次数
	Size        int   // 当前元素数量（ShardedCache 合并统计时填充）
}

// HitRate 返回命中率 = Hits / (Hits + Misses)，无访问时返回 0。
func (s *Stats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// stats 内部原子计数器，锁外也可安全读取
type stats struct {
	hits        atomic.Int64
	misses      atomic.Int64
	evictions   atomic.Int64
	expirations atomic.Int64
	removals    atomic.Int64
}

func (s *stats) snapshot() Stats {
	return Stats{
		Hits:        s.hits.Load(),
		Misses:      s.misses.Load(),
		Evictions:   s.evictions.Load(),
		Expirations: s.expirations.Load(),
		Removals:    s.removals.Load(),
	}
}

func (s *stats) reset() {
	s.hits.Store(0)
	s.misses.Store(0)
	s.evictions.Store(0)
	s.expirations.Store(0)
	s.removals.Store(0)
}
