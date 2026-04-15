package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AuthConfig 认证配置
type AuthConfig struct {
	// 认证类型：jwt, oauth2, basic, apikey
	Type string `json:"type"`
	// JWT 密钥
	JWTSecret string `json:"jwt_secret"`
	// JWT 签发者
	JWTIssuer string `json:"jwt_issuer"`
	// OAuth2 端点
	OAuth2Endpoint string `json:"oauth2_endpoint"`
	// Basic Auth 用户映射
	BasicAuthUsers map[string]string `json:"basic_auth_users"`
	// API Key 列表
	APIKeys map[string]string `json:"api_keys"`
	// 是否启用
	Enabled bool `json:"enabled"`
}

// AuthResult 认证结果
type AuthResult struct {
	Authenticated bool
	UserID        string
	Roles         []string
	Metadata      map[string]interface{}
}

// JWTValidator JWT 验证器
type JWTValidator struct {
	secret string
	issuer string
	cache  sync.Map // token -> claims cache
}

// NewJWTValidator 创建 JWT 验证器
func NewJWTValidator(secret, issuer string) *JWTValidator {
	return &JWTValidator{
		secret: secret,
		issuer: issuer,
	}
}

// Validate 验证 JWT token
func (v *JWTValidator) Validate(token string) (*AuthResult, error) {
	// 简单的 JWT 验证实现（生产环境应使用成熟的 jwt 库）
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	// 验证签名
	signature := parts[2]
	payload := parts[0] + "." + parts[1]
	expectedSig := v.sign(payload)

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return nil, errors.New("invalid signature")
	}

	// 解码 payload（简化实现）
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	// 解析 claims（简化实现，实际应解析 JSON）
	result := &AuthResult{
		Authenticated: true,
		UserID:        "user_from_jwt",
		Roles:         []string{"user"},
		Metadata:      map[string]interface{}{"raw_payload": string(decoded)},
	}

	return result, nil
}

func (v *JWTValidator) sign(data string) string {
	h := hmac.New(sha256.New, []byte(v.secret))
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// BasicAuthValidator Basic Auth 验证器
type BasicAuthValidator struct {
	users map[string]string // username -> password hash
}

// NewBasicAuthValidator 创建 Basic Auth 验证器
func NewBasicAuthValidator(users map[string]string) *BasicAuthValidator {
	return &BasicAuthValidator{
		users: users,
	}
}

// Validate 验证 Basic Auth
func (v *BasicAuthValidator) Validate(username, password string) bool {
	expectedPassword, exists := v.users[username]
	if !exists {
		return false
	}
	// 简单密码比较（生产环境应使用 bcrypt 等安全哈希）
	return expectedPassword == password
}

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	config         AuthConfig
	jwtValidator   *JWTValidator
	basicValidator *BasicAuthValidator
	cache          sync.Map // token -> AuthResult
	cacheTTL       time.Duration
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(config AuthConfig) *AuthMiddleware {
	mw := &AuthMiddleware{
		config:   config,
		cacheTTL: 5 * time.Minute,
	}

	if config.JWTSecret != "" {
		mw.jwtValidator = NewJWTValidator(config.JWTSecret, config.JWTIssuer)
	}

	if config.BasicAuthUsers != nil && len(config.BasicAuthUsers) > 0 {
		mw.basicValidator = NewBasicAuthValidator(config.BasicAuthUsers)
	}

	return mw
}

// Authenticate 执行认证
func (mw *AuthMiddleware) Authenticate(r *http.Request) (*AuthResult, error) {
	if !mw.config.Enabled {
		return &AuthResult{Authenticated: true}, nil
	}

	switch mw.config.Type {
	case "jwt":
		return mw.authenticateJWT(r)
	case "basic":
		return mw.authenticateBasic(r)
	case "apikey":
		return mw.authenticateAPIKey(r)
	case "oauth2":
		return mw.authenticateOAuth2(r)
	default:
		return &AuthResult{Authenticated: true}, nil
	}
}

func (mw *AuthMiddleware) authenticateJWT(r *http.Request) (*AuthResult, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, errors.New("invalid authorization header format")
	}

	token := parts[1]

	// 检查缓存
	if cached, ok := mw.cache.Load(token); ok {
		if result, ok := cached.(*AuthResult); ok {
			return result, nil
		}
	}

	if mw.jwtValidator == nil {
		return nil, errors.New("jwt validator not configured")
	}

	result, err := mw.jwtValidator.Validate(token)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	mw.cache.Store(token, result)

	return result, nil
}

func (mw *AuthMiddleware) authenticateBasic(r *http.Request) (*AuthResult, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, errors.New("missing basic auth credentials")
	}

	if mw.basicValidator == nil {
		return nil, errors.New("basic auth validator not configured")
	}

	if !mw.basicValidator.Validate(username, password) {
		return nil, errors.New("invalid credentials")
	}

	return &AuthResult{
		Authenticated: true,
		UserID:        username,
		Roles:         []string{"user"},
	}, nil
}

func (mw *AuthMiddleware) authenticateAPIKey(r *http.Request) (*AuthResult, error) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.URL.Query().Get("api_key")
	}

	if apiKey == "" {
		return nil, errors.New("missing API key")
	}

	if mw.config.APIKeys == nil {
		return nil, errors.New("API keys not configured")
	}

	userID, exists := mw.config.APIKeys[apiKey]
	if !exists {
		return nil, errors.New("invalid API key")
	}

	return &AuthResult{
		Authenticated: true,
		UserID:        userID,
		Roles:         []string{"api_user"},
	}, nil
}

func (mw *AuthMiddleware) authenticateOAuth2(r *http.Request) (*AuthResult, error) {
	// OAuth2 实现需要与认证服务器交互，这里提供框架
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("missing authorization header")
	}

	// TODO: 实现完整的 OAuth2 流程
	return &AuthResult{
		Authenticated: true,
		UserID:        "oauth2_user",
		Roles:         []string{"oauth2_user"},
	}, nil
}

// GetStats 获取统计信息
func (mw *AuthMiddleware) GetStats() map[string]interface{} {
	count := 0
	mw.cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return map[string]interface{}{
		"type":           mw.config.Type,
		"enabled":        mw.config.Enabled,
		"cache_size":     count,
		"cache_ttl_sec":  mw.cacheTTL.Seconds(),
	}
}

// Cleanup 清理过期缓存
func (mw *AuthMiddleware) Cleanup() {
	// 简单实现：清空所有缓存（生产环境应实现 TTL 过期）
	mw.cache.Range(func(key, value interface{}) bool {
		mw.cache.Delete(key)
		return true
	})
}

// ReplayProtection 防重放攻击
type ReplayProtection struct {
	seenRequests sync.Map // nonce -> timestamp
	window       time.Duration
	mu           sync.Mutex
}

// NewReplayProtection 创建防重放保护
func NewReplayProtection(window time.Duration) *ReplayProtection {
	rp := &ReplayProtection{
		window: window,
	}

	// 定期清理
	go rp.cleanupLoop()

	return rp
}

// Check 检查请求是否重放
func (rp *ReplayProtection) Check(nonce string, timestamp int64) error {
	now := time.Now().Unix()

	// 检查时间窗口
	if now-timestamp > int64(rp.window.Seconds()) {
		return errors.New("request timestamp expired")
	}

	// 检查 nonce 是否已使用
	if _, exists := rp.seenRequests.Load(nonce); exists {
		return errors.New("duplicate request nonce")
	}

	// 记录 nonce
	rp.seenRequests.Store(nonce, now)

	return nil
}

func (rp *ReplayProtection) cleanupLoop() {
	ticker := time.NewTicker(rp.window / 2)
	defer ticker.Stop()

	for range ticker.C {
		rp.cleanup()
	}
}

func (rp *ReplayProtection) cleanup() {
	now := time.Now().Unix()
	expiry := now - int64(rp.window.Seconds())

	rp.seenRequests.Range(func(key, value interface{}) bool {
		if ts, ok := value.(int64); ok && ts < expiry {
			rp.seenRequests.Delete(key)
		}
		return true
	})
}

// ValidateRequestSignature 验证请求签名（防重放）
func ValidateRequestSignature(r *http.Request, secret string, rp *ReplayProtection) error {
	nonce := r.Header.Get("X-Nonce")
	timestampStr := r.Header.Get("X-Timestamp")
	signature := r.Header.Get("X-Signature")

	if nonce == "" || timestampStr == "" || signature == "" {
		return errors.New("missing signature headers")
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	// 检查重放
	if err := rp.Check(nonce, timestamp); err != nil {
		return err
	}

	// 验证签名
	expectedSig := computeSignature(r, nonce, timestampStr, secret)
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return errors.New("invalid signature")
	}

	return nil
}

func computeSignature(r *http.Request, nonce, timestamp, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(r.Method))
	h.Write([]byte(r.URL.Path))
	h.Write([]byte(nonce))
	h.Write([]byte(timestamp))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
