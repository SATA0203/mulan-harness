package compliance

import (
	"errors"
	"sync"
)

// Config 合规校验配置
type Config struct {
	Enabled            bool     `json:"enabled"`
	RequiredHeaders    []string `json:"required_headers"`
	BlockedPaths       []string `json:"blocked_paths"`
	MaxBodySize        int64    `json:"max_body_size"`
	AllowedMethods     []string `json:"allowed_methods"`
	RequireContentType bool     `json:"require_content_type"`
}

// ComplianceEngine 合规引擎
type ComplianceEngine struct {
	config Config
	mu     sync.RWMutex
}

// NewComplianceEngine 创建合规引擎
func NewComplianceEngine(cfg Config) *ComplianceEngine {
	return &ComplianceEngine{
		config: cfg,
	}
}

// ValidationResult 校验结果
type ValidationResult struct {
	Compliant   bool
	Violations  []error
	Warnings    []string
}

// Validate 验证请求合规性
func (c *ComplianceEngine) Validate(req interface{}, context map[string]interface{}) (bool, []error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var violations []error

	if !c.config.Enabled {
		return true, nil
	}

	// 检查必需头部
	if headers, ok := context["headers"].(map[string]string); ok {
		for _, required := range c.config.RequiredHeaders {
			if _, exists := headers[required]; !exists {
				violations = append(violations, errors.New("missing required header: "+required))
			}
		}
	}

	// 检查 blocked paths
	if path, ok := context["path"].(string); ok {
		for _, blocked := range c.config.BlockedPaths {
			if path == blocked || len(blocked) > 0 && path[:len(blocked)] == blocked {
				violations = append(violations, errors.New("path blocked: "+path))
			}
		}
	}

	// 检查 HTTP 方法
	if method, ok := context["method"].(string); ok {
		if len(c.config.AllowedMethods) > 0 {
			allowed := false
			for _, m := range c.config.AllowedMethods {
				if m == method {
					allowed = true
					break
				}
			}
			if !allowed {
				violations = append(violations, errors.New("method not allowed: "+method))
			}
		}
	}

	// 检查 Content-Type
	if c.config.RequireContentType {
		if headers, ok := context["headers"].(map[string]string); ok {
			if _, exists := headers["Content-Type"]; !exists {
				violations = append(violations, errors.New("Content-Type header required"))
			}
		}
	}

	return len(violations) == 0, violations
}

// UpdateConfig 更新配置
func (c *ComplianceEngine) UpdateConfig(cfg Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = cfg
}

// GetConfig 获取当前配置
func (c *ComplianceEngine) GetConfig() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}
