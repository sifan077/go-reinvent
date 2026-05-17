package pool

import (
	"errors"
	"fmt"
	"time"
)

// ErrTimeout 是 WaitTimeout 超时时返回的哨兵错误。
var ErrTimeout = errors.New("pool: future wait timeout")

// ErrPoolStopped 是 SubmitFuture 在池已停止时返回的错误。
var ErrPoolStopped = errors.New("pool: pool is stopped")

// Future 表示一个异步任务的未来结果。
// 通过 SubmitFuture 创建，调用 Wait/WaitTimeout 阻塞获取结果，或通过 Done channel 监听完成事件。
type Future[T any] struct {
	done   chan struct{} // 关闭表示任务完成
	result T             // 任务返回值
	err    error         // 任务返回的 error
}

// Wait 阻塞直到任务完成，返回结果。
func (f *Future[T]) Wait() (T, error) {
	<-f.done
	return f.result, f.err
}

// WaitTimeout 阻塞等待，超时返回零值和 ErrTimeout。
func (f *Future[T]) WaitTimeout(d time.Duration) (T, error) {
	select {
	case <-f.done:
		return f.result, f.err
	case <-time.After(d):
		var zero T
		return zero, ErrTimeout
	}
}

// Done 返回任务完成的通知 channel，可配合 select 使用。
func (f *Future[T]) Done() <-chan struct{} {
	return f.done
}

// SubmitFuture 提交有返回值的任务，返回 Future。
// 独立泛型函数（Go 方法不支持自有类型参数）。
// 池已停止时返回一个已完成的 Future，携带 ErrPoolStopped 错误。
func SubmitFuture[T any](p *FixedPool, task func() (T, error)) *Future[T] {
	f := &Future[T]{done: make(chan struct{})}

	if p.stopped.Load() {
		f.err = ErrPoolStopped
		close(f.done)
		return f
	}

	p.Submit(func() {
		defer close(f.done)
		defer func() {
			if r := recover(); r != nil {
				f.err = fmt.Errorf("pool: task panicked: %v", r)
			}
		}()
		f.result, f.err = task()
	})

	return f
}
