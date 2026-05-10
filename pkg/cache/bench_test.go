package cache

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	hclru "github.com/hashicorp/golang-lru/v2"
)

const benchCapacity = 1000000

// --- 我们的 LRU ---

func BenchmarkOurLRU_PutSequential(b *testing.B) {
	c := New[int, int](benchCapacity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Put(i, i)
	}
}

func BenchmarkOurLRU_GetHit(b *testing.B) {
	c := New[int, int](benchCapacity)
	for i := 0; i < benchCapacity; i++ {
		c.Put(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i % benchCapacity)
	}
}

func BenchmarkOurLRU_GetMiss(b *testing.B) {
	c := New[int, int](benchCapacity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i)
	}
}

func BenchmarkOurLRU_Mixed(b *testing.B) {
	c := New[int, int](benchCapacity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%4 == 0 {
			c.Put(i, i)
		} else {
			c.Get(i % benchCapacity)
		}
	}
}

func BenchmarkOurLRU_Parallel(b *testing.B) {
	c := New[int, int](benchCapacity)
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(benchCapacity)
		for pb.Next() {
			if i%4 == 0 {
				c.Put(i, i)
			} else {
				c.Get(i % benchCapacity)
			}
			i++
		}
	})
}

func BenchmarkOurSharded_Parallel(b *testing.B) {
	c := NewSharded[int, int](benchCapacity, 16, nil)
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(benchCapacity)
		for pb.Next() {
			if i%4 == 0 {
				c.Put(i, i)
			} else {
				c.Get(i % benchCapacity)
			}
			i++
		}
	})
}

// --- HashiCorp golang-lru ---

func BenchmarkHashiCorp_PutSequential(b *testing.B) {
	c, _ := hclru.New[int, int](benchCapacity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Add(i, i)
	}
}

func BenchmarkHashiCorp_GetHit(b *testing.B) {
	c, _ := hclru.New[int, int](benchCapacity)
	for i := 0; i < benchCapacity; i++ {
		c.Add(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i % benchCapacity)
	}
}

func BenchmarkHashiCorp_GetMiss(b *testing.B) {
	c, _ := hclru.New[int, int](benchCapacity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i)
	}
}

func BenchmarkHashiCorp_Mixed(b *testing.B) {
	c, _ := hclru.New[int, int](benchCapacity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%4 == 0 {
			c.Add(i, i)
		} else {
			c.Get(i % benchCapacity)
		}
	}
}

func BenchmarkHashiCorp_Parallel(b *testing.B) {
	c, _ := hclru.New[int, int](benchCapacity)
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Intn(benchCapacity)
		for pb.Next() {
			if i%4 == 0 {
				c.Add(i, i)
			} else {
				c.Get(i % benchCapacity)
			}
			i++
		}
	})
}

// --- 对比报告 ---

func TestCompareReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skip report in short mode")
	}

	const n = 1000000
	const capacity = 1000000

	// 我们的 LRU
	ours := New[int, int](capacity)

	// HashiCorp
	hc, _ := hclru.New[int, int](capacity)

	// 顺序写入
	for i := 0; i < n; i++ {
		ours.Put(i, i)
		hc.Add(i, i)
	}

	// 顺序读取
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			ours.Get(i % capacity)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			hc.Get(i % capacity)
		}
	}()

	wg.Wait()

	fmt.Println("=== 对比测试完成 ===")
	fmt.Println("运行 benchmark 查看详细性能：")
	fmt.Println("  go test -bench=BenchmarkOur -benchmem -count=3")
	fmt.Println("  go test -bench=BenchmarkHashiCorp -benchmem -count=3")
}
