package loadbalancer

import (
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"smartgateway/pkg/config"
)

// BackendNode 后端节点
type BackendNode struct {
	Address      string
	URL          *url.URL
	Weight       int
	CurrentWeight int
	Healthy      bool
	ActiveConns  int64
	FailCount    int
	mu           sync.RWMutex
}

// IsHealthy 检查节点是否健康
func (n *BackendNode) IsHealthy() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Healthy
}

// SetHealthy 设置节点健康状态
func (n *BackendNode) SetHealthy(healthy bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Healthy = healthy
}

// IncrConns 增加活跃连接数
func (n *BackendNode) IncrConns() {
	atomic.AddInt64(&n.ActiveConns, 1)
}

// DecrConns 减少活跃连接数
func (n *BackendNode) DecrConns() {
	atomic.AddInt64(&n.ActiveConns, -1)
}

// GetActiveConns 获取活跃连接数
func (n *BackendNode) GetActiveConns() int64 {
	return atomic.LoadInt64(&n.ActiveConns)
}

// IncrFail 增加失败计数
func (n *BackendNode) IncrFail() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.FailCount++
}

// ResetFail 重置失败计数
func (n *BackendNode) ResetFail() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.FailCount = 0
}

// GetFailCount 获取失败计数
func (n *BackendNode) GetFailCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.FailCount
}

// LBAlgorithm 负载均衡算法类型
type LBAlgorithm string

const (
	RoundRobin     LBAlgorithm = "round_robin"
	Random         LBAlgorithm = "random"
	LeastConn      LBAlgorithm = "least_conn"
	ConsistentHash LBAlgorithm = "consistent_hash"
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	// Next 选择下一个后端节点
	Next(req *http.Request) *BackendNode
	// Add 添加后端节点
	Add(node *BackendNode)
	// Remove 移除后端节点
	Remove(address string)
	// List 列出所有节点
	List() []*BackendNode
	// UpdateHealth 更新节点健康状态
	UpdateHealth(address string, healthy bool)
}

// BaseLB 负载均衡器基础结构
type BaseLB struct {
	mu      sync.RWMutex
	nodes   []*BackendNode
	current int
}

// RoundRobinLB 轮询负载均衡器
type RoundRobinLB struct {
	BaseLB
}

// NewRoundRobinLB 创建轮询负载均衡器
func NewRoundRobinLB() *RoundRobinLB {
	return &RoundRobinLB{
		BaseLB: BaseLB{
			nodes:   make([]*BackendNode, 0),
			current: -1,
		},
	}
}

// Next 选择下一个后端节点（轮询）
func (lb *RoundRobinLB) Next(req *http.Request) *BackendNode {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.nodes) == 0 {
		return nil
	}

	// 轮询选择健康节点
	for i := 0; i < len(lb.nodes); i++ {
		lb.current = (lb.current + 1) % len(lb.nodes)
		node := lb.nodes[lb.current]
		if node.IsHealthy() {
			return node
		}
	}

	// 如果没有健康节点，返回第一个节点（降级）
	if len(lb.nodes) > 0 {
		return lb.nodes[0]
	}
	return nil
}

// Add 添加后端节点
func (lb *RoundRobinLB) Add(node *BackendNode) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.nodes = append(lb.nodes, node)
}

// Remove 移除后端节点
func (lb *RoundRobinLB) Remove(address string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, node := range lb.nodes {
		if node.Address == address {
			lb.nodes = append(lb.nodes[:i], lb.nodes[i+1:]...)
			break
		}
	}
}

// List 列出所有节点
func (lb *RoundRobinLB) List() []*BackendNode {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]*BackendNode, len(lb.nodes))
	copy(result, lb.nodes)
	return result
}

// UpdateHealth 更新节点健康状态
func (lb *RoundRobinLB) UpdateHealth(address string, healthy bool) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for _, node := range lb.nodes {
		if node.Address == address {
			node.SetHealthy(healthy)
			break
		}
	}
}

// RandomLB 随机负载均衡器
type RandomLB struct {
	BaseLB
}

// NewRandomLB 创建随机负载均衡器
func NewRandomLB() *RandomLB {
	return &RandomLB{
		BaseLB: BaseLB{
			nodes: make([]*BackendNode, 0),
		},
	}
}

// Next 选择下一个后端节点（随机）
func (lb *RandomLB) Next(req *http.Request) *BackendNode {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.nodes) == 0 {
		return nil
	}

	// 使用当前时间作为随机种子
	seed := time.Now().UnixNano() % int64(len(lb.nodes))
	
	// 尝试随机选择健康节点
	for i := 0; i < len(lb.nodes)*2; i++ {
		idx := int((seed + int64(i)) % int64(len(lb.nodes)))
		node := lb.nodes[idx]
		if node.IsHealthy() {
			return node
		}
	}

	// 降级返回第一个节点
	if len(lb.nodes) > 0 {
		return lb.nodes[0]
	}
	return nil
}

// Add 添加后端节点
func (lb *RandomLB) Add(node *BackendNode) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.nodes = append(lb.nodes, node)
}

// Remove 移除后端节点
func (lb *RandomLB) Remove(address string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, node := range lb.nodes {
		if node.Address == address {
			lb.nodes = append(lb.nodes[:i], lb.nodes[i+1:]...)
			break
		}
	}
}

// List 列出所有节点
func (lb *RandomLB) List() []*BackendNode {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]*BackendNode, len(lb.nodes))
	copy(result, lb.nodes)
	return result
}

// UpdateHealth 更新节点健康状态
func (lb *RandomLB) UpdateHealth(address string, healthy bool) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for _, node := range lb.nodes {
		if node.Address == address {
			node.SetHealthy(healthy)
			break
		}
	}
}

// LeastConnLB 最小连接数负载均衡器
type LeastConnLB struct {
	BaseLB
}

// NewLeastConnLB 创建最小连接数负载均衡器
func NewLeastConnLB() *LeastConnLB {
	return &LeastConnLB{
		BaseLB: BaseLB{
			nodes: make([]*BackendNode, 0),
		},
	}
}

// Next 选择下一个后端节点（最小连接数）
func (lb *LeastConnLB) Next(req *http.Request) *BackendNode {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(lb.nodes) == 0 {
		return nil
	}

	var minConns int64 = -1
	var selected *BackendNode

	for _, node := range lb.nodes {
		if !node.IsHealthy() {
			continue
		}
		conns := node.GetActiveConns()
		if minConns == -1 || conns < minConns {
			minConns = conns
			selected = node
		}
	}

	if selected != nil {
		return selected
	}

	// 降级返回第一个节点
	if len(lb.nodes) > 0 {
		return lb.nodes[0]
	}
	return nil
}

// Add 添加后端节点
func (lb *LeastConnLB) Add(node *BackendNode) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.nodes = append(lb.nodes, node)
}

// Remove 移除后端节点
func (lb *LeastConnLB) Remove(address string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i, node := range lb.nodes {
		if node.Address == address {
			lb.nodes = append(lb.nodes[:i], lb.nodes[i+1:]...)
			break
		}
	}
}

// List 列出所有节点
func (lb *LeastConnLB) List() []*BackendNode {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]*BackendNode, len(lb.nodes))
	copy(result, lb.nodes)
	return result
}

// UpdateHealth 更新节点健康状态
func (lb *LeastConnLB) UpdateHealth(address string, healthy bool) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for _, node := range lb.nodes {
		if node.Address == address {
			node.SetHealthy(healthy)
			break
		}
	}
}

// NewLoadBalancer 根据算法类型创建负载均衡器
func NewLoadBalancer(algorithm string) LoadBalancer {
	switch LBAlgorithm(algorithm) {
	case RoundRobin, "":
		return NewRoundRobinLB()
	case Random:
		return NewRandomLB()
	case LeastConn:
		return NewLeastConnLB()
	case ConsistentHash:
		// TODO: 实现一致性哈希
		return NewRoundRobinLB()
	default:
		return NewRoundRobinLB()
	}
}

// CreateBackendsFromConfig 从配置创建后端节点列表
func CreateBackendsFromConfig(backends []config.Backend) []*BackendNode {
	nodes := make([]*BackendNode, 0, len(backends))
	for _, b := range backends {
		u, err := url.Parse(b.Address)
		if err != nil {
			u, _ = url.Parse("http://" + b.Address)
		}
		weight := b.Weight
		if weight == 0 {
			weight = 1
		}
		nodes = append(nodes, &BackendNode{
			Address: b.Address,
			URL:     u,
			Weight:  weight,
			Healthy: true,
		})
	}
	return nodes
}
