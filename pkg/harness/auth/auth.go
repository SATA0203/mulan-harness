package auth

import (
	"errors"
	"sync"
)

// Config 权限控制配置
type Config struct {
	Enabled       bool     `json:"enabled"`
	AllowedRoles  []string `json:"allowed_roles"`
	DeniedRoles   []string `json:"denied_roles"`
	RequireAuth   bool     `json:"require_auth"`
	DefaultPolicy string   `json:"default_policy"` // allow/deny
}

// AuthController 权限控制器
type AuthController struct {
	config Config
	mu     sync.RWMutex
}

// NewAuthController 创建权限控制器
func NewAuthController(cfg Config) *AuthController {
	return &AuthController{
		config: cfg,
	}
}

// CheckPermission 检查权限
func (a *AuthController) CheckPermission(req interface{}, context map[string]interface{}) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.config.Enabled {
		return true, nil
	}

	// 获取角色信息
	role, ok := context["role"].(string)
	if !ok {
		if a.config.RequireAuth {
			return false, errors.New("authentication required")
		}
		// 使用默认策略
		return a.config.DefaultPolicy == "allow", nil
	}

	// 检查是否在拒绝列表中
	for _, denied := range a.config.DeniedRoles {
		if denied == role {
			return false, errors.New("role denied")
		}
	}

	// 检查是否在允许列表中（如果配置了）
	if len(a.config.AllowedRoles) > 0 {
		for _, allowed := range a.config.AllowedRoles {
			if allowed == role {
				return true, nil
			}
		}
		return false, errors.New("role not allowed")
	}

	return true, nil
}

// UpdateConfig 更新配置
func (a *AuthController) UpdateConfig(cfg Config) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config = cfg
}

// GetConfig 获取当前配置
func (a *AuthController) GetConfig() Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}
