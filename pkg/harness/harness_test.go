package harness

import (
	"testing"

	"smartgateway/pkg/harness/audit"
	"smartgateway/pkg/harness/auth"
	"smartgateway/pkg/harness/compliance"
)

func TestNewHarness(t *testing.T) {
	cfg := Config{
		AuthConfig: auth.Config{
			Enabled:       true,
			DefaultPolicy: "allow",
		},
		ComplianceConfig: compliance.Config{
			Enabled: true,
		},
		AuditConfig: audit.Config{
			Enabled: true,
		},
	}

	h := NewHarness(cfg)
	if h == nil {
		t.Fatal("NewHarness returned nil")
	}

	if h.Auth() == nil {
		t.Error("Auth controller is nil")
	}
	if h.Compliance() == nil {
		t.Error("Compliance engine is nil")
	}
	if h.Audit() == nil {
		t.Error("Audit logger is nil")
	}
}

func TestHarnessCheckRequest(t *testing.T) {
	cfg := Config{
		AuthConfig: auth.Config{
			Enabled:       false,
			DefaultPolicy: "allow",
		},
		ComplianceConfig: compliance.Config{
			Enabled: false,
		},
		AuditConfig: audit.Config{
			Enabled: false,
		},
	}

	h := NewHarness(cfg)

	// 测试允许请求
	allowed, err := h.CheckRequest(nil, map[string]interface{}{
		"role":   "admin",
		"path":   "/api/test",
		"method": "GET",
	})

	if !allowed {
		t.Error("Expected request to be allowed")
	}
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestHarnessUpdateConfig(t *testing.T) {
	cfg := Config{
		AuthConfig: auth.Config{
			Enabled: false,
		},
		ComplianceConfig: compliance.Config{
			Enabled: false,
		},
		AuditConfig: audit.Config{
			Enabled: false,
		},
	}

	h := NewHarness(cfg)

	// 更新配置
	newCfg := Config{
		AuthConfig: auth.Config{
			Enabled: true,
		},
		ComplianceConfig: compliance.Config{
			Enabled: true,
		},
		AuditConfig: audit.Config{
			Enabled: true,
		},
	}

	h.UpdateConfig(newCfg)

	// 验证配置已更新
	if h.Auth().GetConfig().Enabled != true {
		t.Error("Auth config not updated")
	}
	if h.Compliance().GetConfig().Enabled != true {
		t.Error("Compliance config not updated")
	}
	if h.Audit().GetConfig().Enabled != true {
		t.Error("Audit config not updated")
	}
}
