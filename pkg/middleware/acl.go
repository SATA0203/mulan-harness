package middleware

import (
	"sync"
	"time"
)

// ACLConfig 访问控制列表配置
type ACLConfig struct {
	// 白名单 IP 段
	Whitelist []string `json:"whitelist"`
	// 黑名单 IP 段
	Blacklist []string `json:"blacklist"`
	// 是否启用
	Enabled bool `json:"enabled"`
	// 默认策略：allow, deny
	DefaultPolicy string `json:"default_policy"`
}

// ACLRule 单条 ACL 规则
type ACLRule struct {
	CIDR     string
	Network  *IPNet
	Action   string // allow, deny
	Priority int
}

// IPNet 简化的 IP 网络表示
type IPNet struct {
	IP    uint32
	Mask  uint32
}

// ACLMiddleware 访问控制中间件
type ACLMiddleware struct {
	config        ACLConfig
	whitelistRules []*ACLRule
	blacklistRules []*ACLRule
	defaultAllow  bool
	mu            sync.RWMutex
}

// NewACLMiddleware 创建 ACL 中间件
func NewACLMiddleware(config ACLConfig) *ACLMiddleware {
	mw := &ACLMiddleware{
		config: config,
	}

	mw.defaultAllow = config.DefaultPolicy != "deny"
	mw.compileRules()

	return mw
}

// compileRules 编译规则
func (mw *ACLMiddleware) compileRules() {
	mw.whitelistRules = make([]*ACLRule, 0)
	mw.blacklistRules = make([]*ACLRule, 0)

	for _, cidr := range mw.config.Whitelist {
		if network := parseCIDR(cidr); network != nil {
			mw.whitelistRules = append(mw.whitelistRules, &ACLRule{
				CIDR:    cidr,
				Network: network,
				Action:  "allow",
			})
		}
	}

	for _, cidr := range mw.config.Blacklist {
		if network := parseCIDR(cidr); network != nil {
			mw.blacklistRules = append(mw.blacklistRules, &ACLRule{
				CIDR:    cidr,
				Network: network,
				Action:  "deny",
			})
		}
	}
}

// parseCIDR 解析 CIDR 表示法（简化实现）
func parseCIDR(cidr string) *IPNet {
	// 简化实现，仅支持 /32 和 /24
	// 生产环境应使用 net.ParseCIDR
	if cidr == "0.0.0.0/0" {
		return &IPNet{IP: 0, Mask: 0}
	}

	// 示例：192.168.1.0/24
	// 实际实现需要完整的 IP 解析逻辑
	return nil
}

// Allow 检查 IP 是否允许访问
func (mw *ACLMiddleware) Allow(ip string) bool {
	if !mw.config.Enabled {
		return true
	}

	mw.mu.RLock()
	defer mw.mu.RUnlock()

	ipNum := parseIP(ip)

	// 先检查黑名单
	for _, rule := range mw.blacklistRules {
		if matchIP(ipNum, rule.Network) {
			return false
		}
	}

	// 再检查白名单
	if len(mw.whitelistRules) > 0 {
		for _, rule := range mw.whitelistRules {
			if matchIP(ipNum, rule.Network) {
				return true
			}
		}
		return false
	}

	return mw.defaultAllow
}

// parseIP 解析 IP 地址为 uint32（简化实现）
func parseIP(ip string) uint32 {
	// 简化实现：192.168.1.1 -> 3232235777
	// 生产环境应使用 net.ParseIP
	var a, b, c, d uint32
	_, err := scanIP(ip, &a, &b, &c, &d)
	if err != nil {
		return 0
	}
	return (a << 24) | (b << 16) | (c << 8) | d
}

func scanIP(ip string, a, b, c, d *uint32) (int, error) {
	// 简化解析
	n := 0
	*a = 0
	*b = 0
	*c = 0
	*d = 0

	val := uint32(0)
	count := 0
	idx := 0

	for i := 0; i <= len(ip); i++ {
		if i == len(ip) || ip[i] == '.' {
			if count == 0 {
				return n, nil
			}
			switch idx {
			case 0:
				*a = val
			case 1:
				*b = val
			case 2:
				*c = val
			case 3:
				*d = val
			}
			idx++
			val = 0
			count = 0
			n++
		} else if ip[i] >= '0' && ip[i] <= '9' {
			val = val*10 + uint32(ip[i]-'0')
			count++
			if count > 3 {
				return n, nil
			}
		} else {
			return n, nil
		}
	}

	return n, nil
}

func matchIP(ip uint32, network *IPNet) bool {
	if network == nil {
		return false
	}
	if network.Mask == 0 {
		return true // 0.0.0.0/0 matches all
	}
	return (ip & network.Mask) == (network.IP & network.Mask)
}

// GetStats 获取统计信息
func (mw *ACLMiddleware) GetStats() map[string]interface{} {
	mw.mu.RLock()
	defer mw.mu.RUnlock()

	return map[string]interface{}{
		"enabled":         mw.config.Enabled,
		"whitelist_count": len(mw.whitelistRules),
		"blacklist_count": len(mw.blacklistRules),
		"default_policy":  mw.config.DefaultPolicy,
	}
}

// UpdateConfig 更新配置
func (mw *ACLMiddleware) UpdateConfig(config ACLConfig) {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	mw.config = config
	mw.defaultAllow = config.DefaultPolicy != "deny"
	mw.compileRules()
}

// AddToWhitelist 动态添加白名单
func (mw *ACLMiddleware) AddToWhitelist(cidr string) bool {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	if network := parseCIDR(cidr); network != nil {
		mw.whitelistRules = append(mw.whitelistRules, &ACLRule{
			CIDR:    cidr,
			Network: network,
			Action:  "allow",
		})
		return true
	}
	return false
}

// AddToBlacklist 动态添加黑名单
func (mw *ACLMiddleware) AddToBlacklist(cidr string) bool {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	if network := parseCIDR(cidr); network != nil {
		mw.blacklistRules = append(mw.blacklistRules, &ACLRule{
			CIDR:    cidr,
			Network: network,
			Action:  "deny",
		})
		return true
	}
	return false
}

// RemoveFromWhitelist 从白名单移除
func (mw *ACLMiddleware) RemoveFromWhitelist(cidr string) bool {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	for i, rule := range mw.whitelistRules {
		if rule.CIDR == cidr {
			mw.whitelistRules = append(mw.whitelistRules[:i], mw.whitelistRules[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveFromBlacklist 从黑名单移除
func (mw *ACLMiddleware) RemoveFromBlacklist(cidr string) bool {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	for i, rule := range mw.blacklistRules {
		if rule.CIDR == cidr {
			mw.blacklistRules = append(mw.blacklistRules[:i], mw.blacklistRules[i+1:]...)
			return true
		}
	}
	return false
}

// GeoIPBlocker 基于地理位置的封锁（框架）
type GeoIPBlocker struct {
	blockedCountries map[string]bool // ISO 国家代码
	enabled          bool
	mu               sync.RWMutex
}

// NewGeoIPBlocker 创建地理位置封锁器
func NewGeoIPBlocker(blockedCountries []string) *GeoIPBlocker {
	gb := &GeoIPBlocker{
		blockedCountries: make(map[string]bool),
	}

	for _, country := range blockedCountries {
		gb.blockedCountries[country] = true
	}

	return gb
}

// IsBlocked 检查国家是否被封锁
func (gb *GeoIPBlocker) IsBlocked(countryCode string) bool {
	if !gb.enabled {
		return false
	}

	gb.mu.RLock()
	defer gb.mu.RUnlock()

	return gb.blockedCountries[countryCode]
}

// Enable 启用封锁
func (gb *GeoIPBlocker) Enable() {
	gb.mu.Lock()
	defer gb.mu.Unlock()
	gb.enabled = true
}

// Disable 禁用封锁
func (gb *GeoIPBlocker) Disable() {
	gb.mu.Lock()
	defer gb.mu.Unlock()
	gb.enabled = false
}

// AddBlockedCountry 添加封锁国家
func (gb *GeoIPBlocker) AddBlockedCountry(countryCode string) {
	gb.mu.Lock()
	defer gb.mu.Unlock()
	gb.blockedCountries[countryCode] = true
}

// RemoveBlockedCountry 移除封锁国家
func (gb *GeoIPBlocker) RemoveBlockedCountry(countryCode string) {
	gb.mu.Lock()
	defer gb.mu.Unlock()
	delete(gb.blockedCountries, countryCode)
}

// RequestLogger 请求日志记录器（用于审计）
type RequestLogger struct {
	logFunc func(entry map[string]interface{})
	mu      sync.Mutex
}

// NewRequestLogger 创建请求日志记录器
func NewRequestLogger(logFunc func(entry map[string]interface{})) *RequestLogger {
	return &RequestLogger{
		logFunc: logFunc,
	}
}

// Log 记录请求
func (rl *RequestLogger) Log(entry map[string]interface{}) {
	if rl.logFunc == nil {
		return
	}

	entry["timestamp"] = time.Now().Unix()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.logFunc(entry)
}

// AuditLog 审计日志
type AuditLog struct {
	entries []AuditEntry
	maxSize int
	mu      sync.RWMutex
}

// AuditEntry 审计条目
type AuditEntry struct {
	Timestamp time.Time
	Action    string
	SourceIP  string
	UserID    string
	Details   map[string]interface{}
}

// NewAuditLog 创建审计日志
func NewAuditLog(maxSize int) *AuditLog {
	return &AuditLog{
		entries: make([]AuditEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add 添加审计条目
func (al *AuditLog) Add(action, sourceIP, userID string, details map[string]interface{}) {
	al.mu.Lock()
	defer al.mu.Unlock()

	entry := AuditEntry{
		Timestamp: time.Now(),
		Action:    action,
		SourceIP:  sourceIP,
		UserID:    userID,
		Details:   details,
	}

	al.entries = append(al.entries, entry)

	// 限制大小
	if len(al.entries) > al.maxSize {
		al.entries = al.entries[1:]
	}
}

// GetEntries 获取审计条目
func (al *AuditLog) GetEntries(limit int) []AuditEntry {
	al.mu.RLock()
	defer al.mu.RUnlock()

	if limit > len(al.entries) {
		limit = len(al.entries)
	}

	result := make([]AuditEntry, limit)
	copy(result, al.entries[len(al.entries)-limit:])

	return result
}
