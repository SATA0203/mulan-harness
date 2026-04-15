package validator

import (
	"sync"
)

// Config 校验器配置
type Config struct {
	Enabled      bool     `json:"enabled"`
	Rules        []string `json:"rules"`
	StrictMode   bool     `json:"strict_mode"`
	FailFast     bool     `json:"fail_fast"`
}

// Validator 校验器 Agent
type Validator struct {
	config Config
	mu     sync.RWMutex
}

// NewValidator 创建校验器
func NewValidator(cfg Config) *Validator {
	return &Validator{
		config: cfg,
	}
}

// Validate 校验执行结果
func (v *Validator) Validate(result interface{}, context map[string]interface{}) (bool, []error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.config.Enabled {
		return true, nil
	}

	// 简化实现：默认通过校验
	return true, nil
}

// UpdateConfig 更新配置
func (v *Validator) UpdateConfig(cfg Config) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.config = cfg
}

// GetConfig 获取当前配置
func (v *Validator) GetConfig() Config {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.config
}
