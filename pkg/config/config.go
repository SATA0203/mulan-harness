package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Backend 后端服务配置
type Backend struct {
	Address string `json:"address"`
	Weight  int    `json:"weight,omitempty"`
}

// RouteConfig 路由配置
type RouteConfig struct {
	Name        string            `json:"name"`
	Host        string            `json:"host,omitempty"`
	Path        string            `json:"path,omitempty"`
	PathPrefix  string            `json:"path_prefix,omitempty"`
	Methods     []string          `json:"methods,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Backends    []Backend         `json:"backends"`
	LBAlgorithm string            `json:"lb_algorithm,omitempty"` // round_robin, random, least_conn, consistent_hash
	Timeout     time.Duration     `json:"timeout,omitempty"`
	Retries     int               `json:"retries,omitempty"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled bool   `json:"enabled"`
	QPS     int    `json:"qps"`
	Burst   int    `json:"burst"`
	KeyType string `json:"key_type,omitempty"` // ip, user_id, api
}

// CircuitBreakerConfig 熔断配置
type CircuitBreakerConfig struct {
	Enabled       bool          `json:"enabled"`
	Threshold     int           `json:"threshold,omitempty"`     // 错误次数阈值
	Window        time.Duration `json:"window,omitempty"`        // 统计窗口
	HalfOpenCount int           `json:"half_open_count,omitempty"` // 半开状态探测请求数
	Timeout       time.Duration `json:"timeout,omitempty"`       // 熔断超时后进入半开状态
}

// AuthConfig 认证配置
type AuthConfig struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type,omitempty"` // jwt, oauth2, basic_auth
	Secret  string `json:"secret,omitempty"`
}

// PluginConfig 插件配置
type PluginConfig struct {
	Name     string                 `json:"name"`
	Enabled  bool                   `json:"enabled"`
	Priority int                    `json:"priority,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// LogConfig 日志配置
type LogConfig struct {
	Enabled      bool   `json:"enabled"`
	Level        string `json:"level,omitempty"`
	Format       string `json:"format,omitempty"` // json, text
	Output       string `json:"output,omitempty"`
	SampleRate   float64 `json:"sample_rate,omitempty"`
	IncludeBody  bool   `json:"include_body,omitempty"`
}

// GatewayConfig 网关总配置
type GatewayConfig struct {
	ServerAddr    string              `json:"server_addr"`
	ReadTimeout   time.Duration       `json:"read_timeout,omitempty"`
	WriteTimeout  time.Duration       `json:"write_timeout,omitempty"`
	IdleTimeout   time.Duration       `json:"idle_timeout,omitempty"`
	Routes        []RouteConfig       `json:"routes"`
	RateLimit     RateLimitConfig     `json:"rate_limit,omitempty"`
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
	Auth          AuthConfig          `json:"auth,omitempty"`
	ACL           ACLConfig           `json:"acl,omitempty"`
	Plugins       []PluginConfig      `json:"plugins,omitempty"`
	Log           LogConfig           `json:"log,omitempty"`
	HealthCheck   HealthCheckConfig   `json:"health_check,omitempty"`
	Harness       HarnessConfig       `json:"harness,omitempty"`
	Agent         AgentConfig         `json:"agent,omitempty"`
	Evolution     EvolutionConfig     `json:"evolution,omitempty"`
}

// ACLConfig 访问控制配置
type ACLConfig struct {
	Enabled       bool     `json:"enabled"`
	Whitelist     []string `json:"whitelist,omitempty"`
	Blacklist     []string `json:"blacklist,omitempty"`
	DefaultPolicy string   `json:"default_policy,omitempty"` // allow, deny
}

// HarnessConfig Harness 管控层配置
type HarnessConfig struct {
	Enabled    bool                   `json:"enabled"`
	AuthConfig map[string]interface{} `json:"auth_config,omitempty"`
	ComplianceConfig map[string]interface{} `json:"compliance_config,omitempty"`
	AuditConfig      map[string]interface{} `json:"audit_config,omitempty"`
}

// AgentConfig 多 Agent 协作配置
type AgentConfig struct {
	Enabled       bool              `json:"enabled"`
	PlannerStrategy string          `json:"planner_strategy,omitempty"` // sequential, parallel, dynamic
	MaxRetries    int               `json:"max_retries,omitempty"`
	Timeout       time.Duration     `json:"timeout,omitempty"`
	Routes        map[string]bool   `json:"routes,omitempty"` // 启用 Agent 处理的路由
}

// EvolutionConfig 自进化底座配置
type EvolutionConfig struct {
	Enabled      bool          `json:"enabled"`
	SkillLimit   int           `json:"skill_limit,omitempty"`
	MemoryLimit  int           `json:"memory_limit,omitempty"`
	OptimizeInterval time.Duration `json:"optimize_interval,omitempty"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Enabled     bool          `json:"enabled"`
	Interval    time.Duration `json:"interval,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	UnhealthyThreshold int    `json:"unhealthy_threshold,omitempty"`
	HealthyThreshold   int    `json:"healthy_threshold,omitempty"`
	Path        string        `json:"path,omitempty"`
}

// ConfigManager 配置管理器，支持热更新
type ConfigManager struct {
	mu       sync.RWMutex
	config   *GatewayConfig
	watchers []chan *GatewayConfig
}

// NewConfigManager 创建配置管理器
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config:   &GatewayConfig{},
		watchers: make([]chan *GatewayConfig, 0),
	}
}

// LoadFromFile 从文件加载配置
func (cm *ConfigManager) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败：%w", err)
	}
	return cm.LoadFromJSON(data)
}

// LoadFromJSON 从 JSON 数据加载配置
func (cm *ConfigManager) LoadFromJSON(data []byte) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var cfg GatewayConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("解析配置失败：%w", err)
	}

	oldConfig := cm.config
	cm.config = &cfg

	// 通知所有观察者配置已更新
	for _, ch := range cm.watchers {
		select {
		case ch <- &cfg:
		default:
			// 如果通道满了，跳过
		}
	}

	_ = oldConfig // 可用于回滚
	return nil
}

// GetConfig 获取当前配置（只读）
func (cm *ConfigManager) GetConfig() *GatewayConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// RegisterWatcher 注册配置变更观察者
func (cm *ConfigManager) RegisterWatcher() chan *GatewayConfig {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	ch := make(chan *GatewayConfig, 1)
	cm.watchers = append(cm.watchers, ch)
	return ch
}

// Validate 验证配置有效性
func (cfg *GatewayConfig) Validate() error {
	if cfg.ServerAddr == "" {
		return fmt.Errorf("server_addr 不能为空")
	}

	for i, route := range cfg.Routes {
		if route.Name == "" {
			return fmt.Errorf("路由 [%d] 缺少 name", i)
		}
		if len(route.Backends) == 0 {
			return fmt.Errorf("路由 [%s] 缺少 backends", route.Name)
		}
		for j, backend := range route.Backends {
			if backend.Address == "" {
				return fmt.Errorf("路由 [%s] 的后端 [%d] 缺少 address", route.Name, j)
			}
		}
	}

	return nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() *GatewayConfig {
	return &GatewayConfig{
		ServerAddr:   ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		Log: LogConfig{
			Enabled:    true,
			Level:      "info",
			Format:     "json",
			SampleRate: 1.0,
		},
		HealthCheck: HealthCheckConfig{
			Enabled:            true,
			Interval:           10 * time.Second,
			Timeout:            5 * time.Second,
			UnhealthyThreshold: 3,
			HealthyThreshold:   2,
			Path:               "/health",
		},
	}
}
