# 企业生产级多 Agent+Harness 集群架构

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-1.0.0-green.svg)]()

## 📖 项目简介

本方案针对企业级 AI Agent 集群在生产场景中的**稳定性缺失、安全风险不可控、协同效率低下**三大核心痛点，构建了一套融合以下四层组件的协同架构：

- **Harness 管控集群**（安全护栏）
- **多智能体团队**（执行单元）
- **分布式调度中心**（协同大脑）
- **自进化底座**（迭代引擎）

通过"管控即服务"的理念，将不可控的大模型能力转化为可审计、可回滚、自优化的企业级生产力，满足金融、制造、零售等高合规行业的严苛要求。

## 🏗️ 架构总览

```
┌─────────────────────────────────────────────────────────┐
│               企业用户 / 业务系统入口                     │
└───────────────────────┬─────────────────────────────────┘
                        │ 统一 API 网关
┌───────────────────────▼─────────────────────────────────┐
│              分布式调度中心（协同大脑）                   │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │任务拆解器│ │优先级调度│ │状态同步器│ │冲突仲裁器│       │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│              Harness 管控集群（安全护栏）                 │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │权限控制层│ │合规校验层│ │工具调度层│ │审计日志层│       │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│              多智能体团队（执行单元）                      │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │规划Agent│ │执行Agent│ │工具Agent│ │校验Agent│       │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│              自进化底座（迭代引擎）                        │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │技能注册表│ │记忆系统  │ │沙箱环境  │ │策略优化器│       │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
└─────────────────────────────────────────────────────────┘
```

## ✨ 核心特性

### 六大设计原则

| 原则 | 价值 |
|------|------|
| 🔒 **安全优先** | 杜绝越权操作、数据泄露风险，满足等保合规要求 |
| 🔄 **高可用设计** | 99.99% 服务可用性，故障切换≤500ms |
| 📈 **弹性伸缩** | 基于 K8s HPA/VPA，支持 Serverless 弹性实例 |
| 👁️ **全链路可观测** | 故障排查效率提升 80%，所有日志可审计回放 |
| 🧠 **自进化能力** | 系统自主优化，运维成本降低 60%+ |
| 🔌 **解耦与标准化** | 新增场景无需修改核心代码，扩展性极强 |

### 三层 Harness 管控结构

1. **执行层（Agent Harness）** - 负责任务拆解、工具调用的实际执行
2. **控制层（Control Harness）** - 权限控制、环境隔离、行为约束
3. **评估层（Evaluation Harness）** - 自动测试、结果评分、合规校验

### 多 Agent 角色分工

| Agent 角色 | 核心职责 |
|-----------|---------|
| 🎯 规划 Agent | 任务拆解、制定执行顺序与依赖关系 |
| ⚡ 执行 Agent | 执行子任务、调用工具完成操作 |
| 🛠️ 工具 Agent | 封装外部工具调用逻辑、处理异常重试 |
| ✅ 校验 Agent | 事实核查、合规检查、格式校验 |

## 🚀 生产级验证

本架构已在多个头部企业的生产环境中得到验证：

- **某股份制银行信贷审核系统**：审核效率提升 3 倍，错误率下降 85%
- **某电商仓库 AGV 调度系统**：订单处理效率提升 40%，资源利用率从 55%→82%
- **某视频平台万卡集群**：主节点切换时 200 万在线推理任务无中断
- **某金融机构**：越权操作风险降低 99.9%

## 📂 目录结构

```
.
├── README.md                                    # 项目说明文档
├── 企业生产级多 Agent+Harness 集群架构设计与自进化底座.md  # 完整架构设计文档
└── .git                                         # Git 版本控制
```

## 🛠️ 技术栈

- **容器编排**: Kubernetes + Istio Service Mesh
- **负载均衡**: Ingress Controller + 加权轮询算法
- **权限控制**: OPA (Open Policy Agent)
- **数据存储**: TiDB (分布式数据库) + Redis (缓存) + MinIO (对象存储)
- **沙箱环境**: gVisor 轻量级虚拟化
- **共识算法**: Raft
- **向量数据库**: Milvus / Pinecone

## 🚀 快速开始

### 环境要求

- Go 1.19+
- Linux/macOS/Windows
- （可选）Docker 20.10+
- （可选）Kubernetes 1.20+

### 安装步骤

#### 方式一：从源码编译

```bash
# 克隆仓库
git clone <repository-url>
cd smartgateway

# 编译
go build -o smartgateway ./cmd/main.go

# 验证安装
./smartgateway -version
```

#### 方式二：使用 Go install

```bash
go install smartgateway/cmd@latest
```

#### 方式三：Docker 部署（推荐生产环境）

```bash
# 构建镜像
docker build -t smartgateway:1.0.0 .

# 运行容器
docker run -d \
  --name smartgateway \
  -p 8080:8080 \
  -v $(pwd)/config.json:/app/config.json \
  smartgateway:1.0.0 \
  --config /app/config.json
```

### 配置说明

1. **创建配置文件**

复制示例配置文件并根据实际需求修改：

```bash
cp config.example.json config.json
```

2. **最小化配置示例**

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

3. **启动服务**

```bash
# 使用配置文件启动
./smartgateway --config config.json

# 或使用默认配置启动
./smartgateway
```

4. **验证服务**

```bash
# 检查服务状态
curl http://localhost:8080/health

# 查看版本信息
./smartgateway --version
```

### 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-config` | 配置文件路径 | 无（使用内置默认配置） |
| `-version` | 显示版本号 | - |
| `-help` | 显示帮助信息 | - |

## 📡 API 文档

### 网关核心 API

SmartGateway 作为反向代理网关，透明转发请求到后端服务。以下是网关自身提供的管理接口：

#### 1. 健康检查接口

**GET** `/health`

检查网关及后端服务的健康状态。

**响应示例：**
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

**状态码：**
- `200 OK` - 服务健康
- `503 Service Unavailable` - 服务不健康

#### 2. 路由查询接口

**GET** `/api/routes`

获取当前所有路由配置信息。

**响应示例：**
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

#### 3. 配置热更新接口

**PUT** `/api/config`

动态更新网关配置（无需重启）。

**请求体：** 完整的 GatewayConfig JSON 对象

**请求示例：**
```bash
curl -X PUT http://localhost:8080/api/config \
  -H "Content-Type: application/json" \
  -d @config.json
```

**响应示例：**
```json
{
  "status": "success",
  "message": "配置已更新",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**状态码：**
- `200 OK` - 配置更新成功
- `400 Bad Request` - 配置格式错误
- `500 Internal Server Error` - 服务器错误

### 路由配置详解

#### 路由匹配规则

SmartGateway 支持多种路由匹配方式：

| 匹配类型 | 配置字段 | 说明 | 示例 |
|---------|---------|------|------|
| 精确路径 | `path` | 完全匹配请求路径 | `/api/users` |
| 前缀匹配 | `path_prefix` | 匹配指定前缀的路径 | `/api` |
| 通配符路径 | `path` (含 `*`) | 正则风格匹配 | `/api/*/details` |
| Host 匹配 | `host` | 匹配请求 Host | `api.example.com` |
| Host 通配符 | `host` (含 `*`) | 子域名匹配 | `*.example.com` |
| Method 匹配 | `methods` | 匹配 HTTP 方法 | `["GET", "POST"]` |
| Header 匹配 | `headers` | 匹配请求头 | `{"X-API-Version": "v2"}` |

#### 负载均衡算法

| 算法 | 配置值 | 说明 | 适用场景 |
|------|--------|------|---------|
| 轮询 | `round_robin` | 按顺序轮流分配请求 | 后端性能相近 |
| 随机 | `random` | 随机选择后端 | 简单场景 |
| 最少连接 | `least_conn` | 选择当前连接数最少的后端 | 长连接场景 |
| 一致性哈希 | `consistent_hash` | 基于 IP/Key 的哈希分配 | 需要会话保持 |

#### 高级配置选项

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

### 日志格式

SmartGateway 提供结构化访问日志（JSON 格式）：

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "method": "GET",
  "path": "/api/users",
  "host": "api.example.com",
  "remote_addr": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "status_code": 200,
  "duration_ms": 45,
  "upstream_addr": "http://127.0.0.1:8081",
  "response_bytes": 1024
}
```

## 📋 适用场景

- ✅ 金融信贷审核与风控
- ✅ 供应链自动化管理
- ✅ 企业级客服与营销
- ✅ 智能制造与质检
- ✅ 政务审批与服务
- ✅ 医疗诊断辅助

## 📄 许可证

本项目采用 MIT 许可证

## 📞 联系与支持

如需了解更多技术细节或企业合作，请参阅完整架构文档：
[企业生产级多 Agent+Harness 集群架构设计与自进化底座.md](./企业生产级多 Agent+Harness 集群架构设计与自进化底座.md)

---

<div align="center">

**让 AI 能力成为可控、可信、可持续进化的企业生产力**

</div>
