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
	c.Put("short", 1)             // 使用全局 TTL 50ms
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
