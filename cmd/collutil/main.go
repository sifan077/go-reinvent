package main

import (
	"fmt"
	"go-reinvent/pkg/collutil"
	"strings"
)

func main() {
	fmt.Println("=== 泛型集合工具演示 ===")

	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// Filter / Reject
	fmt.Println("\n--- 过滤 ---")
	evens := collutil.Filter(nums, func(n int) bool { return n%2 == 0 })
	fmt.Printf("偶数: %v\n", evens)
	odds := collutil.Reject(nums, func(n int) bool { return n%2 == 0 })
	fmt.Printf("奇数: %v\n", odds)

	// Find
	fmt.Println("\n--- 查找 ---")
	found, ok := collutil.Find(nums, func(n int) bool { return n > 5 })
	fmt.Printf("第一个大于5的数: %d (found=%v)\n", found, ok)

	// Every / Some
	fmt.Println("\n--- 判断 ---")
	fmt.Printf("所有数都大于0: %v\n", collutil.Every(nums, func(n int) bool { return n > 0 }))
	fmt.Printf("存在大于8的数: %v\n", collutil.Some(nums, func(n int) bool { return n > 8 }))

	// Map / Reduce
	fmt.Println("\n--- 映射/归约 ---")
	doubled := collutil.Map(nums[:5], func(n int) int { return n * 2 })
	fmt.Printf("翻倍: %v\n", doubled)
	sum := collutil.Reduce(nums, 0, func(acc, n int) int { return acc + n })
	fmt.Printf("求和: %d\n", sum)

	// ForEach
	fmt.Println("\n--- 遍历 ---")
	var parts []string
	collutil.ForEach([]string{"Go", "泛型", "真好用"}, func(s string) {
		parts = append(parts, strings.ToUpper(s))
	})
	fmt.Printf("大写: %v\n", parts)

	// Unique
	fmt.Println("\n--- 去重 ---")
	duped := []int{1, 2, 2, 3, 3, 3, 4}
	fmt.Printf("去重前: %v → 去重后: %v\n", duped, collutil.Unique(duped))

	// Contains
	fmt.Println("\n--- 包含 ---")
	fmt.Printf("包含3: %v\n", collutil.Contains(nums, 3))
	fmt.Printf("包含99: %v\n", collutil.Contains(nums, 99))

	// 集合运算
	fmt.Println("\n--- 集合运算 ---")
	a := []int{1, 2, 3, 4, 5}
	b := []int{4, 5, 6, 7, 8}
	fmt.Printf("A: %v\n", a)
	fmt.Printf("B: %v\n", b)
	fmt.Printf("交集: %v\n", collutil.Intersect(a, b))
	fmt.Printf("并集: %v\n", collutil.Union(a, b))
	fmt.Printf("A-B 差集: %v\n", collutil.Diff(a, b))

	// GroupBy
	fmt.Println("\n--- 分组 ---")
	grouped := collutil.GroupBy(nums, func(n int) string {
		if n%2 == 0 {
			return "偶数"
		}
		return "奇数"
	})
	for k, v := range grouped {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// Chunk
	fmt.Println("\n--- 分块 ---")
	chunks := collutil.Chunk([]int{1, 2, 3, 4, 5, 6, 7}, 3)
	fmt.Printf("每3个一块: %v\n", chunks)

	// Flatten
	fmt.Println("\n--- 展平 ---")
	matrix := [][]int{{1, 2}, {3, 4}, {5}}
	fmt.Printf("展平 %v → %v\n", matrix, collutil.Flatten(matrix))

	// ToMap
	fmt.Println("\n--- 转Map ---")
	type User struct {
		ID   int
		Name string
	}
	users := []User{{1, "Alice"}, {2, "Bob"}, {3, "Charlie"}}
	userMap := collutil.ToMap(users, func(u User) (int, User) { return u.ID, u })
	for id, u := range userMap {
		fmt.Printf("  ID=%d → %s\n", id, u.Name)
	}
}
