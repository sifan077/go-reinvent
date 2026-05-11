package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket 是令牌桶限流器。
//
// 令牌桶以固定速率生成令牌，桶满则丢弃多余令牌。
// 每次请求消耗指定数量的令牌，桶空则拒绝。
// 允许突发流量（消耗桶中积攒的令牌）。
type TokenBucket struct {
	mu       sync.Mutex
	rate     float64   // 每秒生成令牌数
	burst    int       // 桶最大容量
	tokens   float64   // 当前令牌数
	lastTime time.Time // 上次计算时间
}

// NewTokenBucket 创建令牌桶限流器。
//
// 默认：rate=100/s，burst=100。
func NewTokenBucket(opts ...Option) *TokenBucket {
	cfg := applyOptions(opts)

	if cfg.rate <= 0 {
		panic("ratelimit: rate must be positive")
	}
	if cfg.burst <= 0 {
		panic("ratelimit: burst must be positive")
	}

	return &TokenBucket{
		rate:     cfg.rate,
		burst:    cfg.burst,
		tokens:   float64(cfg.burst), // 初始满桶
		lastTime: time.Now(),
	}
}

// Allow 判断当前请求是否允许通过。
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 判断当前请求是否允许消耗 n 个令牌。
func (tb *TokenBucket) AllowN(n int) bool {
	if n <= 0 {
		return true
	}
	if n > tb.burst {
		return false
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	tb.refill(now)
	tb.lastTime = now

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

// refill 根据时间差生成新令牌。
func (tb *TokenBucket) refill(now time.Time) {
	elapsed := now.Sub(tb.lastTime).Seconds()
	if elapsed <= 0 {
		return
	}
	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.burst) {
		tb.tokens = float64(tb.burst)
	}
}

// Rate 返回每秒令牌生成速率。
func (tb *TokenBucket) Rate() float64 {
	return tb.rate
}

// Burst 返回桶最大容量。
func (tb *TokenBucket) Burst() int {
	return tb.burst
}
