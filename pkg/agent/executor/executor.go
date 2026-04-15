package executor

import (
	"sync"
)

// Config 执行器配置
type Config struct {
	Enabled        bool   `json:"enabled"`
	MaxConcurrency int    `json:"max_concurrency"`
	Timeout        int    `json:"timeout"`
	RetryCount     int    `json:"retry_count"`
	Strategy       string `json:"strategy"` // sync/async
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success   bool
	Data      interface{}
	Metadata  map[string]interface{}
	Error     error
}

// Executor 执行器 Agent
type Executor struct {
	config Config
	mu     sync.RWMutex
}

// NewExecutor 创建执行器
func NewExecutor(cfg Config) *Executor {
	return &Executor{
		config: cfg,
	}
}

// Execute 执行计划
func (e *Executor) Execute(plan interface{}, context map[string]interface{}) (*ExecutionResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.config.Enabled {
		return &ExecutionResult{
			Success: true,
			Data:    nil,
		}, nil
	}

	// 简化实现：返回成功结果
	return &ExecutionResult{
		Success: true,
		Data:    map[string]interface{}{"status": "executed"},
		Metadata: map[string]interface{}{
			"executor": "default",
		},
	}, nil
}

// UpdateConfig 更新配置
func (e *Executor) UpdateConfig(cfg Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = cfg
}

// GetConfig 获取当前配置
func (e *Executor) GetConfig() Config {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}
