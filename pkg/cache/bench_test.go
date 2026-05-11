package cache

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	hclru "github.com/hashicorp/golang-lru/v2"
)

const benchCapacity = 1000000

// ============================================================
// 公平对比：Our LRU vs HashiCorp golang-lru v2
// 两者均为 单分片 + sync.RWMutex，Get 操作均需排他锁（会移动链表节点）
// ============================================================

func BenchmarkOurLRU_PutSequential(b *testing.B) {
	c := New[int, int](benchCapacity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Put(i, i)
	}
}

func BenchmarkHashiCorp_PutSequential(b *testing.B) {
	c, _ := hclru.New[int, int](benchCapacity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Add(i, i)
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

func BenchmarkOurLRU_GetMiss(b *testing.B) {
	c := New[int, int](benchCapacity)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i)
	}
}

func BenchmarkHashiCorp_GetMiss(b *testing.B) {
	c, _ := hclru.New[int, int](benchCapacity)
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

// ============================================================
// 分片对比：HashiCorp golang-lru v2 没有内置分片实现，
// 此处展示分片在单线程下的额外开销和高并发下的吞吐提升。
// ============================================================

func BenchmarkOurSharded_PutSequential(b *testing.B) {
	c := NewSharded[int, int](benchCapacity, 16, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Put(i, i)
	}
}

func BenchmarkOurSharded_GetHit(b *testing.B) {
	c := NewSharded[int, int](benchCapacity, 16, nil)
	for i := 0; i < benchCapacity; i++ {
		c.Put(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i % benchCapacity)
	}
}

func BenchmarkOurSharded_GetMiss(b *testing.B) {
	c := NewSharded[int, int](benchCapacity, 16, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i)
	}
}

func BenchmarkOurSharded_Mixed(b *testing.B) {
	c := NewSharded[int, int](benchCapacity, 16, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%4 == 0 {
			c.Put(i, i)
		} else {
			c.Get(i % benchCapacity)
		}
	}
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

// ============================================================
// 对比报告
// ============================================================

func TestCompareReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skip report in short mode")
	}

	const n = 1000000
	const capacity = 1000000

	ours := New[int, int](capacity)
	hc, _ := hclru.New[int, int](capacity)

	for i := 0; i < n; i++ {
		ours.Put(i, i)
		hc.Add(i, i)
	}

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
	fmt.Println("公平对比（单分片 vs 单分片，均使用 sync.RWMutex）：")
	fmt.Println("  go test -bench='Benchmark(OurLRU|HashiCorp)' -benchmem -count=3")
	fmt.Println("")
	fmt.Println("分片开销与优势（单线程展示 hash 开销，并行展示锁分散收益）：")
	fmt.Println("  go test -bench='Benchmark(OurLRU|OurSharded)' -benchmem -count=3")
}
