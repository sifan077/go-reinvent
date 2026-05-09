# LRU 缓存库 · 设计规划

## 概述

LRU（Least Recently Used）是一种经典的缓存淘汰策略：当缓存满时，优先淘汰**最久未被访问**的数据。

本计划从单机 MVP 出发，分 5 个阶段逐步构建一个生产级 LRU 缓存库，每个阶段都可以独立编译、测试和使用。

---

## 阶段一：MVP — 基础 LRU 缓存

### 目标

实现一个**单机、单并发、无过期**的 LRU 缓存，理解 LRU 的核心数据结构。

### 核心思路

LRU 的本质是两个操作必须是 O(1)：

1. **快速查找** — 给定 key，立即找到对应的 value → 哈希表 `map[K]*Node`
2. **快速排序** — 访问后，立即将该元素移到"最近使用"的位置 → **双向链表**

为什么是双向链表而不是数组？

- 数组删除中间元素需要移动后续所有元素 → O(n)
- 双向链表删除任意节点只需修改前后指针 → O(1)

```
哈希表:  key → *Node（指向链表节点）

双向链表（按访问时间排序）:
  HEAD ⟷ [最近使用] ⟷ [...] ⟷ [最久未使用] ⟷ TAIL

Get(key): 哈希表O(1)找到节点 → 把节点移到链表头部
Put(key, val): 若已存在则更新并移到头部；若不存在则新建节点插入头部，
               若超出容量则删除链表尾部节点（最久未使用）
```

### 目录结构

```
pkg/cache/
├── lru.go          # LRU 结构体、双向链表、核心 Get/Put/Remove
├── option.go       # Option 模式配置（容量等）
└── cache_test.go   # 单元测试
```

### 核心设计

#### 1. 双向链表节点

```go
// entry 是双向链表中的一个节点，同时存储 key 是为了在淘汰时能反向删除哈希表中的记录
type entry[K comparable, V any] struct {
    key   K
    value V
    prev  *entry[K, V]
    next  *entry[K, V]
}
```

为什么 node 里要存 key？

> 淘汰链表尾部节点时，需要同时从哈希表中删除对应 key。如果 node 里没有 key，就无法反查哈希表。

#### 2. LRU 结构体

```go
type LRU[K comparable, V any] struct {
    capacity int
    cache    map[K]*entry[K, V]  // 哈希表：key → 链表节点
    head     *entry[K, V]        // 哨兵头（简化边界判断）
    tail     *entry[K, V]        // 哨兵尾
    size     int                 // 当前元素数量
}
```

为什么用哨兵节点（sentinel node）？

> 哨兵头和哨兵尾是不存储数据的空节点。有了它们，链表永远不为空，插入/删除操作不需要判断"链表是否为空"、"是否是第一个/最后一个节点"，代码大幅简化，边界 bug 几乎为零。

#### 3. 核心方法

```go
// New 创建 LRU 缓存
func New[K comparable, V any](capacity int, opts ...Option) *LRU[K, V]

// Get 查找缓存，命中后将节点移到链表头部（标记为最近使用）
func (c *LRU[K, V]) Get(key K) (V, bool)

// Put 写入缓存，若已存在则更新并移到头部；若不存在则新建；超出容量则淘汰尾部
func (c *LRU[K, V]) Put(key K, value V)

// Remove 手动删除指定 key
func (c *LRU[K, V]) Remove(key K) bool

// Len 返回当前缓存元素数量
func (c *LRU[K, V]) Len() int
```

#### 4. 链表内部操作（私有）

```go
// moveToFront 将已有节点移到链表头部
func (c *LRU[K, V]) moveToFront(e *entry[K, V])

// pushFront 在链表头部插入新节点
func (c *LRU[K, V]) pushFront(e *entry[K, V])

// removeElement 从链表中删除节点
func (c *LRU[K, V]) removeElement(e *entry[K, V])

// removeTail 删除链表尾部节点（最久未使用），并返回其 key 用于删除哈希表
func (c *LRU[K, V]) removeTail() K
```

#### 5. Option 配置

```go
type Option func(*config)

type config struct {
    // 后续阶段扩展：onEvict、ttl 等
}

func New[K comparable, V any](capacity int, opts ...Option) *LRU[K, V]
```

### 测试计划

- [ ] 基本 Put/Get 功能
- [ ] Get 不存在的 key 返回 false
- [ ] Put 已存在的 key 会更新 value
- [ ] 超出容量时自动淘汰最久未使用的元素
- [ ] 淘汰顺序验证：连续 Put n+1 个 key，第 1 个被淘汰
- [ ] Get 操作会刷新访问时间（不会被淘汰）
- [ ] Remove 手动删除
- [ ] 容量为 1 的边界情况
- [ ] 容量为 0 的 panic 或拒绝

---

## 阶段二：并发安全 + TTL 过期

### 目标

加上 `sync.RWMutex` 保证并发安全，支持 key 级别的 TTL（Time-To-Live）过期。

### 核心思路

#### 并发安全

最简单的方案：整个缓存加一把读写锁。

```
Get → RLock（读锁，允许并发读）
Put → Lock（写锁，互斥写）
```

> 为什么是 RWMutex 而不是 Mutex？
> 缓存场景读多写少，RWMutex 允许多个 goroutine 同时读，性能更好。

#### TTL 过期

给每个 entry 增加一个过期时间字段：

```go
type entry[K comparable, V any] struct {
    key      K
    value    V
    prev     *entry[K, V]
    next     *entry[K, V]
    expiresAt time.Time  // 零值表示永不过期
}
```

过期判断时机（惰性删除）：

1. **Get 时检查** — 如果已过期，删除并返回未命中
2. **不主动扫描** — 简单实现不需要后台 goroutine 定期清理

> 为什么用惰性删除而不是主动扫描？
> 惰性删除零开销，不需要额外 goroutine，实现简单。生产级缓存才会加主动清理（阶段五）。

### 修改内容

```
pkg/cache/
├── lru.go          # 增加 sync.RWMutex，entry 增加 expiresAt 字段
├── option.go       # 增加 WithTTL 全局默认 TTL 选项
└── cache_test.go   # 增加并发测试、TTL 测试
```

### 新增/修改 API

```go
// Put 增加可选 TTL 参数
func (c *LRU[K, V]) Put(key K, value V, ttl ...time.Duration)

// Option 新增
func WithTTL(d time.Duration) Option  // 全局默认 TTL
```

### 测试计划

- [ ] 并发安全：100 个 goroutine 同时 Put/Get，不 panic
- [ ] TTL 过期：Put 后等待过期，Get 返回 false
- [ ] 零 TTL（默认）表示永不过期
- [ ] 单个 key 自定义 TTL 覆盖全局 TTL
- [ ] 过期 key 在 Get 时被正确清理（不占容量）

---

## 阶段三：淘汰回调 + 访问统计

### 目标

支持淘汰回调（onEvict），让用户在元素被驱逐时执行清理逻辑（如关闭连接、写日志）。
加入命中率统计，便于监控缓存效果。

### 核心思路

#### 淘汰回调

通过 Option 注入回调函数：

```go
type EvictCallback[K comparable, V any] func(key K, value V, reason EvictReason)

type EvictReason int

const (
    EvictReasonCapacity EvictReason = iota  // 容量满被淘汰
    EvictReasonExpired                       // TTL 过期被淘汰
    EvictReasonRemoved                       // 被手动 Remove 删除
)
```

回调在什么时候触发？

- Put 超出容量 → 淘汰尾部节点 → 触发回调（reason = Capacity）
- Get 发现过期 → 删除节点 → 触发回调（reason = Expired）
- 手动 Remove → 删除节点 → 触发回调（reason = Removed）

> 回调函数在锁内执行还是锁外执行？
> **锁内执行**更简单，但回调如果做耗时操作会阻塞其他请求。
> **锁外执行**更安全，但需要先拷贝出 key/value 再释放锁后调用。
> MVP 选择锁内执行，后续可优化为锁外。

#### 访问统计

```go
type Stats struct {
    Hits      int64  // 命中次数
    Misses    int64  // 未命中次数
    Evictions int64  // 淘汰次数（容量触发）
    Expirations int64 // 过期次数
    Size      int    // 当前元素数量
}

func (s *Stats) HitRate() float64  // 命中率 = Hits / (Hits + Misses)
```

用 `atomic.Int64` 实现无锁计数，性能开销极小。

### 修改内容

```
pkg/cache/
├── lru.go          # 增加回调触发逻辑
├── stats.go        # Stats 结构体、原子计数
├── option.go       # 增加 WithOnEvict 选项
└── cache_test.go   # 增加回调测试、统计测试
```

### 新增 API

```go
func WithOnEvict[K comparable, V any](cb EvictCallback[K, V]) Option
func (c *LRU[K, V]) Stats() Stats
func (c *LRU[K, V]) ResetStats()
```

### 测试计划

- [ ] 容量淘汰触发回调，reason == EvictReasonCapacity
- [ ] 过期淘汰触发回调，reason == EvictReasonExpired
- [ ] 手动删除触发回调，reason == EvictReasonRemoved
- [ ] 回调中再次操作缓存不死锁（如果锁内执行则需注意）
- [ ] 命中率统计正确
- [ ] 统计计数器并发安全

---

## 阶段四：分片锁 + 批量操作

### 目标

用分片锁（sharded lock）替代全局锁，提升高并发场景下的吞吐量。
增加批量操作 API。

### 核心思路

#### 分片锁原理

全局锁的问题：所有 goroutine 竞争同一把锁，高并发下锁争用严重。

分片锁的思路：把缓存分成 N 个独立的子缓存（分片），每个分片有自己的锁。

```
全局哈希:  hash(key) → shard_index (0 ~ N-1)

Shard[0]:  { lock, map, linkedList }
Shard[1]:  { lock, map, linkedList }
...
Shard[N-1]: { lock, map, linkedList }
```

不同 key 大概率落在不同分片，可以并行读写，锁争用降低 N 倍。

分片数量怎么选？

- 通常是 16 或 32，取 2 的幂方便位运算取模
- 分片太多浪费内存（每个分片都有一个 map + 哨兵节点）
- 分片太少并发提升有限

> 位运算取模：`shardIndex = hash(key) & (shardCount - 1)`，比 `%` 更快

#### 哈希函数

Go 的 map 内部已经有哈希函数，但不对外暴露。选择方案：

1. **简单方案**：对 comparable 类型用 `fmt.Sprintf("%v", key)` 然后取 FNV 哈希
2. **泛型方案**：让用户通过 Option 注入自定义哈希函数

```go
type Hasher[K comparable] func(key K) uint64

func WithHasher[K comparable](h Hasher[K]) Option
```

默认实现用 FNV-1a 哈希。

#### 批量操作

```go
// GetMulti 批量获取
func (c *LRU[K, V]) GetMulti(keys ...K) map[K]V

// PutMulti 批量写入
func (c *LRU[K, V]) PutMulti(entries map[K]V, ttl ...time.Duration)

// RemoveMulti 批量删除
func (c *LRU[K, V]) RemoveMulti(keys ...K) int
```

### 修改内容

```
pkg/cache/
├── lru.go          # 重构为 LRU（单分片）+ ShardedCache（多分片）
├── shard.go        # ShardedCache 结构体，分片逻辑
├── hash.go         # 默认哈希函数（FNV-1a）
├── stats.go        # 统计合并多个分片的计数
├── option.go       # 增加 WithShards、WithHasher 选项
└── cache_test.go   # 增加分片并发测试、批量操作测试
```

### 测试计划

- [ ] 分片缓存基本功能与单分片一致
- [ ] 分片并发性能优于全局锁（可选 benchmark）
- [ ] 批量操作原子性（GetMulti 要么全在要么不在）
- [ ] 自定义哈希函数生效
- [ ] 分片数为 0 或负数的默认处理

---

## 阶段五：主动清理 + 接口抽象

### 目标

增加后台 goroutine 定期清理过期 key（主动清理），定义 `Cache` 接口便于替换实现。

### 核心思路

#### 主动清理

惰性删除的缺点：大量过期 key 会一直占用内存，直到被访问才清理。

主动清理方案：启动一个后台 goroutine，每隔一段时间扫描并删除过期 key。

```go
// 清理策略
type Janitor struct {
    interval time.Duration  // 扫描间隔
    stop     chan struct{}   // 停止信号
}
```

扫描策略选择：

1. **全量扫描**：遍历所有 key，删除过期的。简单但 key 很多时耗时。
2. **采样扫描**：随机抽取 N 个 key 检查过期。生产级做法（Redis 的策略）。
3. **惰性 + 采样**：Get 时惰性删除 + 每次扫描随机采样 20 个 key，删除其中过期的。

> MVP 选择方案 3（惰性 + 采样），这是 Redis 的 eviction 策略之一，简单高效。

#### 接口抽象

```go
// Cache 是缓存的通用接口
type Cache[K comparable, V any] interface {
    Get(key K) (V, bool)
    Put(key K, value V, ttl ...time.Duration)
    Remove(key K) bool
    Len() int
    Stats() Stats
    Close() error  // 关闭后台 goroutine
}
```

有了接口，用户可以在不同实现之间切换：

- `LRU` — 基础单机 LRU
- `ShardedLRU` — 分片 LRU
- 未来可扩展：`LFU`、`FIFO`、`ARC` 等

### 修改内容

```
pkg/cache/
├── cache.go        # Cache 接口定义
├── lru.go          # LRU 实现 Cache 接口
├── shard.go        # ShardedLRU 实现 Cache 接口
├── hash.go         # 哈希函数
├── stats.go        # 统计
├── janitor.go      # 后台清理 goroutine
├── option.go       # 增加 WithJanitorInterval 选项
└── cache_test.go   # 增加清理测试、接口测试
```

### 测试计划

- [ ] 后台清理 goroutine 能正确启动和停止
- [ ] Close() 后不再清理，资源正确释放
- [ ] 采样清理能在合理时间内清理大量过期 key
- [ ] Cache 接口多态：LRU 和 ShardedLRU 都满足接口
- [ ] 资源泄漏测试：反复 New/Close 不泄漏 goroutine

---

## 最终目录结构

```
pkg/cache/
├── cache.go        # Cache 接口
├── lru.go          # 单分片 LRU 实现（双向链表 + 哈希表）
├── shard.go        # 分片 LRU 封装
├── hash.go         # 默认 FNV-1a 哈希
├── stats.go        # 访问统计
├── janitor.go      # 后台过期清理
├── option.go       # Option 模式配置
└── cache_test.go   # 全量测试

cmd/cache/
└── main.go         # 演示程序
```

## 对外 API 总览

```go
// 创建
cache := cache.New[string, User](1000,
    cache.WithTTL(5*time.Minute),
    cache.WithShards(16),
    cache.WithOnEvict(func(key string, val User, reason cache.EvictReason) {
        log.Printf("evicted: %s, reason: %d", key, reason)
    }),
    cache.WithJanitorInterval(1*time.Minute),
)

// 基本操作
cache.Put("user:1", user1)
cache.Put("user:2", user2, 10*time.Minute)  // 自定义 TTL
val, ok := cache.Get("user:1")
cache.Remove("user:1")

// 批量操作
vals := cache.GetMulti("user:1", "user:2", "user:3")
cache.PutMulti(map[string]User{"user:4": u4, "user:5": u5})

// 统计
stats := cache.Stats()
fmt.Printf("命中率: %.2f%%\n", stats.HitRate()*100)

// 关闭
cache.Close()
```

## 关键设计决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| 核心数据结构 | 哈希表 + 双向链表 | O(1) 查找 + O(1) 移动/删除 |
| 并发方案 | 分片 RWMutex | 读多写少场景下性能优于单锁和 sync.Map |
| 过期策略 | 惰性删除 + 采样清理 | 简单高效，Redis 同款策略 |
| 哈希函数 | FNV-1a + 可注入 | 默认够用，极端场景可自定义 |
| 接口设计 | 泛型 Cache 接口 | 类型安全，多态，可替换实现 |
| 零依赖 | 仅标准库 | 符合项目原则 |

## 参考

- GroupCache：https://github.com/golang/groupcache — Go 社区经典 LRU 实现
- Ristretto：https://github.com/dgraph-io/ristretto — 高性能并发缓存
- Redis 淘汰策略：惰性删除 + 定期删除 + 采样淘汰
- 标准库 `container/list` — 可参考但本项目自己实现双向链表（学习目的）
