package middleware

import (
	"sync"
	"time"
)

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// 失败阈值（连续失败次数）
	FailureThreshold int `json:"failure_threshold"`
	// 成功阈值（半开状态下需要的成功次数）
	SuccessThreshold int `json:"success_threshold"`
	// 超时时间（秒）
	Timeout time.Duration `json:"timeout"`
	// 请求超时时间
	RequestTimeout time.Duration `json:"request_timeout"`
	// 是否启用
	Enabled bool `json:"enabled"`
}

// CircuitState 熔断器状态
type CircuitState int

const (
	StateClosed CircuitState = iota // 关闭状态（正常）
	StateOpen                       // 打开状态（熔断）
	StateHalfOpen                   // 半开状态（试探）
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config          CircuitBreakerConfig
	state           CircuitState
	failures        int
	successes       int
	lastFailureTime time.Time
	mu              sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}

	if config.SuccessThreshold <= 0 {
		cb.config.SuccessThreshold = 1
	}

	return cb
}

// Allow 检查是否允许请求通过
func (cb *CircuitBreaker) Allow() bool {
	if !cb.config.Enabled {
		return true
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateOpen:
		// 检查是否超过超时时间，如果是则进入半开状态
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			return true
		}
		return false

	case StateHalfOpen:
		// 半开状态允许有限请求通过
		return true

	default: // StateClosed
		return true
	}
}

// RecordSuccess 记录成功请求
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}

	case StateClosed:
		cb.failures = 0 // 重置失败计数
	}
}

// RecordFailure 记录失败请求
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = StateOpen
		}

	case StateHalfOpen:
		// 半开状态下任何失败都会重新打开熔断器
		cb.state = StateOpen
		cb.successes = 0
	}
}

// GetState 获取当前状态
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats 获取统计信息
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":            cb.state.String(),
		"failures":         cb.failures,
		"successes":        cb.successes,
		"failure_threshold": cb.config.FailureThreshold,
		"success_threshold": cb.config.SuccessThreshold,
		"timeout_seconds":  cb.config.Timeout.Seconds(),
		"last_failure_time": cb.lastFailureTime,
		"enabled":          cb.config.Enabled,
	}
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
}

// UpdateConfig 更新配置
func (cb *CircuitBreaker) UpdateConfig(config CircuitBreakerConfig) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.config = config
}
