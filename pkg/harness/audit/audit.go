package audit

import (
	"encoding/json"
	"sync"
	"time"

	"smartgateway/pkg/logging"
)

// Config 审计日志配置
type Config struct {
	Enabled        bool   `json:"enabled"`
	LogLevel       string `json:"log_level"`
	IncludeBody    bool   `json:"include_body"`
	MaxBodyLength  int    `json:"max_body_length"`
	RetentionDays  int    `json:"retention_days"`
	OutputFormat   string `json:"output_format"` // json/text
}

// AuditLogger 审计日志器
type AuditLogger struct {
	config Config
	mu     sync.RWMutex
}

// AuditEvent 审计事件
type AuditEvent struct {
	Timestamp   time.Time                `json:"timestamp"`
	EventType   string                   `json:"event_type"`
	RequestID   string                   `json:"request_id,omitempty"`
	User        string                   `json:"user,omitempty"`
	Action      string                   `json:"action"`
	Resource    string                   `json:"resource,omitempty"`
	Context     map[string]interface{}   `json:"context,omitempty"`
	Violations  []string                 `json:"violations,omitempty"`
	Status      string                   `json:"status"`
}

// NewAuditLogger 创建审计日志器
func NewAuditLogger(cfg Config) *AuditLogger {
	return &AuditLogger{
		config: cfg,
	}
}

// LogAccessGranted 记录访问授权
func (a *AuditLogger) LogAccessGranted(req interface{}, context map[string]interface{}) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.config.Enabled {
		return
	}

	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: "access_granted",
		Action:    "request_processed",
		Context:   context,
		Status:    "success",
	}

	a.logEvent(event)
}

// LogAccessDenied 记录访问拒绝
func (a *AuditLogger) LogAccessDenied(req interface{}, context map[string]interface{}, err error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.config.Enabled {
		return
	}

	violations := []string{}
	if err != nil {
		violations = append(violations, err.Error())
	}

	event := AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "access_denied",
		Action:     "request_blocked",
		Context:    context,
		Violations: violations,
		Status:     "denied",
	}

	a.logEvent(event)
}

// LogComplianceViolation 记录合规违规
func (a *AuditLogger) LogComplianceViolation(req interface{}, context map[string]interface{}, violations []error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.config.Enabled {
		return
	}

	violationStrs := make([]string, len(violations))
	for i, v := range violations {
		violationStrs[i] = v.Error()
	}

	event := AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "compliance_violation",
		Action:     "compliance_check_failed",
		Context:    context,
		Violations: violationStrs,
		Status:     "violation",
	}

	a.logEvent(event)
}

// LogCustomEvent 记录自定义事件
func (a *AuditLogger) LogCustomEvent(eventType, action string, context map[string]interface{}) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.config.Enabled {
		return
	}

	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		Action:    action,
		Context:   context,
		Status:    "logged",
	}

	a.logEvent(event)
}

// logEvent 记录事件
func (a *AuditLogger) logEvent(event AuditEvent) {
	if a.config.OutputFormat == "json" {
		data, err := json.Marshal(event)
		if err != nil {
			logging.Error("Failed to marshal audit event", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		logging.Info("AUDIT: "+string(data), nil)
	} else {
		logging.Info("AUDIT", map[string]interface{}{
			"event_type": event.EventType,
			"action":     event.Action,
			"status":     event.Status,
			"context":    event.Context,
		})
	}
}

// UpdateConfig 更新配置
func (a *AuditLogger) UpdateConfig(cfg Config) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config = cfg
}

// GetConfig 获取当前配置
func (a *AuditLogger) GetConfig() Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}
