package middleware

import (
	"sync"
	"time"
)

// RateLimiterConfig 限流配置
type RateLimiterConfig struct {
	// 每秒请求数限制
	RequestsPerSecond int `json:"requests_per_second"`
	// 桶容量（突发流量）
	BurstSize int `json:"burst_size"`
	// 限流维度：ip, user, api
	Dimension string `json:"dimension"`
	// 是否启用
	Enabled bool `json:"enabled"`
}

// TokenBucket 令牌桶实现
type TokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(capacity, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow 检查是否允许请求通过
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// GetTokens 获取当前令牌数（用于监控）
func (tb *TokenBucket) GetTokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens
}

// RateLimiter 限流器
type RateLimiter struct {
	config    RateLimiterConfig
	buckets   map[string]*TokenBucket
	globalBkt *TokenBucket
	mu        sync.RWMutex
}

// NewRateLimiter 创建限流器
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		config:  config,
		buckets: make(map[string]*TokenBucket),
	}

	if config.Enabled && config.RequestsPerSecond > 0 {
		capacity := float64(config.BurstSize)
		if capacity <= 0 {
			capacity = float64(config.RequestsPerSecond)
		}
		rl.globalBkt = NewTokenBucket(capacity, float64(config.RequestsPerSecond))
	}

	return rl
}

// Allow 检查请求是否允许通过
func (rl *RateLimiter) Allow(key string) bool {
	if !rl.config.Enabled {
		return true
	}

	// 全局限流
	if rl.globalBkt != nil && !rl.globalBkt.Allow() {
		return false
	}

	// 按维度限流
	if rl.config.Dimension != "global" {
		rl.mu.RLock()
		bucket, exists := rl.buckets[key]
		rl.mu.RUnlock()

		if !exists {
			// 创建新的桶
			rl.mu.Lock()
			// 双重检查
			if bucket, exists = rl.buckets[key]; !exists {
				capacity := float64(rl.config.BurstSize)
				if capacity <= 0 {
					capacity = float64(rl.config.RequestsPerSecond)
				}
				bucket = NewTokenBucket(capacity, float64(rl.config.RequestsPerSecond))
				rl.buckets[key] = bucket
			}
			rl.mu.Unlock()
		}

		if !bucket.Allow() {
			return false
		}
	}

	return true
}

// Cleanup 清理长时间未使用的桶（防止内存泄漏）
func (rl *RateLimiter) Cleanup(idleTimeout time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		if now.Sub(bucket.lastRefill) > idleTimeout {
			delete(rl.buckets, key)
		}
		bucket.mu.Unlock()
	}
}

// GetStats 获取限流统计信息
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := map[string]interface{}{
		"enabled":       rl.config.Enabled,
		"dimension":     rl.config.Dimension,
		"active_keys":   len(rl.buckets),
		"requests_per_second": rl.config.RequestsPerSecond,
		"burst_size":    rl.config.BurstSize,
	}

	if rl.globalBkt != nil {
		stats["global_tokens"] = rl.globalBkt.GetTokens()
	}

	return stats
}

// UpdateConfig 更新配置
func (rl *RateLimiter) UpdateConfig(config RateLimiterConfig) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.config = config

	if config.Enabled && config.RequestsPerSecond > 0 {
		capacity := float64(config.BurstSize)
		if capacity <= 0 {
			capacity = float64(config.RequestsPerSecond)
		}
		rl.globalBkt = NewTokenBucket(capacity, float64(config.RequestsPerSecond))
	} else {
		rl.globalBkt = nil
	}
}
