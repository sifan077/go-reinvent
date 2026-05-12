package ratelimit

import (
	"sync"
	"time"
)

// LeakyBucket 是漏桶限流器。
//
// 漏桶以固定速率处理请求（漏水），超出桶容量的请求被拒绝。
// 与令牌桶不同，漏桶不允许突发流量，输出速率严格平滑。
type LeakyBucket struct {
	mu       sync.Mutex
	rate     float64   // 每秒漏出请求数
	burst    int       // 桶最大容量（最大排队数）
	water    float64   // 当前水量（排队请求数）
	lastTime time.Time // 上次漏水时间
}

// NewLeakyBucket 创建漏桶限流器。
//
// 默认：rate=100/s，burst=100。
func NewLeakyBucket(opts ...Option) *LeakyBucket {
	cfg := applyOptions(opts)

	if cfg.rate <= 0 {
		panic("ratelimit: rate must be positive")
	}
	if cfg.burst <= 0 {
		panic("ratelimit: burst must be positive")
	}

	return &LeakyBucket{
		rate:     cfg.rate,
		burst:    cfg.burst,
		water:    0, // 初始空桶
		lastTime: time.Now(),
	}
}

// Allow 判断当前请求是否允许通过。
func (lb *LeakyBucket) Allow() bool {
	return lb.AllowN(1)
}

// AllowN 判断当前请求是否允许加入 n 个排队。
func (lb *LeakyBucket) AllowN(n int) bool {
	if n <= 0 {
		return true
	}
	if n > lb.burst {
		return false
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	now := time.Now()
	lb.leak(now)
	lb.lastTime = now

	if lb.water+float64(n) <= float64(lb.burst) {
		lb.water += float64(n)
		return true
	}
	return false
}

// leak 根据时间差漏出水。
func (lb *LeakyBucket) leak(now time.Time) {
	elapsed := now.Sub(lb.lastTime).Seconds()
	if elapsed <= 0 {
		return
	}
	lb.water -= elapsed * lb.rate
	if lb.water < 0 {
		lb.water = 0
	}
}

// Rate 返回每秒漏出速率。
func (lb *LeakyBucket) Rate() float64 {
	return lb.rate
}

// Burst 返回桶最大容量。
func (lb *LeakyBucket) Burst() int {
	return lb.burst
}
