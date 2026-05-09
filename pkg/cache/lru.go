// Package cache 实现了一个基于 LRU（最近最少使用）策略的泛型缓存。
// 核心数据结构：哈希表 + 双向链表，保证 Get/Put 均为 O(1) 时间复杂度。
package cache

import "fmt"

// entry 是双向链表中的节点，同时存储 key 用于淘汰时反向删除哈希表记录。
type entry[K comparable, V any] struct {
	key   K
	value V
	prev  *entry[K, V]
	next  *entry[K, V]
}

// LRU 是一个基于最近最少使用策略的泛型缓存。
type LRU[K comparable, V any] struct {
	capacity int
	cache    map[K]*entry[K, V] // 哈希表：O(1) 查找
	head     *entry[K, V]       // 哨兵头，head.next 才是最近使用的节点
	tail     *entry[K, V]       // 哨兵尾，tail.prev 才是最久未使用的节点
	size     int
}

// New 创建一个指定容量的 LRU 缓存。capacity 必须大于 0。
func New[K comparable, V any](capacity int, opts ...Option) *LRU[K, V] {
	if capacity <= 0 {
		panic(fmt.Sprintf("cache: capacity must be > 0, got %d", capacity))
	}
	_ = applyOptions(opts...) // 阶段一暂不使用配置
	c := &LRU[K, V]{
		capacity: capacity,
		cache:    make(map[K]*entry[K, V], capacity),
		head:     &entry[K, V]{},
		tail:     &entry[K, V]{},
	}
	c.head.next = c.tail
	c.tail.prev = c.head
	return c
}

// Peek 只读查找缓存，不刷新访问顺序。
func (c *LRU[K, V]) Peek(key K) (V, bool) {
	e, ok := c.cache[key]
	if !ok {
		var zero V
		return zero, false
	}
	return e.value, true
}

// Get 查找缓存。命中后将节点移到链表头部（标记为最近使用），返回值和 true；
// 未命中返回零值和 false。
func (c *LRU[K, V]) Get(key K) (V, bool) {
	e, ok := c.cache[key]
	if !ok {
		var zero V
		return zero, false
	}
	c.moveToFront(e)
	return e.value, true
}

// Put 写入缓存。若 key 已存在则更新值并移到头部；若不存在则新建。
// 超出容量时自动淘汰最久未使用的元素。
func (c *LRU[K, V]) Put(key K, value V) {
	if e, ok := c.cache[key]; ok {
		e.value = value
		c.moveToFront(e)
		return
	}
	e := &entry[K, V]{key: key, value: value}
	c.cache[key] = e
	c.pushFront(e)
	c.size++
	if c.size > c.capacity {
		c.removeOldest()
	}
}

// Remove 手动删除指定 key，返回是否删除成功。
func (c *LRU[K, V]) Remove(key K) bool {
	e, ok := c.cache[key]
	if !ok {
		return false
	}
	c.removeElement(e)
	delete(c.cache, e.key)
	c.size--
	return true
}

// Len 返回当前缓存中的元素数量。
func (c *LRU[K, V]) Len() int {
	return c.size
}

// Keys 返回链表中的 key 列表（从最近使用到最久未使用），用于调试。
func (c *LRU[K, V]) Keys() []K {
	keys := make([]K, 0, c.size)
	for e := c.head.next; e != c.tail; e = e.next {
		keys = append(keys, e.key)
	}
	return keys
}

// --- 以下为链表内部操作 ---

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

// removeOldest 删除链表尾部节点（最久未使用），并从哈希表中删除对应 key。
func (c *LRU[K, V]) removeOldest() {
	victim := c.tail.prev
	c.removeElement(victim)
	delete(c.cache, victim.key)
	c.size--
}
