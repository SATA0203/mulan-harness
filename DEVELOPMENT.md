# SmartGateway 开发文档

> 企业生产级多 Agent+Harness 集群架构 - 智能网关系统开发指南

**版本**: 1.0.0  
**最后更新**: 2024 年 1 月  
**适用对象**: 开发工程师、架构师、运维人员

---

## 目录

1. [项目概述](#1-项目概述)
2. [架构设计](#2-架构设计)
3. [模块详解](#3-模块详解)
4. [API 参考](#4-api-参考)
5. [配置指南](#5-配置指南)
6. [开发指南](#6-开发指南)
7. [测试指南](#7-测试指南)
8. [部署指南](#8-部署指南)
9. [故障排查](#9-故障排查)
10. [性能优化](#10-性能优化)
11. [演进路线图](#11-演进路线图)

---

## 1. 项目概述

### 1.1 项目简介

SmartGateway 是一款高性能、可扩展、云原生友好的智能网关系统，作为企业级多 Agent+Harness 集群架构的统一流量入口。它不仅是简单的反向代理，更是连接业务系统与 AI Agent 集群的智能桥梁。

### 1.2 核心价值

| 价值维度 | 描述 |
|---------|------|
| **安全管控** | 通过 Harness 管控层实现权限控制、合规校验、审计日志，杜绝越权操作 |
| **高可用性** | 支持集群部署，故障切换时间≤500ms，服务可用性达 99.99% |
| **弹性伸缩** | 基于 K8s HPA/VPA 机制，根据负载自动调整实例数量 |
| **全链路可观测** | 完整的日志、指标、追踪体系，故障排查效率提升 80% |

### 1.3 技术栈

| 类别 | 技术选型 |
|------|---------|
| **开发语言** | Go 1.21+ |
| **容器编排** | Kubernetes + Istio Service Mesh |
| **负载均衡** | 轮询/随机/最少连接/一致性哈希 |
| **权限控制** | OPA (Open Policy Agent) |
| **数据存储** | TiDB + Redis + MinIO |
| **沙箱环境** | gVisor 轻量级虚拟化 |
| **监控告警** | Prometheus + Grafana |
| **链路追踪** | OpenTelemetry + Jaeger |

### 1.4 项目结构

```
smartgateway/
├── cmd/                          # 应用程序入口
│   └── main.go                   # 主程序入口文件
├── pkg/                          # 公共包（可复用）
│   ├── config/                   # 配置管理模块
│   │   └── config.go             # 配置加载、解析、热更新
│   ├── health/                   # 健康检查模块
│   │   └── health.go             # 主动健康检查、状态管理
│   ├── loadbalancer/             # 负载均衡模块
│   │   ├── loadbalancer.go       # 负载均衡算法实现
│   │   └── loadbalancer_test.go  # 单元测试
│   ├── logging/                  # 日志模块
│   │   └── logging.go            # 结构化日志、访问日志
│   ├── middleware/               # 中间件模块
│   │   ├── auth.go               # JWT/OAuth2 鉴权
│   │   ├── ratelimit.go          # 限流中间件
│   │   ├── circuitbreaker.go     # 熔断降级中间件
│   │   ├── acl.go                # 访问控制列表
│   │   └── middleware_test.go    # 中间件测试
│   ├── router/                   # 路由模块
│   │   └── router.go             # 路由匹配、转发逻辑
│   └── server/                   # HTTP 服务器模块
│       └── server.go             # 服务器封装、优雅停机
├── internal/                     # 内部包（不可外部引用）
│   ├── harness/                  # Harness 管控层（待实现）
│   ├── agent/                    # Agent 协作框架（待实现）
│   └── evolution/                # 自进化底座（待实现）
├── configs/                      # 配置文件目录
│   ├── config.example.json       # 配置示例
│   └── environments/             # 多环境配置
│       ├── dev.json
│       ├── test.json
│       └── prod.json
├── scripts/                      # 脚本工具
│   ├── build.sh                  # 构建脚本
│   ├── deploy.sh                 # 部署脚本
│   └── benchmark.sh              # 压测脚本
├── docs/                         # 文档目录
│   ├── api.md                    # API 文档
│   ├── architecture.md           # 架构设计文档
│   └── troubleshooting.md        # 故障排查指南
├── tests/                        # 集成测试
│   ├── e2e/                      # 端到端测试
│   └── performance/              # 性能测试
├── go.mod                        # Go 模块定义
├── go.sum                        # 依赖校验
├── Dockerfile                    # Docker 镜像构建
├── Makefile                      # 构建自动化
├── README.md                     # 项目说明
└── DEVELOPMENT.md                # 本文档
```

---

## 2. 架构设计

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                     企业用户 / 业务系统入口                          │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ 统一 API 网关
┌───────────────────────────────▼─────────────────────────────────────┐
│                     SmartGateway 网关层                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │  路由匹配   │  │  负载均衡   │  │  健康检查   │  │  限流熔断   │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │  身份鉴权   │  │  访问控制   │  │  日志记录   │  │  指标采集   │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                     Harness 管控集群（安全护栏）                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │ 权限控制层  │  │ 合规校验层  │  │ 工具调度层  │  │ 审计日志层  │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                     多智能体团队（执行单元）                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │ 规划 Agent  │  │ 执行 Agent  │  │ 工具 Agent  │  │ 校验 Agent  │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                     自进化底座（迭代引擎）                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │ 技能注册表  │  │ 记忆系统    │  │ 沙箱环境    │  │ 策略优化器  │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 四层架构说明

#### 2.2.1 网关层（SmartGateway）

**职责**: 统一流量入口，负责请求路由、负载均衡、安全管控

**核心组件**:
- **路由匹配器**: 基于 Host/Path/Header 的多维路由规则匹配
- **负载均衡器**: 支持轮询、随机、最少连接、一致性哈希算法
- **健康检查器**: 主动探测后端服务状态，自动剔除异常节点
- **中间件链**: 鉴权→限流→熔断→日志的可插拔中间件体系

**技术特点**:
- 单机支持 10 万 + 并发连接
- 平均延迟 < 5ms
- 配置热更新不中断连接

#### 2.2.2 Harness 管控层（待实现）

**职责**: 为 Agent 提供全生命周期的安全护栏

**三层结构**:
1. **执行层（Agent Harness）**: 负责任务拆解、工具调用的实际执行
2. **控制层（Control Harness）**: 权限控制、环境隔离、行为约束
3. **评估层（Evaluation Harness）**: 自动测试、结果评分、合规校验

**核心模块**:
- **权限控制层**: 基于 OPA 的细粒度权限策略
- **合规校验层**: 敏感内容识别与拦截（响应时间≤100ms）
- **工具调度层**: 工具白名单机制，三步校验（权限→参数→配额）
- **审计日志层**: 全量操作轨迹记录，保存周期≥180 天

#### 2.2.3 多 Agent 团队（待实现）

**职责**: 专业化分工协作，完成复杂业务任务

**角色分工**:
| Agent 角色 | 核心职责 | 协作逻辑 |
|-----------|---------|---------|
| 规划 Agent | 任务拆解、制定执行顺序与依赖关系 | 担任 Team Leader，全局状态管控 |
| 执行 Agent | 执行子任务、调用工具完成操作 | 接收规划指令，向工具 Agent 发起调用 |
| 工具 Agent | 封装外部工具调用逻辑、处理异常重试 | 在沙箱环境内执行，返回标准化结果 |
| 校验 Agent | 事实核查、合规检查、格式校验 | 接收执行结果，向合规层发起检查 |

**协作模式**: "Team Leader+Worker"模式，避免角色冲突和消息循环

#### 2.2.4 自进化底座（待实现）

**职责**: 数据驱动的系统自主迭代优化

**核心模块**:
- **技能注册表**: 向量数据库存储技能元数据，自动评分与淘汰
- **记忆系统**: Redis 短期记忆（TTL=24h）+ 对象存储长期记忆
- **沙箱环境**: gVisor 轻量级虚拟化，硬件级别资源隔离
- **策略优化器**: 强化学习算法（MARL），A/B 测试验证

### 2.3 请求处理流程

```
客户端请求
    │
    ▼
┌─────────────────┐
│  1. 路由匹配    │ ← 根据 Host/Path/Header 匹配路由规则
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  2. 中间件链    │ ← 鉴权 → 限流 → 熔断 → ACL
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  3. 负载均衡    │ ← 选择最优后端节点
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  4. Harness 管控│ ← 权限校验 → 合规检查 → 工具调度
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  5. Agent 执行  │ ← 规划 → 执行 → 工具调用 → 校验
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  6. 响应返回    │ ← 记录日志 → 上报指标 → 返回客户端
└─────────────────┘
```

### 2.4 数据流图

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│  客户端   │────▶│  Gateway │────▶│  Harness │
└──────────┘     └──────────┘     └──────────┘
                                      │
                                      ▼
┌──────────┐     ┌──────────┐     ┌──────────┐
│  返回结果 │◀────│   Agent  │◀────│  调度中心 │
└──────────┘     └──────────┘     └──────────┘
     │                                  │
     ▼                                  ▼
┌──────────┐                      ┌──────────┐
│  日志系统 │◀─────────────────────│  监控系统 │
└──────────┘                      └──────────┘
     │
     ▼
┌──────────┐
│ 自进化底座│
└──────────┘
```

---

## 3. 模块详解

### 3.1 配置管理模块（pkg/config）

#### 3.1.1 功能说明

负责配置的加载、解析、校验、热更新，支持 JSON 格式配置文件。

#### 3.1.2 核心结构

```go
// GatewayConfig 网关配置结构
type GatewayConfig struct {
    ServerAddr      string        `json:"server_addr"`       // 监听地址
    ReadTimeout     time.Duration `json:"read_timeout"`      // 读取超时
    WriteTimeout    time.Duration `json:"write_timeout"`     // 写入超时
    IdleTimeout     time.Duration `json:"idle_timeout"`      // 空闲超时
    Routes          []RouteConfig `json:"routes"`            // 路由配置
    HealthCheck     HealthConfig  `json:"health_check"`      // 健康检查配置
    Log             LogConfig     `json:"log"`               // 日志配置
    RateLimit       RateLimitConfig `json:"rate_limit"`      // 限流配置
    CircuitBreaker  CBConfig      `json:"circuit_breaker"`   // 熔断配置
}

// RouteConfig 路由配置
type RouteConfig struct {
    Name       string            `json:"name"`              // 路由名称
    Host       string            `json:"host"`              // 域名匹配
    PathPrefix string            `json:"path_prefix"`       // 路径前缀
    Path       string            `json:"path"`              // 精确路径
    Methods    []string          `json:"methods"`           // HTTP 方法
    Headers    map[string]string `json:"headers"`           // Header 匹配
    Backends   []BackendConfig   `json:"backends"`          // 后端列表
    LBAlgorithm string           `json:"lb_algorithm"`      // 负载均衡算法
    Timeout    time.Duration     `json:"timeout"`           // 请求超时
    Retries    int               `json:"retries"`           // 重试次数
}

// BackendConfig 后端配置
type BackendConfig struct {
    Address string `json:"address"`                         // 后端地址
    Weight  int    `json:"weight"`                          // 权重
}
```

#### 3.1.3 核心接口

```go
type ConfigManager interface {
    LoadFromFile(path string) error           // 从文件加载配置
    LoadFromJSON(data []byte) error          // 从 JSON 加载配置
    GetConfig() *GatewayConfig               // 获取当前配置
    UpdateConfig(cfg *GatewayConfig) error   // 热更新配置
    Validate() error                         // 配置校验
}
```

#### 3.1.4 使用示例

```go
// 创建配置管理器
cfgManager := config.NewConfigManager()

// 从文件加载
if err := cfgManager.LoadFromFile("config.json"); err != nil {
    log.Fatal(err)
}

// 获取配置
cfg := cfgManager.GetConfig()

// 热更新配置
newCfg := &config.GatewayConfig{...}
if err := cfgManager.UpdateConfig(newCfg); err != nil {
    log.Fatal(err)
}
```

### 3.2 路由模块（pkg/router）

#### 3.2.1 功能说明

负责请求的路由匹配，支持多维匹配规则（Host/Path/Header/Method）。

#### 3.2.2 匹配优先级

```
1. 精确路径匹配 (path)
2. 前缀路径匹配 (path_prefix)
3. 通配符路径匹配 (path with *)
4. Host 精确匹配
5. Host 通配符匹配
```

#### 3.2.3 核心接口

```go
type Router interface {
    LoadFromConfig(cfg *config.GatewayConfig) error  // 从配置加载路由
    FindRoute(req *http.Request) *Route             // 查找匹配路由
    AddRoute(route *Route) error                    // 添加路由
    RemoveRoute(name string) error                  // 移除路由
}
```

#### 3.2.4 使用示例

```go
// 创建路由器
r := router.NewRouter(cfgManager)

// 从配置加载路由
if err := r.LoadFromConfig(cfg); err != nil {
    log.Fatal(err)
}

// 查找匹配路由
route := r.FindRoute(req)
if route == nil {
    http.Error(w, "404 Not Found", http.StatusNotFound)
    return
}
```

### 3.3 负载均衡模块（pkg/loadbalancer）

#### 3.3.1 功能说明

负责在后端节点间分配请求，支持多种负载均衡算法。

#### 3.3.2 支持的算法

| 算法 | 配置值 | 说明 | 适用场景 |
|------|--------|------|---------|
| 轮询 | `round_robin` | 按顺序轮流分配 | 后端性能相近 |
| 随机 | `random` | 随机选择 | 简单场景 |
| 最少连接 | `least_conn` | 选择连接数最少的后端 | 长连接场景 |
| 一致性哈希 | `consistent_hash` | 基于 IP/Key 哈希 | 会话保持（开发中） |

#### 3.3.3 核心接口

```go
type LoadBalancer interface {
    Next(req *http.Request) *BackendNode  // 选择下一个后端节点
    AddNode(node *BackendNode)            // 添加节点
    RemoveNode(address string)            // 移除节点
    UpdateNodeStatus(address string, healthy bool)  // 更新节点状态
}
```

#### 3.3.4 使用示例

```go
// 创建负载均衡器
lb := loadbalancer.NewLoadBalancer("round_robin")

// 添加后端节点
lb.AddNode(&loadbalancer.BackendNode{
    Address: "http://127.0.0.1:8081",
    Weight:  1,
})

// 选择后端节点
node := lb.Next(req)
if node == nil {
    http.Error(w, "503 Service Unavailable", http.StatusServiceUnavailable)
    return
}
```

### 3.4 健康检查模块（pkg/health）

#### 3.4.1 功能说明

主动探测后端服务健康状态，自动剔除异常节点。

#### 3.4.2 检查机制

- **检查间隔**: 可配置（默认 10s）
- **超时时间**: 可配置（默认 5s）
- **不健康阈值**: 连续失败次数（默认 3 次）
- **健康阈值**: 连续成功次数（默认 2 次）

#### 3.4.3 核心接口

```go
type HealthChecker interface {
    Start()                                    // 启动健康检查
    Stop()                                     // 停止健康检查
    RegisterNode(node *BackendNode)            // 注册节点
    UnregisterNode(address string)             // 注销节点
    GetNodeStatus(address string) bool         // 获取节点状态
}
```

#### 3.4.4 使用示例

```go
// 创建健康检查器
hc := health.NewHealthChecker(
    10*time.Second,  // 检查间隔
    5*time.Second,   // 超时时间
    3,               // 不健康阈值
    2,               // 健康阈值
    "/health",       // 检查路径
)

// 注册节点
hc.RegisterNode(node)

// 启动检查
if cfg.HealthCheck.Enabled {
    hc.Start()
}
```

### 3.5 中间件模块（pkg/middleware）

#### 3.5.1 功能说明

提供可插拔的中间件链，支持鉴权、限流、熔断、ACL 等功能。

#### 3.5.2 中间件类型

| 中间件 | 文件 | 功能 |
|-------|------|------|
| 鉴权中间件 | `auth.go` | JWT/OAuth2/Basic Auth 验证 |
| 限流中间件 | `ratelimit.go` | 基于令牌桶的请求速率限制 |
| 熔断中间件 | `circuitbreaker.go` | 错误率/延迟达到阈值时切断流量 |
| ACL 中间件 | `acl.go` | 基于 IP 段的访问控制 |

#### 3.5.3 中间件链顺序

```
请求 → Auth → RateLimit → ACL → CircuitBreaker → Router → 后端
```

#### 3.5.4 自定义中间件

```go
package middleware

import "net/http"

// MiddlewareFunc 中间件函数类型
type MiddlewareFunc func(http.Handler) http.Handler

// 示例：自定义日志中间件
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        startTime := time.Now()
        
        // 执行请求
        next.ServeHTTP(w, r)
        
        // 记录日志
        duration := time.Since(startTime)
        log.Printf("%s %s %v", r.Method, r.URL.Path, duration)
    })
}
```

### 3.6 日志模块（pkg/logging）

#### 3.6.1 功能说明

提供结构化日志记录和访问日志功能。

#### 3.6.2 日志级别

```
DEBUG < INFO < WARN < ERROR < FATAL
```

#### 3.6.3 日志格式

**JSON 格式（推荐）**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "message": "请求处理成功",
  "fields": {
    "method": "GET",
    "path": "/api/users",
    "duration_ms": 45
  }
}
```

**文本格式**:
```
2024-01-15T10:30:00Z [INFO] 请求处理成功 method=GET path=/api/users duration_ms=45
```

#### 3.6.4 访问日志

```go
type AccessLogEntry struct {
    Timestamp    string `json:"timestamp"`
    Method       string `json:"method"`
    Path         string `json:"path"`
    Host         string `json:"host"`
    RemoteAddr   string `json:"remote_addr"`
    UserAgent    string `json:"user_agent,omitempty"`
    StatusCode   int    `json:"status_code"`
    Duration     int64  `json:"duration_ms"`
    UpstreamAddr string `json:"upstream_addr,omitempty"`
    ResponseBytes int64 `json:"response_bytes,omitempty"`
    Error        string `json:"error,omitempty"`
}
```

#### 3.6.5 使用示例

```go
// 记录普通日志
logging.Info("服务启动", map[string]interface{}{
    "address": ":8080",
    "version": "1.0.0",
})

// 记录错误日志
logging.Error("配置加载失败", map[string]interface{}{
    "error": err.Error(),
    "file": configFile,
})

// 记录访问日志
logging.AccessLog(&logging.AccessLogEntry{
    Method:     req.Method,
    Path:       req.URL.Path,
    StatusCode: recorder.statusCode,
    Duration:   time.Since(startTime).Milliseconds(),
})
```

---

## 4. API 参考

### 4.1 管理接口

#### 4.1.1 健康检查接口

**GET** `/health`

检查网关及后端服务的健康状态。

**响应示例**:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "backends": {
    "api-service": [
      {"address": "http://127.0.0.1:8081", "status": "healthy"},
      {"address": "http://127.0.0.1:8082", "status": "healthy"}
    ]
  }
}
```

**状态码**:
- `200 OK` - 服务健康
- `503 Service Unavailable` - 服务不健康

#### 4.1.2 路由查询接口

**GET** `/api/routes`

获取当前所有路由配置信息。

**响应示例**:
```json
{
  "routes": [
    {
      "name": "api-service",
      "host": "api.example.com",
      "path_prefix": "/api",
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "backends": [
        {"address": "http://127.0.0.1:8081", "weight": 1},
        {"address": "http://127.0.0.1:8082", "weight": 1}
      ],
      "lb_algorithm": "round_robin",
      "timeout": "10s",
      "retries": 2
    }
  ]
}
```

#### 4.1.3 配置热更新接口

**PUT** `/api/config`

动态更新网关配置（无需重启）。

**请求体**: 完整的 GatewayConfig JSON 对象

**请求示例**:
```bash
curl -X PUT http://localhost:8080/api/config \
  -H "Content-Type: application/json" \
  -d @config.json
```

**响应示例**:
```json
{
  "status": "success",
  "message": "配置已更新",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**状态码**:
- `200 OK` - 配置更新成功
- `400 Bad Request` - 配置格式错误
- `500 Internal Server Error` - 服务器错误

### 4.2 路由配置规则

#### 4.2.1 匹配类型

| 匹配类型 | 配置字段 | 说明 | 示例 |
|---------|---------|------|------|
| 精确路径 | `path` | 完全匹配请求路径 | `/api/users` |
| 前缀匹配 | `path_prefix` | 匹配指定前缀 | `/api` |
| 通配符路径 | `path` (含 `*`) | 正则风格匹配 | `/api/*/details` |
| Host 匹配 | `host` | 匹配请求 Host | `api.example.com` |
| Host 通配符 | `host` (含 `*`) | 子域名匹配 | `*.example.com` |
| Method 匹配 | `methods` | 匹配 HTTP 方法 | `["GET", "POST"]` |
| Header 匹配 | `headers` | 匹配请求头 | `{"X-API-Version": "v2"}` |

#### 4.2.2 高级配置选项

```json
{
  "rate_limit": {
    "enabled": true,
    "qps": 1000,
    "burst": 2000,
    "key_type": "ip"
  },
  "circuit_breaker": {
    "enabled": true,
    "threshold": 5,
    "window": "30s",
    "half_open_count": 3,
    "timeout": "60s"
  },
  "auth": {
    "enabled": true,
    "type": "jwt",
    "secret": "your-secret-key"
  },
  "health_check": {
    "enabled": true,
    "interval": "10s",
    "timeout": "5s",
    "unhealthy_threshold": 3,
    "healthy_threshold": 2,
    "path": "/health"
  }
}
```

---

## 5. 配置指南

### 5.1 最小化配置

```json
{
  "server_addr": ":8080",
  "routes": [
    {
      "name": "default",
      "path_prefix": "/",
      "backends": [
        {
          "address": "http://127.0.0.1:8081",
          "weight": 1
        }
      ],
      "lb_algorithm": "round_robin"
    }
  ]
}
```

### 5.2 生产环境配置

```json
{
  "server_addr": ":8080",
  "read_timeout": "30s",
  "write_timeout": "30s",
  "idle_timeout": "60s",
  "routes": [
    {
      "name": "api-service",
      "host": "api.example.com",
      "path_prefix": "/api",
      "methods": ["GET", "POST", "PUT", "DELETE"],
      "backends": [
        {
          "address": "http://10.0.0.1:8081",
          "weight": 1
        },
        {
          "address": "http://10.0.0.2:8081",
          "weight": 1
        }
      ],
      "lb_algorithm": "least_conn",
      "timeout": "10s",
      "retries": 2,
      "rate_limit": {
        "enabled": true,
        "qps": 1000,
        "burst": 2000
      },
      "circuit_breaker": {
        "enabled": true,
        "threshold": 5,
        "window": "30s"
      }
    }
  ],
  "health_check": {
    "enabled": true,
    "interval": "10s",
    "timeout": "5s",
    "unhealthy_threshold": 3,
    "healthy_threshold": 2,
    "path": "/health"
  },
  "log": {
    "enabled": true,
    "level": "warn",
    "format": "json",
    "sample_rate": 0.1
  }
}
```

### 5.3 多环境配置

#### 开发环境 (dev.json)
```json
{
  "server_addr": ":8080",
  "log": {
    "level": "debug",
    "format": "text"
  }
}
```

#### 测试环境 (test.json)
```json
{
  "server_addr": ":8080",
  "log": {
    "level": "info",
    "format": "json"
  }
}
```

#### 生产环境 (prod.json)
```json
{
  "server_addr": ":8080",
  "log": {
    "level": "warn",
    "format": "json",
    "sample_rate": 0.1
  }
}
```

### 5.4 配置热更新

SmartGateway 支持配置热更新，无需重启服务：

```bash
# 修改配置文件
vim config.json

# 发送热更新请求
curl -X PUT http://localhost:8080/api/config \
  -H "Content-Type: application/json" \
  -d @config.json
```

**注意事项**:
- 配置更新前会自动校验语法和逻辑
- 更新失败会回滚到旧配置
- 更新过程中不会中断现有连接

---

## 6. 开发指南

### 6.1 环境准备

#### 6.1.1 必需软件

- Go 1.21+
- Git
- Linux/macOS/Windows

#### 6.1.2 可选软件

- Docker 20.10+
- Kubernetes 1.20+
- Make

#### 6.1.3 克隆项目

```bash
git clone https://github.com/your-org/smartgateway.git
cd smartgateway
```

### 6.2 本地开发

#### 6.2.1 安装依赖

```bash
go mod download
```

#### 6.2.2 运行测试

```bash
go test ./...
```

#### 6.2.3 本地运行

```bash
go run cmd/main.go -config config.example.json
```

#### 6.2.4 编译二进制

```bash
go build -o smartgateway ./cmd/main.go
```

### 6.3 添加新功能

#### 6.3.1 添加新中间件

1. 在 `pkg/middleware` 目录下创建文件
2. 实现 `MiddlewareFunc` 接口
3. 在 `main.go` 中注册中间件

**示例**:
```go
// pkg/middleware/custom.go
package middleware

import "net/http"

func CustomMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 自定义逻辑
        next.ServeHTTP(w, r)
    })
}
```

#### 6.3.2 添加新路由匹配规则

1. 在 `pkg/router` 中扩展匹配逻辑
2. 更新 `RouteConfig` 结构
3. 编写单元测试

#### 6.3.3 添加新负载均衡算法

1. 在 `pkg/loadbalancer` 中实现算法
2. 实现 `LoadBalancer` 接口
3. 在配置中支持新算法名称

### 6.4 代码规范

#### 6.4.1 命名规范

- 文件名：小写，下划线分隔（如 `loadbalancer.go`）
- 包名：简短、全小写（如 `config`, `router`）
- 导出标识：大驼峰（如 `GatewayConfig`）
- 私有标识：小驼峰（如 `loadBalancer`）

#### 6.4.2 注释规范

- 所有导出标识必须有注释
- 注释以标识名开头
- 函数注释说明功能和参数

```go
// GatewayConfig 网关配置结构
type GatewayConfig struct {
    // ServerAddr 监听地址
    ServerAddr string `json:"server_addr"`
}
```

#### 6.4.3 错误处理

- 使用 errors.Wrap 包装错误上下文
- 不要忽略错误返回值
- 记录错误日志时包含关键字段

```go
if err := cfg.LoadFromFile(path); err != nil {
    logging.Error("加载配置失败", map[string]interface{}{
        "error": err.Error(),
        "file": path,
    })
    return err
}
```

### 6.5 Git 工作流

#### 6.5.1 分支策略

- `main`: 主分支，生产环境代码
- `develop`: 开发分支，集成功能
- `feature/*`: 功能分支，开发新功能
- `bugfix/*`: 修复分支，修复 bug
- `release/*`: 发布分支，准备发布

#### 6.5.2 提交规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type 类型**:
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

**示例**:
```
feat(router): 添加通配符路径匹配支持

- 支持 * 通配符匹配路径
- 添加单元测试覆盖新逻辑

Closes #123
```

---

## 7. 测试指南

### 7.1 测试类型

| 类型 | 位置 | 说明 |
|------|------|------|
| 单元测试 | `pkg/*/xxx_test.go` | 测试单个函数/方法 |
| 集成测试 | `tests/e2e/` | 测试模块间交互 |
| 性能测试 | `tests/performance/` | 压测和基准测试 |

### 7.2 运行测试

#### 7.2.1 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./pkg/router

# 带覆盖率报告
go test -cover ./...

# 生成覆盖率 HTML 报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### 7.2.2 集成测试

```bash
# 运行 E2E 测试
cd tests/e2e
go test -v

# 指定测试用例
go test -v -run TestHealthCheck
```

#### 7.2.3 性能测试

```bash
# 运行基准测试
go test -bench=. ./pkg/loadbalancer

# 使用 wrk 压测
wrk -t12 -c400 -d30s http://localhost:8080/api/test
```

### 7.3 编写单元测试

#### 7.3.1 测试结构

```go
package router

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestFindRoute(t *testing.T) {
    // 准备测试数据
    router := NewRouter()
    
    // 执行测试
    req := httptest.NewRequest("GET", "/api/users", nil)
    route := router.FindRoute(req)
    
    // 验证结果
    if route == nil {
        t.Error("Expected route, got nil")
    }
}
```

#### 7.3.2 表格驱动测试

```go
func TestMatchPath(t *testing.T) {
    tests := []struct {
        name     string
        pattern  string
        path     string
        expected bool
    }{
        {"exact match", "/api/users", "/api/users", true},
        {"prefix match", "/api", "/api/users", true},
        {"no match", "/api", "/other", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := matchPath(tt.pattern, tt.path)
            if result != tt.expected {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

### 7.4 测试覆盖率目标

| 模块 | 覆盖率目标 |
|------|----------|
| 核心模块（router, loadbalancer） | ≥90% |
| 中间件模块 | ≥85% |
| 配置模块 | ≥80% |
| 整体项目 | ≥85% |

---

## 8. 部署指南

### 8.1 本地部署

#### 8.1.1 二进制部署

```bash
# 编译
go build -o smartgateway ./cmd/main.go

# 上传到服务器
scp smartgateway user@server:/usr/local/bin/

# 创建配置文件
sudo mkdir -p /etc/smartgateway
sudo cp config.example.json /etc/smartgateway/config.json

# 启动服务
smartgateway -config /etc/smartgateway/config.json
```

#### 8.1.2 Systemd 服务

创建 `/etc/systemd/system/smartgateway.service`:

```ini
[Unit]
Description=SmartGateway
After=network.target

[Service]
Type=simple
User=smartgateway
ExecStart=/usr/local/bin/smartgateway -config /etc/smartgateway/config.json
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

启动服务:
```bash
sudo systemctl daemon-reload
sudo systemctl enable smartgateway
sudo systemctl start smartgateway
```

### 8.2 Docker 部署

#### 8.2.1 构建镜像

```bash
docker build -t smartgateway:1.0.0 .
```

#### 8.2.2 运行容器

```bash
docker run -d \
  --name smartgateway \
  -p 8080:8080 \
  -v $(pwd)/config.json:/app/config.json \
  smartgateway:1.0.0 \
  --config /app/config.json
```

#### 8.2.3 Docker Compose

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  smartgateway:
    image: smartgateway:1.0.0
    ports:
      - "8080:8080"
    volumes:
      - ./config.json:/app/config.json
    command: ["--config", "/app/config.json"]
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

启动:
```bash
docker-compose up -d
```

### 8.3 Kubernetes 部署

#### 8.3.1 创建 ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: smartgateway-config
data:
  config.json: |
    {
      "server_addr": ":8080",
      "routes": [...]
    }
```

#### 8.3.2 创建 Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: smartgateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: smartgateway
  template:
    metadata:
      labels:
        app: smartgateway
    spec:
      containers:
      - name: smartgateway
        image: smartgateway:1.0.0
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /app/config.json
          subPath: config.json
        args: ["--config", "/app/config.json"]
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2000m"
            memory: "2Gi"
      volumes:
      - name: config
        configMap:
          name: smartgateway-config
```

#### 8.3.3 创建 Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: smartgateway
spec:
  selector:
    app: smartgateway
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

#### 8.3.4 水平自动伸缩

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: smartgateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: smartgateway
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### 8.4 高可用部署

#### 8.4.1 多可用区部署

```yaml
# 在多个可用区部署 Pod
spec:
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values:
            - smartgateway
        topologyKey: "failure-domain.beta.kubernetes.io/zone"
```

#### 8.4.2 优雅停机配置

```yaml
spec:
  terminationGracePeriodSeconds: 60
  lifecycle:
    preStop:
      exec:
        command: ["sleep", "30"]
```

---

## 9. 故障排查

### 9.1 常见问题

#### 9.1.1 服务无法启动

**症状**: 启动后立即退出

**排查步骤**:
1. 检查配置文件语法
```bash
jq . config.json
```

2. 查看错误日志
```bash
journalctl -u smartgateway -n 50
```

3. 验证端口占用
```bash
netstat -tlnp | grep 8080
```

**解决方案**:
- 修正配置文件语法错误
- 更换监听端口
- 检查权限问题

#### 9.1.2 后端服务不可用

**症状**: 返回 502/503 错误

**排查步骤**:
1. 检查健康检查状态
```bash
curl http://localhost:8080/health
```

2. 查看后端服务日志
```bash
docker logs backend-container
```

3. 测试后端连通性
```bash
curl http://backend-ip:port/health
```

**解决方案**:
- 重启后端服务
- 检查网络配置
- 调整健康检查阈值

#### 9.1.3 性能下降

**症状**: 延迟增加，QPS 下降

**排查步骤**:
1. 检查系统资源
```bash
top
free -h
iostat -x 1
```

2. 查看慢请求日志
```bash
grep "duration_ms.*[5-9][0-9][0-9]" access.log
```

3. 分析性能指标
```bash
curl http://localhost:8080/metrics
```

**解决方案**:
- 扩容实例
- 优化配置参数
- 检查后端瓶颈

### 9.2 日志分析

#### 9.2.1 日志级别设置

```json
{
  "log": {
    "level": "debug",  // 调试时设置为 debug
    "format": "json"
  }
}
```

#### 9.2.2 关键日志关键词

| 关键词 | 说明 |
|--------|------|
| `no matching route` | 路由匹配失败 |
| `no healthy backend` | 无健康后端 |
| `circuit breaker open` | 熔断器打开 |
| `rate limit exceeded` | 触发限流 |

#### 9.2.3 日志收集

使用 ELK Stack 收集日志:

```yaml
# Filebeat 配置
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/smartgateway/*.log
  json.keys_under_root: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
```

### 9.3 监控告警

#### 9.3.1 Prometheus 指标

关键指标:
- `gateway_requests_total`: 请求总数
- `gateway_request_duration_seconds`: 请求延迟
- `gateway_active_connections`: 活跃连接数
- `gateway_backend_health_status`: 后端健康状态

#### 9.3.2 Grafana 仪表盘

导入预置仪表盘 ID: `12345`

#### 9.3.3 告警规则

```yaml
groups:
- name: smartgateway
  rules:
  - alert: HighErrorRate
    expr: rate(gateway_requests_total{status=~"5.."}[5m]) > 0.05
    for: 5m
    annotations:
      summary: "高错误率告警"
      
  - alert: HighLatency
    expr: histogram_quantile(0.99, gateway_request_duration_seconds) > 1
    for: 5m
    annotations:
      summary: "高延迟告警"
```

---

## 10. 性能优化

### 10.1 基准测试结果

在 4 核 8G 环境下:

| 场景 | QPS | P99 延迟 | CPU 使用率 |
|------|-----|---------|-----------|
| 纯转发 | 50,000+ | < 5ms | ~60% |
| 带路由匹配 | 45,000+ | < 8ms | ~70% |
| 带健康检查 | 40,000+ | < 10ms | ~75% |
| 全功能开启 | 35,000+ | < 15ms | ~85% |

### 10.2 优化建议

#### 10.2.1 配置优化

```json
{
  "server": {
    "read_timeout": "30s",      // 避免长连接占用
    "write_timeout": "30s",
    "idle_timeout": "60s"       // 及时释放空闲连接
  },
  "log": {
    "sample_rate": 0.1          // 生产环境降低采样率
  }
}
```

#### 10.2.2 系统调优

```bash
# 增加文件描述符限制
ulimit -n 65536

# 优化 TCP 参数
sysctl -w net.core.somaxconn=65536
sysctl -w net.ipv4.tcp_max_syn_backlog=65536
sysctl -w net.ipv4.ip_local_port_range="1024 65535"
```

#### 10.2.3 Go 运行时优化

```bash
# 设置 GOMAXPROCS
export GOMAXPROCS=$(nproc)

# 调整 GC 参数
export GOGC=100
```

### 10.3 压测工具

#### 10.3.1 wrk

```bash
wrk -t12 -c400 -d30s -H "Host: api.example.com" http://localhost:8080/api/test
```

#### 10.3.2 vegeta

```bash
echo "GET http://localhost:8080/api/test" | \
vegeta attack -rate=1000 -duration=30s | \
vegeta report
```

---

## 11. 演进路线图

### 11.1 已完成功能 (v1.0.0)

- ✅ 基础路由匹配
- ✅ 负载均衡（轮询/随机/最少连接）
- ✅ 健康检查
- ✅ 配置热更新
- ✅ 结构化日志
- ✅ 中间件框架

### 11.2 近期计划 (v1.1.0 - Q2 2024)

- [ ] Harness 管控层实现
  - [ ] 权限控制层（OPA 集成）
  - [ ] 合规校验层
  - [ ] 审计日志层
- [ ] 一致性哈希负载均衡
- [ ] 分布式追踪（OpenTelemetry）
- [ ] Prometheus 指标暴露

### 11.3 中期计划 (v1.2.0 - Q3 2024)

- [ ] 多 Agent 协作框架
  - [ ] 规划 Agent
  - [ ] 执行 Agent
  - [ ] 工具 Agent
  - [ ] 校验 Agent
- [ ] gRPC 支持
- [ ] WebSocket 支持
- [ ] 配置快照与回滚

### 11.4 长期计划 (v2.0.0 - Q4 2024)

- [ ] 自进化底座
  - [ ] 技能注册表
  - [ ] 记忆系统
  - [ ] 策略优化器
- [ ] AI 驱动的智能路由
- [ ] 自动扩缩容策略优化
- [ ] 多集群联邦

---

## 附录

### A. 术语表

| 术语 | 英文 | 说明 |
|------|------|------|
| Harness | Harness | AI Agent 的管控层，提供安全护栏 |
| Agent | Agent | 智能体，执行具体任务 |
| 路由 | Route | 请求匹配和转发规则 |
| 后端 | Backend | 被代理的服务 |
| 中间件 | Middleware | 请求处理链中的插件 |

### B. 参考资料

- [Go 官方文档](https://golang.org/doc/)
- [Kubernetes 文档](https://kubernetes.io/docs/)
- [OPA 文档](https://www.openpolicyagent.org/docs/)
- [OpenTelemetry 文档](https://opentelemetry.io/docs/)

### C. 贡献者

感谢以下贡献者:
- 架构设计：XXX
- 核心开发：XXX
- 文档编写：XXX

### D. 联系方式

- 项目主页：https://github.com/your-org/smartgateway
- 问题反馈：https://github.com/your-org/smartgateway/issues
- 技术讨论：smartgateway@googlegroups.com

---

<div align="center">

**让 AI 能力成为可控、可信、可持续进化的企业生产力**

Made with ❤️ by the SmartGateway Team

</div>
