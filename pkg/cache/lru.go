// Package cache 实现了一个基于 LRU（最近最少使用）策略的泛型缓存。
// 核心数据结构：哈希表 + 双向链表，保证 Get/Put 均为 O(1) 时间复杂度。
// 支持并发安全（sync.RWMutex）和 key 级别 TTL 过期（惰性删除）。
package cache

import (
	"fmt"
	"sync"
	"time"
)

// entry 是双向链表中的节点，同时存储 key 用于淘汰时反向删除哈希表记录。
type entry[K comparable, V any] struct {
	key       K
	value     V
	prev      *entry[K, V]
	next      *entry[K, V]
	expiresAt time.Time // 零值表示永不过期
}

// LRU 是一个基于最近最少使用策略的泛型缓存，支持并发安全和 TTL 过期。
type LRU[K comparable, V any] struct {
	mu       sync.RWMutex
	capacity int
	cache    map[K]*entry[K, V] // 哈希表：O(1) 查找
	head     *entry[K, V]       // 哨兵头，head.next 才是最近使用的节点
	tail     *entry[K, V]       // 哨兵尾，tail.prev 才是最久未使用的节点
	size     int
	ttl      time.Duration       // 全局默认 TTL
	onEvict  EvictCallback[K, V] // 淘汰回调
	stats    stats               // 访问统计（原子计数器）
	janitor  *Janitor[K, V]      // 后台清理 goroutine
}

// New 创建一个指定容量的 LRU 缓存。capacity 必须大于 0。
func New[K comparable, V any](capacity int, opts ...Option) *LRU[K, V] {
	if capacity <= 0 {
		panic(fmt.Sprintf("cache: capacity must be > 0, got %d", capacity))
	}
	cfg := applyOptions(opts...)
	c := &LRU[K, V]{
		capacity: capacity,
		cache:    make(map[K]*entry[K, V], capacity),
		head:     &entry[K, V]{},
		tail:     &entry[K, V]{},
		ttl:      cfg.ttl,
	}
	if cb, ok := cfg.onEvict.(EvictCallback[K, V]); ok {
		c.onEvict = cb
	}
	c.head.next = c.tail
	c.tail.prev = c.head

	// 启动后台清理 goroutine（如果配置了 janitor 间隔）
	if cfg.janitorInterval > 0 {
		c.janitor = newJanitor[K, V](cfg.janitorInterval, cfg.janitorSamples)
		c.janitor.start(c)
	}

	return c
}

// Peek 只读查找缓存，不刷新访问顺序。过期 key 会被惰性删除。
func (c *LRU[K, V]) Peek(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.cache[key]
	if !ok {
		c.stats.misses.Add(1)
		var zero V
		return zero, false
	}
	// 检查是否过期
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		v := e.value
		c.removeElement(e)
		delete(c.cache, e.key)
		c.size--
		c.stats.misses.Add(1)
		c.stats.expirations.Add(1)
		c.fireEvict(key, v, EvictReasonExpired)
		var zero V
		return zero, false
	}
	c.stats.hits.Add(1)
	return e.value, true
}

// Get 查找缓存。命中后将节点移到链表头部（标记为最近使用），返回值和 true；
// 未命中返回零值和 false。过期 key 会被惰性删除。
func (c *LRU[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.cache[key]
	if !ok {
		c.stats.misses.Add(1)
		var zero V
		return zero, false
	}
	// 检查是否过期
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		v := e.value
		c.removeElement(e)
		delete(c.cache, e.key)
		c.size--
		c.stats.misses.Add(1)
		c.stats.expirations.Add(1)
		c.fireEvict(key, v, EvictReasonExpired)
		var zero V
		return zero, false
	}
	c.stats.hits.Add(1)
	c.moveToFront(e)
	return e.value, true
}

// Put 写入缓存。若 key 已存在则更新值并移到头部；若不存在则新建。
// 超出容量时自动淘汰最久未使用的元素。
// ttl 参数可选：传入则覆盖全局默认 TTL，不传则使用全局 TTL。
func (c *LRU[K, V]) Put(key K, value V, ttl ...time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 确定本次写入的 TTL
	effectiveTTL := c.ttl
	if len(ttl) > 0 && ttl[0] > 0 {
		effectiveTTL = ttl[0]
	}

	// 计算过期时间
	var expiresAt time.Time
	if effectiveTTL > 0 {
		expiresAt = time.Now().Add(effectiveTTL)
	}

	if e, ok := c.cache[key]; ok {
		e.value = value
		e.expiresAt = expiresAt
		c.moveToFront(e)
		return
	}
	e := &entry[K, V]{key: key, value: value, expiresAt: expiresAt}
	c.cache[key] = e
	c.pushFront(e)
	c.size++
	if c.size > c.capacity {
		c.removeOldest()
	}
}

// Remove 手动删除指定 key，返回是否删除成功。
func (c *LRU[K, V]) Remove(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.cache[key]
	if !ok {
		return false
	}
	c.removeElement(e)
	delete(c.cache, e.key)
	c.size--
	c.stats.removals.Add(1)
	c.fireEvict(key, e.value, EvictReasonRemoved)
	return true
}

// Len 返回当前缓存中的元素数量。
func (c *LRU[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// Keys 返回链表中的 key 列表（从最近使用到最久未使用），用于调试。
func (c *LRU[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, c.size)
	for e := c.head.next; e != c.tail; e = e.next {
		keys = append(keys, e.key)
	}
	return keys
}

// --- 以下为链表内部操作（调用方需持有锁） ---

// pushFront 在链表头部（哨兵头之后）插入新节点。
func (c *LRU[K, V]) pushFront(e *entry[K, V]) {
	e.next = c.head.next
	e.prev = c.head
	c.head.next.prev = e
	c.head.next = e
}

// removeElement 从链表中摘除节点（不操作哈希表）。
func (c *LRU[K, V]) removeElement(e *entry[K, V]) {
	e.prev.next = e.next
	e.next.prev = e.prev
}

// moveToFront 将已有节点移到链表头部。
func (c *LRU[K, V]) moveToFront(e *entry[K, V]) {
	c.removeElement(e)
	c.pushFront(e)
}

// Stats 返回缓存访问统计快照。
func (c *LRU[K, V]) Stats() Stats {
	return c.stats.snapshot()
}

// ResetStats 重置所有访问统计计数器。
func (c *LRU[K, V]) ResetStats() {
	c.stats.reset()
}

// fireEvict 触发淘汰回调（调用方需持有锁）。
func (c *LRU[K, V]) fireEvict(key K, value V, reason EvictReason) {
	if c.onEvict != nil {
		c.onEvict(key, value, reason)
	}
}

// removeOldest 删除链表尾部节点（最久未使用），并从哈希表中删除对应 key。
func (c *LRU[K, V]) removeOldest() {
	victim := c.tail.prev
	c.removeElement(victim)
	delete(c.cache, victim.key)
	c.size--
	c.stats.evictions.Add(1)
	c.fireEvict(victim.key, victim.value, EvictReasonCapacity)
}

// Close 关闭缓存，停止后台清理 goroutine。
// 关闭后缓存仍可正常读写，但不再主动清理过期 key。
func (c *LRU[K, V]) Close() error {
	if c.janitor != nil {
		c.janitor.stopCleanup()
		c.janitor = nil
	}
	return nil
}
