# Nginx Control Backend

一个基于 Go + Gin 的 Nginx 管理后端服务，面向私有化部署场景，提供 Nginx 实例管理、配置建模、配置渲染与发布、运行时操作、指标采集、日志解析、审计记录和事件通知能力。

项目目标不是把浏览器变成远程 Shell，而是提供一层可审计、可回滚、受约束的 Nginx 运维 API，让前端控制台能够安全地完成常见管理动作。

## 功能特性

- **实例管理**：支持命令行、systemd、Docker 三种 Nginx 运行模式。
- **多节点管理**：支持服务器节点建模，远程节点可通过 Agent 接入中心端。
- **运行时操作**：支持状态查询、配置测试、reload、start、restart、stop，并记录操作历史。
- **结构化配置**：管理站点、location 规则、upstream、upstream server、HTTPS 证书等对象。
- **配置生命周期**：支持配置渲染、语法校验、差异对比、发布、回滚、版本历史和发布任务记录。
- **观测能力**：采集进程状态、`stub_status`、upstream 健康状态和短期指标样本。
- **日志与审计**：读取 access/error log，解析结构化日志，记录敏感操作审计日志。
- **事件通知**：支持通知列表、已读/全部已读、SSE 和 WebSocket 推送。
- **运行时设置**：提供可扩展的 key-value 设置接口，用于业务运行参数维护。

## 技术栈

- Go `1.26+`
- Gin
- GORM
- SQLite 默认存储
- robfig/cron 定时任务
- zap 日志
- `github.com/wfu-work/nav-common-go-lib` 公共启动、数据库、响应、日志与 CRUD 能力

## 目录结构

```text
.
├── apis/        # Gin Handler：参数绑定、响应封装
├── domains/     # GORM 模型与领域实体
├── inits/       # 应用初始化、自动迁移、路由注册、定时任务
├── routers/     # API 路由分组
├── services/    # 核心业务逻辑
├── utils/       # 命令执行等工具函数
├── config.yaml  # 本地运行配置
├── go.mod
└── main.go
```

## 快速开始

### 环境要求

- Go `1.26+`
- 本机已安装 Nginx，或通过 systemd/Docker 管理 Nginx
- 如需指标采集，请在 Nginx 中开启 `stub_status`

### 启动服务

```bash
go run .
```

默认读取当前目录下的 `config.yaml`，监听端口为 `3007`，接口前缀为 `/api`。

### 构建

```bash
go build ./...
```

### 测试

```bash
go test ./...
```

## 配置说明

核心配置位于 [config.yaml](./config.yaml)。

```yaml
system:
  app-name: "nginx-go"
  addr: 3007
  db-type: sqlite
  router-prefix: /api

sqlite:
  db-name: nginx-go
  path: ./data/

nginx:
  mode: command
  service-name: nginx
  bin: nginx
  systemctl: systemctl
  docker-bin: docker
  main-config: ""
  managed-config: "./data/nginx/generated.conf"
  temp-dir: "./data/nginx/tmp"
  backup-dir: "./data/nginx/backups"
  access-log: "/var/log/nginx/access.log"
  error-log: "/var/log/nginx/error.log"
  stub-status-url: ""
  command-timeout-seconds: 10
  require-confirm-actions:
    - restart
    - stop
    - rollback

metrics:
  collect-interval-seconds: 30
  retention-hours: 72

agent:
  shared-token: ""
```

### Nginx 运行模式

- `command`：直接执行 `nginx` 命令，适合本机管理。
- `systemd`：通过 `systemctl` 管理服务，适合 Linux 服务化部署。
- `docker`：通过 Docker CLI 操作容器内 Nginx。

## 多服务器与 Agent 模式

单机模式下，后台服务会直接读取本机 Nginx 配置、日志和进程信息。如果需要管理多台服务器，推荐使用 **中心端 + Agent** 模式：

```text
nginx-web
   |
nginx-go 中心端
   |
   | HTTPS / 内网 API
   |
多台 nginx-agent
   |
本机 nginx / 配置文件 / access.log / error.log / stub_status
```

中心端新增了两个核心资源：

- `nodes`：服务器节点，描述节点名称、Agent ID、在线状态、标签和接入模式。
- `agent tasks`：中心端下发给 Agent 的任务队列，Agent 主动轮询、执行并回传结果。

推荐流程：

1. 在中心端配置 `agent.shared-token`。
2. 在每台 Nginx 服务器部署 Agent，并使用相同 token 注册到中心端。
3. 在中心端创建或更新 Nginx 实例，将 `nodeGuid` 绑定到对应节点。
4. 后续状态查询、reload、日志读取、指标采集和配置发布会自动转发给目标 Agent。

### Agent 接入接口

以下接口用于 Agent 与中心端通信。若配置了 `agent.shared-token`，Agent 需要在请求头中携带：

```text
X-Agent-Token: your-token
```

```text
POST /api/agent/register
POST /api/agent/heartbeat
GET  /api/agent/tasks/poll?nodeGuid=xxx
POST /api/agent/tasks/:guid/complete
```

本仓库内置了一个轻量 Agent 命令，可在 Nginx 所在服务器运行：

```bash
go run ./cmd/nginx-agent \
  -center http://127.0.0.1:3007/api \
  -token your-token \
  -agent-id prod-nginx-01 \
  -name 生产-Nginx-01
```

也可以使用环境变量：

```bash
NGINX_AGENT_CENTER=http://127.0.0.1:3007/api \
NGINX_AGENT_TOKEN=your-token \
NGINX_AGENT_ID=prod-nginx-01 \
NGINX_AGENT_NAME=生产-Nginx-01 \
go run ./cmd/nginx-agent
```

注册示例：

```json
{
  "agentId": "prod-nginx-01",
  "name": "生产 Nginx 01",
  "address": "10.0.0.12",
  "labels": "env=prod,region=cn",
  "version": "0.1.0"
}
```

任务完成示例：

```json
{
  "success": true,
  "response": {
    "success": true,
    "message": "nginx reload success",
    "output": "nginx: configuration file syntax is ok",
    "durationMs": 120
  }
}
```

当前中心端已支持通过 Agent 下发以下任务类型：

```text
nginx.status
nginx.operation
nginx.stub_status
nginx.process
nginx.log_tail
nginx.config_validate
nginx.config_publish
```

Agent 需要根据任务中的 `runtime` 字段在本机执行白名单动作，并把结构化结果回传给中心端。

### 发布配置前的准备

如果使用配置发布功能，请确保 `nginx.managed-config` 对应文件会被真实 Nginx 主配置引用。例如：

```nginx
http {
    include /path/to/generated.conf;
}
```

发布流程会先执行配置校验，再写入托管配置文件并 reload Nginx。reload 失败时，会尝试恢复发布前的配置。

## API 概览

以下路径默认带有 `/api` 前缀。

### Nginx 运行时

```text
GET    /api/nginx/status
POST   /api/nginx/refresh
POST   /api/nginx/test
POST   /api/nginx/reload
POST   /api/nginx/restart
POST   /api/nginx/start
POST   /api/nginx/stop
GET    /api/nginx/operations/list
GET    /api/nginx/operations/:guid
```

`restart`、`stop`、`rollback` 等高风险操作默认需要显式确认：

```json
{
  "instanceGuid": "default",
  "confirm": true,
  "reason": "发布反向代理配置"
}
```

### 实例

```text
GET    /api/nginx/instances/list
POST   /api/nginx/instances
GET    /api/nginx/instances/:guid
PUT    /api/nginx/instances/:guid
DELETE /api/nginx/instances/:guid
```

实例支持通过 `nodeGuid` 绑定远程节点。未绑定节点时保持本机模式；绑定 `accessMode=agent` 的节点时，中心端会通过 Agent 执行对应操作。

### 节点

```text
GET    /api/nodes/list
POST   /api/nodes
GET    /api/nodes/:guid
PUT    /api/nodes/:guid
DELETE /api/nodes/:guid
```

### 站点与 Location

```text
GET    /api/sites/list
POST   /api/sites
GET    /api/sites/:guid
PUT    /api/sites/:guid
DELETE /api/sites/:guid
POST   /api/sites/:guid/enable
POST   /api/sites/:guid/disable
POST   /api/sites/:guid/locations
PUT    /api/sites/:guid/locations/:locationGuid
DELETE /api/sites/:guid/locations/:locationGuid
```

### Upstream

```text
GET    /api/upstreams/list
POST   /api/upstreams
GET    /api/upstreams/:guid
PUT    /api/upstreams/:guid
DELETE /api/upstreams/:guid
GET    /api/upstreams/:guid/health
POST   /api/upstreams/:guid/servers
PUT    /api/upstreams/:guid/servers/:serverGuid
DELETE /api/upstreams/:guid/servers/:serverGuid
```

### 证书

```text
GET    /api/certificates/list
POST   /api/certificates
GET    /api/certificates/:guid
PUT    /api/certificates/:guid
DELETE /api/certificates/:guid
```

### 配置

```text
POST   /api/configs/render
POST   /api/configs/validate
GET    /api/configs/diff
POST   /api/configs/diff
POST   /api/configs/publish
POST   /api/configs/rollback
GET    /api/configs/versions/list
GET    /api/configs/versions/:guid
GET    /api/configs/tasks/list
```

### 指标与日志

```text
GET    /api/metrics/summary
GET    /api/metrics/nginx
GET    /api/metrics/stub-status
GET    /api/metrics/process
GET    /api/metrics/samples/list
GET    /api/logs/access
GET    /api/logs/access/list
GET    /api/logs/access/records
GET    /api/logs/error
GET    /api/logs/error/list
GET    /api/logs/error/records
POST   /api/logs/sync
GET    /api/logs/audit/list
```

### 事件通知

```text
GET    /api/events/stream
GET    /api/events/ws
GET    /api/ws
GET    /api/events/notifications/list
POST   /api/events/notifications/:guid/read
POST   /api/events/notifications/read-all
```

当前通知来源包括 Nginx 操作、配置发布/回滚、upstream 健康异常和 error log 告警等场景。

### 设置

```text
GET    /api/settings/list
POST   /api/settings
DELETE /api/settings/:guid
```

## 安全设计

本项目会操作真实 Nginx 进程和配置文件，因此默认采用相对保守的执行模型：

- 不提供任意 Shell 命令执行接口。
- 浏览器请求不能直接传入任意命令，只能触发白名单动作。
- 远程节点由 Agent 主动轮询任务，中心端不需要直接访问各服务器 SSH 或开放端口。
- 绑定到远程节点的实例，如果节点不存在或被禁用，会直接报错，不会回退到本机执行。
- 命令路径来自服务端配置，例如 `nginx.bin`、`nginx.systemctl`、`nginx.docker-bin`。
- 命令执行带有超时时间，默认 `10s`。
- `reload` 前会先执行 `nginx -t`。
- 发布配置前会备份目标文件，失败时尽量恢复。
- 高风险动作需要 `confirm: true`。
- 关键操作会写入操作历史、审计记录和事件通知。

生产环境建议：

- 使用最小权限账号运行服务。
- 明确限制 `managed-config`、`backup-dir`、`temp-dir` 的读写范围。
- 不要把接口直接暴露到公网。
- 在网关或上层系统增加认证、授权与访问审计。
- 发布前先在测试环境验证生成配置。

## 定时任务

服务启动时会注册以下后台任务：

- 定时采集 Nginx 指标。
- 定时清理过期指标样本。
- 定时检查 upstream 健康状态。
- 定时扫描 error log 关键错误。

指标采集周期由 `metrics.collect-interval-seconds` 控制，指标保留时间由 `metrics.retention-hours` 控制。

## 前端项目

配套前端位于同级目录 `nginx-web`，基于 Angular 实现，通过 `/api` 前缀访问本服务。

本地开发时通常为：

```text
nginx-go   -> http://127.0.0.1:3007/api
nginx-web  -> Angular dev server
```

## 开发指南

格式化代码：

```bash
gofmt -w apis domains inits routers services utils
```

构建：

```bash
go build ./...
```

测试：

```bash
go test ./...
```

推荐贡献流程：

1. 先通过 Issue 描述问题、需求或设计思路。
2. 保持接口兼容，非必要不破坏已有字段和路径。
3. 涉及核心服务逻辑时补充或更新测试。
4. 提交前执行格式化、构建和测试。
5. Pull Request 中说明变更范围、验证方式和潜在风险。

## 路线图

- 更细粒度的高风险操作 RBAC。
- Prometheus exporter 与更长周期指标存储。
- 更丰富的日志检索、聚合与告警规则。
- 证书自动续期集成。
- 多节点远程 Nginx 管理。
- 配置发布、restart、stop 的审批流。
- Docker API Client 模式，减少对 Docker CLI 的依赖。

## 许可证

当前仓库尚未包含明确的开源许可证文件。正式公开发布前，建议补充 `LICENSE`，可根据项目定位选择 MIT、Apache-2.0、AGPL-3.0 等许可证。

## 免责声明

本项目具备操作 Nginx 进程和写入配置文件的能力。请在生产环境使用前认真检查路径配置、系统权限、访问控制和回滚策略。因误配置、权限过大或未授权访问造成的问题，需要由部署方自行承担风险。
