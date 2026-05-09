package cache

import "testing"

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
	// 依次写入 1,2,3,4,5，验证淘汰顺序
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
