package loadbalancer

import (
	"net/http"
	"testing"

	"smartgateway/pkg/config"
)

func TestRoundRobinLB(t *testing.T) {
	lb := NewRoundRobinLB()

	// 添加三个后端节点
	node1 := &BackendNode{Address: "http://backend1:8080", Healthy: true}
	node2 := &BackendNode{Address: "http://backend2:8080", Healthy: true}
	node3 := &BackendNode{Address: "http://backend3:8080", Healthy: true}

	lb.Add(node1)
	lb.Add(node2)
	lb.Add(node3)

	// 测试轮询是否按顺序返回
	req, _ := http.NewRequest("GET", "http://test.com", nil)

	n1 := lb.Next(req)
	n2 := lb.Next(req)
	n3 := lb.Next(req)
	n4 := lb.Next(req)

	if n1.Address != node1.Address {
		t.Errorf("Expected backend1, got %s", n1.Address)
	}
	if n2.Address != node2.Address {
		t.Errorf("Expected backend2, got %s", n2.Address)
	}
	if n3.Address != node3.Address {
		t.Errorf("Expected backend3, got %s", n3.Address)
	}
	if n4.Address != node1.Address {
		t.Errorf("Expected backend1 (cycle), got %s", n4.Address)
	}
}

func TestRoundRobinLBSkipUnhealthy(t *testing.T) {
	lb := NewRoundRobinLB()

	node1 := &BackendNode{Address: "http://backend1:8080", Healthy: true}
	node2 := &BackendNode{Address: "http://backend2:8080", Healthy: false} // 不健康
	node3 := &BackendNode{Address: "http://backend3:8080", Healthy: true}

	lb.Add(node1)
	lb.Add(node2)
	lb.Add(node3)

	req, _ := http.NewRequest("GET", "http://test.com", nil)

	// 应该跳过不健康的节点
	n1 := lb.Next(req)
	n2 := lb.Next(req)

	if n1.Address != node1.Address {
		t.Errorf("Expected backend1, got %s", n1.Address)
	}
	if n2.Address != node3.Address {
		t.Errorf("Expected backend3 (skipping unhealthy backend2), got %s", n2.Address)
	}
}

func TestRandomLB(t *testing.T) {
	lb := NewRandomLB()

	node1 := &BackendNode{Address: "http://backend1:8080", Healthy: true}
	node2 := &BackendNode{Address: "http://backend2:8080", Healthy: true}
	node3 := &BackendNode{Address: "http://backend3:8080", Healthy: true}

	lb.Add(node1)
	lb.Add(node2)
	lb.Add(node3)

	req, _ := http.NewRequest("GET", "http://test.com", nil)

	// 随机选择，多次调用应该能选中所有节点
	selected := make(map[string]bool)
	for i := 0; i < 100; i++ {
		node := lb.Next(req)
		selected[node.Address] = true
	}

	if len(selected) != 3 {
		t.Errorf("Expected all 3 backends to be selected, got %d", len(selected))
	}
}

func TestLeastConnLB(t *testing.T) {
	lb := NewLeastConnLB()

	node1 := &BackendNode{Address: "http://backend1:8080", Healthy: true}
	node2 := &BackendNode{Address: "http://backend2:8080", Healthy: true}
	node3 := &BackendNode{Address: "http://backend3:8080", Healthy: true}

	lb.Add(node1)
	lb.Add(node2)
	lb.Add(node3)

	req, _ := http.NewRequest("GET", "http://test.com", nil)

	// 初始所有节点连接数相同，应该返回第一个
	n1 := lb.Next(req)
	if n1.Address != node1.Address {
		t.Errorf("Expected backend1, got %s", n1.Address)
	}

	// 模拟 node1 有较多连接
	node1.ActiveConns = 100
	node2.ActiveConns = 50
	node3.ActiveConns = 10

	// 应该选择连接数最少的 node3
	n2 := lb.Next(req)
	if n2.Address != node3.Address {
		t.Errorf("Expected backend3 (least connections), got %s", n2.Address)
	}
}

func TestLoadBalancerRemove(t *testing.T) {
	lb := NewRoundRobinLB()

	node1 := &BackendNode{Address: "http://backend1:8080", Healthy: true}
	node2 := &BackendNode{Address: "http://backend2:8080", Healthy: true}

	lb.Add(node1)
	lb.Add(node2)

	if len(lb.List()) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(lb.List()))
	}

	lb.Remove("http://backend1:8080")

	if len(lb.List()) != 1 {
		t.Errorf("Expected 1 node after removal, got %d", len(lb.List()))
	}

	if lb.List()[0].Address != node2.Address {
		t.Errorf("Expected backend2, got %s", lb.List()[0].Address)
	}
}

func TestCreateBackendsFromConfig(t *testing.T) {
	backends := []config.Backend{
		{Address: "http://127.0.0.1:8081", Weight: 1},
		{Address: "http://127.0.0.1:8082", Weight: 2},
		{Address: "127.0.0.1:8083", Weight: 1}, // 不带协议前缀
	}

	for _, b := range backends {
		nodes := CreateBackendsFromConfig([]config.Backend{b})

		if len(nodes) != 1 {
			t.Errorf("Expected 1 node, got %d", len(nodes))
			continue
		}

		if nodes[0].Address != b.Address {
			t.Errorf("Expected address %s, got %s", b.Address, nodes[0].Address)
		}

		expectedWeight := b.Weight
		if expectedWeight == 0 {
			expectedWeight = 1
		}
		if nodes[0].Weight != expectedWeight {
			t.Errorf("Expected weight %d, got %d", expectedWeight, nodes[0].Weight)
		}
	}
}
