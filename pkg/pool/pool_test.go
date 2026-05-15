package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ==================== 阶段一：固定大小协程池测试 ====================

// TestPool_Submit 验证基本的任务提交和执行。
func TestPool_Submit(t *testing.T) {
	p := New(5)
	defer p.Stop()

	var count atomic.Int32
	n := 100

	for i := 0; i < n; i++ {
		p.Submit(func() {
			count.Add(1)
		})
	}

	// 等待所有任务完成
	p.Stop()

	if int(count.Load()) != n {
		t.Fatalf("期望执行 %d 个任务，实际执行 %d", n, count.Load())
	}
}

// TestPool_ConcurrencyLimit 验证并发执行的 worker 数不超过池大小。
func TestPool_ConcurrencyLimit(t *testing.T) {
	size := 3
	p := New(size)
	defer p.Stop()

	var maxConcurrent atomic.Int32
	var current atomic.Int32
	var mu sync.Mutex

	n := 50
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		p.Submit(func() {
			cur := current.Add(1)
			mu.Lock()
			if cur > maxConcurrent.Load() {
				maxConcurrent.Store(cur)
			}
			mu.Unlock()

			time.Sleep(10 * time.Millisecond)
			current.Add(-1)
			wg.Done()
		})
	}

	wg.Wait()

	if maxConcurrent.Load() > int32(size) {
		t.Fatalf("最大并发数 %d 超过池大小 %d", maxConcurrent.Load(), size)
	}
}

// TestPool_TrySubmit 验证非阻塞提交：队列未满时成功，队列满时返回 false。
func TestPool_TrySubmit(t *testing.T) {
	// 创建一个很小的池，队列也小
	p := New(1, WithQueueSize(1))
	defer p.StopNow()

	// 阻塞唯一的 worker，让它一直忙
	started := make(chan struct{})
	p.Submit(func() {
		close(started)
		time.Sleep(2 * time.Second)
	})
	<-started

	// 队列还有空间，应该成功
	if !p.TrySubmit(func() {}) {
		t.Fatal("队列未满时 TrySubmit 应该成功")
	}

	// 队列已满，应该失败
	if p.TrySubmit(func() {}) {
		t.Fatal("队列满时 TrySubmit 应该返回 false")
	}
}

// TestPool_Stop 验证优雅停止：等待所有已提交任务完成后退出，之后提交的任务被忽略。
func TestPool_Stop(t *testing.T) {
	p := New(3)

	var count atomic.Int32
	n := 30

	for i := 0; i < n; i++ {
		p.Submit(func() {
			time.Sleep(10 * time.Millisecond)
			count.Add(1)
		})
	}

	p.Stop()

	// Stop 后所有已提交的任务应该执行完毕
	if int(count.Load()) != n {
		t.Fatalf("Stop 后期望执行 %d 个任务，实际执行 %d", n, count.Load())
	}

	// Stop 后提交的任务应该被忽略
	p.Submit(func() {
		t.Fatal("Stop 后不应执行新任务")
	})
}

// TestPool_StopNow 验证立即停止：不等待队列中剩余任务，快速返回。
func TestPool_StopNow(t *testing.T) {
	p := New(1)

	var count atomic.Int32

	// 阻塞唯一的 worker
	started := make(chan struct{})
	p.Submit(func() {
		close(started)
		time.Sleep(5 * time.Second)
	})
	<-started

	// 提交更多任务到队列
	for i := 0; i < 10; i++ {
		p.TrySubmit(func() {
			count.Add(1)
		})
	}

	p.StopNow()

	// 队列中的任务不保证执行
	// 但 StopNow 应该快速返回，不等 5 秒
}

// TestPool_Running 验证 Running() 返回当前正在执行任务的 worker 数。
func TestPool_Running(t *testing.T) {
	p := New(3)
	defer p.Stop()

	// 初始状态没有任务在执行
	if p.Running() != 0 {
		t.Fatalf("初始 Running 应为 0，实际为 %d", p.Running())
	}

	started := make(chan struct{})
	block := make(chan struct{})

	// 提交一个会阻塞的任务
	p.Submit(func() {
		close(started)
		<-block
	})

	<-started
	time.Sleep(10 * time.Millisecond) // 等待 running 计数更新

	if p.Running() != 1 {
		t.Fatalf("Running 应为 1，实际为 %d", p.Running())
	}

	// 释放任务
	close(block)
	time.Sleep(10 * time.Millisecond)

	if p.Running() != 0 {
		t.Fatalf("任务完成后 Running 应为 0，实际为 %d", p.Running())
	}
}

// TestPool_Waiting 验证 Waiting() 返回等待队列中的任务数。
func TestPool_Waiting(t *testing.T) {
	p := New(1, WithQueueSize(10))
	defer p.StopNow()

	// 阻塞唯一的 worker
	started := make(chan struct{})
	p.Submit(func() {
		close(started)
		time.Sleep(5 * time.Second)
	})
	<-started

	// 提交 5 个任务到队列（worker 被阻塞，任务会堆积在队列中）
	for i := 0; i < 5; i++ {
		p.Submit(func() {
			time.Sleep(100 * time.Millisecond)
		})
	}

	time.Sleep(10 * time.Millisecond)
	w := p.Waiting()
	// worker 被阻塞时，队列中应有 4-5 个任务（取决于时序）
	if w < 4 || w > 5 {
		t.Fatalf("Waiting 应约为 4-5，实际为 %d", w)
	}
}

// TestPool_LargeTaskVolume 验证大量任务提交不丢任务。
func TestPool_LargeTaskVolume(t *testing.T) {
	p := New(10)
	defer p.Stop()

	var count atomic.Int32
	n := 10000

	for i := 0; i < n; i++ {
		p.Submit(func() {
			count.Add(1)
		})
	}

	p.Stop()

	if int(count.Load()) != n {
		t.Fatalf("期望执行 %d 个任务，实际执行 %d", n, count.Load())
	}
}

// TestPool_ConcurrentSubmit 验证多 goroutine 同时 Submit 的并发安全性。
func TestPool_ConcurrentSubmit(t *testing.T) {
	p := New(10)
	defer p.Stop()

	var count atomic.Int32
	n := 1000
	goroutines := 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < n/goroutines; i++ {
				p.Submit(func() {
					count.Add(1)
				})
			}
		}()
	}

	wg.Wait()
	p.Stop()

	if int(count.Load()) != n {
		t.Fatalf("期望执行 %d 个任务，实际执行 %d", n, count.Load())
	}
}

// TestPool_InterfaceCompliance 验证 FixedPool 实现了 Pool 接口。
func TestPool_InterfaceCompliance(t *testing.T) {
	var _ Pool = &FixedPool{}
	var _ Pool = New(1)
}

// TestNew_PanicOnInvalidSize 验证 size <= 0 时 New 会 panic。
func TestNew_PanicOnInvalidSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("size <= 0 应该 panic")
		}
	}()
	New(0)
}

// TestPool_SubmitAfterStop 验证 Stop 后 Submit 不会阻塞。
func TestPool_SubmitAfterStop(t *testing.T) {
	p := New(2)
	p.Stop()

	done := make(chan struct{})
	go func() {
		p.Submit(func() {})
		close(done)
	}()

	select {
	case <-done:
		// 正常返回
	case <-time.After(time.Second):
		t.Fatal("Stop 后 Submit 阻塞了")
	}
}

// TestPool_TrySubmitAfterStop 验证 Stop 后 TrySubmit 返回 false。
func TestPool_TrySubmitAfterStop(t *testing.T) {
	p := New(2)
	p.Stop()

	if p.TrySubmit(func() {}) {
		t.Fatal("Stop 后 TrySubmit 应返回 false")
	}
}

// TestPool_Size 验证 Size() 返回配置的 worker 数量。
func TestPool_Size(t *testing.T) {
	p := New(7)
	defer p.Stop()

	if p.Size() != 7 {
		t.Fatalf("Size 应为 7，实际为 %d", p.Size())
	}
}

// TestPool_StopIdempotent 验证多次调用 Stop 不会 panic。
func TestPool_StopIdempotent(t *testing.T) {
	p := New(2)

	var count atomic.Int32
	for i := 0; i < 10; i++ {
		p.Submit(func() {
			count.Add(1)
		})
	}

	// 多次 Stop 不应 panic
	p.Stop()
	p.Stop()
	p.Stop()
}

// TestPool_StopNowIdempotent 验证多次调用 StopNow 不会 panic。
func TestPool_StopNowIdempotent(t *testing.T) {
	p := New(2)

	p.Submit(func() {
		time.Sleep(time.Second)
	})

	p.StopNow()
	p.StopNow()
	p.StopNow()
}
