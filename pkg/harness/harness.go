package harness

import (
	"sync"

	"smartgateway/pkg/harness/audit"
	"smartgateway/pkg/harness/auth"
	"smartgateway/pkg/harness/compliance"
)

// Harness 管控层 - 负责权限控制、合规校验和审计日志
type Harness struct {
	auth       *auth.AuthController
	compliance *compliance.ComplianceEngine
	audit      *audit.AuditLogger
	mu         sync.RWMutex
}

// Config Harness 配置
type Config struct {
	AuthConfig       auth.Config
	ComplianceConfig compliance.Config
	AuditConfig      audit.Config
}

// NewHarness 创建 Harness 管控实例
func NewHarness(cfg Config) *Harness {
	return &Harness{
		auth:       auth.NewAuthController(cfg.AuthConfig),
		compliance: compliance.NewComplianceEngine(cfg.ComplianceConfig),
		audit:      audit.NewAuditLogger(cfg.AuditConfig),
	}
}

// Auth 获取权限控制器
func (h *Harness) Auth() *auth.AuthController {
	return h.auth
}

// Compliance 获取合规引擎
func (h *Harness) Compliance() *compliance.ComplianceEngine {
	return h.compliance
}

// Audit 获取审计日志器
func (h *Harness) Audit() *audit.AuditLogger {
	return h.audit
}

// CheckRequest 检查请求（权限 + 合规）
func (h *Harness) CheckRequest(req interface{}, context map[string]interface{}) (bool, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 1. 权限检查
	allowed, err := h.auth.CheckPermission(req, context)
	if err != nil || !allowed {
		h.audit.LogAccessDenied(req, context, err)
		return false, err
	}

	// 2. 合规检查
	compliant, violations := h.compliance.Validate(req, context)
	if !compliant {
		h.audit.LogComplianceViolation(req, context, violations)
		return false, violations[0]
	}

	// 3. 记录审计日志
	h.audit.LogAccessGranted(req, context)

	return true, nil
}

// UpdateConfig 热更新配置
func (h *Harness) UpdateConfig(cfg Config) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.auth.UpdateConfig(cfg.AuthConfig)
	h.compliance.UpdateConfig(cfg.ComplianceConfig)
	h.audit.UpdateConfig(cfg.AuditConfig)
}
