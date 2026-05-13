package ratelimit

import (
	"sync"
	"time"
)

// SlidingWindow 是滑动窗口计数器限流器。
//
// 将时间划分为固定大小的窗口（如 1 秒），记录当前窗口和上一个窗口的请求数，
// 用时间权重加权平均估算当前有效请求数，超过限制则拒绝。
//
// 与固定窗口的区别：
//
//	固定窗口按整秒/整分钟硬切，窗口边界处可能出现 2 倍突发。
//	例如限制 100/s，用户在 0.9s 发 100 个、1.0s 又发 100 个，
//	实际 0.2s 内通过了 200 个请求。
//	滑动窗口通过加权平均平滑两个窗口的边界，避免此问题。
//
// 算法核心：
//
//	只维护两个计数器（prevCount 上一窗口、currCount 当前窗口），
//	用当前时刻在窗口中的位置计算上一窗口的"残留权重"，
//	有效请求数 = prevCount × weight + currCount。
//	这样在窗口切换时不会出现计数突变，而是平滑过渡。
type SlidingWindow struct {
	mu sync.Mutex

	rate     int           // 每个窗口允许的最大请求数
	interval time.Duration // 窗口大小（如 1 秒、1 分钟）

	prevCount int       // 上一个窗口的请求数
	currCount int       // 当前窗口的请求数
	currStart time.Time // 当前窗口的起始时间（Truncate 对齐后的时间）
}

// NewSlidingWindow 创建滑动窗口限流器。
//
// 参数：
//
//	rate     — 每个窗口允许的最大请求数（如 1000）
//	interval — 窗口大小（如 time.Second、time.Minute）
//
// 默认：rate=100/窗口，interval=1秒。
//
// 示例：
//
//	sw := ratelimit.NewSlidingWindow(
//	    ratelimit.WithRate(1000),            // 每窗口 1000 次
//	    ratelimit.WithInterval(time.Minute), // 窗口大小 1 分钟
//	)
func NewSlidingWindow(opts ...Option) *SlidingWindow {
	cfg := applyOptions(opts)

	if cfg.rate <= 0 {
		panic("ratelimit: rate must be positive")
	}
	if cfg.interval <= 0 {
		panic("ratelimit: interval must be positive")
	}

	now := time.Now()
	return &SlidingWindow{
		rate:      int(cfg.rate),
		interval:  cfg.interval,
		prevCount: 0,
		currCount: 0,
		// Truncate 将时间向下取整到 interval 的整数倍，作为当前窗口的起点。
		// 例如 interval=1s，now=1.37s → currStart=1.0s
		currStart: now.Truncate(cfg.interval),
	}
}

// Allow 判断当前请求是否允许通过（消耗 1 个配额）。
func (sw *SlidingWindow) Allow() bool {
	return sw.AllowN(1)
}

// AllowN 判断当前请求是否允许消耗 n 个配额。
//
// 算法步骤：
//  1. 快速路径：n<=0 直接放行，n>rate 直接拒绝
//  2. 加锁，推进窗口（检查是否需要切换到新窗口）
//  3. 加权计算当前有效请求数
//  4. 判断有效请求数 + n 是否超过限制
func (sw *SlidingWindow) AllowN(n int) bool {
	// 快速路径：不消耗配额的请求直接放行
	if n <= 0 {
		return true
	}
	// 单次请求超过窗口上限，永远不可能满足
	if n > sw.rate {
		return false
	}

	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	// 先推进窗口状态，确保 currStart/prevCount/currCount 反映当前窗口
	sw.advance(now)

	// === 加权计算有效请求数 ===
	//
	// elapsed: 当前时刻在窗口中已经过去的时间（范围 [0, interval) ）
	//   - 因为 advance 已经将 currStart 对齐到窗口左边界，
	//     所以 elapsed 一定严格小于 interval，不会越界。
	//
	// weight: 上一窗口的残留权重（范围 (0, 1.0] ）
	//   - elapsed=0 时 weight=1.0（刚进新窗口，上一窗口完全生效）
	//   - elapsed→interval 时 weight→0（即将离开窗口，上一窗口影响趋零）
	//   - 这个线性衰减正是"滑动"的含义：越远的窗口影响越小
	//
	// count: 有效请求数的近似值
	//   = 上一窗口计数 × 残留权重 + 当前窗口精确计数
	//
	// 举例：interval=1s, rate=100, prevCount=80, currCount=10, elapsed=0.3s
	//   weight = 1.0 - 0.3/1.0 = 0.7
	//   count  = 80 × 0.7 + 10 = 56 + 10 = 66
	//   剩余配额 = 100 - 66 = 34
	elapsed := now.Sub(sw.currStart).Seconds()
	weight := 1.0 - elapsed/sw.interval.Seconds()
	count := float64(sw.prevCount)*weight + float64(sw.currCount)

	// 判断是否超限：用估算的有效计数 + 本次请求量 与 rate 比较
	if int(count)+n > sw.rate {
		return false
	}
	// 允许通过，在当前窗口精确计数（currCount 是整数，不是估算值）
	sw.currCount += n
	return true
}

// advance 处理窗口切换。
//
// 每次 AllowN 调用时先执行此方法，确保计数器状态反映当前所处的窗口。
//
// 窗口切换逻辑：
//
//	用 Truncate(now) 判断当前时刻属于哪个窗口。
//	如果窗口没变（now 还在同一个窗口内），什么都不做。
//	如果窗口变了，需要滚动计数器：
//
//	正常滚动（只跳了 1 个窗口）：
//	  prevCount = currCount  （当前窗口变为上一窗口）
//	  currCount = 0          （新窗口从零开始）
//
//	跳过多个窗口（跳了 2+ 个窗口）：
//	  prevCount = 0          （上一窗口已过期太久，残留权重无意义，直接清零）
//	  currCount = 0
//
//	示例时间线（interval=1s）：
//	  窗口 1: currCount=80
//	  然后 2.5 秒没有请求...
//	  窗口 3: advance 被调用，发现跳了 2 个窗口
//	  → prevCount=0, currCount=0，不会让窗口 1 的旧数据污染计算
func (sw *SlidingWindow) advance(now time.Time) {
	// Truncate 将时间向下取整到 interval 的整数倍
	// 例如 interval=1s: 1.37s→1.0s, 2.0s→2.0s, 2.99s→2.0s
	windowStart := now.Truncate(sw.interval)

	// 窗口没变，无需操作
	if windowStart != sw.currStart {
		// 判断是否跳过了多个窗口
		// windowStart - currStart >= 2*interval 表示中间至少隔了一个完整窗口
		if windowStart.Sub(sw.currStart) >= sw.interval*2 {
			// 跳了 2+ 个窗口：上一窗口的计数已过期，清零
			// 否则 weight 公式会用一个"来自很久以前"的 prevCount 参与计算，结果不准
			sw.prevCount = 0
		} else {
			// 正常滚动：当前窗口变成上一窗口
			sw.prevCount = sw.currCount
		}
		// 新窗口从零开始计数
		sw.currCount = 0
		sw.currStart = windowStart
	}
}

// Rate 返回每窗口最大请求数。
func (sw *SlidingWindow) Rate() float64 {
	return float64(sw.rate)
}

// Burst 返回每窗口最大请求数。
//
// 滑动窗口没有"桶容量"的概念（不像令牌桶可以攒令牌、漏桶可以排队），
// 所以 Burst 等于 Rate，仅为满足 Limiter 接口。
func (sw *SlidingWindow) Burst() int {
	return sw.rate
}
