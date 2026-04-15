package middleware

import (
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	// 创建容量为 5，补充速率为 10 tokens/秒的桶
	bucket := NewTokenBucket(5, 10)

	// 初始应该有 5 个令牌
	for i := 0; i < 5; i++ {
		if !bucket.Allow() {
			t.Errorf("Expected Allow() to return true for request %d", i+1)
		}
	}

	// 第 6 个请求应该被拒绝
	if bucket.Allow() {
		t.Error("Expected Allow() to return false after exhausting tokens")
	}

	// 等待令牌补充
	time.Sleep(200 * time.Millisecond)

	// 应该至少有 1 个令牌了
	if !bucket.Allow() {
		t.Error("Expected Allow() to return true after token refill")
	}
}

func TestRateLimiter(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		Dimension:         "ip",
		Enabled:           true,
	}

	rl := NewRateLimiter(config)

	// 测试全局限流
	for i := 0; i < 5; i++ {
		if !rl.Allow("192.168.1.1") {
			t.Errorf("Expected Allow() to return true for request %d", i+1)
		}
	}

	// 突发流量耗尽后应该被限制
	if rl.Allow("192.168.1.1") {
		t.Error("Expected Allow() to return false after burst exhausted")
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		Dimension:         "ip",
		Enabled:           false,
	}

	rl := NewRateLimiter(config)

	// 禁用状态下应该始终允许
	for i := 0; i < 100; i++ {
		if !rl.Allow("192.168.1.1") {
			t.Errorf("Expected Allow() to return true when disabled, request %d", i+1)
		}
	}
}

func TestRateLimiterStats(t *testing.T) {
	config := RateLimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		Dimension:         "ip",
		Enabled:           true,
	}

	rl := NewRateLimiter(config)

	stats := rl.GetStats()

	if stats["enabled"] != true {
		t.Error("Expected enabled to be true")
	}

	if stats["dimension"] != "ip" {
		t.Errorf("Expected dimension to be 'ip', got %v", stats["dimension"])
	}
}

func TestCircuitBreaker(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          time.Second,
		Enabled:          true,
	}

	cb := NewCircuitBreaker(config)

	// 初始状态应该是关闭的
	if cb.GetState() != StateClosed {
		t.Error("Expected initial state to be closed")
	}

	// 允许请求通过
	if !cb.Allow() {
		t.Error("Expected Allow() to return true in closed state")
	}

	// 记录 3 次失败，应该触发熔断
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.GetState() != StateOpen {
		t.Error("Expected state to be open after threshold failures")
	}

	// 熔断状态下应该拒绝请求
	if cb.Allow() {
		t.Error("Expected Allow() to return false in open state")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
		Enabled:          true,
	}

	cb := NewCircuitBreaker(config)

	// 触发熔断
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetState() != StateOpen {
		t.Error("Expected state to be open")
	}

	// 等待超时
	time.Sleep(150 * time.Millisecond)

	// 应该进入半开状态
	if !cb.Allow() {
		t.Error("Expected Allow() to return true after timeout")
	}

	if cb.GetState() != StateHalfOpen {
		t.Error("Expected state to be half-open")
	}

	// 记录成功
	cb.RecordSuccess()
	cb.RecordSuccess()

	// 应该恢复到关闭状态
	if cb.GetState() != StateClosed {
		t.Error("Expected state to be closed after successful requests")
	}
}

func TestCircuitBreakerDisabled(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		Enabled:          false,
	}

	cb := NewCircuitBreaker(config)

	// 禁用状态下应该始终允许
	for i := 0; i < 10; i++ {
		cb.RecordFailure()
		if !cb.Allow() {
			t.Errorf("Expected Allow() to return true when disabled, failure %d", i+1)
		}
	}
}

func TestAuthMiddleware(t *testing.T) {
	// 测试 API Key 认证
	config := AuthConfig{
		Type:    "apikey",
		Enabled: true,
		APIKeys: map[string]string{
			"valid-key-123": "user1",
		},
	}

	mw := NewAuthMiddleware(config)

	// 由于 Authenticate 需要 http.Request，这里只测试基本功能
	if mw == nil {
		t.Error("Expected middleware to be created")
	}

	stats := mw.GetStats()
	if stats["type"] != "apikey" {
		t.Errorf("Expected type to be 'apikey', got %v", stats["type"])
	}
}

func TestACLMiddleware(t *testing.T) {
	config := ACLConfig{
		Enabled:       true,
		DefaultPolicy: "allow",
	}

	mw := NewACLMiddleware(config)

	// 没有规则时应该允许所有 IP
	if !mw.Allow("192.168.1.1") {
		t.Error("Expected Allow() to return true with no rules")
	}

	stats := mw.GetStats()
	if stats["enabled"] != true {
		t.Error("Expected enabled to be true")
	}
}

func TestReplayProtection(t *testing.T) {
	rp := NewReplayProtection(time.Minute)

	nonce := "unique-nonce-123"
	timestamp := time.Now().Unix()

	// 第一次检查应该通过
	if err := rp.Check(nonce, timestamp); err != nil {
		t.Errorf("Expected Check() to return nil, got %v", err)
	}

	// 重复使用相同的 nonce 应该失败
	if err := rp.Check(nonce, timestamp); err == nil {
		t.Error("Expected Check() to return error for duplicate nonce")
	}

	// 过期的时间戳应该失败
	expiredTimestamp := time.Now().Add(-2 * time.Minute).Unix()
	if err := rp.Check("new-nonce", expiredTimestamp); err == nil {
		t.Error("Expected Check() to return error for expired timestamp")
	}
}

func TestGeoIPBlocker(t *testing.T) {
	gb := NewGeoIPBlocker([]string{"CN", "RU"})

	gb.Enable()

	if !gb.IsBlocked("CN") {
		t.Error("Expected CN to be blocked")
	}

	if !gb.IsBlocked("RU") {
		t.Error("Expected RU to be blocked")
	}

	if gb.IsBlocked("US") {
		t.Error("Expected US to not be blocked")
	}

	// 禁用后应该都不封锁
	gb.Disable()
	if gb.IsBlocked("CN") {
		t.Error("Expected CN to not be blocked after disable")
	}
}

func TestAuditLog(t *testing.T) {
	al := NewAuditLog(10)

	// 添加条目
	al.Add("login", "192.168.1.1", "user1", map[string]interface{}{
		"success": true,
	})

	al.Add("logout", "192.168.1.1", "user1", nil)

	entries := al.GetEntries(10)
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	// 测试大小限制
	for i := 0; i < 20; i++ {
		al.Add("action", "192.168.1.1", "user1", nil)
	}

	entries = al.GetEntries(100)
	if len(entries) > 10 {
		t.Errorf("Expected max 10 entries, got %d", len(entries))
	}
}
