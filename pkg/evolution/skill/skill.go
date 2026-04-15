package skill

import (
	"sync"
)

// Skill 技能定义
type Skill struct {
	ID          string
	Name        string
	Description string
	Handler     interface{}
	Metadata    map[string]interface{}
}

// Config 技能注册表配置
type Config struct {
	Enabled      bool `json:"enabled"`
	MaxSkills    int  `json:"max_skills"`
	AutoRegister bool `json:"auto_register"`
}

// Registry 技能注册表
type Registry struct {
	config Config
	skills map[string]*Skill
	mu     sync.RWMutex
}

// NewRegistry 创建技能注册表
func NewRegistry(cfg Config) *Registry {
	return &Registry{
		config: cfg,
		skills: make(map[string]*Skill),
	}
}

// Register 注册技能
func (r *Registry) Register(skill *Skill) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.config.Enabled {
		return false
	}

	// 检查是否超过最大数量
	if len(r.skills) >= r.config.MaxSkills {
		return false
	}

	r.skills[skill.ID] = skill
	return true
}

// Unregister 注销技能
func (r *Registry) Unregister(skillID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.skills[skillID]
	if exists {
		delete(r.skills, skillID)
	}
	return exists
}

// Get 获取技能
func (r *Registry) Get(skillID string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[skillID]
	return skill, exists
}

// List 列出所有技能
func (r *Registry) List() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.skills))
	for _, s := range r.skills {
		skills = append(skills, s)
	}
	return skills
}

// Count 获取技能数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// UpdateConfig 更新配置
func (r *Registry) UpdateConfig(cfg Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cfg
}

// GetConfig 获取当前配置
func (r *Registry) GetConfig() Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}
