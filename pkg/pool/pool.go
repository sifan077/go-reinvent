// Package pool 实现协程池，用于控制并发 goroutine 数量，避免无限制创建导致资源耗尽。
// 核心设计：预先启动固定数量的 worker goroutine，通过有缓冲 channel 分发任务。
// worker 常驻不退出，避免 goroutine 反复创建销毁的开销；任务 panic 由 worker 捕获，不影响池整体运行。
package pool

import (
	"sync"
	"sync/atomic"
)

// Pool 是协程池的通用接口。
type Pool interface {
	// Submit 提交任务，池满时阻塞等待。
	Submit(task func())

	// TrySubmit 非阻塞提交，池满时返回 false。
	TrySubmit(task func()) bool

	// Stop 优雅停止：等待队列中已提交的任务执行完毕后关闭。
	Stop()

	// StopNow 立即停止：不再执行队列中剩余任务。
	StopNow()

	// Running 返回当前正在执行任务的 worker 数。
	Running() int

	// Waiting 返回等待队列中的任务数。
	Waiting() int

	// Size 返回池的 worker 容量。
	Size() int
}

// FixedPool 是固定大小的协程池实现。
//
// 内部维护 size 个常驻 worker goroutine，每个 worker 从 taskCh 中竞争获取任务。
// 任务通过 Submit/TrySubmit 提交到 taskCh，worker 取出后在 safeRun 中执行并捕获 panic。
// 停止时关闭 taskCh（优雅）或 quit 通道（立即），worker 检测到信号后自行退出。
type FixedPool struct {
	size         int            // 配置的 worker 数量
	taskCh       chan func()     // 任务队列
	wg           sync.WaitGroup // 等待 worker 退出
	quit         chan struct{}   // 通知 worker 停止
	stopped      atomic.Bool    // 是否已停止
	running      atomic.Int32   // 当前执行任务的 worker 数
	panicHandler func(r any)    // panic 回调
}

// 接口合规检查
var _ Pool = &FixedPool{}

// New 创建固定大小的协程池。
//
// size 是 worker 数量（并发上限），必须大于 0。
// 默认 queueSize = size * 2。
func New(size int, opts ...Option) *FixedPool {
	if size <= 0 {
		panic("pool: size must be positive")
	}

	cfg := applyOptions(opts)

	queueSize := cfg.queueSize
	if queueSize == 0 {
		queueSize = size * 2
	}

	p := &FixedPool{
		size:         size,
		taskCh:       make(chan func(), queueSize),
		quit:         make(chan struct{}),
		panicHandler: cfg.panicHandler,
	}

	// 启动 worker
	p.wg.Add(size)
	for i := 0; i < size; i++ {
		go p.worker()
	}

	return p
}

// worker 是 worker goroutine 的主循环。
// 每个 worker 持续从 taskCh 中取任务执行，直到收到 quit 信号或 taskCh 被关闭。
func (p *FixedPool) worker() {
	defer p.wg.Done()
	for {
		select {
		case task, ok := <-p.taskCh:
			if !ok {
				return
			}
			p.safeRun(task)
		case <-p.quit:
			return
		}
	}
}

// safeRun 在 panic 保护下执行任务。
// running 计数通过 defer 保证无论任务正常完成还是 panic 都能正确递减。
// panic 时调用用户注册的 panicHandler（如果有的话），否则静默吞掉。
func (p *FixedPool) safeRun(task func()) {
	p.running.Add(1)
	defer p.running.Add(-1)

	defer func() {
		if r := recover(); r != nil {
			if p.panicHandler != nil {
				p.panicHandler(r)
			}
		}
	}()

	task()
}

// Submit 提交任务，池满时阻塞等待。
func (p *FixedPool) Submit(task func()) {
	if p.stopped.Load() {
		return
	}
	p.taskCh <- task
}

// TrySubmit 非阻塞提交，池满或已停止时返回 false。
func (p *FixedPool) TrySubmit(task func()) bool {
	if p.stopped.Load() {
		return false
	}
	select {
	case p.taskCh <- task:
		return true
	default:
		return false
	}
}

// Stop 优雅停止：关闭任务队列，等待所有已提交的任务执行完毕后退出。
// Stop 是幂等的，多次调用不会 panic。
func (p *FixedPool) Stop() {
	if !p.stopped.CompareAndSwap(false, true) {
		return
	}
	// 关闭 taskCh：worker 消费完剩余任务后会收到 ok=false 从而退出
	close(p.taskCh)
	p.wg.Wait()
}

// StopNow 立即停止：通知所有 worker 立即退出，不等待队列中剩余任务。
// StopNow 是幂等的，多次调用不会 panic。
func (p *FixedPool) StopNow() {
	if !p.stopped.CompareAndSwap(false, true) {
		return
	}
	// 关闭 quit 通道，所有 worker 的 select 会收到通知并退出
	close(p.quit)
	// 排空 taskCh：防止有 goroutine 阻塞在发送端（如 Submit）
	go func() {
		for range p.taskCh {
		}
	}()
	p.wg.Wait()
}

// Running 返回当前正在执行任务的 worker 数。
func (p *FixedPool) Running() int {
	return int(p.running.Load())
}

// Waiting 返回等待队列中的任务数。
func (p *FixedPool) Waiting() int {
	return len(p.taskCh)
}

// Size 返回池的 worker 容量。
func (p *FixedPool) Size() int {
	return p.size
}
