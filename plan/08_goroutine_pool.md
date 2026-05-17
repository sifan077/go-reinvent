# 协程池 · 设计规划

## Context

go-reinvent 项目路线图第 6 个模块。协程池用于控制并发 goroutine 数量，避免无限制创建 goroutine 导致资源耗尽。参考业界常见实现（ants、gammazero/workerpool），从 MVP 出发，逐步构建一个轻量、零依赖的协程池。

---

## 阶段一：MVP — 固定大小协程池

### 目标

实现最基本的协程池：固定数量的 worker，通过 channel 接收任务。

### 核心思路

```
协程池本质：
  预先启动 N 个 worker goroutine，每个 worker 从任务 channel 中取任务执行
  提交任务时往 channel 中写入，worker 竞争消费
  避免每次请求都 go func()，控制并发上限

关键参数：
  size      — worker 数量（并发上限）
  queueSize — 任务队列容量（默认 size * 2）
```

### 目录结构

```
pkg/pool/
├── pool.go       # Pool 接口 + 固定池实现
├── option.go     # Option 模式配置
└── pool_test.go
```

### 核心设计

#### 1. Pool 接口

```go
// Pool 是协程池接口。
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
```

#### 2. FixedPool 结构体

```go
type FixedPool struct {
    size     int            // 配置的 worker 数量
    taskCh   chan func()     // 任务队列
    wg       sync.WaitGroup // 等待 worker 退出
    quit     chan struct{}   // 通知 worker 停止
    stopped  atomic.Bool    // 是否已停止
    running  atomic.Int32   // 当前执行任务的 worker 数
}
```

#### 3. worker 循环

```go
func (p *FixedPool) worker() {
    defer p.wg.Done()
    for {
        select {
        case task, ok := <-p.taskCh:
            if !ok {
                return // channel 已关闭
            }
            p.safeRun(task)
        case <-p.quit:
            return
        }
    }
}
```

> 注：阶段一直接内联 `running.Add(1)` / `task()` / `running.Add(-1)`，后为阶段二引入 safeRun 重构，running 计数和 panic 捕获统一收口到 safeRun 内部。

#### 4. 构造函数

```go
func New(size int, opts ...Option) *FixedPool
```

默认 queueSize = size * 2（有缓冲 channel，避免 Submit 立即阻塞）。

### 测试计划

- [ ] 基本 Submit：任务被正确执行
- [ ] 并发数不超过 Size()
- [ ] TrySubmit：队列满时返回 false
- [ ] Stop：等待所有已提交任务完成后退出
- [ ] StopNow：立即退出，队列中剩余任务不执行
- [ ] Running() 返回正确值
- [ ] Waiting() 返回正确值
- [ ] 提交大量任务不丢任务
- [ ] 并发安全：多 goroutine 同时 Submit
- [ ] 接口合规检查：var _ Pool = &FixedPool{}
- [ ] Stop 后 Submit 不阻塞
- [ ] Stop 后 TrySubmit 返回 false
- [ ] Size() 返回正确值
- [ ] Stop/StopNow 幂等调用不 panic
- [ ] size <= 0 时 New panic

---

## 阶段二：Panic 捕获 + 任务超时

### 目标

任务 panic 不会导致整个池崩溃；支持单个任务超时控制。

### 核心思路

```
Panic 捕获：
  worker 中用 defer recover() 捕获 panic
  默认行为：静默吞掉 panic（和 go func() 行为一致）
  可选：通过 WithPanicHandler 注册回调
  panic 后 running 计数通过 defer 保证正确递减

任务超时：
  提供 SubmitWithTimeout(task, timeout) 方法
  内部用 context.WithTimeout 包装任务
  超时后任务的 context 被取消，但 worker 不会被 kill（任务需要自己监听 ctx.Done()）
  这是协作式取消：如果任务不监听 ctx.Done()，worker 仍会被占用直到任务自行退出
```

### 修改内容

```
pkg/pool/
├── pool.go       # worker 增加 recover，新增 SubmitWithTimeout
├── option.go     # 新增 WithPanicHandler
└── pool_test.go  # 新增 panic/timeout 测试
```

### 核心设计

#### 1. Panic 捕获

```go
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

func (p *FixedPool) safeRun(task func()) {
    p.running.Add(1)
    defer p.running.Add(-1) // defer 保护：recover 内部可能 panic
    defer func() {
        if r := recover(); r != nil {
            if p.panicHandler != nil {
                p.panicHandler(r)
            }
        }
    }()
    task()
}
```

#### 2. 任务超时

```go
// SubmitWithTimeout 提交带超时的任务。
// task 接收 context.Context，应监听 ctx.Done() 以响应超时。
// 注意：这是协作式取消，如果 task 不监听 ctx.Done()，worker 仍会被占用。
func (p *FixedPool) SubmitWithTimeout(task func(ctx context.Context), timeout time.Duration) bool
```

#### 3. 新增 Option

```go
func WithPanicHandler(fn func(r interface{})) Option  // panic 回调
func WithQueueSize(size int) Option                    // 队列容量
```

### 测试计划

- [ ] 任务 panic 不影响池继续工作
- [ ] PanicHandler 被正确调用
- [ ] 不同类型的 panic 值都能被捕获
- [ ] SubmitWithTimeout：超时后 context 被取消
- [ ] SubmitWithTimeout：任务在超时内完成则正常结束
- [ ] SubmitWithTimeout：任务可通过 ctx.Done() 检测超时并提前退出
- [ ] 多个超时任务并发执行互不影响
- [ ] 池停止后 SubmitWithTimeout 返回 false
- [ ] 超时任务中 panic 被捕获

---

## 阶段三：任务结果 + Future 模式

### 目标

支持提交有返回值的任务，提供 Future/AsyncResult 模式获取结果。

### 核心思路

```
场景：
  并发调用多个微服务，收集结果
  并行计算，汇总结果

设计：
  SubmitFuture[T](pool, task func() (T, error)) *Future[T]
  注意：Go 不允许方法有自己的类型参数，所以 SubmitFuture 是独立泛型函数而非方法

  Future 提供：
    - Wait() (T, error)         — 阻塞等待结果
    - WaitTimeout(d) (T, error) — 带超时等待，超时返回 ErrTimeout
    - Done() <-chan struct{}     — 完成通知 channel
```

### 新增文件

```
pkg/pool/
├── future.go     # Future 实现
└── pool_test.go  # 新增 Future 测试
```

### 核心设计

#### 1. 哨兵错误

```go
var ErrTimeout = errors.New("pool: future wait timeout")      // WaitTimeout 超时
var ErrPoolStopped = errors.New("pool: pool is stopped")      // 池已停止时 SubmitFuture
```

#### 2. Future 结构体

```go
type Future[T any] struct {
    done    chan struct{} // 关闭表示任务完成
    result  T            // 任务返回值
    err     error        // 任务返回的 error
}

func (f *Future[T]) Wait() (T, error)                        // 阻塞等待结果
func (f *Future[T]) WaitTimeout(d time.Duration) (T, error)  // 带超时等待
func (f *Future[T]) Done() <-chan struct{}                    // 完成通知 channel
```

#### 3. SubmitFuture 实现

```go
func SubmitFuture[T any](p *FixedPool, task func() (T, error)) *Future[T] {
    f := &Future[T]{done: make(chan struct{})}

    // 池已停止时返回已完成的 Future，携带 ErrPoolStopped
    if p.stopped.Load() {
        f.err = ErrPoolStopped
        close(f.done)
        return f
    }

    p.Submit(func() {
        defer close(f.done)           // 保证 done 一定被关闭
        defer func() {                // 在 safeRun 的 recover 之前执行（LIFO）
            if r := recover(); r != nil {
                f.err = fmt.Errorf("pool: task panicked: %v", r)
            }
        }()
        f.result, f.err = task()
    })

    return f
}
```

> 关键：wrapper 内部自行 recover，不依赖 safeRun 的 recover。因为 safeRun 捕获 panic 后静默吞掉，Future 的 done 永远不会被关闭，Wait 会永久阻塞。wrapper 的 defer recover 在 safeRun 的 defer recover 之前执行（LIFO 顺序），确保 Future 始终能完成。

### 测试计划

- [ ] SubmitFuture：正确返回结果
- [ ] SubmitFuture：任务返回 error 时 Future 携带 error
- [ ] Wait：阻塞直到任务完成
- [ ] WaitTimeout：超时返回 ErrTimeout
- [ ] WaitTimeout：未超时时正常返回
- [ ] Done channel：可配合 select 使用
- [ ] 多个 Future 并发执行
- [ ] 池已停止时返回 ErrPoolStopped
- [ ] 任务 panic 时 Future 不永久阻塞

---

## 阶段四：动态扩缩容 + 统计

### 目标

支持动态调整池大小；加入任务统计信息。

### 核心思路

```
动态扩缩容：
  Resize(newSize int) — 运行时调整 worker 数

  设计：
  - 用 atomic.Int32 维护 currentWorkers（当前活跃 worker 数）和 targetSize（目标 worker 数）
  - 扩容：spawn 新的 worker goroutine，递增 currentWorkers
  - 缩容：递减 targetSize，worker 在完成当前任务后检查 targetSize，超出则自行退出
  - quit channel 仅用于 Stop/StopNow，不参与扩缩容

  worker 退出判断：
  func (p *FixedPool) worker() {
      defer func() {
          p.currentWorkers.Add(-1)
          p.wg.Done()
      }()
      for {
          select {
          case task, ok := <-p.taskCh:
              if !ok { return }
              p.safeRun(task)
              // 完成任务后检查是否需要缩容
              if p.currentWorkers.Load() > p.targetSize.Load() {
                  return
              }
          case <-p.quit:
              return
          }
      }
  }

统计：
  type Stats struct {
      TotalSubmitted  int64  // 累计提交任务数
      TotalCompleted  int64  // 累计完成任务数
      TotalPanicked   int64  // 累计 panic 次数
      TotalTimedOut   int64  // 累计超时次数
      Running         int    // 当前执行中
      Waiting         int    // 队列等待中
      Size            int    // 池容量（目标 worker 数）
  }
```

### 修改内容

```
pkg/pool/
├── pool.go       # 新增 Resize、Stats 方法
├── stats.go      # Stats 结构体
└── pool_test.go  # 新增扩缩容、统计测试
```

### 测试计划

- [ ] Resize 扩容：新 worker 启动并处理任务
- [ ] Resize 缩容：多余 worker 退出，不丢任务
- [ ] Stats 返回正确的统计数据
- [ ] 扩缩容期间任务不丢失
- [ ] 并发 Resize 安全

---

## 阶段五：Option 完善 + 演示程序

### 目标

完善配置选项，编写演示程序。

### 新增文件

```
cmd/pool/
└── main.go       # 演示程序
```

### 演示程序

```go
// cmd/pool/main.go
func main() {
    // 1. 基本用法：提交 100 个任务到 10 worker 的池
    p := pool.New(10, pool.WithQueueSize(50))
    for i := 0; i < 100; i++ {
        i := i
        p.Submit(func() {
            fmt.Printf("task %d running on goroutine\n", i)
            time.Sleep(100 * time.Millisecond)
        })
    }
    p.Stop()

    // 2. Future 模式：并发获取结果
    p2 := pool.New(5)
    futures := make([]*pool.Future[string], 10)
    for i := 0; i < 10; i++ {
        i := i
        futures[i] = pool.SubmitFuture(p2, func() (string, error) {
            return fmt.Sprintf("result-%d", i), nil
        })
    }
    for _, f := range futures {
        r, _ := f.Wait()
        fmt.Println(r)
    }
    p2.Stop()

    // 3. Panic 捕获
    p3 := pool.New(3, pool.WithPanicHandler(func(r interface{}) {
        fmt.Printf("task panicked: %v\n", r)
    }))
    p3.Submit(func() { panic("boom") })
    time.Sleep(time.Second)
    p3.Stop()

    // 4. 动态扩缩容
    p4 := pool.New(5)
    // ... 提交任务 ...
    p4.Resize(20) // 扩容
    // ... 提交更多任务 ...
    p4.Resize(3)  // 缩容
    p4.Stop()
}
```

---

## 当前目录结构

```
pkg/pool/
├── pool.go       # Pool 接口 + FixedPool 实现
├── future.go     # Future 泛型实现 + 哨兵错误
├── option.go     # Option 模式配置
└── pool_test.go  # 全量测试
```

## 对外 API 总览

```go
// 创建池
p := pool.New(10,                          // 10 个 worker
    pool.WithQueueSize(100),               // 队列容量 100
    pool.WithPanicHandler(func(r interface{}) {
        log.Printf("panic: %v", r)
    }),
)

// 提交任务
p.Submit(func() { /* do work */ })

// 非阻塞提交
if !p.TrySubmit(func() { /* do work */ }) {
    fmt.Println("pool is busy")
}

// 超时任务（协作式取消，任务需监听 ctx.Done()）
p.SubmitWithTimeout(func(ctx context.Context) {
    select {
    case <-ctx.Done():
        fmt.Println("timed out")
        return
    default:
        // do work
    }
}, 5*time.Second)

// Future 模式（独立泛型函数，非方法）
f := pool.SubmitFuture(p, func() (string, error) {
    return "result", nil
})
result, err := f.Wait()

// Future 带超时等待
result, err := f.WaitTimeout(5 * time.Second)
if errors.Is(err, pool.ErrTimeout) {
    fmt.Println("future timed out")
}

// 状态查询
fmt.Printf("running=%d, waiting=%d, size=%d\n",
    p.Running(), p.Waiting(), p.Size())

// 停止
p.Stop()     // 优雅停止
p.StopNow()  // 立即停止
```

## 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 任务队列 | 有缓冲 channel | 简单高效，天然并发安全 |
| 默认队列大小 | size * 2 | 避免 Submit 立即阻塞，留出缓冲 |
| Worker 生命周期 | 常驻 | 避免 goroutine 反复创建销毁的开销 |
| Panic 处理 | 默认 recover + 可选回调 | 与 go func() 行为一致，不崩溃 |
| 超时机制 | context 传递（协作式取消） | 任务自行监听 ctx，不强制 kill goroutine |
| SubmitFuture | 独立泛型函数 | Go 方法不支持自有类型参数 |
| Future.WaitTimeout | 超时返回 ErrTimeout | 符合 Go error 惯例，返回值顺序 (T, error) |
| Future panic | wrapper 自行 recover，不依赖 safeRun | 防止 safeRun 静默吞掉 panic 导致 Wait 永久阻塞 |
| 池停止时 SubmitFuture | 返回已完成 Future + ErrPoolStopped | 不阻塞调用方，错误可检查 |
| 零依赖 | 仅标准库 | 符合项目原则 |

## 验证方式

```bash
# 运行全量测试
go test ./pkg/pool/ -v

# 并发测试
go test ./pkg/pool/ -race -v

# 运行演示程序
go run ./cmd/pool/

# 性能基准（可选）
go test ./pkg/pool/ -bench=.
```
