// Package ratelimit 实现单机限流器，支持令牌桶、漏桶、滑动窗口三种经典算法。
package ratelimit

// Limiter 是限流器的通用接口。
type Limiter interface {
	// Allow 判断当前请求是否允许通过（消耗 1 个令牌/配额）。
	Allow() bool

	// AllowN 判断当前请求是否允许消耗 n 个令牌/配额。
	AllowN(n int) bool

	// Rate 返回限流速率（每秒允许的请求数）。
	Rate() float64

	// Burst 返回桶容量（最大突发数）。
	Burst() int
}
