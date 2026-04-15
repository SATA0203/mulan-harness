package planner

import (
	"sync"
)

// Config 规划器配置
type Config struct {
	Enabled       bool     `json:"enabled"`
	Strategy      string   `json:"strategy"` // sequential/parallel/dynamic
	MaxSteps      int      `json:"max_steps"`
	TimeoutPerStep int     `json:"timeout_per_step"`
	PriorityRules []string `json:"priority_rules"`
}

// Plan 执行计划
type Plan struct {
	ID          string
	Steps       []*PlanStep
	Strategy    string
	CreatedAt   int64
	Status      string
}

// PlanStep 计划步骤
type PlanStep struct {
	ID       string
	Name     string
	Action   string
	Params   map[string]interface{}
	DependsOn []string
	Status   string
	Result   interface{}
	Error    error
}

// Planner 规划器 Agent
type Planner struct {
	config Config
	mu     sync.RWMutex
}

// NewPlanner 创建规划器
func NewPlanner(cfg Config) *Planner {
	return &Planner{
		config: cfg,
	}
}

// CreatePlan 创建执行计划
func (p *Planner) CreatePlan(task interface{}, context map[string]interface{}) (*Plan, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.config.Enabled {
		// 返回简单计划
		return &Plan{
			ID:       "simple-plan",
			Steps:    []*PlanStep{},
			Strategy: "sequential",
			Status:   "ready",
		}, nil
	}

	// 根据策略创建计划
	steps := p.createSteps(task, context)

	return &Plan{
		ID:       generatePlanID(),
		Steps:    steps,
		Strategy: p.config.Strategy,
		Status:   "ready",
	}, nil
}

// createSteps 创建执行步骤
func (p *Planner) createSteps(task interface{}, context map[string]interface{}) []*PlanStep {
	// 简化实现：根据任务类型创建步骤
	steps := []*PlanStep{
		{
			ID:     "step-1",
			Name:   "预处理",
			Action: "preprocess",
			Status: "pending",
		},
		{
			ID:     "step-2",
			Name:   "主处理",
			Action: "process",
			Status: "pending",
		},
		{
			ID:     "step-3",
			Name:   "后处理",
			Action: "postprocess",
			Status: "pending",
		},
	}

	return steps
}

// generatePlanID 生成计划 ID
func generatePlanID() string {
	return "plan-" + randomString(8)
}

// UpdateConfig 更新配置
func (p *Planner) UpdateConfig(cfg Config) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = cfg
}

// GetConfig 获取当前配置
func (p *Planner) GetConfig() Config {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}
