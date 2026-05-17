package pool

import (
	"context"
	"errors"
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

// ==================== 阶段二：Panic 捕获 + 任务超时测试 ====================

// TestPool_PanicDoesNotAffectPool 验证任务 panic 后池继续正常工作。
func TestPool_PanicDoesNotAffectPool(t *testing.T) {
	p := New(3)
	defer p.Stop()

	// 先提交一个会 panic 的任务
	p.Submit(func() {
		panic("boom")
	})

	// 再提交正常任务，验证池仍然可用
	var count atomic.Int32
	var wg sync.WaitGroup
	n := 50
	wg.Add(n)
	for i := 0; i < n; i++ {
		p.Submit(func() {
			count.Add(1)
			wg.Done()
		})
	}

	wg.Wait()
	if int(count.Load()) != n {
		t.Fatalf("panic 后期望执行 %d 个任务，实际执行 %d", n, count.Load())
	}
}

// TestPool_PanicHandler 验证 PanicHandler 被正确调用。
func TestPool_PanicHandler(t *testing.T) {
	var handlerCalled atomic.Bool
	var handlerValue atomic.Value

	p := New(2, WithPanicHandler(func(r any) {
		handlerCalled.Store(true)
		handlerValue.Store(r)
	}))
	defer p.Stop()

	p.Submit(func() {
		panic("test panic")
	})

	// 等待 panic 被捕获
	time.Sleep(50 * time.Millisecond)

	if !handlerCalled.Load() {
		t.Fatal("PanicHandler 应该被调用")
	}
	if handlerValue.Load() != "test panic" {
		t.Fatalf("PanicHandler 参数应为 'test panic'，实际为 %v", handlerValue.Load())
	}
}

// TestPool_PanicHandlerWithDifferentTypes 验证不同类型的 panic 值都能被捕获。
func TestPool_PanicHandlerWithDifferentTypes(t *testing.T) {
	var values []any
	var mu sync.Mutex

	p := New(2, WithPanicHandler(func(r any) {
		mu.Lock()
		values = append(values, r)
		mu.Unlock()
	}))
	defer p.Stop()

	p.Submit(func() { panic("string") })
	p.Submit(func() { panic(42) })
	p.Submit(func() { panic(errors.New("error")) })

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(values) != 3 {
		t.Fatalf("期望 3 次 panic，实际 %d 次", len(values))
	}
}

// TestPool_SubmitWithTimeout_NormalCompletion 验证任务在超时内完成则正常结束。
func TestPool_SubmitWithTimeout_NormalCompletion(t *testing.T) {
	p := New(3)
	defer p.Stop()

	done := make(chan struct{})
	ok := p.SubmitWithTimeout(func(ctx context.Context) {
		// 任务在超时前完成
		close(done)
	}, time.Second)

	if !ok {
		t.Fatal("SubmitWithTimeout 应返回 true")
	}

	select {
	case <-done:
		// 正常完成
	case <-time.After(500 * time.Millisecond):
		t.Fatal("任务应在超时前完成")
	}
}

// TestPool_SubmitWithTimeout_Timeout 验证超时后 context 被取消。
func TestPool_SubmitWithTimeout_Timeout(t *testing.T) {
	p := New(2)
	defer p.Stop()

	ctxCancelled := make(chan struct{})
	p.SubmitWithTimeout(func(ctx context.Context) {
		// 等待 ctx 被取消
		<-ctx.Done()
		close(ctxCancelled)
	}, 50*time.Millisecond)

	select {
	case <-ctxCancelled:
		// 超时后 ctx 被取消
	case <-time.After(time.Second):
		t.Fatal("context 应在超时后被取消")
	}
}

// TestPool_SubmitWithTimeout_TaskListensToCtx 验证任务可以通过 ctx.Done() 检测超时并提前退出。
func TestPool_SubmitWithTimeout_TaskListensToCtx(t *testing.T) {
	p := New(2)
	defer p.Stop()

	taskDone := make(chan struct{})
	p.SubmitWithTimeout(func(ctx context.Context) {
		select {
		case <-ctx.Done():
			// 超时退出
		case <-time.After(10 * time.Second):
			// 不应该走到这里
		}
		close(taskDone)
	}, 100*time.Millisecond)

	select {
	case <-taskDone:
		// 任务因超时退出
	case <-time.After(time.Second):
		t.Fatal("任务应因超时在 1 秒内退出")
	}
}

// TestPool_SubmitWithTimeout_MultipleConcurrent 验证多个超时任务并发执行互不影响。
func TestPool_SubmitWithTimeout_MultipleConcurrent(t *testing.T) {
	p := New(5)
	defer p.Stop()

	var completed atomic.Int32
	var wg sync.WaitGroup
	n := 20

	for i := 0; i < n; i++ {
		wg.Add(1)
		timeout := 50*time.Millisecond + time.Duration(i*10)*time.Millisecond
		p.SubmitWithTimeout(func(ctx context.Context) {
			defer wg.Done()
			<-ctx.Done()
			completed.Add(1)
		}, timeout)
	}

	wg.Wait()

	if int(completed.Load()) != n {
		t.Fatalf("期望 %d 个任务完成，实际 %d", n, completed.Load())
	}
}

// TestPool_SubmitWithTimeout_PoolStopped 验证池停止后 SubmitWithTimeout 返回 false。
func TestPool_SubmitWithTimeout_PoolStopped(t *testing.T) {
	p := New(2)
	p.Stop()

	ok := p.SubmitWithTimeout(func(ctx context.Context) {}, time.Second)
	if ok {
		t.Fatal("池停止后 SubmitWithTimeout 应返回 false")
	}
}

// TestPool_SubmitWithTimeout_PanicInTask 验证超时任务中 panic 被捕获。
func TestPool_SubmitWithTimeout_PanicInTask(t *testing.T) {
	var handlerCalled atomic.Bool
	p := New(2, WithPanicHandler(func(r any) {
		handlerCalled.Store(true)
	}))
	defer p.Stop()

	p.SubmitWithTimeout(func(ctx context.Context) {
		panic("timeout task panic")
	}, time.Second)

	time.Sleep(50 * time.Millisecond)
	if !handlerCalled.Load() {
		t.Fatal("超时任务中的 panic 也应被捕获")
	}
}

// ==================== 阶段三：Future 模式测试 ====================

// TestSubmitFuture_基本结果 验证 SubmitFuture 正确返回结果。
func TestSubmitFuture_基本结果(t *testing.T) {
	p := New(3)
	defer p.Stop()

	f := SubmitFuture(p, func() (string, error) {
		return "hello", nil
	})

	result, err := f.Wait()
	if err != nil {
		t.Fatalf("期望无 error，实际: %v", err)
	}
	if result != "hello" {
		t.Fatalf("期望 'hello'，实际: %q", result)
	}
}

// TestSubmitFuture_错误传递 验证 task 返回 error 时 Future 携带 error。
func TestSubmitFuture_错误传递(t *testing.T) {
	p := New(3)
	defer p.Stop()

	myErr := errors.New("task failed")
	f := SubmitFuture(p, func() (int, error) {
		return 0, myErr
	})

	result, err := f.Wait()
	if !errors.Is(err, myErr) {
		t.Fatalf("期望 error %v，实际: %v", myErr, err)
	}
	if result != 0 {
		t.Fatalf("期望零值 0，实际: %d", result)
	}
}

// TestFuture_Wait_阻塞 验证 Wait 阻塞直到任务完成。
func TestFuture_Wait_阻塞(t *testing.T) {
	p := New(2)
	defer p.Stop()

	started := make(chan struct{})
	f := SubmitFuture(p, func() (string, error) {
		close(started)
		time.Sleep(100 * time.Millisecond)
		return "done", nil
	})

	<-started
	// Wait 应该阻塞直到任务完成
	select {
	case <-f.Done():
		// 任务已完成
	case <-time.After(time.Second):
		t.Fatal("Wait 应在 1 秒内返回")
	}

	result, _ := f.Wait()
	if result != "done" {
		t.Fatalf("期望 'done'，实际: %q", result)
	}
}

// TestFuture_WaitTimeout_超时 验证超时返回 ErrTimeout。
func TestFuture_WaitTimeout_超时(t *testing.T) {
	p := New(1)
	defer p.Stop()

	// 阻塞唯一的 worker
	started := make(chan struct{})
	p.Submit(func() {
		close(started)
		time.Sleep(5 * time.Second)
	})
	<-started

	// 提交一个 Future，队列满时会阻塞
	f := SubmitFuture(p, func() (int, error) {
		return 42, nil
	})

	_, err := f.WaitTimeout(50 * time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("期望 ErrTimeout，实际: %v", err)
	}
}

// TestFuture_WaitTimeout_正常完成 验证未超时时正常返回。
func TestFuture_WaitTimeout_正常完成(t *testing.T) {
	p := New(3)
	defer p.Stop()

	f := SubmitFuture(p, func() (string, error) {
		return "fast", nil
	})

	result, err := f.WaitTimeout(time.Second)
	if err != nil {
		t.Fatalf("期望无 error，实际: %v", err)
	}
	if result != "fast" {
		t.Fatalf("期望 'fast'，实际: %q", result)
	}
}

// TestFuture_Done_channel 验证 Done channel 可配合 select 使用。
func TestFuture_Done_channel(t *testing.T) {
	p := New(3)
	defer p.Stop()

	f := SubmitFuture(p, func() (int, error) {
		return 99, nil
	})

	select {
	case <-f.Done():
		result, err := f.Wait()
		if err != nil {
			t.Fatalf("期望无 error，实际: %v", err)
		}
		if result != 99 {
			t.Fatalf("期望 99，实际: %d", result)
		}
	case <-time.After(time.Second):
		t.Fatal("Done channel 应在 1 秒内关闭")
	}
}

// TestSubmitFuture_并发多任务 验证多个 Future 并发执行互不影响。
func TestSubmitFuture_并发多任务(t *testing.T) {
	p := New(5)
	defer p.Stop()

	n := 50
	futures := make([]*Future[int], n)
	for i := 0; i < n; i++ {
		i := i
		futures[i] = SubmitFuture(p, func() (int, error) {
			return i * 2, nil
		})
	}

	for i, f := range futures {
		result, err := f.Wait()
		if err != nil {
			t.Fatalf("future[%d] 期望无 error，实际: %v", i, err)
		}
		if result != i*2 {
			t.Fatalf("future[%d] 期望 %d，实际: %d", i, i*2, result)
		}
	}
}

// TestSubmitFuture_池已停止 验证池已停止时 SubmitFuture 返回已完成的 Future 携带错误。
func TestSubmitFuture_池已停止(t *testing.T) {
	p := New(2)
	p.Stop()

	f := SubmitFuture(p, func() (string, error) {
		t.Fatal("任务不应被执行")
		return "", nil
	})

	select {
	case <-f.Done():
		// 已完成
	case <-time.After(time.Second):
		t.Fatal("池已停止时 Future 应立即完成")
	}

	_, err := f.Wait()
	if !errors.Is(err, ErrPoolStopped) {
		t.Fatalf("期望 ErrPoolStopped，实际: %v", err)
	}
}

// TestSubmitFuture_Panic恢复 验证任务 panic 时 Future 不会永久阻塞。
func TestSubmitFuture_Panic恢复(t *testing.T) {
	p := New(3)
	defer p.Stop()

	f := SubmitFuture(p, func() (string, error) {
		panic("future panic")
	})

	result, err := f.Wait()
	if err == nil {
		t.Fatal("期望有 error，实际: nil")
	}
	if result != "" {
		t.Fatalf("期望零值，实际: %q", result)
	}
	// 池应继续正常工作
	p.Stop()
}
