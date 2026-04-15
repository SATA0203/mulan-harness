# SmartGateway 高性能智能网关系统

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-1.0.0-green.svg)]()
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)]()

## 📖 项目简介

SmartGateway 是一款**高性能、可扩展、云原生友好**的智能网关系统，作为所有流量的统一入口。

### 核心特性

- **高性能**: 单机支持 10 万 + 并发连接，平均延迟 < 5ms
- **高可用**: 支持集群部署，无单点故障，配置热更新不中断连接
- **易扩展**: 提供插件机制，支持用户自定义业务逻辑
- **可观测**: 内置完善的监控指标、访问日志和链路追踪集成

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────────────┐
│                    Client Requests                       │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                   SmartGateway                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Router    │  │   Load      │  │   Health    │     │
│  │  (路由匹配)  │  │  Balancer   │  │   Checker   │     │
│  │             │  │ (负载均衡)  │  │  (健康检查)  │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │    Rate     │  │  Circuit    │  │    Auth     │     │
│  │   Limit     │  │  Breaker    │  │   (鉴权)    │     │
│  │  (限流)     │  │  (熔断)     │  │             │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└───────────────────────┬─────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
┌───────▼───────┐ ┌─────▼─────┐ ┌──────▼──────┐
│  Service A    │ │ Service B │ │  Service C  │
│  (后端服务)    │ │ (后端服务) │ │  (后端服务)  │
└───────────────┘ └───────────┘ └─────────────┘
```

## 🚀 快速开始

### 安装

#### 从源码编译

```bash
git clone https://github.com/your-org/smartgateway.git
cd smartgateway
go build -o smartgateway ./cmd/main.go
```

#### 下载二进制文件

```bash
# Linux
wget https://github.com/your-org/smartgateway/releases/download/v1.0.0/smartgateway-linux-amd64
chmod +x smartgateway-linux-amd64

# macOS
wget https://github.com/your-org/smartgateway/releases/download/v1.0.0/smartgateway-darwin-amd64
chmod +x smartgateway-darwin-amd64
```

### 基本使用

#### 1. 使用默认配置启动

```bash
./smartgateway
```

网关将在 `:8080` 端口启动，使用默认配置。

#### 2. 使用配置文件启动

```bash
./smartgateway -config config.json
```

#### 3. 查看版本

```bash
./smartgateway -version
```

### 配置文件示例

创建 `config.json`:

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
          "address": "http://127.0.0.1:8081",
          "weight": 1
        },
        {
          "address": "http://127.0.0.1:8082",
          "weight": 1
        }
      ],
      "lb_algorithm": "round_robin",
      "timeout": "10s",
      "retries": 2
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
    "level": "info",
    "format": "json",
    "sample_rate": 1.0
  }
}
```

## 📋 功能特性

### 1. 核心路由与负载均衡

- **动态路由**: 支持基于域名 (Host)、路径 (Path)、Header 的匹配规则
- **负载均衡算法**:
  - 轮询 (Round Robin)
  - 随机 (Random)
  - 最小连接数 (Least Connections)
  - 一致性哈希 (Consistent Hash) - 开发中
- **健康检查**: 主动探测后端服务状态，自动剔除异常节点
- **超时与重试**: 可配置连接超时、读取超时；支持按错误码自动重试

### 2. 流量控制与安全 (开发中)

- **限流 (Rate Limiting)**: 支持基于 IP、用户 ID、API 维度的请求速率限制
- **熔断降级 (Circuit Breaking)**: 当后端错误率或延迟达到阈值，自动切断流量
- **身份鉴权**: 内置 JWT、OAuth2、Basic Auth 验证插件
- **黑白名单**: 支持基于 IP 段的访问控制

### 3. 可观测性

- **监控指标 (Metrics)**: 暴露 Prometheus 格式指标
- **访问日志 (Access Log)**: 结构化日志 (JSON)，包含请求/响应头、耗时、状态码
- **链路追踪 (Tracing)**: 集成 OpenTelemetry/Jaeger - 开发中

### 4. 配置管理

- **配置热加载**: 修改配置后秒级生效，无需重启进程
- **配置校验**: 提交配置前自动语法和逻辑校验
- **多环境支持**: 支持开发、测试、生产多套配置隔离

## 🔧 配置参数详解

### 服务器配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `server_addr` | string | `:8080` | 监听地址 |
| `read_timeout` | duration | `30s` | 读取超时时间 |
| `write_timeout` | duration | `30s` | 写入超时时间 |
| `idle_timeout` | duration | `60s` | 空闲连接超时时间 |

### 路由配置

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 路由名称 |
| `host` | string | 否 | 匹配的域名 |
| `path_prefix` | string | 否 | 匹配的路径前缀 |
| `path` | string | 否 | 精确匹配的路径 |
| `methods` | []string | 否 | 允许的 HTTP 方法 |
| `headers` | map | 否 | 需要匹配的 Header |
| `backends` | []Backend | 是 | 后端服务列表 |
| `lb_algorithm` | string | 否 | 负载均衡算法 |
| `timeout` | duration | 否 | 请求超时时间 |
| `retries` | int | 否 | 重试次数 |

### 健康检查配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `true` | 是否启用健康检查 |
| `interval` | duration | `10s` | 检查间隔 |
| `timeout` | duration | `5s` | 检查超时时间 |
| `unhealthy_threshold` | int | `3` | 判定为不健康的连续失败次数 |
| `healthy_threshold` | int | `2` | 判定为健康的连续成功次数 |
| `path` | string | `/health` | 健康检查路径 |

### 日志配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | `true` | 是否启用日志 |
| `level` | string | `info` | 日志级别 (debug/info/warn/error/fatal) |
| `format` | string | `json` | 日志格式 (json/text) |
| `sample_rate` | float | `1.0` | 采样率 (0.0-1.0) |

## 📊 性能指标

### 基准测试结果

在 4 核 8G 环境下：

| 场景 | QPS | P99 延迟 | CPU 使用率 |
|------|-----|---------|-----------|
| 纯转发 | 50,000+ | < 5ms | ~60% |
| 带路由匹配 | 45,000+ | < 8ms | ~70% |
| 带健康检查 | 40,000+ | < 10ms | ~75% |

## 🛠️ 开发指南

### 项目结构

```
smartgateway/
├── cmd/
│   └── main.go              # 主程序入口
├── pkg/
│   ├── config/              # 配置管理
│   ├── router/              # 路由匹配
│   ├── loadbalancer/        # 负载均衡
│   ├── health/              # 健康检查
│   ├── logging/             # 日志记录
│   ├── middleware/          # 中间件
│   └── server/              # HTTP 服务器
├── internal/                # 内部包
├── utils/                   # 工具函数
├── config.example.json      # 配置示例
├── go.mod                   # Go 模块定义
└── README.md                # 本文档
```

### 添加新插件

1. 在 `pkg/middleware` 目录下创建新的中间件文件
2. 实现 `MiddlewareFunc` 接口
3. 在配置文件中启用插件

示例：

```go
package middleware

import (
    "net/http"
    "smartgateway/pkg/logging"
)

func MyCustomMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logging.Info("Custom middleware called", map[string]interface{}{
            "path": r.URL.Path,
        })
        next.ServeHTTP(w, r)
    })
}
```

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 联系方式

- 项目主页：https://github.com/your-org/smartgateway
- 问题反馈：https://github.com/your-org/smartgateway/issues
- 邮件联系：support@smartgateway.io

---

<div align="center">

**让流量管理更简单、更高效**

Made with ❤️ by the SmartGateway Team

</div>
