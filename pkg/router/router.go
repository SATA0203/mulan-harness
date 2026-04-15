package router

import (
	"net/http"
	"regexp"
	"strings"
	"sync"

	"smartgateway/pkg/config"
	"smartgateway/pkg/loadbalancer"
)

// RouteMatcher 路由匹配器
type RouteMatcher interface {
	// Match 检查请求是否匹配该路由
	Match(req *http.Request) bool
}

// SimpleRouteMatcher 简单路由匹配器（基于 Host、Path、Method、Headers）
type SimpleRouteMatcher struct {
	Host       string
	Path       string
	PathPrefix string
	Methods    []string
	Headers    map[string]string
	hostRegex  *regexp.Regexp
	pathRegex  *regexp.Regexp
}

// NewSimpleRouteMatcher 创建简单路由匹配器
func NewSimpleRouteMatcher(route config.RouteConfig) *SimpleRouteMatcher {
	matcher := &SimpleRouteMatcher{
		Host:       route.Host,
		Path:       route.Path,
		PathPrefix: route.PathPrefix,
		Methods:    route.Methods,
		Headers:    route.Headers,
	}

	// 编译正则表达式（如果配置了正则）
	if route.Host != "" && strings.Contains(route.Host, "*") {
		pattern := strings.ReplaceAll(route.Host, ".", "\\.")
		pattern = strings.ReplaceAll(pattern, "*", ".*")
		matcher.hostRegex = regexp.MustCompile("^" + pattern + "$")
	}

	if route.Path != "" && strings.Contains(route.Path, "*") {
		pattern := strings.ReplaceAll(route.Path, "*", ".*")
		matcher.pathRegex = regexp.MustCompile("^" + pattern + "$")
	}

	return matcher
}

// Match 检查请求是否匹配
func (m *SimpleRouteMatcher) Match(req *http.Request) bool {
	// 检查 Host
	if m.Host != "" {
		if m.hostRegex != nil {
			if !m.hostRegex.MatchString(req.Host) {
				return false
			}
		} else {
			if req.Host != m.Host {
				return false
			}
		}
	}

	// 检查 Path
	if m.Path != "" {
		if m.pathRegex != nil {
			if !m.pathRegex.MatchString(req.URL.Path) {
				return false
			}
		} else {
			if req.URL.Path != m.Path {
				return false
			}
		}
	}

	// 检查 PathPrefix
	if m.PathPrefix != "" {
		if !strings.HasPrefix(req.URL.Path, m.PathPrefix) {
			return false
		}
	}

	// 检查 Method
	if len(m.Methods) > 0 {
		found := false
		for _, method := range m.Methods {
			if req.Method == method {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查 Headers
	for key, value := range m.Headers {
		if req.Header.Get(key) != value {
			return false
		}
	}

	return true
}

// Route 路由条目
type Route struct {
	Name       string
	Matcher    RouteMatcher
	LB         loadbalancer.LoadBalancer
	Config     config.RouteConfig
	Timeout    int64 // 超时时间（毫秒）
	Retries    int
	Middleware []MiddlewareFunc
}

// MiddlewareFunc 中间件函数类型
type MiddlewareFunc func(http.Handler) http.Handler

// Router 路由器
type Router struct {
	mu            sync.RWMutex
	routes        []*Route
	configManager *config.ConfigManager
}

// NewRouter 创建路由器
func NewRouter(configManager *config.ConfigManager) *Router {
	router := &Router{
		routes:        make([]*Route, 0),
		configManager: configManager,
	}

	// 注册配置变更监听
	go router.watchConfigChanges()

	return router
}

// watchConfigChanges 监听配置变更
func (r *Router) watchConfigChanges() {
	watcher := r.configManager.RegisterWatcher()
	for cfg := range watcher {
		r.UpdateRoutes(cfg)
	}
}

// UpdateRoutes 更新路由配置
func (r *Router) UpdateRoutes(cfg *config.GatewayConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	newRoutes := make([]*Route, 0, len(cfg.Routes))

	for _, routeCfg := range cfg.Routes {
		matcher := NewSimpleRouteMatcher(routeCfg)
		lb := loadbalancer.NewLoadBalancer(routeCfg.LBAlgorithm)

		// 添加后端节点
		nodes := loadbalancer.CreateBackendsFromConfig(routeCfg.Backends)
		for _, node := range nodes {
			lb.Add(node)
		}

		timeout := int64(routeCfg.Timeout.Milliseconds())
		if timeout == 0 {
			timeout = 30000 // 默认 30 秒
		}

		retries := routeCfg.Retries
		if retries == 0 {
			retries = 1
		}

		route := &Route{
			Name:    routeCfg.Name,
			Matcher: matcher,
			LB:      lb,
			Config:  routeCfg,
			Timeout: timeout,
			Retries: retries,
		}

		newRoutes = append(newRoutes, route)
	}

	r.routes = newRoutes
}

// LoadFromConfig 从配置加载路由
func (r *Router) LoadFromConfig(cfg *config.GatewayConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = make([]*Route, 0, len(cfg.Routes))

	for _, routeCfg := range cfg.Routes {
		matcher := NewSimpleRouteMatcher(routeCfg)
		lb := loadbalancer.NewLoadBalancer(routeCfg.LBAlgorithm)

		// 添加后端节点
		nodes := loadbalancer.CreateBackendsFromConfig(routeCfg.Backends)
		for _, node := range nodes {
			lb.Add(node)
		}

		timeout := int64(routeCfg.Timeout.Milliseconds())
		if timeout == 0 {
			timeout = 30000
		}

		retries := routeCfg.Retries
		if retries == 0 {
			retries = 1
		}

		route := &Route{
			Name:    routeCfg.Name,
			Matcher: matcher,
			LB:      lb,
			Config:  routeCfg,
			Timeout: timeout,
			Retries: retries,
		}

		r.routes = append(r.routes, route)
	}

	return nil
}

// FindRoute 查找匹配的路由
func (r *Router) FindRoute(req *http.Request) *Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.routes {
		if route.Matcher.Match(req) {
			return route
		}
	}

	return nil
}

// AddMiddleware 为指定路由添加中间件
func (r *Router) AddMiddleware(routeName string, middleware MiddlewareFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, route := range r.routes {
		if route.Name == routeName {
			route.Middleware = append(route.Middleware, middleware)
			return nil
		}
	}

	return nil
}

// GetAllRoutes 获取所有路由（用于监控和管理）
func (r *Router) GetAllRoutes() []*Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Route, len(r.routes))
	copy(result, r.routes)
	return result
}

// GetRouteByName 根据名称获取路由
func (r *Router) GetRouteByName(name string) *Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.routes {
		if route.Name == name {
			return route
		}
	}

	return nil
}
