package cache

import (
	"sync"
	"testing"
	"time"
)

// --- 阶段一基础测试 ---

func TestBasicPutGet(t *testing.T) {
	c := New[string, int](3)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	tests := []struct {
		name string
		key  string
		want int
		ok   bool
	}{
		{"命中a", "a", 1, true},
		{"命中b", "b", 2, true},
		{"命中c", "c", 3, true},
		{"未命中", "x", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := c.Get(tt.key)
			if ok != tt.ok {
				t.Fatalf("Get(%q) ok = %v, want %v", tt.key, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("Get(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestEviction(t *testing.T) {
	c := New[string, int](3)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)
	c.Put("d", 4) // a 应被淘汰

	if _, ok := c.Get("a"); ok {
		t.Fatal("key 'a' should have been evicted")
	}
	if v, ok := c.Get("d"); !ok || v != 4 {
		t.Fatalf("key 'd' should exist with value 4, got %v, %v", v, ok)
	}
	if c.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", c.Len())
	}
}

func TestGetRefreshesAccess(t *testing.T) {
	c := New[string, int](3)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	c.Get("a") // a 移到头部，现在顺序：a, c, b

	c.Put("d", 4) // b 是最久未使用的，应被淘汰

	if _, ok := c.Get("b"); ok {
		t.Fatal("key 'b' should have been evicted after 'a' was accessed")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("key 'a' should still exist (was refreshed by Get)")
	}
}

func TestPutUpdatesExistingKey(t *testing.T) {
	c := New[string, int](2)
	c.Put("a", 1)
	c.Put("a", 10) // 更新值

	v, ok := c.Get("a")
	if !ok || v != 10 {
		t.Fatalf("Get('a') = %v, %v, want 10, true", v, ok)
	}
	if c.Len() != 1 {
		t.Fatalf("Len() = %d, want 1 (should not create duplicate)", c.Len())
	}

	c.Put("b", 2)
	c.Put("a", 100) // a 被更新并移到头部，b 变成最久未使用

	c.Put("c", 3) // b 应被淘汰
	if _, ok := c.Get("b"); ok {
		t.Fatal("key 'b' should have been evicted")
	}
}

func TestRemove(t *testing.T) {
	c := New[string, int](3)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	ok := c.Remove("b")
	if !ok {
		t.Fatal("Remove('b') should return true")
	}
	if c.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", c.Len())
	}
	if _, ok := c.Get("b"); ok {
		t.Fatal("key 'b' should not exist after Remove")
	}

	// Remove 不存在的 key
	ok = c.Remove("x")
	if ok {
		t.Fatal("Remove('x') should return false")
	}
}

func TestCapacityOne(t *testing.T) {
	c := New[string, int](1)
	c.Put("a", 1)

	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("Get('a') = %v, %v, want 1, true", v, ok)
	}

	c.Put("b", 2) // a 应被淘汰
	if _, ok := c.Get("a"); ok {
		t.Fatal("key 'a' should have been evicted when capacity is 1")
	}
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Fatalf("Get('b') = %v, %v, want 2, true", v, ok)
	}
}

func TestCapacityZeroPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("New with capacity 0 should panic")
		}
	}()
	New[string, int](0)
}

func TestNegativeCapacityPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("New with negative capacity should panic")
		}
	}()
	New[string, int](-1)
}

func TestEvictionOrder(t *testing.T) {
	c := New[int, int](3)
	c.Put(1, 10)
	c.Put(2, 20)
	c.Put(3, 30)
	c.Put(4, 40) // 淘汰 1
	c.Put(5, 50) // 淘汰 2

	wantEvicted := []int{1, 2}
	for _, key := range wantEvicted {
		if _, ok := c.Get(key); ok {
			t.Fatalf("key %d should have been evicted", key)
		}
	}
	wantPresent := []int{3, 4, 5}
	for _, key := range wantPresent {
		if _, ok := c.Get(key); !ok {
			t.Fatalf("key %d should still exist", key)
		}
	}
}

func TestLen(t *testing.T) {
	c := New[string, int](5)
	if c.Len() != 0 {
		t.Fatalf("empty cache Len() = %d, want 0", c.Len())
	}
	c.Put("a", 1)
	c.Put("b", 2)
	if c.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", c.Len())
	}
	c.Remove("a")
	if c.Len() != 1 {
		t.Fatalf("after Remove Len() = %d, want 1", c.Len())
	}
}

// --- 阶段二：并发安全测试 ---

func TestConcurrentPutGet(t *testing.T) {
	c := New[int, int](100)
	var wg sync.WaitGroup

	// 50 个 goroutine 并发写入
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Put(n, n*10)
		}(i)
	}
	wg.Wait()

	// 50 个 goroutine 并发读取
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Get(n)
		}(i)
	}
	wg.Wait()
}

func TestConcurrentPutGetLarge(t *testing.T) {
	c := New[string, int](50)
	var wg sync.WaitGroup

	// 100 个 goroutine 混合读写
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := string(rune('A' + n%26))
			c.Put(key, n)
			c.Get(key)
			c.Peek(key)
			c.Remove(key)
			c.Len()
		}(i)
	}
	wg.Wait()
}

func TestConcurrentLen(t *testing.T) {
	c := New[int, int](10)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Put(n, n)
			_ = c.Len()
		}(i)
	}
	wg.Wait()

	// 容量为 10，最终 Len 应该是 10
	if c.Len() != 10 {
		t.Fatalf("Len() = %d, want 10", c.Len())
	}
}

// --- 阶段二：TTL 过期测试 ---

func TestTTLExpired(t *testing.T) {
	c := New[string, int](10)
	c.Put("short", 1, 50*time.Millisecond)
	c.Put("long", 2, 5*time.Second)

	// 立即查找，都应命中
	if v, ok := c.Get("short"); !ok || v != 1 {
		t.Fatalf("Get('short') immediately = %v, %v, want 1, true", v, ok)
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	// short 应过期
	if _, ok := c.Get("short"); ok {
		t.Fatal("key 'short' should have expired")
	}
	// long 应仍在
	if v, ok := c.Get("long"); !ok || v != 2 {
		t.Fatalf("Get('long') = %v, %v, want 2, true", v, ok)
	}
}

func TestTTLPeekExpired(t *testing.T) {
	c := New[string, int](10)
	c.Put("k", 42, 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	// Peek 也应检测过期并删除
	if _, ok := c.Peek("k"); ok {
		t.Fatal("Peek('k') should return false after expiration")
	}
	// 确认已从缓存中删除
	if c.Len() != 0 {
		t.Fatalf("Len() = %d, want 0 (expired key should be removed)", c.Len())
	}
}

func TestTTLZeroMeansNoExpiry(t *testing.T) {
	c := New[string, int](10)
	c.Put("forever", 99) // 不传 TTL，零值表示永不过期

	time.Sleep(50 * time.Millisecond)

	if v, ok := c.Get("forever"); !ok || v != 99 {
		t.Fatalf("Get('forever') = %v, %v, want 99, true", v, ok)
	}
}

func TestGlobalTTL(t *testing.T) {
	c := New[string, int](10, WithTTL(50*time.Millisecond))
	c.Put("a", 1)

	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Get("a"); ok {
		t.Fatal("key 'a' should have expired via global TTL")
	}
}

func TestPerKeyTTLOverridesGlobal(t *testing.T) {
	c := New[string, int](10, WithTTL(50*time.Millisecond))
	c.Put("short", 1)               // 使用全局 TTL 50ms
	c.Put("long", 2, 5*time.Second) // 覆盖为 5s

	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Get("short"); ok {
		t.Fatal("key 'short' should have expired via global TTL")
	}
	if v, ok := c.Get("long"); !ok || v != 2 {
		t.Fatalf("Get('long') = %v, %v, want 2, true (custom TTL should override)", v, ok)
	}
}

func TestTTLUpdateRefreshesExpiry(t *testing.T) {
	c := New[string, int](10)
	c.Put("k", 1, 50*time.Millisecond)

	time.Sleep(30 * time.Millisecond)
	c.Put("k", 2, 50*time.Millisecond) // 重新设置，刷新过期时间

	time.Sleep(30 * time.Millisecond) // 总共 60ms，但第一次的 50ms 已过
	// 因为 Put 刷新了过期时间，应该还在
	if v, ok := c.Get("k"); !ok || v != 2 {
		t.Fatalf("Get('k') = %v, %v, want 2, true (TTL should be refreshed)", v, ok)
	}
}

func TestTTLLenCorrectAfterExpiry(t *testing.T) {
	c := New[string, int](10)
	c.Put("a", 1, 50*time.Millisecond)
	c.Put("b", 2, 50*time.Millisecond)
	c.Put("c", 3) // 永不过期

	if c.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", c.Len())
	}

	time.Sleep(100 * time.Millisecond)

	// a, b 过期但还没被访问，Len 仍为 3（惰性删除）
	if c.Len() != 3 {
		t.Fatalf("Len() before access = %d, want 3 (lazy deletion)", c.Len())
	}

	// 访问触发惰性删除
	c.Get("a")
	c.Get("b")

	if c.Len() != 1 {
		t.Fatalf("Len() after access = %d, want 1", c.Len())
	}
}

func TestConcurrentTTL(t *testing.T) {
	c := New[int, int](100, WithTTL(50*time.Millisecond))
	var wg sync.WaitGroup

	// 并发写入带 TTL 的 key
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Put(n, n)
		}(i)
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	// 并发读取，应全部过期
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if _, ok := c.Get(n); ok {
				t.Errorf("key %d should have expired", n)
			}
		}(i)
	}
	wg.Wait()
}

// --- Phase 3: 淘汰回调 + 访问统计 ---

func TestEvictCallbackCapacity(t *testing.T) {
	type evictRecord struct {
		key    int
		value  int
		reason EvictReason
	}
	var records []evictRecord

	c := New[int, int](2, WithOnEvict(func(key, value int, reason EvictReason) {
		records = append(records, evictRecord{key, value, reason})
	}))

	c.Put(1, 10)
	c.Put(2, 20)
	c.Put(3, 30) // 触发淘汰 key=1

	if len(records) != 1 {
		t.Fatalf("expected 1 eviction, got %d", len(records))
	}
	if records[0].key != 1 || records[0].value != 10 {
		t.Errorf("expected evicted key=1 value=10, got key=%d value=%d", records[0].key, records[0].value)
	}
	if records[0].reason != EvictReasonCapacity {
		t.Errorf("expected reason=Capacity, got %v", records[0].reason)
	}
}

func TestEvictCallbackExpired(t *testing.T) {
	type evictRecord struct {
		key    string
		value  string
		reason EvictReason
	}
	var records []evictRecord

	c := New[string, string](10, WithOnEvict(func(key, value string, reason EvictReason) {
		records = append(records, evictRecord{key, value, reason})
	}))

	c.Put("k1", "v1", 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	// Get 触发惰性删除
	c.Get("k1")

	if len(records) != 1 {
		t.Fatalf("expected 1 eviction, got %d", len(records))
	}
	if records[0].key != "k1" || records[0].value != "v1" {
		t.Errorf("expected evicted key=k1 value=v1, got key=%s value=%s", records[0].key, records[0].value)
	}
	if records[0].reason != EvictReasonExpired {
		t.Errorf("expected reason=Expired, got %v", records[0].reason)
	}
}

func TestEvictCallbackRemoved(t *testing.T) {
	type evictRecord struct {
		key    int
		value  int
		reason EvictReason
	}
	var records []evictRecord

	c := New[int, int](10, WithOnEvict(func(key, value int, reason EvictReason) {
		records = append(records, evictRecord{key, value, reason})
	}))

	c.Put(42, 100)
	c.Remove(42)

	if len(records) != 1 {
		t.Fatalf("expected 1 eviction, got %d", len(records))
	}
	if records[0].key != 42 || records[0].value != 100 {
		t.Errorf("expected evicted key=42 value=100, got key=%d value=%d", records[0].key, records[0].value)
	}
	if records[0].reason != EvictReasonRemoved {
		t.Errorf("expected reason=Removed, got %v", records[0].reason)
	}
}

func TestStatsHitRate(t *testing.T) {
	c := New[string, int](10)

	c.Put("a", 1)
	c.Put("b", 2)

	// 2 hits
	c.Get("a")
	c.Peek("b")

	// 2 misses (1 not found + 1 expired)
	c.Get("c")
	c.Put("d", 4, 10*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	c.Get("d") // expired → miss

	s := c.Stats()
	if s.Hits != 2 {
		t.Errorf("expected Hits=2, got %d", s.Hits)
	}
	if s.Misses != 2 {
		t.Errorf("expected Misses=2, got %d", s.Misses)
	}
	if s.Expirations != 1 {
		t.Errorf("expected Expirations=1, got %d", s.Expirations)
	}

	hr := s.HitRate()
	if hr < 0.49 || hr > 0.51 {
		t.Errorf("expected HitRate≈0.5, got %f", hr)
	}
}

func TestStatsReset(t *testing.T) {
	c := New[int, int](10)

	c.Put(1, 1)
	c.Get(1) // hit
	c.Get(2) // miss
	c.Remove(1)

	s := c.Stats()
	if s.Hits != 1 || s.Misses != 1 || s.Removals != 1 {
		t.Fatalf("before reset: unexpected stats %+v", s)
	}

	c.ResetStats()
	s = c.Stats()
	if s.Hits != 0 || s.Misses != 0 || s.Evictions != 0 || s.Expirations != 0 || s.Removals != 0 {
		t.Errorf("after reset: expected all zeros, got %+v", s)
	}
}

func TestStatsConcurrency(t *testing.T) {
	c := New[int, int](50)
	var wg sync.WaitGroup

	// 50 goroutine 写入
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Put(n, n)
		}(i)
	}
	wg.Wait()

	// 100 goroutine 混合读写 + 读统计
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Get(n)
			c.Get(n + 100) // miss
			c.Stats()
		}(i)
	}
	wg.Wait()

	s := c.Stats()
	total := s.Hits + s.Misses
	if total != 200 {
		t.Errorf("expected total accesses=200, got %d (hits=%d, misses=%d)", total, s.Hits, s.Misses)
	}
}

// --- Phase 4: 分片锁 + 批量操作 ---

func TestShardedBasicPutGet(t *testing.T) {
	sc := NewSharded[string, int](100, 4, nil)
	sc.Put("a", 1)
	sc.Put("b", 2)
	sc.Put("c", 3)

	if v, ok := sc.Get("a"); !ok || v != 1 {
		t.Fatalf("Get('a') = %v, %v, want 1, true", v, ok)
	}
	if v, ok := sc.Get("b"); !ok || v != 2 {
		t.Fatalf("Get('b') = %v, %v, want 2, true", v, ok)
	}
	if _, ok := sc.Get("x"); ok {
		t.Fatal("Get('x') should return false")
	}
}

func TestShardedEviction(t *testing.T) {
	// 每个分片容量 1，4 个分片，共 4 个 entry
	sc := NewSharded[string, int](4, 4, nil)

	// 同一个 key 反复 Put，只占一个分片的一个槽位
	sc.Put("a", 1)
	sc.Put("a", 2) // 更新，不增加 size

	if sc.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", sc.Len())
	}
	if v, ok := sc.Get("a"); !ok || v != 2 {
		t.Fatalf("Get('a') = %v, %v, want 2, true", v, ok)
	}
}

func TestShardedLen(t *testing.T) {
	sc := NewSharded[int, int](100, 4, nil)

	if sc.Len() != 0 {
		t.Fatalf("empty Len() = %d, want 0", sc.Len())
	}

	for i := 0; i < 20; i++ {
		sc.Put(i, i*10)
	}

	if sc.Len() != 20 {
		t.Fatalf("Len() = %d, want 20", sc.Len())
	}

	sc.Remove(5)
	if sc.Len() != 19 {
		t.Fatalf("after Remove Len() = %d, want 19", sc.Len())
	}
}

func TestShardedRemove(t *testing.T) {
	sc := NewSharded[string, int](100, 4, nil)
	sc.Put("x", 42)

	if ok := sc.Remove("x"); !ok {
		t.Fatal("Remove('x') should return true")
	}
	if ok := sc.Remove("x"); ok {
		t.Fatal("Remove('x') again should return false")
	}
	if _, ok := sc.Get("x"); ok {
		t.Fatal("Get('x') should return false after Remove")
	}
}

func TestShardedPeek(t *testing.T) {
	sc := NewSharded[string, int](100, 4, nil)
	sc.Put("k", 99)

	v, ok := sc.Peek("k")
	if !ok || v != 99 {
		t.Fatalf("Peek('k') = %v, %v, want 99, true", v, ok)
	}
}

func TestShardedConcurrent(t *testing.T) {
	sc := NewSharded[int, int](1000, 16, nil)
	var wg sync.WaitGroup

	// 100 个 goroutine 并发写入
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sc.Put(n, n*10)
		}(i)
	}
	wg.Wait()

	// 100 个 goroutine 并发读取
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sc.Get(n)
		}(i)
	}
	wg.Wait()

	// 100 个 goroutine 混合操作
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sc.Put(n+200, n)
			sc.Get(n + 200)
			sc.Peek(n + 200)
			sc.Remove(n + 200)
			sc.Len()
			sc.Stats()
		}(i)
	}
	wg.Wait()
}

func TestShardedStats(t *testing.T) {
	sc := NewSharded[int, int](100, 4, nil)
	sc.Put(1, 10)
	sc.Put(2, 20)

	sc.Get(1) // hit
	sc.Get(2) // hit
	sc.Get(3) // miss

	s := sc.Stats()
	if s.Hits != 2 {
		t.Errorf("Hits = %d, want 2", s.Hits)
	}
	if s.Misses != 1 {
		t.Errorf("Misses = %d, want 1", s.Misses)
	}
	if s.Size != 2 {
		t.Errorf("Size = %d, want 2", s.Size)
	}
}

func TestShardedResetStats(t *testing.T) {
	sc := NewSharded[int, int](100, 4, nil)
	sc.Put(1, 10)
	sc.Get(1)
	sc.Get(999)

	sc.ResetStats()
	s := sc.Stats()
	if s.Hits != 0 || s.Misses != 0 {
		t.Errorf("after reset: expected zeros, got %+v", s)
	}
}

func TestShardedCustomHasher(t *testing.T) {
	// 自定义哈希：所有 key 路由到分片 0
	alwaysZero := func(_ string) uint64 { return 0 }
	sc := NewSharded[string, int](100, 4, alwaysZero)

	sc.Put("a", 1)
	sc.Put("b", 2)

	if v, ok := sc.Get("a"); !ok || v != 1 {
		t.Fatalf("Get('a') = %v, %v, want 1, true", v, ok)
	}
	if v, ok := sc.Get("b"); !ok || v != 2 {
		t.Fatalf("Get('b') = %v, %v, want 2, true", v, ok)
	}
}

func TestShardedDefaultShardCount(t *testing.T) {
	// shardCnt <= 0 应使用默认值
	sc := NewSharded[string, int](100, 0, nil)
	sc.Put("a", 1)
	if v, ok := sc.Get("a"); !ok || v != 1 {
		t.Fatalf("Get('a') = %v, %v, want 1, true", v, ok)
	}

	sc2 := NewSharded[string, int](100, -1, nil)
	sc2.Put("b", 2)
	if v, ok := sc2.Get("b"); !ok || v != 2 {
		t.Fatalf("Get('b') = %v, %v, want 2, true", v, ok)
	}
}

func TestShardedCapacityPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("NewSharded with capacity 0 should panic")
		}
	}()
	NewSharded[string, int](0, 4, nil)
}

func TestShardedWithTTL(t *testing.T) {
	sc := NewSharded[string, int](100, 4, nil, WithTTL(50*time.Millisecond))
	sc.Put("k", 42)

	time.Sleep(100 * time.Millisecond)

	if _, ok := sc.Get("k"); ok {
		t.Fatal("Get('k') should return false after TTL expiration")
	}
}

func TestNextPowerOfTwo(t *testing.T) {
	tests := []struct {
		in, want int
	}{
		{0, 1}, {1, 1}, {2, 2}, {3, 4}, {4, 4},
		{5, 8}, {7, 8}, {8, 8}, {9, 16}, {16, 16},
		{17, 32}, {31, 32}, {32, 32}, {33, 64},
	}
	for _, tt := range tests {
		if got := nextPowerOfTwo(tt.in); got != tt.want {
			t.Errorf("nextPowerOfTwo(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
