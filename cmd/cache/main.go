package main

import (
	"fmt"
	"time"

	"go-reinvent/pkg/cache"
)

func main() {
	fmt.Println("=== LRU 缓存演示 ===")

	// 创建容量为 3 的缓存
	c := cache.New[string, string](3)

	// 写入 3 个元素
	c.Put("name", "张三")
	c.Put("age", "25")
	c.Put("city", "北京")
	printCache(c, "初始状态")

	// 访问 "name"，刷新其位置
	c.Get("name")
	printCache(c, "访问 name 后")

	// 写入第 4 个元素，"age" 应被淘汰（最久未使用）
	c.Put("email", "zhangsan@example.com")
	printCache(c, "写入 email 后（age 应被淘汰）")

	// 更新已存在的 key
	c.Put("city", "上海")
	printCache(c, "更新 city 为上海")

	// 手动删除
	c.Remove("name")
	printCache(c, "删除 name 后")

	// 演示不同类型
	fmt.Println("\n=== 整数类型缓存 ===")
	nums := cache.New[int, int](2)
	nums.Put(1, 100)
	nums.Put(2, 200)
	nums.Get(1)
	nums.Put(3, 300) // 2 应被淘汰

	for _, k := range []int{1, 2, 3} {
		if v, ok := nums.Get(k); ok {
			fmt.Printf("  Get(%d) = %d\n", k, v)
		} else {
			fmt.Printf("  Get(%d) = 未命中\n", k)
		}
	}

	// 演示 TTL 过期
	fmt.Println("\n=== TTL 过期演示 ===")
	tc := cache.New[string, string](10, cache.WithTTL(100*time.Millisecond))
	tc.Put("token", "abc123")
	tc.Put("session", "xyz789", 200*time.Millisecond) // 自定义 TTL

	fmt.Println("  写入 token(TTL=100ms) 和 session(TTL=200ms)")
	if v, ok := tc.Peek("token"); ok {
		fmt.Printf("  Peek(token) = %s\n", v)
	}

	time.Sleep(150 * time.Millisecond)
	fmt.Println("  等待 150ms...")

	if _, ok := tc.Peek("token"); !ok {
		fmt.Println("  Peek(token) = 已过期")
	}
	if v, ok := tc.Peek("session"); ok {
		fmt.Printf("  Peek(session) = %s（自定义 TTL，尚未过期）\n", v)
	}

	// 演示淘汰回调
	fmt.Println("\n=== 淘汰回调演示 ===")
	evictions := 0
	sc := cache.New[string, string](2, cache.WithOnEvict(func(key, value string, reason cache.EvictReason) {
		evictions++
		fmt.Printf("  [回调] key=%s value=%s reason=%s\n", key, value, reason)
	}))
	sc.Put("a", "1")
	sc.Put("b", "2")
	sc.Put("c", "3") // 容量淘汰 a
	sc.Remove("b")   // 手动删除
	fmt.Printf("  共触发 %d 次淘汰回调\n", evictions)

	// 演示访问统计
	fmt.Println("\n=== 访问统计演示 ===")
	pc := cache.New[string, int](10, cache.WithTTL(100*time.Millisecond))
	pc.Put("x", 100)
	pc.Put("y", 200)
	pc.Put("z", 300, 50*time.Millisecond)

	pc.Get("x") // hit
	pc.Get("y") // hit
	pc.Get("w") // miss
	time.Sleep(80 * time.Millisecond)
	pc.Get("z") // expired → miss

	s := pc.Stats()
	fmt.Printf("  Hits=%d Misses=%d Expirations=%d HitRate=%.0f%%\n",
		s.Hits, s.Misses, s.Expirations, s.HitRate()*100)

	pc.ResetStats()
	s = pc.Stats()
	fmt.Printf("  ResetStats 后: Hits=%d Misses=%d\n", s.Hits, s.Misses)

	// 演示分片缓存
	fmt.Println("\n=== 分片缓存演示 ===")
	sharded := cache.NewSharded[string, string](100, 8, nil,
		cache.WithTTL(5*time.Minute),
	)
	sharded.Put("user:1", "张三")
	sharded.Put("user:2", "李四")
	sharded.Put("user:3", "王五")
	fmt.Printf("  Len=%d\n", sharded.Len())
	if v, ok := sharded.Get("user:1"); ok {
		fmt.Printf("  Get(user:1) = %s\n", v)
	}

	// Stats
	s2 := sharded.Stats()
	fmt.Printf("  Stats: Hits=%d Misses=%d Size=%d\n", s2.Hits, s2.Misses, s2.Size)

	// 演示 Janitor 主动清理
	fmt.Println("\n=== Janitor 主动清理演示 ===")
	jc := cache.New[string, string](10,
		cache.WithTTL(100*time.Millisecond),
		cache.WithJanitorInterval(200*time.Millisecond), // 每 200ms 清理一次
		cache.WithOnEvict(func(key, value string, reason cache.EvictReason) {
			fmt.Printf("  [淘汰] key=%s value=%s reason=%s\n", key, value, reason)
		}),
	)

	jc.Put("token:1", "abc")
	jc.Put("token:2", "def")
	jc.Put("token:3", "ghi")
	fmt.Printf("  写入 3 个 key（TTL=100ms），Len=%d\n", jc.Len())

	time.Sleep(150 * time.Millisecond)
	fmt.Println("  等待 150ms... key 已过期但尚未清理")
	fmt.Printf("  Len=%d（惰性删除：过期 key 仍占位）\n", jc.Len())

	time.Sleep(150 * time.Millisecond) // 等待 janitor 清理周期
	fmt.Println("  再等待 150ms... janitor 已清理过期 key")
	fmt.Printf("  Len=%d\n", jc.Len())

	// 关闭 janitor
	jc.Close()
	fmt.Println("  Close() 后 janitor 停止，缓存仍可读写")
	jc.Put("new", "value")
	fmt.Printf("  Put('new', 'value')，Len=%d\n", jc.Len())

	// 接口多态演示
	fmt.Println("\n=== 接口多态演示 ===")

	// LRU 实现
	var ic cache.Cache[string, int] = cache.New[string, int](10)
	ic.Put("key", 42)
	fmt.Printf("  LRU: Get('key') = %d\n", mustGetInt(ic.Get("key")))
	ic.Close()

	// ShardedCache 实现
	ic = cache.NewSharded[string, int](100, 4, nil)
	ic.Put("key", 99)
	fmt.Printf("  Sharded: Get('key') = %d\n", mustGetInt(ic.Get("key")))
	ic.Close()

	fmt.Println("\n=== 演示结束 ===")
}

func mustGetInt(v int, ok bool) int {
	if !ok {
		return 0
	}
	return v
}

func printCache(c *cache.LRU[string, string], label string) {
	fmt.Printf("\n[%s] Len=%d Keys=%v\n", label, c.Len(), c.Keys())
	for _, k := range []string{"name", "age", "city", "email"} {
		if v, ok := c.Peek(k); ok {
			fmt.Printf("  %s = %s\n", k, v)
		}
	}
}
