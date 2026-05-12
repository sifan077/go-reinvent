package ratelimit

import (
	"sync"
	"testing"
	"time"
)

// ==================== 阶段一：令牌桶测试 ====================

func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket(WithRate(10), WithBurst(10))

	// 初始满桶，burst 个请求应该全部通过
	for i := 0; i < 10; i++ {
		if !tb.Allow() {
			t.Fatalf("第 %d 个请求应该通过", i+1)
		}
	}

	// 第 11 个请求应该被拒绝
	if tb.Allow() {
		t.Fatal("超出 burst 的请求应该被拒绝")
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	tests := []struct {
		name   string
		rate   float64
		burst  int
		n      int
		want   bool
	}{
		{"消耗 1 个令牌", 10, 10, 1, true},
		{"消耗 5 个令牌", 10, 10, 5, true},
		{"消耗等于 burst 的令牌", 10, 10, 10, true},
		{"消耗超过 burst 的令牌", 10, 10, 11, false},
		{"消耗 0 个令牌", 10, 10, 0, true},
		{"消耗负数令牌", 10, 10, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTokenBucket(WithRate(tt.rate), WithBurst(tt.burst))
			got := tb.AllowN(tt.n)
			if got != tt.want {
				t.Fatalf("AllowN(%d) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(WithRate(100), WithBurst(100))

	// 消耗所有令牌
	for i := 0; i < 100; i++ {
		tb.Allow()
	}
	if tb.Allow() {
		t.Fatal("令牌耗尽后应该拒绝")
	}

	// 等待 100ms，应该恢复约 10 个令牌
	time.Sleep(110 * time.Millisecond)

	allowed := 0
	for i := 0; i < 20; i++ {
		if tb.Allow() {
			allowed++
		}
	}

	if allowed < 8 {
		t.Fatalf("等待后应该恢复令牌，实际通过 %d 个（期望至少 8）", allowed)
	}
}

func TestTokenBucket_PartialConsume(t *testing.T) {
	tb := NewTokenBucket(WithRate(10), WithBurst(10))

	// 消耗 7 个
	if !tb.AllowN(7) {
		t.Fatal("消耗 7 个应该成功")
	}

	// 剩余 3 个，消耗 4 个应该失败
	if tb.AllowN(4) {
		t.Fatal("剩余 3 个令牌时消耗 4 个应该失败")
	}

	// 消耗 3 个应该成功
	if !tb.AllowN(3) {
		t.Fatal("剩余 3 个令牌时消耗 3 个应该成功")
	}
}

func TestTokenBucket_Rate(t *testing.T) {
	tb := NewTokenBucket(WithRate(50), WithBurst(100))
	if tb.Rate() != 50 {
		t.Fatalf("Rate() = %v, want 50", tb.Rate())
	}
}

func TestTokenBucket_Burst(t *testing.T) {
	tb := NewTokenBucket(WithRate(50), WithBurst(100))
	if tb.Burst() != 100 {
		t.Fatalf("Burst() = %v, want 100", tb.Burst())
	}
}

func TestTokenBucket_DefaultOptions(t *testing.T) {
	tb := NewTokenBucket()
	if tb.Rate() != 100 {
		t.Fatalf("默认 Rate() = %v, want 100", tb.Rate())
	}
	if tb.Burst() != 100 {
		t.Fatalf("默认 Burst() = %v, want 100", tb.Burst())
	}
}

func TestTokenBucket_PanicOnInvalidRate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("rate <= 0 应该 panic")
		}
	}()
	NewTokenBucket(WithRate(0))
}

func TestTokenBucket_PanicOnInvalidBurst(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("burst <= 0 应该 panic")
		}
	}()
	NewTokenBucket(WithBurst(-1))
}

func TestTokenBucket_Concurrent(t *testing.T) {
	tb := NewTokenBucket(WithRate(1000), WithBurst(1000))

	var wg sync.WaitGroup
	allowed := make(chan bool, 1000)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- tb.Allow()
		}()
	}

	wg.Wait()
	close(allowed)

	trueCount := 0
	for v := range allowed {
		if v {
			trueCount++
		}
	}

	if trueCount == 0 || trueCount > 1000 {
		t.Fatalf("并发 Allow 结果异常: 通过 %d 个", trueCount)
	}
}

func TestTokenBucket_ConcurrentAllowN(t *testing.T) {
	tb := NewTokenBucket(WithRate(100), WithBurst(100))

	var wg sync.WaitGroup
	var mu sync.Mutex
	totalConsumed := 0

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.AllowN(3) {
				mu.Lock()
				totalConsumed += 3
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if totalConsumed > 100 {
		t.Fatalf("并发消耗超过 burst: 消耗 %d, burst %d", totalConsumed, 100)
	}
}

func TestTokenBucket_SmoothRate(t *testing.T) {
	tb := NewTokenBucket(WithRate(100), WithBurst(1))

	// burst=1，初始 1 个令牌
	if !tb.Allow() {
		t.Fatal("初始应该有 1 个令牌")
	}
	if tb.Allow() {
		t.Fatal("初始只有 1 个令牌，第二次应该拒绝")
	}

	// 每 10ms 应该生成 1 个令牌
	time.Sleep(15 * time.Millisecond)
	if !tb.Allow() {
		t.Fatal("等待后应该有 1 个令牌")
	}
}

// ==================== 阶段二：漏桶测试 ====================

func TestLeakyBucket_Allow(t *testing.T) {
	lb := NewLeakyBucket(WithRate(10), WithBurst(10))

	// 初始空桶，burst 个请求应该全部通过
	for i := 0; i < 10; i++ {
		if !lb.Allow() {
			t.Fatalf("第 %d 个请求应该通过", i+1)
		}
	}

	// 第 11 个请求应该被拒绝（桶满）
	if lb.Allow() {
		t.Fatal("超出 burst 的请求应该被拒绝")
	}
}

func TestLeakyBucket_AllowN(t *testing.T) {
	tests := []struct {
		name  string
		rate  float64
		burst int
		n     int
		want  bool
	}{
		{"消耗 1 个配额", 10, 10, 1, true},
		{"消耗 5 个配额", 10, 10, 5, true},
		{"消耗等于 burst 的配额", 10, 10, 10, true},
		{"消耗超过 burst 的配额", 10, 10, 11, false},
		{"消耗 0 个配额", 10, 10, 0, true},
		{"消耗负数配额", 10, 10, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := NewLeakyBucket(WithRate(tt.rate), WithBurst(tt.burst))
			got := lb.AllowN(tt.n)
			if got != tt.want {
				t.Fatalf("AllowN(%d) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

func TestLeakyBucket_Leak(t *testing.T) {
	lb := NewLeakyBucket(WithRate(100), WithBurst(100))

	// 消耗所有配额
	for i := 0; i < 100; i++ {
		lb.Allow()
	}
	if lb.Allow() {
		t.Fatal("桶满后应该拒绝")
	}

	// 等待 100ms，应该漏出约 10 个请求
	time.Sleep(110 * time.Millisecond)

	allowed := 0
	for i := 0; i < 20; i++ {
		if lb.Allow() {
			allowed++
		}
	}

	if allowed < 8 {
		t.Fatalf("等待后应该漏出请求，实际通过 %d 个（期望至少 8）", allowed)
	}
}

func TestLeakyBucket_NoBurst(t *testing.T) {
	// 漏桶的核心特性：不允许突发
	// burst=5, rate=1000/s（漏得很快）
	lb := NewLeakyBucket(WithRate(1000), WithBurst(5))

	// 填满桶
	for i := 0; i < 5; i++ {
		if !lb.Allow() {
			t.Fatalf("第 %d 个请求应该通过", i+1)
		}
	}

	// 立即再请求，即使 rate 很高也应该拒绝
	// 因为水还没来得及漏出去
	if lb.Allow() {
		t.Fatal("桶满后立即请求应该拒绝，漏桶不允许突发")
	}
}

func TestLeakyBucket_PartialConsume(t *testing.T) {
	lb := NewLeakyBucket(WithRate(10), WithBurst(10))

	// 加入 7 个请求
	if !lb.AllowN(7) {
		t.Fatal("加入 7 个请求应该成功")
	}

	// 剩余 3 个容量，加入 4 个应该失败
	if lb.AllowN(4) {
		t.Fatal("剩余 3 个容量时加入 4 个应该失败")
	}

	// 加入 3 个应该成功
	if !lb.AllowN(3) {
		t.Fatal("剩余 3 个容量时加入 3 个应该成功")
	}
}

func TestLeakyBucket_Rate(t *testing.T) {
	lb := NewLeakyBucket(WithRate(50), WithBurst(100))
	if lb.Rate() != 50 {
		t.Fatalf("Rate() = %v, want 50", lb.Rate())
	}
}

func TestLeakyBucket_Burst(t *testing.T) {
	lb := NewLeakyBucket(WithRate(50), WithBurst(100))
	if lb.Burst() != 100 {
		t.Fatalf("Burst() = %v, want 100", lb.Burst())
	}
}

func TestLeakyBucket_DefaultOptions(t *testing.T) {
	lb := NewLeakyBucket()
	if lb.Rate() != 100 {
		t.Fatalf("默认 Rate() = %v, want 100", lb.Rate())
	}
	if lb.Burst() != 100 {
		t.Fatalf("默认 Burst() = %v, want 100", lb.Burst())
	}
}

func TestLeakyBucket_PanicOnInvalidRate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("rate <= 0 应该 panic")
		}
	}()
	NewLeakyBucket(WithRate(0))
}

func TestLeakyBucket_PanicOnInvalidBurst(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("burst <= 0 应该 panic")
		}
	}()
	NewLeakyBucket(WithBurst(-1))
}

func TestLeakyBucket_Concurrent(t *testing.T) {
	lb := NewLeakyBucket(WithRate(1000), WithBurst(1000))

	var wg sync.WaitGroup
	allowed := make(chan bool, 1000)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- lb.Allow()
		}()
	}

	wg.Wait()
	close(allowed)

	trueCount := 0
	for v := range allowed {
		if v {
			trueCount++
		}
	}

	if trueCount == 0 || trueCount > 1000 {
		t.Fatalf("并发 Allow 结果异常: 通过 %d 个", trueCount)
	}
}

func TestLeakyBucket_ConcurrentAllowN(t *testing.T) {
	lb := NewLeakyBucket(WithRate(100), WithBurst(100))

	var wg sync.WaitGroup
	var mu sync.Mutex
	totalConsumed := 0

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if lb.AllowN(3) {
				mu.Lock()
				totalConsumed += 3
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if totalConsumed > 100 {
		t.Fatalf("并发消耗超过 burst: 消耗 %d, burst %d", totalConsumed, 100)
	}
}

func TestLeakyBucket_SmoothOutput(t *testing.T) {
	// 漏桶的输出是严格平滑的
	lb := NewLeakyBucket(WithRate(1000), WithBurst(5))

	// 填满
	for i := 0; i < 5; i++ {
		lb.Allow()
	}

	// 等待 10ms，应该漏出 10 个（1000/s * 0.01s）
	// 但桶最多加 5 个，所以应该能通过 5 个
	time.Sleep(10 * time.Millisecond)

	allowed := 0
	for i := 0; i < 10; i++ {
		if lb.Allow() {
			allowed++
		}
	}

	if allowed < 5 {
		t.Fatalf("漏桶平滑输出异常: 通过 %d 个（期望至少 5）", allowed)
	}
}

