package evolution

import (
	"sync"

	"smartgateway/pkg/evolution/memory"
	"smartgateway/pkg/evolution/skill"
	"smartgateway/pkg/evolution/strategy"
)

// SelfEvolutionBase 自进化底座
type SelfEvolutionBase struct {
	skillRegistry  *skill.Registry
	memorySystem   *memory.MemorySystem
	strategyOptimizer *strategy.Optimizer
	mu             sync.RWMutex
}

// Config 自进化配置
type Config struct {
	SkillConfig    skill.Config
	MemoryConfig   memory.Config
	StrategyConfig strategy.Config
}

// NewSelfEvolutionBase 创建自进化底座实例
func NewSelfEvolutionBase(cfg Config) *SelfEvolutionBase {
	return &SelfEvolutionBase{
		skillRegistry:     skill.NewRegistry(cfg.SkillConfig),
		memorySystem:      memory.NewMemorySystem(cfg.MemoryConfig),
		strategyOptimizer: strategy.NewOptimizer(cfg.StrategyConfig),
	}
}

// Evolve 执行进化流程
func (seb *SelfEvolutionBase) Evolve(feedback interface{}, context map[string]interface{}) error {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	// 1. 记录经验到记忆系统
	seb.memorySystem.StoreExperience(feedback, context)

	// 2. 分析反馈，优化策略
	err := seb.strategyOptimizer.AnalyzeAndOptimize(feedback, context)
	if err != nil {
		return err
	}

	// 3. 根据优化结果注册新技能（简化实现，暂不注册）
	// newSkills := seb.strategyOptimizer.GenerateNewSkills()
	// for _, s := range newSkills {
	// 	seb.skillRegistry.Register(s)
	// }

	return nil
}

// GetSkillRegistry 获取技能注册表
func (seb *SelfEvolutionBase) GetSkillRegistry() *skill.Registry {
	return seb.skillRegistry
}

// GetMemorySystem 获取记忆系统
func (seb *SelfEvolutionBase) GetMemorySystem() *memory.MemorySystem {
	return seb.memorySystem
}

// GetStrategyOptimizer 获取策略优化器
func (seb *SelfEvolutionBase) GetStrategyOptimizer() *strategy.Optimizer {
	return seb.strategyOptimizer
}

// UpdateConfig 更新配置
func (seb *SelfEvolutionBase) UpdateConfig(cfg Config) {
	seb.mu.Lock()
	defer seb.mu.Unlock()

	seb.skillRegistry.UpdateConfig(cfg.SkillConfig)
	seb.memorySystem.UpdateConfig(cfg.MemoryConfig)
	seb.strategyOptimizer.UpdateConfig(cfg.StrategyConfig)
}

// GetStats 获取统计信息
func (seb *SelfEvolutionBase) GetStats() map[string]interface{} {
	seb.mu.RLock()
	defer seb.mu.RUnlock()

	return map[string]interface{}{
		"skills_count":    seb.skillRegistry.Count(),
		"memories_count":  seb.memorySystem.Count(),
		"strategies_count": seb.strategyOptimizer.Count(),
	}
}
