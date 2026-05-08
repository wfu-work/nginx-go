# nginx-go

基于 `github.com/wfu-work/nav-common-go-lib` 的可视化 Nginx 管理后端。项目采用前后端分离：后端提供 Nginx 配置建模、配置生成、语法校验、发布回滚、进程控制、性能观测、日志查询和审计能力；前端负责可视化编辑、配置预览、操作确认和监控展示。

代码结构和开发方式优先参考本地项目：

```text
/Users/wfu/Documents/works/xiaoxi/code/aegis/aegis-go
```

## 目标

- 界面可视化配置 Nginx：站点、端口、域名、反向代理、负载均衡、HTTPS、静态目录、缓存、限流、访问控制、日志。
- 界面管理 Nginx 进程：启动、停止、重启、reload、配置测试、状态刷新。
- 配置安全发布：结构化配置、配置预览、语法校验、版本保存、diff、发布、回滚。
- 性能观测：连接数、请求量、状态码、带宽、上游状态、进程 CPU/内存、access/error 日志。
- 尽量复用第三方库和 Nginx 原生命令，不自己实现完整 Nginx 解析器、进程管理器或监控系统。

## 基础框架

后端统一采用 `github.com/wfu-work/nav-common-go-lib`，不要重新搭一套通用框架。

优先复用能力：

- `commonInits.SysInit`：统一初始化数据库、路由、定时任务、清理任务。
- `global.NAV_DB`：数据库访问。
- `global.NAV_LOG`：日志。
- `response.Ok` / `response.FailWithMessage`：统一响应格式。
- `commonDomains.BaseDataEntity`：领域模型基础字段。
- `commonServices.CrudService[T]`：常规 CRUD 能复用就复用。
- `middlewares.ApiLogger()`：关键操作路由记录 API 日志。
- `scheduleds`：定时采集 Nginx 指标、清理历史数据。
- `commonUtils.ToPageInfo`、`commonDomains.PageResult`：分页查询。

保留第三方库的使用，但框架层能力优先来自 `nav-common-go-lib`。

## 推荐依赖

核心依赖：

- `github.com/wfu-work/nav-common-go-lib`
- `github.com/gin-gonic/gin`
- `github.com/robfig/cron/v3`
- `go.uber.org/zap`
- `gorm.io/gorm`

Nginx 与系统能力：

- 配置模板：Go 标准库 `text/template`
- 配置 diff：`github.com/sergi/go-diff`
- 文件监听：`github.com/fsnotify/fsnotify`
- 系统进程指标：`github.com/shirou/gopsutil/v4`
- Prometheus 指标：`github.com/prometheus/client_golang`
- Docker 管理：`github.com/docker/docker/client`，仅在容器部署模式启用。

不要自己实现：

- 完整 Nginx 语法解析器：配置校验交给 `nginx -t`。
- 复杂进程管理：本机模式调用 systemd / nginx 命令，容器模式调用 Docker API。
- 指标长期存储系统：后续接 Prometheus，当前只做短期聚合和展示。

## 目录规划

对齐 `aegis-go` 的分层风格：

```text
.
├── apis/                 # Gin Handler，负责参数绑定和响应
├── domains/              # GORM 模型和业务实体
├── inits/                # SysInit 注册表、路由、定时任务
├── routers/              # 路由分组
├── services/             # 业务逻辑
├── templates/nginx/      # Nginx 配置模板
├── utils/                # 命令执行、文件操作、日志解析等工具
├── config.yaml           # 本地配置
├── go.mod
└── main.go
```

入口保持简单：

```go
package main

import "nginx-go/inits"

func main() {
	inits.Init()
}
```

初始化方式参考：

```go
func Init() {
	sysInit := commonInits.SysInit{}
	sysInit.OnTableInit(registerTables)
	sysInit.OnRouterInit(func(publicGroup *gin.RouterGroup, privateGroup *gin.RouterGroup) {
		routers.RouterGroupApp.InitSiteRouter(privateGroup)
		routers.RouterGroupApp.InitUpstreamRouter(privateGroup)
		routers.RouterGroupApp.InitConfigRouter(privateGroup)
		routers.RouterGroupApp.InitNginxRouter(privateGroup)
		routers.RouterGroupApp.InitMetricRouter(privateGroup)
		routers.RouterGroupApp.InitLogRouter(privateGroup)
	})
	sysInit.OnScheInit(registerSchedules)
	sysInit.Init()
}
```

## 核心领域模型

所有业务表默认嵌入 `commonDomains.BaseDataEntity`，保持和 `aegis-go` 一致。

建议模型：

- `NginxInstance`：Nginx 实例，本机、远程、Docker 容器等。
- `Site`：站点配置。
- `ServerBlock`：server 块。
- `LocationRule`：location 规则。
- `Upstream`：上游服务组。
- `UpstreamServer`：上游节点。
- `Certificate`：证书配置。
- `ConfigVersion`：配置版本。
- `PublishTask`：发布任务。
- `NginxOperation`：启动、停止、重启、reload、test 等操作记录。
- `MetricSample`：短期指标采样。
- `AccessLogRecord`：access log 解析结果，可按需落库。
- `ErrorLogRecord`：error log 解析结果，可按需落库。
- `AuditLog`：审计日志。
- `Setting`：系统配置。

初期建议把复杂 Nginx 配置存为 JSON 字段，加快开发；稳定后再拆细表。

## Nginx 操作能力

界面需要支持以下操作，后端统一封装到 `services.NginxService`。

| 操作 | 说明 | 建议实现 |
| --- | --- | --- |
| `status` | 查看运行状态 | `systemctl is-active nginx`、`nginx -V`、进程检查、Docker 状态 |
| `refresh` | 刷新状态和指标 | 重新采集进程、stub_status、配置文件时间戳 |
| `test` | 测试配置 | `nginx -t -c <config>` |
| `reload` | 平滑重载配置 | 发布前必须先 `test`，再 `nginx -s reload` 或 `systemctl reload nginx` |
| `restart` | 重启 Nginx | `systemctl restart nginx` 或 Docker restart |
| `start` | 启动 Nginx | `systemctl start nginx` 或 Docker start |
| `stop` | 停止 Nginx | `systemctl stop nginx` 或 Docker stop |

安全边界：

- 所有进程操作必须走后端白名单，不允许前端传任意 shell 命令。
- `reload`、`restart`、`stop` 属于高危操作，必须记录 `NginxOperation` 和 `AuditLog`。
- `reload` 前强制执行 `nginx -t`，失败则拒绝执行。
- `stop` 和 `restart` 建议要求二次确认，前端传 `confirm=true`。
- 命令执行需要超时控制，避免接口阻塞。
- 命令输出只返回必要摘要，完整输出落库或写日志。

## 配置发布流程

主流程不让前端直接编辑完整 Nginx 文本。前端编辑结构化配置，后端负责生成、校验和发布。

1. 前端提交站点、server、location、upstream 等结构化数据。
2. 后端保存草稿。
3. 后端使用 `text/template` 生成 Nginx 配置预览。
4. 后端写入临时目录。
5. 执行 `nginx -t -c <temp-config>`。
6. 校验通过后生成 `ConfigVersion`。
7. 发布时将配置写入目标目录。
8. 执行 reload。
9. 写入 `PublishTask`、`NginxOperation`、`AuditLog`。
10. 失败时返回错误并保留上一版配置。

关键原则：

- 任何发布必须可回滚。
- 写配置使用临时文件 + 原子替换。
- 配置文件目录、Nginx binary、systemctl、Docker container id 都来自后端配置，不允许前端指定。
- 高级模式可以保存手写配置，但仍必须经过 `nginx -t`。

## API 规划

接口风格参考 `aegis-go`：`routers` 负责分组，`apis` 负责 Handler，`services` 执行业务，响应使用 `response` 包。

### Nginx 实例

```text
GET    /api/v1/nginx/instances/list
POST   /api/v1/nginx/instances
GET    /api/v1/nginx/instances/:guid
PUT    /api/v1/nginx/instances/:guid
DELETE /api/v1/nginx/instances/:guid
```

### Nginx 进程操作

```text
GET    /api/v1/nginx/status
POST   /api/v1/nginx/refresh
POST   /api/v1/nginx/test
POST   /api/v1/nginx/reload
POST   /api/v1/nginx/restart
POST   /api/v1/nginx/start
POST   /api/v1/nginx/stop
GET    /api/v1/nginx/operations/list
GET    /api/v1/nginx/operations/:guid
```

请求示例：

```json
{
  "instanceGuid": "default",
  "confirm": true,
  "reason": "publish new reverse proxy config"
}
```

响应示例：

```json
{
  "operationGuid": "op_xxx",
  "action": "reload",
  "success": true,
  "message": "nginx reload success",
  "durationMs": 328
}
```

### 站点配置

```text
GET    /api/v1/sites/list
POST   /api/v1/sites
GET    /api/v1/sites/:guid
PUT    /api/v1/sites/:guid
DELETE /api/v1/sites/:guid
POST   /api/v1/sites/:guid/enable
POST   /api/v1/sites/:guid/disable
```

### 上游服务

```text
GET    /api/v1/upstreams/list
POST   /api/v1/upstreams
GET    /api/v1/upstreams/:guid
PUT    /api/v1/upstreams/:guid
DELETE /api/v1/upstreams/:guid
GET    /api/v1/upstreams/:guid/health
```

### 配置预览、校验、发布

```text
POST   /api/v1/configs/render
POST   /api/v1/configs/validate
GET    /api/v1/configs/diff
POST   /api/v1/configs/publish
POST   /api/v1/configs/rollback
GET    /api/v1/configs/versions/list
GET    /api/v1/configs/versions/:guid
```

### 性能与日志

```text
GET    /api/v1/metrics/summary
GET    /api/v1/metrics/nginx
GET    /api/v1/metrics/process
GET    /api/v1/logs/access/list
GET    /api/v1/logs/error/list
GET    /api/v1/events/stream
```

`/api/v1/events/stream` 初期建议用 SSE，推送运行状态、发布状态、5xx 告警和上游健康变化。

## Service 规划

建议先实现这些服务：

- `SiteService`：站点 CRUD，优先复用 `commonServices.CrudService[domains.Site]`。
- `UpstreamService`：上游 CRUD 和健康检查。
- `ConfigService`：配置渲染、diff、版本管理。
- `PublishService`：配置发布、回滚、发布任务记录。
- `NginxService`：status、refresh、test、reload、restart、start、stop。
- `MetricService`：stub_status、进程指标、短期指标聚合。
- `LogService`：access/error 日志读取、过滤、解析。
- `AuditService`：关键行为审计。
- `SettingService`：Nginx 路径、日志路径、stub_status URL、运行模式等配置。

## 路由规划

```go
type RouterGroup struct {
	SiteRouter
	UpstreamRouter
	ConfigRouter
	NginxRouter
	MetricRouter
	LogRouter
	SettingRouter
}

var RouterGroupApp = new(RouterGroup)
```

高危操作路由必须加 `middlewares.ApiLogger()`：

```go
func (s *NginxRouter) InitNginxRouter(router *gin.RouterGroup) {
	groupLogger := router.Group("nginx").Use(middlewares.ApiLogger())
	group := router.Group("nginx")

	group.GET("status", nginxApi.Status)
	group.POST("refresh", nginxApi.Refresh)

	groupLogger.POST("test", nginxApi.Test)
	groupLogger.POST("reload", nginxApi.Reload)
	groupLogger.POST("restart", nginxApi.Restart)
	groupLogger.POST("start", nginxApi.Start)
	groupLogger.POST("stop", nginxApi.Stop)
	group.GET("operations/list", nginxApi.OperationList)
	group.GET("operations/:guid", nginxApi.OperationGet)
}
```

## 配置文件规划

```yaml
server:
  port: 8080

nginx:
  mode: "systemd" # systemd | command | docker
  default_instance: "default"
  bin: "/usr/sbin/nginx"
  config_dir: "/etc/nginx"
  conf_d_dir: "/etc/nginx/conf.d"
  main_config: "/etc/nginx/nginx.conf"
  temp_dir: "/tmp/nginx-go"
  access_log: "/var/log/nginx/access.log"
  error_log: "/var/log/nginx/error.log"
  stub_status_url: "http://127.0.0.1/nginx_status"
  command_timeout_seconds: 10
  systemctl: "/bin/systemctl"
  docker_container: ""

metrics:
  collect_interval_seconds: 10
  retention_hours: 72

security:
  require_confirm_for:
    - "restart"
    - "stop"
    - "rollback"
```

## 定时任务

使用 `SysInit.OnScheInit` 注册：

- 每 10 秒采集 Nginx status、进程 CPU/内存、stub_status。
- 每 30 秒检查 upstream 健康。
- 每 1 分钟扫描 error log 关键错误。
- 每 1 小时清理超过保留周期的 `MetricSample` 和操作日志。

## 开发阶段

### Phase 1：框架骨架

- 引入 `github.com/wfu-work/nav-common-go-lib`。
- 改造 `main.go` 为 `inits.Init()`。
- 建立 `apis/domains/inits/routers/services/utils` 目录。
- 注册健康检查、表迁移、路由。

### Phase 2：Nginx 操作

- 实现 `NginxService.Status`、`Refresh`、`Test`。
- 实现 `Reload`、`Restart`、`Start`、`Stop`。
- 增加 `NginxOperation` 和审计记录。
- 所有命令执行加白名单、超时和日志。

### Phase 3：配置建模

- 实现 `Site`、`ServerBlock`、`LocationRule`、`Upstream` CRUD。
- 使用 `text/template` 生成配置。
- 实现配置预览和 `nginx -t` 校验。

### Phase 4：发布回滚

- 实现 `ConfigVersion`。
- 实现 diff、publish、rollback。
- 发布成功后自动 reload。
- 发布失败保留错误输出。

### Phase 5：性能观测

- 接入 `stub_status`。
- 接入 `gopsutil` 采集进程指标。
- 实现日志查询和基础聚合。
- 增加 SSE 实时推送。

### Phase 6：生产增强

- 多实例管理。
- Docker 模式。
- Prometheus 集成。
- 操作审批。
- 更细的 RBAC 权限。

## 设计原则

- `nav-common-go-lib` 有的能力优先复用，不重复写框架代码。
- 常规 CRUD 优先用 `CrudService[T]`，复杂业务再单独写 Service。
- Nginx 配置主流程使用结构化模型，文本编辑只作为高级模式。
- Nginx 语法校验必须调用 `nginx -t`。
- 界面上的启动、停止、重启、reload 必须经过后端白名单和审计。
- 高危操作必须可追踪、可诊断，发布必须可回滚。
