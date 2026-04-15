package agent

import (
	"sync"

	"smartgateway/pkg/agent/coordinator"
	"smartgateway/pkg/agent/executor"
	"smartgateway/pkg/agent/planner"
	"smartgateway/pkg/agent/validator"
)

// AgentFramework 多 Agent 协作框架
type AgentFramework struct {
	planner     *planner.Planner
	executor    *executor.Executor
	validator   *validator.Validator
	coordinator *coordinator.Coordinator
	mu          sync.RWMutex
}

// Config Agent 框架配置
type Config struct {
	PlannerConfig     planner.Config
	ExecutorConfig    executor.Config
	ValidatorConfig   validator.Config
	CoordinatorConfig coordinator.Config
}

// NewAgentFramework 创建 Agent 框架实例
func NewAgentFramework(cfg Config) *AgentFramework {
	return &AgentFramework{
		planner:     planner.NewPlanner(cfg.PlannerConfig),
		executor:    executor.NewExecutor(cfg.ExecutorConfig),
		validator:   validator.NewValidator(cfg.ValidatorConfig),
		coordinator: coordinator.NewCoordinator(cfg.CoordinatorConfig),
	}
}

// ProcessTask 处理任务（完整协作流程）
func (af *AgentFramework) ProcessTask(task interface{}, context map[string]interface{}) (*TaskResult, error) {
	af.mu.RLock()
	defer af.mu.RUnlock()

	// 1. 规划阶段
	plan, err := af.planner.CreatePlan(task, context)
	if err != nil {
		return nil, err
	}

	// 2. 执行阶段
	execResult, err := af.executor.Execute(plan, context)
	if err != nil {
		return nil, err
	}

	// 3. 校验阶段
	valid, validationErrors := af.validator.Validate(execResult, context)
	if !valid {
		return &TaskResult{
			Success: false,
			Data:    nil,
			Errors:  validationErrors,
		}, nil
	}

	// 4. 协调记录
	af.coordinator.RecordCompletion(task, execResult)

	return &TaskResult{
		Success: true,
		Data:    execResult,
		Errors:  nil,
	}, nil
}

// TaskResult 任务结果
type TaskResult struct {
	Success bool
	Data    interface{}
	Errors  []error
}

// GetPlanner 获取规划器
func (af *AgentFramework) GetPlanner() *planner.Planner {
	return af.planner
}

// GetExecutor 获取执行器
func (af *AgentFramework) GetExecutor() *executor.Executor {
	return af.executor
}

// GetValidator 获取校验器
func (af *AgentFramework) GetValidator() *validator.Validator {
	return af.validator
}

// GetCoordinator 获取协调器
func (af *AgentFramework) GetCoordinator() *coordinator.Coordinator {
	return af.coordinator
}

// UpdateConfig 更新配置
func (af *AgentFramework) UpdateConfig(cfg Config) {
	af.mu.Lock()
	defer af.mu.Unlock()

	af.planner.UpdateConfig(cfg.PlannerConfig)
	af.executor.UpdateConfig(cfg.ExecutorConfig)
	af.validator.UpdateConfig(cfg.ValidatorConfig)
	af.coordinator.UpdateConfig(cfg.CoordinatorConfig)
}
