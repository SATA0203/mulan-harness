package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"smartgateway/pkg/config"
	"smartgateway/pkg/health"
	"smartgateway/pkg/loadbalancer"
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

	// 注册健康检查
	for _, routeCfg := range cfg.Routes {
		nodes := loadbalancer.CreateBackendsFromConfig(routeCfg.Backends)
		for _, node := range nodes {
			server.healthChecker.RegisterBackend(node)
		}
	}

	return server, nil
}

// updateRoutes 更新路由配置
func (s *GatewayServer) updateRoutes(cfg *config.GatewayConfig) error {
	return s.router.LoadFromConfig(cfg)
}

// Start 启动服务器
func (s *GatewayServer) Start() error {
	handler := s.buildHandler()

	s.httpServer = &http.Server{
		Addr:         s.serverConfig.Addr,
		Handler:      handler,
		ReadTimeout:  s.serverConfig.ReadTimeout,
		WriteTimeout: s.serverConfig.WriteTimeout,
		IdleTimeout:  s.serverConfig.IdleTimeout,
	}

	// 启动健康检查
	s.healthChecker.Start()

	logging.Info("Gateway server starting", map[string]interface{}{"addr": s.serverConfig.Addr})

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// Shutdown 优雅停机
func (s *GatewayServer) Shutdown(ctx context.Context) error {
	logging.Info("Gateway server shutting down")

	// 停止健康检查
	s.healthChecker.Stop()

	// 优雅关闭 HTTP 服务器
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// buildHandler 构建 HTTP 处理器
func (s *GatewayServer) buildHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// 提取客户端 IP
		clientIP := getClientIP(r)

		// ACL 检查
		if s.aclMiddleware != nil && !s.aclMiddleware.Allow(clientIP) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			s.recordMetrics(r, 403, time.Since(startTime))
			return
		}

		// 限流检查
		if s.rateLimiter != nil && !s.rateLimiter.Allow(clientIP) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			s.recordMetrics(r, 429, time.Since(startTime))
			return
		}

		// 熔断检查
		if s.circuitBreaker != nil && !s.circuitBreaker.Allow() {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			s.recordMetrics(r, 503, time.Since(startTime))
			return
		}

		// 认证检查
		if s.authMiddleware != nil {
			if _, err := s.authMiddleware.Authenticate(r); err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				s.recordMetrics(r, 401, time.Since(startTime))
				return
			}
		}

		// 路由匹配
		route := s.router.FindRoute(r)
		if route == nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			s.recordMetrics(r, 404, time.Since(startTime))
			return
		}

		// 选择后端
		node := route.LB.Next(r)
		if node == nil {
			http.Error(w, "No Available Backend", http.StatusBadGateway)
			s.recordMetrics(r, 502, time.Since(startTime))
			return
		}

		// 增加连接数
		node.IncrConns()
		defer node.DecrConns()

		// 转发请求
		s.proxyRequest(w, r, node, route)
	})
}

// proxyRequest 代理请求到后端
func (s *GatewayServer) proxyRequest(w http.ResponseWriter, r *http.Request, node *loadbalancer.BackendNode, route *router.Route) {
	backendURL := node.URL
	
	// 创建新的请求
	newURL := *r.URL
	newURL.Scheme = backendURL.Scheme
	newURL.Host = backendURL.Host

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, newURL.String(), r.Body)
	if err != nil {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		if s.circuitBreaker != nil {
			s.circuitBreaker.RecordFailure()
		}
		return
	}

	// 复制请求头
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// 添加 X-Forwarded-* 头
	proxyReq.Header.Set("X-Forwarded-For", getClientIP(r))
	proxyReq.Header.Set("X-Forwarded-Proto", getProto(r))
	proxyReq.Header.Set("X-Forwarded-Host", r.Host)

	// 执行请求
	timeout := time.Duration(route.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	
	client := &http.Client{
		Timeout: timeout,
	}

	startTime := time.Now()
	resp, err := client.Do(proxyReq)
	latency := time.Since(startTime)
	
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		if s.circuitBreaker != nil {
			s.circuitBreaker.RecordFailure()
		}
		node.IncrFail()
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 复制状态码
	w.WriteHeader(resp.StatusCode)

	// 复制响应体
	io.Copy(w, resp.Body)

	// 记录成功/失败
	if s.circuitBreaker != nil {
		if resp.StatusCode < 500 {
			s.circuitBreaker.RecordSuccess()
		} else {
			s.circuitBreaker.RecordFailure()
		}
	}

	// 记录指标
	s.recordMetrics(r, resp.StatusCode, latency)
	
	// 记录访问日志
	logging.AccessLog(&logging.AccessLogEntry{
		Method:       r.Method,
		Path:         r.URL.Path,
		Host:         r.Host,
		RemoteAddr:   getClientIP(r),
		UserAgent:    r.UserAgent(),
		StatusCode:   resp.StatusCode,
		Duration:     latency.Milliseconds(),
		UpstreamAddr: backendURL.Host,
	})
}

// stripPrefix 剥离路径前缀
func stripPrefix(path, prefix string) string {
	if prefix == "" {
		return path
	}
	if len(path) > len(prefix) && path[:len(prefix)] == prefix {
		return path[len(prefix):]
	}
	return path
}

// getClientIP 获取客户端 IP
func getClientIP(r *http.Request) string {
	// 尝试 X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// 尝试 X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// 使用 RemoteAddr
	return r.RemoteAddr
}

// getProto 获取请求协议
func getProto(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// recordMetrics 记录监控指标
func (s *GatewayServer) recordMetrics(r *http.Request, statusCode int, latency time.Duration) {
	routeName := "unknown"
	if route := s.router.FindRoute(r); route != nil {
		routeName = route.Name
	}
	s.metrics.RecordRequest(routeName, statusCode, latency)
}

// GetMetrics 获取监控指标
func (s *GatewayServer) GetMetrics() map[string]interface{} {
	return s.metrics.GetStats()
}

// GetHealthStatus 获取健康状态
func (s *GatewayServer) GetHealthStatus() map[string]interface{} {
	return s.healthChecker.GetAllStatuses()
}

// ReloadConfig 热更新配置
func (s *GatewayServer) ReloadConfig(cfg *config.GatewayConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 更新路由
	if err := s.updateRoutes(cfg); err != nil {
		return fmt.Errorf("failed to update routes: %w", err)
	}

	// 更新限流配置
	if s.rateLimiter != nil && cfg.RateLimit.Enabled {
		rateConfig := middleware.RateLimiterConfig{
			RequestsPerSecond: cfg.RateLimit.QPS,
			BurstSize:         cfg.RateLimit.Burst,
			Dimension:         cfg.RateLimit.KeyType,
			Enabled:           cfg.RateLimit.Enabled,
		}
		s.rateLimiter.UpdateConfig(rateConfig)
	}

	// 更新熔断配置
	if s.circuitBreaker != nil && cfg.CircuitBreaker.Enabled {
		cbConfig := middleware.CircuitBreakerConfig{
			FailureThreshold: cfg.CircuitBreaker.Threshold,
			SuccessThreshold: cfg.CircuitBreaker.HalfOpenCount,
			Timeout:          cfg.CircuitBreaker.Timeout,
			Enabled:          cfg.CircuitBreaker.Enabled,
		}
		s.circuitBreaker.UpdateConfig(cbConfig)
	}

	// 更新 ACL 配置
	if s.aclMiddleware != nil && cfg.ACL.Enabled {
		aclConfig := middleware.ACLConfig{
			Whitelist:     cfg.ACL.Whitelist,
			Blacklist:     cfg.ACL.Blacklist,
			Enabled:       cfg.ACL.Enabled,
			DefaultPolicy: cfg.ACL.DefaultPolicy,
		}
		s.aclMiddleware.UpdateConfig(aclConfig)
	}

	logging.Info("Configuration reloaded successfully")
	return nil
}
