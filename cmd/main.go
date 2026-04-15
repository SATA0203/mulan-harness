package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"smartgateway/pkg/config"
	"smartgateway/pkg/health"
	"smartgateway/pkg/logging"
	"smartgateway/pkg/router"
)

var (
	configFile string
	version    bool
)

func init() {
	flag.StringVar(&configFile, "config", "", "配置文件路径")
	flag.BoolVar(&version, "version", false, "显示版本号")
}

func main() {
	flag.Parse()

	if version {
		fmt.Println("SmartGateway v1.0.0")
		os.Exit(0)
	}

	// 初始化配置管理器
	cfgManager := config.NewConfigManager()

	// 如果指定了配置文件，从文件加载
	if configFile != "" {
		if err := cfgManager.LoadFromFile(configFile); err != nil {
			logging.Error("加载配置文件失败", map[string]interface{}{
				"error": err.Error(),
				"file":  configFile,
			})
			// 使用默认配置继续
			logging.Info("使用默认配置启动")
		} else {
			logging.Info("配置加载成功", map[string]interface{}{
				"file": configFile,
			})
		}
	} else {
		// 没有配置文件，使用默认配置
		cfgManager.LoadFromJSON([]byte(`{
			"server_addr": ":8080",
			"routes": []
		}`))
		logging.Info("使用默认配置启动（未指定配置文件）")
	}

	cfg := cfgManager.GetConfig()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		logging.Error("配置验证失败", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// 初始化路由器
	r := router.NewRouter(cfgManager)
	if err := r.LoadFromConfig(cfg); err != nil {
		logging.Error("加载路由失败", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// 初始化健康检查器
	hc := health.NewHealthChecker(
		cfg.HealthCheck.Interval,
		cfg.HealthCheck.Timeout,
		cfg.HealthCheck.UnhealthyThreshold,
		cfg.HealthCheck.HealthyThreshold,
		cfg.HealthCheck.Path,
	)

	// 注册所有后端节点进行健康检查
	for _, route := range cfg.Routes {
		for _, backend := range route.Backends {
			// 这里需要从 router 获取 backend node 引用
			// 简化处理：暂时跳过，后续通过 router 集成
			_ = backend
		}
	}

	// 启动健康检查
	if cfg.HealthCheck.Enabled {
		hc.Start()
		logging.Info("健康检查已启动")
	}

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:         cfg.ServerAddr,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		Handler:      createGatewayHandler(r, cfg),
	}

	// 优雅停机
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh

		logging.Info("收到退出信号，开始优雅停机", map[string]interface{}{
			"signal": sig.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logging.Error("服务器关闭失败", map[string]interface{}{
				"error": err.Error(),
			})
		}

		hc.Stop()
		logging.Info("服务器已停止")
	}()

	// 启动服务器
	logging.Info("SmartGateway 启动中", map[string]interface{}{
		"address": cfg.ServerAddr,
		"version": "1.0.0",
	})

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logging.Fatal("服务器启动失败", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// createGatewayHandler 创建网关处理器
func createGatewayHandler(r *router.Router, cfg *config.GatewayConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		startTime := time.Now()

		// 查找匹配的路由
		route := r.FindRoute(req)
		if route == nil {
			logging.AccessLog(&logging.AccessLogEntry{
				Method:     req.Method,
				Path:       req.URL.Path,
				Host:       req.Host,
				RemoteAddr: req.RemoteAddr,
				StatusCode: 404,
				Duration:   time.Since(startTime).Milliseconds(),
				Error:      "no matching route found",
			})
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return
		}

		// 选择后端节点
		node := route.LB.Next(req)
		if node == nil {
			logging.AccessLog(&logging.AccessLogEntry{
				Method:     req.Method,
				Path:       req.URL.Path,
				Host:       req.Host,
				RemoteAddr: req.RemoteAddr,
				StatusCode: 503,
				Duration:   time.Since(startTime).Milliseconds(),
				Error:      "no healthy backend available",
			})
			http.Error(w, "503 Service Unavailable", http.StatusServiceUnavailable)
			return
		}

		// 增加连接数
		node.IncrConns()
		defer node.DecrConns()

		// 创建反向代理
		proxy := httputil.NewSingleHostReverseProxy(node.URL)

		// 自定义错误处理
		originalErrorHandler := proxy.ErrorHandler
		proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
			if originalErrorHandler != nil {
				originalErrorHandler(w, req, err)
			} else {
				logging.Error("代理请求失败", map[string]interface{}{
					"error":   err.Error(),
					"upstream": node.Address,
					"path":    req.URL.Path,
				})
				http.Error(w, "502 Bad Gateway", http.StatusBadGateway)
			}

			// 记录访问日志
			logging.AccessLog(&logging.AccessLogEntry{
				Method:       req.Method,
				Path:         req.URL.Path,
				Host:         req.Host,
				RemoteAddr:   req.RemoteAddr,
				StatusCode:   502,
				Duration:     time.Since(startTime).Milliseconds(),
				UpstreamAddr: node.Address,
				Error:        err.Error(),
			})
		}

		// 记录访问日志（成功情况）
		recorder := &responseRecorder{ResponseWriter: w, statusCode: 200}
		
		// 执行代理
		proxy.ServeHTTP(recorder, req)

		// 记录访问日志
		logging.AccessLog(&logging.AccessLogEntry{
			Method:       req.Method,
			Path:         req.URL.Path,
			Host:         req.Host,
			RemoteAddr:   req.RemoteAddr,
			UserAgent:    req.UserAgent(),
			StatusCode:   recorder.statusCode,
			ResponseBody: recorder.written,
			Duration:     time.Since(startTime).Milliseconds(),
			UpstreamAddr: node.Address,
		})
	})
}

// responseRecorder 响应记录器
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)
	return n, err
}
