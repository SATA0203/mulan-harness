package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"smartgateway/pkg/loadbalancer"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	mu                 sync.RWMutex
	checkers           map[string]*backendChecker
	interval           time.Duration
	timeout            time.Duration
	unhealthyThreshold int
	healthyThreshold   int
	path               string
	stopCh             chan struct{}
}

type backendChecker struct {
	node              *loadbalancer.BackendNode
	consecutiveFails  int
	consecutiveSuccesses int
	lastCheckTime     time.Time
	lastStatus        bool
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(
	interval, timeout time.Duration,
	unhealthyThreshold, healthyThreshold int,
	path string,
) *HealthChecker {
	return &HealthChecker{
		checkers:           make(map[string]*backendChecker),
		interval:           interval,
		timeout:            timeout,
		unhealthyThreshold: unhealthyThreshold,
		healthyThreshold:   healthyThreshold,
		path:               path,
		stopCh:             make(chan struct{}),
	}
}

// Start 启动健康检查
func (hc *HealthChecker) Start() {
	go hc.run()
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}

// RegisterBackend 注册后端节点进行健康检查
func (hc *HealthChecker) RegisterBackend(node *loadbalancer.BackendNode) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if _, exists := hc.checkers[node.Address]; !exists {
		hc.checkers[node.Address] = &backendChecker{
			node:       node,
			lastStatus: true, // 初始状态为健康
		}
	}
}

// UnregisterBackend 移除后端节点的健康检查
func (hc *HealthChecker) UnregisterBackend(address string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.checkers, address)
}

// run 执行定期健康检查
func (hc *HealthChecker) run() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkAll()
		case <-hc.stopCh:
			return
		}
	}
}

// checkAll 检查所有后端节点
func (hc *HealthChecker) checkAll() {
	hc.mu.RLock()
	checkers := make([]*backendChecker, 0, len(hc.checkers))
	for _, checker := range hc.checkers {
		checkers = append(checkers, checker)
	}
	hc.mu.RUnlock()

	for _, checker := range checkers {
		hc.checkOne(checker)
	}
}

// checkOne 检查单个后端节点
func (hc *HealthChecker) checkOne(checker *backendChecker) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	healthy := hc.doCheck(ctx, checker.node)

	checker.lastCheckTime = time.Now()

	if healthy {
		checker.consecutiveFails = 0
		checker.consecutiveSuccesses++

		// 如果连续成功次数达到阈值，标记为健康
		if checker.consecutiveSuccesses >= hc.healthyThreshold {
			if !checker.node.IsHealthy() {
				checker.node.SetHealthy(true)
				checker.lastStatus = true
			}
		}
	} else {
		checker.consecutiveSuccesses = 0
		checker.consecutiveFails++

		// 如果连续失败次数达到阈值，标记为不健康
		if checker.consecutiveFails >= hc.unhealthyThreshold {
			if checker.node.IsHealthy() {
				checker.node.SetHealthy(false)
				checker.lastStatus = false
			}
		}
	}
}

// doCheck 执行实际的健康检查
func (hc *HealthChecker) doCheck(ctx context.Context, node *loadbalancer.BackendNode) bool {
	// 构建健康检查 URL
	checkPath := hc.path
	if checkPath == "" {
		checkPath = "/health"
	}

	url := node.URL.Scheme + "://" + node.URL.Host + checkPath

	client := &http.Client{
		Timeout: hc.timeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		node.IncrFail()
		return false
	}
	defer resp.Body.Close()

	// 2xx 和 3xx 状态码视为健康
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		node.ResetFail()
		return true
	}

	node.IncrFail()
	return false
}

// GetStatus 获取后端节点的健康状态
func (hc *HealthChecker) GetStatus(address string) (bool, time.Time, int, int) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if checker, exists := hc.checkers[address]; exists {
		return checker.lastStatus, checker.lastCheckTime, checker.consecutiveFails, checker.consecutiveSuccesses
	}

	return false, time.Time{}, 0, 0
}

// GetAllStatuses 获取所有后端节点的健康状态
func (hc *HealthChecker) GetAllStatuses() map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	statuses := make(map[string]interface{})
	for addr, checker := range hc.checkers {
		statuses[addr] = map[string]interface{}{
			"healthy":            checker.node.IsHealthy(),
			"last_check_time":    checker.lastCheckTime,
			"consecutive_fails":  checker.consecutiveFails,
			"consecutive_successes": checker.consecutiveSuccesses,
			"fail_count":         checker.node.GetFailCount(),
		}
	}

	return statuses
}
