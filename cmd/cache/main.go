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
}

func printCache(c *cache.LRU[string, string], label string) {
	fmt.Printf("\n[%s] Len=%d Keys=%v\n", label, c.Len(), c.Keys())
	for _, k := range []string{"name", "age", "city", "email"} {
		if v, ok := c.Peek(k); ok {
			fmt.Printf("  %s = %s\n", k, v)
		}
	}
}
