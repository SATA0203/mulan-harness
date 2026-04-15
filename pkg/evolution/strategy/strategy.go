package strategy

import (
	"sync"
)

// Strategy 策略定义
type Strategy struct {
	ID          string
	Name        string
	Description string
	Params      map[string]interface{}
	Score       float64
}

// Config 策略优化器配置
type Config struct {
	Enabled        bool     `json:"enabled"`
	OptimizationAlgo string `json:"optimization_algo"` // rule_based/ml/reinforcement
	LearningRate   float64  `json:"learning_rate"`
	MinScore       float64  `json:"min_score"`
}

// Optimizer 策略优化器
type Optimizer struct {
	config     Config
	strategies map[string]*Strategy
	mu         sync.RWMutex
}

// NewOptimizer 创建策略优化器
func NewOptimizer(cfg Config) *Optimizer {
	return &Optimizer{
		config:     cfg,
		strategies: make(map[string]*Strategy),
	}
}

// AnalyzeAndOptimize 分析反馈并优化策略
func (o *Optimizer) AnalyzeAndOptimize(feedback interface{}, context map[string]interface{}) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.config.Enabled {
		return nil
	}

	// 简化实现：更新策略分数
	for _, strategy := range o.strategies {
		strategy.Score += 0.1
	}

	return nil
}

// GenerateNewSkills 生成新技能
func (o *Optimizer) GenerateNewSkills() []*SkillStub {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// 返回空列表（简化实现）
	return []*SkillStub{}
}

// SkillStub 技能桩（避免循环依赖）
type SkillStub struct {
	ID   string
	Name string
}

// RegisterStrategy 注册策略
func (o *Optimizer) RegisterStrategy(strategy *Strategy) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.strategies[strategy.ID] = strategy
}

// GetStrategy 获取策略
func (o *Optimizer) GetStrategy(id string) (*Strategy, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	s, exists := o.strategies[id]
	return s, exists
}

// ListStrategies 列出所有策略
func (o *Optimizer) ListStrategies() []*Strategy {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]*Strategy, 0, len(o.strategies))
	for _, s := range o.strategies {
		result = append(result, s)
	}
	return result
}

// Count 获取策略数量
func (o *Optimizer) Count() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.strategies)
}

// UpdateConfig 更新配置
func (o *Optimizer) UpdateConfig(cfg Config) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.config = cfg
}

// GetConfig 获取当前配置
func (o *Optimizer) GetConfig() Config {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.config
}
