package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"smartgateway/pkg/agent"
	"smartgateway/pkg/agent/coordinator"
	"smartgateway/pkg/agent/executor"
	"smartgateway/pkg/agent/planner"
	"smartgateway/pkg/agent/validator"
	"smartgateway/pkg/config"
	"smartgateway/pkg/evolution"
	"smartgateway/pkg/evolution/memory"
	"smartgateway/pkg/evolution/skill"
	"smartgateway/pkg/evolution/strategy"
	"smartgateway/pkg/harness"
	"smartgateway/pkg/harness/audit"
	"smartgateway/pkg/harness/auth"
	"smartgateway/pkg/harness/compliance"
	"smartgateway/pkg/health"
	"smartgateway/pkg/logging"
	"smartgateway/pkg/middleware"
	"smartgateway/pkg/router"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Addr         string        `json:"addr"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// GatewayServer 网关服务器
type GatewayServer struct {
	configMgr      *config.ConfigManager
	serverConfig   ServerConfig
	httpServer     *http.Server
	router         *router.Router
	healthChecker  *health.HealthChecker
	rateLimiter    *middleware.RateLimiter
	circuitBreaker *middleware.CircuitBreaker
	authMiddleware *middleware.AuthMiddleware
	aclMiddleware  *middleware.ACLMiddleware
	harness        *harness.Harness
	agent          *agent.AgentFramework
	evolution      *evolution.SelfEvolutionBase
	metrics        *Metrics
	mu             sync.RWMutex
}

// Metrics 监控指标
type Metrics struct {
	RequestCount   map[string]int64
	ErrorCount     map[string]int64
	LatencySum     map[string]float64
	LatencyCount   map[string]int64
	mu             sync.RWMutex
}

// NewMetrics 创建监控指标
func NewMetrics() *Metrics {
	return &Metrics{
		RequestCount: make(map[string]int64),
		ErrorCount:   make(map[string]int64),
		LatencySum:   make(map[string]float64),
		LatencyCount: make(map[string]int64),
	}
}

// RecordRequest 记录请求
func (m *Metrics) RecordRequest(route string, statusCode int, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RequestCount[route]++
	m.LatencySum[route] += float64(latency.Microseconds())
	m.LatencyCount[route]++

	if statusCode >= 400 {
		m.ErrorCount[route]++
	}
}

// GetStats 获取统计信息
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	for route, count := range m.RequestCount {
		latencyAvg := float64(0)
		if m.LatencyCount[route] > 0 {
			latencyAvg = m.LatencySum[route] / float64(m.LatencyCount[route])
		}

		stats[route] = map[string]interface{}{
			"request_count":   count,
			"error_count":     m.ErrorCount[route],
			"avg_latency_us":  latencyAvg,
			"error_rate":      float64(m.ErrorCount[route]) / float64(count),
		}
	}

	return stats
}

// NewGatewayServer 创建网关服务器
func NewGatewayServer(cfgMgr *config.ConfigManager) (*GatewayServer, error) {
	cfg := cfgMgr.GetConfig()
	
	server := &GatewayServer{
		configMgr: cfgMgr,
		serverConfig: ServerConfig{
			Addr:         cfg.ServerAddr,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		router:        router.NewRouter(cfgMgr),
		healthChecker: health.NewHealthChecker(
			cfg.HealthCheck.Interval,
			cfg.HealthCheck.Timeout,
			cfg.HealthCheck.UnhealthyThreshold,
			cfg.HealthCheck.HealthyThreshold,
			cfg.HealthCheck.Path,
		),
		metrics: NewMetrics(),
	}

	// 初始化中间件
	if cfg.RateLimit.Enabled {
		rateConfig := middleware.RateLimiterConfig{
			RequestsPerSecond: cfg.RateLimit.QPS,
			BurstSize:         cfg.RateLimit.Burst,
			Dimension:         cfg.RateLimit.KeyType,
			Enabled:           cfg.RateLimit.Enabled,
		}
		server.rateLimiter = middleware.NewRateLimiter(rateConfig)
	}

	if cfg.CircuitBreaker.Enabled {
		cbConfig := middleware.CircuitBreakerConfig{
			FailureThreshold: cfg.CircuitBreaker.Threshold,
			SuccessThreshold: cfg.CircuitBreaker.HalfOpenCount,
			Timeout:          cfg.CircuitBreaker.Timeout,
			Enabled:          cfg.CircuitBreaker.Enabled,
		}
		server.circuitBreaker = middleware.NewCircuitBreaker(cbConfig)
	}

	if cfg.Auth.Enabled {
		authConfig := middleware.AuthConfig{
			Type:    cfg.Auth.Type,
			JWTSecret: cfg.Auth.Secret,
			Enabled: cfg.Auth.Enabled,
		}
		server.authMiddleware = middleware.NewAuthMiddleware(authConfig)
	}

	if cfg.ACL.Enabled {
		aclConfig := middleware.ACLConfig{
			Whitelist:     cfg.ACL.Whitelist,
			Blacklist:     cfg.ACL.Blacklist,
			Enabled:       cfg.ACL.Enabled,
			DefaultPolicy: cfg.ACL.DefaultPolicy,
		}
		server.aclMiddleware = middleware.NewACLMiddleware(aclConfig)
	}

	// 加载路由配置
	if err := server.router.LoadFromConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to load routes: %w", err)
	}

	// 初始化 Harness 管控层
	if cfg.Harness.Enabled {
		harnessCfg := harness.Config{
			AuthConfig: auth.Config{
				Enabled:       true,
				AllowedRoles:  []string{"admin", "user"},
				DeniedRoles:   []string{},
				RequireAuth:   false,
				DefaultPolicy: "allow",
			},
			ComplianceConfig: compliance.Config{
				Enabled:           true,
				RequiredHeaders:   []string{},
				BlockedPaths:      []string{},
				MaxBodySize:       10 << 20, // 10MB
				AllowedMethods:    []string{"GET", "POST", "PUT", "DELETE"},
			},
			AuditConfig: audit.Config{
				Enabled:        true,
				LogLevel:       "info",
				IncludeBody:    false,
				MaxBodyLength:  1024,
				RetentionDays:  30,
				OutputFormat:   "json",
			},
		}
		server.harness = harness.NewHarness(harnessCfg)
		logging.Info("Harness 管控层已初始化")
	}

	// 初始化多 Agent 协作框架
	if cfg.Agent.Enabled {
		agentCfg := agent.Config{
			PlannerConfig: planner.Config{
				Enabled:        true,
				Strategy:       cfg.Agent.PlannerStrategy,
				MaxSteps:       10,
				TimeoutPerStep: int(cfg.Agent.Timeout.Seconds()),
				PriorityRules:  []string{},
			},
			ExecutorConfig: executor.Config{
				Enabled:        true,
				MaxConcurrency: 5,
				Timeout:        int(cfg.Agent.Timeout.Seconds()),
				RetryCount:     cfg.Agent.MaxRetries,
				Strategy:       "sync",
			},
			ValidatorConfig: validator.Config{
				Enabled:    true,
				Rules:      []string{},
				StrictMode: true,
				FailFast:   true,
			},
			CoordinatorConfig: coordinator.Config{
				Enabled:        true,
				MaxHistorySize: 100,
				EnableTracking: true,
			},
		}
		server.agent = agent.NewAgentFramework(agentCfg)
		logging.Info("多 Agent 协作框架已初始化", map[string]interface{}{
			"strategy": cfg.Agent.PlannerStrategy,
		})
	}

	// 初始化自进化底座
	if cfg.Evolution.Enabled {
		evolutionCfg := evolution.Config{
			SkillConfig: skill.Config{
				Enabled:      true,
				MaxSkills:    cfg.Evolution.SkillLimit,
				AutoRegister: true,
			},
			MemoryConfig: memory.Config{
				Enabled:          true,
				MaxMemories:      cfg.Evolution.MemoryLimit,
				RetentionDays:    7,
				EnableForgetting: true,
						},
			StrategyConfig: strategy.Config{
				Enabled:          true,
				OptimizationAlgo: "rule_based",
				LearningRate:     0.01,
				MinScore:         0.8,
			},
		}
		server.evolution = evolution.NewSelfEvolutionBase(evolutionCfg)
		logging.Info("自进化底座已初始化", map[string]interface{}{
			"skill_limit":  cfg.Evolution.SkillLimit,
			"memory_limit": cfg.Evolution.MemoryLimit,
		})
	}

	return server, nil
}
