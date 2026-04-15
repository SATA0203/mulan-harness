package memory

import (
	"sync"
	"time"
)

// MemoryEntry 记忆条目
type MemoryEntry struct {
	ID        string
	Type      string // experience/lesson/pattern
	Content   interface{}
	Context   map[string]interface{}
	CreatedAt time.Time
	Weight    float64
}

// Config 记忆系统配置
type Config struct {
	Enabled        bool `json:"enabled"`
	MaxMemories    int  `json:"max_memories"`
	RetentionDays  int  `json:"retention_days"`
	EnableForgetting bool `json:"enable_forgetting"`
}

// MemorySystem 记忆系统
type MemorySystem struct {
	config   Config
	memories []*MemoryEntry
	mu       sync.RWMutex
}

// NewMemorySystem 创建记忆系统
func NewMemorySystem(cfg Config) *MemorySystem {
	return &MemorySystem{
		config:   cfg,
		memories: make([]*MemoryEntry, 0),
	}
}

// StoreExperience 存储经验
func (m *MemorySystem) StoreExperience(content interface{}, context map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		return
	}

	entry := &MemoryEntry{
		ID:        generateMemoryID(),
		Type:      "experience",
		Content:   content,
		Context:   context,
		CreatedAt: time.Now(),
		Weight:    1.0,
	}

	m.memories = append(m.memories, entry)

	// 限制记忆数量
	if len(m.memories) > m.config.MaxMemories {
		m.memories = m.memories[1:]
	}
}

// StoreLesson 存储教训
func (m *MemorySystem) StoreLesson(content interface{}, context map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		return
	}

	entry := &MemoryEntry{
		ID:        generateMemoryID(),
		Type:      "lesson",
		Content:   content,
		Context:   context,
		CreatedAt: time.Now(),
		Weight:    2.0, // 教训权重更高
	}

	m.memories = append(m.memories, entry)
}

// Retrieve 检索记忆
func (m *MemorySystem) Retrieve(queryType string, limit int) []*MemoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*MemoryEntry, 0)
	for _, entry := range m.memories {
		if entry.Type == queryType {
			result = append(result, entry)
			if len(result) >= limit {
				break
			}
		}
	}

	return result
}

// Count 获取记忆数量
func (m *MemorySystem) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.memories)
}

// Clear 清空记忆
func (m *MemorySystem) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.memories = make([]*MemoryEntry, 0)
}

// UpdateConfig 更新配置
func (m *MemorySystem) UpdateConfig(cfg Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = cfg
}

// GetConfig 获取当前配置
func (m *MemorySystem) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}
