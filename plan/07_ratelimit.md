# 单机限流器 · 设计规划

## Context

go-reinvent 项目路线图第 5 个模块。参考 LRU 缓存的分阶段增量开发模式，从 MVP 出发，逐步构建包含三种经典限流算法（令牌桶、漏桶、滑动窗口）的单机限流器，最终加上按 key 独立限流能力。

---

## 阶段一：MVP — 令牌桶（Token Bucket）

### 目标

实现最常用的令牌桶限流器，理解限流核心原理。

### 核心思路

令牌桶是最流行的限流算法，允许突发流量，平滑限流：

```
桶容量 = maxBurst + 1
令牌以固定速率 (rate) 填充，桶满则丢弃多余令牌
每次请求消耗 1 个令牌，桶空则拒绝

关键参数:
  rate    — 每秒生成令牌数（如 100/s）
  burst   — 桶最大容量（允许的最大突发数）
```

**时间计算**：用 `time.Now()` 计算上次调用到现在应该生成多少令牌，避免启动 goroutine 定时填充。

### 目录结构

```
pkg/ratelimit/
├── limiter.go      # Limiter 接口定义
├── token_bucket.go # 令牌桶实现
├── option.go       # Option 模式配置
└── ratelimit_test.go
```

### 核心设计

#### 1. Limiter 接口

```go
// Limiter 是限流器的通用接口
type Limiter interface {
    // Allow 判断当前请求是否允许通过（消耗 1 个令牌）
    Allow() bool

    // AllowN 判断当前请求是否允许消耗 n 个令牌
    AllowN(n int) bool

    // Rate 返回限流速率（每秒）
    Rate() float64

    // Burst 返回桶容量
    Burst() int
}
```

#### 2. TokenBucket 结构体

```go
type TokenBucket struct {
    mu       sync.Mutex
    rate     float64    // 每秒生成令牌数
    burst    int        // 桶最大容量
    tokens   float64    // 当前令牌数（float64 支持小数累积）
    lastTime time.Time  // 上次计算时间
}
```

核心算法（`allowN`）：

```go
func (tb *TokenBucket) allowN(now time.Time, n int) bool {
    // 1. 计算时间差，生成新令牌
    elapsed := now.Sub(tb.lastTime)
    tb.tokens += elapsed.Seconds() * tb.rate
    if tb.tokens > float64(tb.burst) {
        tb.tokens = float64(tb.burst)
    }
    tb.lastTime = now

    // 2. 尝试消耗 n 个令牌
    if tb.tokens >= float64(n) {
        tb.tokens -= float64(n)
        return true
    }
    return false
}
```

#### 3. Option 配置

```go
type config struct {
    rate  float64
    burst int
}

type Option func(*config)

func WithRate(rate float64) Option   // 每秒令牌数
func WithBurst(burst int) Option     // 突发容量
```

### 构造函数

```go
func NewTokenBucket(opts ...Option) *TokenBucket
```

### 测试计划

- [x] 基本 Allow：速率内请求全部通过
- [x] 超过 burst 的请求被拒绝
- [x] 等待一段时间后令牌恢复，请求通过
- [x] AllowN 消耗多个令牌
- [x] AllowN(n > burst) 始终拒绝
- [x] 并发安全：多 goroutine 同时 Allow
- [x] Rate() 和 Burst() 返回正确值
- [x] 默认参数处理（rate <= 0 或 burst <= 0 应 panic）

---

## 阶段二：漏桶（Leaky Bucket）

### 目标

实现漏桶限流器。漏桶以固定速率处理请求，超出部分排队或拒绝。

### 核心思路

```
漏桶 vs 令牌桶的区别：
  令牌桶：允许突发（桶里可以攒令牌），输出速率可变
  漏桶：  输出速率恒定，严格平滑，不允许突发

算法：
  桶有一个固定容量（burst），请求进入桶排队
  桶以固定速率"漏水"（处理请求）
  桶满则拒绝新请求

关键参数:
  rate  — 每秒处理请求数
  burst — 桶容量（最大排队数）
```

### 新增文件

```
pkg/ratelimit/
├── leaky_bucket.go # 漏桶实现
└── ratelimit_test.go # 增加漏桶测试
```

### 核心设计

```go
type LeakyBucket struct {
    mu       sync.Mutex
    rate     float64    // 每秒漏出请求数
    burst    int        // 桶最大容量
    water    float64    // 当前水量（排队请求数）
    lastTime time.Time  // 上次漏水时间
}
```

核心算法：

```go
func (lb *LeakyBucket) allowN(now time.Time, n int) bool {
    // 1. 计算时间差，漏出水
    elapsed := now.Sub(lb.lastTime)
    lb.water -= elapsed.Seconds() * lb.rate
    if lb.water < 0 {
        lb.water = 0
    }
    lb.lastTime = now

    // 2. 尝试加入 n 个请求
    if lb.water+float64(n) <= float64(lb.burst) {
        lb.water += float64(n)
        return true
    }
    return false
}
```

### 测试计划

- [x] 基本 Allow：稳定速率下请求均匀通过
- [x] 突发请求：超过 burst 的请求被拒绝
- [x] 与令牌桶的区别验证：漏桶不允许突发
- [x] 并发安全
- [x] Rate() 和 Burst() 返回正确值

---

## 阶段三：滑动窗口（Sliding Window Counter）

### 目标

实现滑动窗口限流器，基于时间窗口精确计数。

### 核心思路

```
滑动窗口 vs 固定窗口：
  固定窗口：按整分钟/整秒划分，窗口边界处可能出现 2x 突发
  滑动窗口：用加权平均平滑两个窗口的边界，更精确

算法（滑动窗口计数器）：
  将时间划分为固定大小的窗口（如 1 秒）
  记录当前窗口和上一个窗口的请求数
  用时间权重加权平均计算当前窗口的请求数
  超过限制则拒绝

关键参数:
  rate     — 每个窗口允许的最大请求数
  interval — 窗口大小（如 1 秒、1 分钟）
```

### 新增文件

```
pkg/ratelimit/
├── sliding_window.go # 滑动窗口实现
└── ratelimit_test.go  # 增加滑动窗口测试
```

### 核心设计

```go
type SlidingWindow struct {
    mu         sync.Mutex
    rate       int           // 每个窗口最大请求数
    interval   time.Duration // 窗口大小
    prevCount  int           // 上一个窗口的请求数
    currCount  int           // 当前窗口的请求数
    currStart  time.Time     // 当前窗口起始时间
}
```

核心算法：

```go
func (sw *SlidingWindow) Allow() bool {
    now := time.Now()
    windowStart := now.Truncate(sw.interval)

    if windowStart != sw.currStart {
        // 窗口切换
        sw.prevCount = sw.currCount
        sw.currCount = 0
        sw.currStart = windowStart
    }

    // 加权计算：上一窗口的残留权重 + 当前窗口计数
    elapsed := now.Sub(windowStart)
    weight := 1.0 - elapsed.Seconds()/sw.interval.Seconds()
    count := float64(sw.prevCount)*weight + float64(sw.currCount)

    if int(count) >= sw.rate {
        return false
    }
    sw.currCount++
    return true
}
```

### 测试计划

- [ ] 基本 Allow：窗口内请求计数正确
- [ ] 窗口切换：新窗口开始时计数重置
- [ ] 加权平均：窗口边界处的平滑效果
- [ ] 并发安全
- [ ] Rate() 和 Burst() 返回正确值（Burst = rate）
- [ ] 不同 interval 大小的行为

---

## 阶段四：按 key 限流（KeyedLimiter）

### 目标

支持按 key 独立限流（如按用户 ID、IP 地址），类似 LRU 的分片设计。

### 核心思路

```
应用场景：
  API 网关：每个用户 100 次/分钟
  登录接口：每个 IP 5 次/分钟
  消息推送：每个设备 10 条/秒

设计：
  KeyedLimiter 维护一个 map[string]Limiter
  每个 key 有独立的限流器实例
  支持配置默认限流参数和自定义参数
  使用 sync.RWMutex 管理 key 映射
```

### 新增文件

```
pkg/ratelimit/
├── keyed.go          # KeyedLimiter 实现
├── ratelimit_test.go  # 增加 keyed 测试
```

### 核心设计

```go
type KeyedLimiter struct {
    mu       sync.RWMutex
    limiters map[string]Limiter
    newFunc  func() Limiter  // 创建新限流器的工厂函数
    maxSize  int             // 最大 key 数量（0 表示不限制）
}
```

API 设计：

```go
// NewKeyedLimiter 创建按 key 限流的限流器
func NewKeyedLimiter(newFunc func() Limiter, opts ...KeyedOption) *KeyedLimiter

// Allow 判断指定 key 是否允许通过
func (kl *KeyedLimiter) Allow(key string) bool

// AllowN 判断指定 key 是否允许消耗 n 个令牌
func (kl *KeyedLimiter) AllowN(key string, n int) bool

// Remove 删除指定 key 的限流器
func (kl *KeyedLimiter) Remove(key string)

// Len 返回当前 key 数量
func (kl *KeyedLimiter) Len() int
```

Option 配置：

```go
func WithMaxKeys(n int) KeyedOption  // 限制最大 key 数量
```

### 测试计划

- [ ] 不同 key 独立限流
- [ ] 同一 key 共享限流
- [ ] Remove 删除 key
- [ ] Len 返回正确数量
- [ ] 并发安全：多 goroutine 同时操作不同 key
- [ ] MaxKeys 限制：超过上限时的行为（拒绝新 key 或淘汰旧 key）

---

## 阶段五：统计 + 接口抽象 + 演示

### 目标

加入限流统计，定义统一接口，编写演示程序。

### 核心思路

```go
type Stats struct {
    Allowed    int64 // 允许通过的请求数
    Rejected   int64 // 被拒绝的请求数
    Total      int64 // 总请求数
}

func (s *Stats) PassRate() float64  // 通过率 = Allowed / Total
```

### 修改内容

```
pkg/ratelimit/
├── limiter.go         # Limiter 接口（已有）+ Stats 方法
├── token_bucket.go    # 实现 Stats
├── leaky_bucket.go    # 实现 Stats
├── sliding_window.go  # 实现 Stats
├── keyed.go           # 聚合各 key 的 Stats
├── stats.go           # Stats 结构体
├── option.go          # Option 统一管理
└── ratelimit_test.go  # 全量测试

cmd/ratelimit/
└── main.go            # 演示程序
```

### 演示程序

```go
// cmd/ratelimit/main.go
func main() {
    // 1. 令牌桶演示
    tb := ratelimit.NewTokenBucket(
        ratelimit.WithRate(10),  // 10/s
        ratelimit.WithBurst(20), // 允许突发 20
    )
    // 模拟突发请求

    // 2. 漏桶演示
    lb := ratelimit.NewLeakyBucket(...)
    // 对比令牌桶的突发行为

    // 3. 滑动窗口演示
    sw := ratelimit.NewSlidingWindow(...)
    // 展示窗口边界的平滑效果

    // 4. 按 key 限流演示
    kl := ratelimit.NewKeyedLimiter(...)
    // 模拟多用户独立限流
}
```

---

## 最终目录结构

```
pkg/ratelimit/
├── limiter.go         # Limiter 接口定义
├── token_bucket.go    # 令牌桶实现
├── leaky_bucket.go    # 漏桶实现
├── sliding_window.go  # 滑动窗口实现
├── keyed.go           # 按 key 限流封装
├── stats.go           # 访问统计
├── option.go          # Option 模式配置
└── ratelimit_test.go  # 全量测试

cmd/ratelimit/
└── main.go            # 演示程序
```

## 对外 API 总览

```go
// 令牌桶
tb := ratelimit.NewTokenBucket(
    ratelimit.WithRate(100),      // 100 请求/秒
    ratelimit.WithBurst(200),     // 允许突发 200
)
tb.Allow()       // bool
tb.AllowN(5)     // bool, 消耗 5 个令牌

// 漏桶
lb := ratelimit.NewLeakyBucket(
    ratelimit.WithRate(50),       // 50 请求/秒
    ratelimit.WithBurst(100),     // 桶容量 100
)
lb.Allow()

// 滑动窗口
sw := ratelimit.NewSlidingWindow(
    ratelimit.WithRate(1000),     // 每窗口 1000 请求
    ratelimit.WithInterval(time.Minute), // 1 分钟窗口
)
sw.Allow()

// 按 key 限流
kl := ratelimit.NewKeyedLimiter(
    func() ratelimit.Limiter {
        return ratelimit.NewTokenBucket(
            ratelimit.WithRate(10),
            ratelimit.WithBurst(20),
        )
    },
    ratelimit.WithMaxKeys(10000),
)
kl.Allow("user:123")
kl.AllowN("user:456", 3)
kl.Remove("user:123")

// 统计
stats := tb.Stats()
fmt.Printf("通过率: %.2f%%\n", stats.PassRate()*100)
```

## 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 核心算法 | 令牌桶 + 漏桶 + 滑动窗口 | 覆盖三种经典限流策略，学习价值高 |
| 时间计算 | 惰性计算（调用时算） | 无需后台 goroutine，零开销 |
| 令牌精度 | float64 | 支持小数累积，避免精度丢失 |
| 按 key 限流 | map + 工厂函数 | 灵活，每个 key 独立配置 |
| 并发方案 | sync.Mutex | 限流器操作都是写操作，RWMutex 无优势 |
| 零依赖 | 仅标准库 | 符合项目原则 |

## 验证方式

```bash
# 运行全量测试
go test ./pkg/ratelimit/ -v

# 运行演示程序
go run ./cmd/ratelimit/

# 并发测试
go test ./pkg/ratelimit/ -race -v

# 性能基准（可选）
go test ./pkg/ratelimit/ -bench=.
```
