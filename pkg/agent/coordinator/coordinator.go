package coordinator

import (
	"sync"
	"time"
)

// Config 协调器配置
type Config struct {
	Enabled         bool `json:"enabled"`
	MaxHistorySize  int  `json:"max_history_size"`
	EnableTracking  bool `json:"enable_tracking"`
}

// TaskRecord 任务记录
type TaskRecord struct {
	TaskID      string
	Task        interface{}
	Result      interface{}
	Status      string
	CreatedAt   time.Time
	CompletedAt time.Time
}

// Coordinator 协调器 Agent
type Coordinator struct {
	config  Config
	history []*TaskRecord
	mu      sync.RWMutex
}

// NewCoordinator 创建协调器
func NewCoordinator(cfg Config) *Coordinator {
	return &Coordinator{
		config:  cfg,
		history: make([]*TaskRecord, 0),
	}
}

// RecordCompletion 记录任务完成
func (c *Coordinator) RecordCompletion(task interface{}, result interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.config.Enabled {
		return
	}

	record := &TaskRecord{
		TaskID:      generateTaskID(),
		Task:        task,
		Result:      result,
		Status:      "completed",
		CreatedAt:   time.Now(),
		CompletedAt: time.Now(),
	}

	c.history = append(c.history, record)

	// 限制历史记录大小
	if len(c.history) > c.config.MaxHistorySize {
		c.history = c.history[1:]
	}
}

// GetHistory 获取历史记录
func (c *Coordinator) GetHistory(limit int) []*TaskRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit > len(c.history) {
		limit = len(c.history)
	}

	return c.history[len(c.history)-limit:]
}

// GetStats 获取统计信息
func (c *Coordinator) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"total_tasks":    len(c.history),
		"enabled":        c.config.Enabled,
		"tracking":       c.config.EnableTracking,
		"max_history":    c.config.MaxHistorySize,
	}
}

// UpdateConfig 更新配置
func (c *Coordinator) UpdateConfig(cfg Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = cfg
}

// GetConfig 获取当前配置
func (c *Coordinator) GetConfig() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}
