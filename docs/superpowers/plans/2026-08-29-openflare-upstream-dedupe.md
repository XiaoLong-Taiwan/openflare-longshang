# server 插件复用上游能力清理计划

**目标**：`backend/OpenFlare/plugins/server` 里大量能力上游 Wavelet 已提供，删除本地副本改为复用；
按规约判定归属——通用能力回流上游，OpenFlare 业务留在插件内。

**事实基线**：`server` 实测 **68,117** 行非测试 Go 代码（早先记的 29.6k 只是 `openflare/` 一个域）。
路由契约冻结基准：`docs/superpowers/specs/baseline/routes-engine.txt`（256 条 方法+路径）与
`routes.txt`（232 条 swagger 操作），由 `plugin_parity_test.go` 逐条把守。

## 关键防线（先读，否则会做错）

直接挂上游插件**会破坏契约**，以下差异必须逐项处理，不允许“上游有就换”：

1. **验证码门禁丢失**：OF 在 `POST /api/v1/user/{login,register,send-email-code}` 上挂了
   `cap.VerifyMiddleware`（`router/v1/user.go:71-73`），上游 `domain/user` 没有该门禁
   （`user/plugin.go:126-129`）。直接替换=安全回归。
2. **路径变化**：`/api/cap/{challenge,redeem}`（OF，挂在 `/api` 组）vs `/api/v1/cap/*`（上游，且多一个 `GET`）；
   `GET /api/health` vs `/healthz` + `/api/healthz`；`/api/v1/upload` 的 `GET /my`、`PUT /:id`、
   `GET /download/:id`、`POST /download/batch` vs 上游 `GET ""`、`POST /batch-download`；
   OF 独有 `GET /api/v1/user/self`。
3. **公共配置响应体不同**：`GET /api/v1/config/public` 路径相同但 JSON 字段不同，SPA 直接消费。
4. 上游额外需要表 `w_message_channels` / `w_message_bindings` / `w_message_pairing_codes`（OF 库中不存在）。
   除此之外 13 张共享 `w_*` 表**列级零差异**；唯一缺口 `w_access_tokens.description/expires_at` 上游 Go 模型也未使用，非阻塞。

## 判定清单

| 包 | 行数 | 上游对应 | 判定 |
| :-- | --: | :-- | :-- |
| `shared/response` | 142 | `pkg/response` | 删（7/7 `Abort*` 逐字一致，仅风格差异） |
| `pkg/logger` | 382 | `pkg/logger` | 删（`Config` 字段完全一致） |
| `pkg/mail` | 266 | `pkg/mail` | 删 |
| `pkg/httppool` | 106 | `pkg/httppool` | 删（逐字节相同） |
| `pkg/trace` | 148 | `pkg/trace` | 删（差异仅默认 tracer 名，由 `app.app_name`/`otel.tracer_name` 决定） |
| `pkg/cache/ram` | 305 | `pkg/cache/ram` | 删（OF 用裸 `go`，上游用 `util.Go`，换过去是修复） |
| `infra/persistence/idgen` | 47 | `pkg/idgen` | 删（上游为超集） |
| `infra/persistence/batchwriter` | 337 | `pkg/batchwriter` | 删 |
| `listener` | 42 | `core/events.go Subscribe[T]` | 删，改事件订阅 |
| `testhelper` | 479 | `pkg/testhelper` | 合并，仅留 `SetupLogStoresForTest` |
| `pkg/cache/disk` | 412 | `pkg/cache/disk` | 先回流 OF 的 5 处类型断言 `ok` 守卫（通用健壮性修复），再删本地副本 |
| `pkg/cap` | 511 | `plugins/domain/cap/pow` | **不能直接复用**（跨插件 import 被禁）。二选一：pow 提取到上游 `pkg/pow`，或留 `server` 内并放弃“复用” |
| `infra/objectstore`、`infra/diskcache` | 1,058 | `plugins/infra/storage/*` | 改走 `contracts.StorageService`/上游 infra（P6） |
| `infra/task`（worker/scheduler/handlers） | 1,125 | `plugins/drivers/driver_asynq_*` | 换内核驱动（P6） |
| `infra/config`（全局 `Config` 单例） | 444 | `core/config.go` + `plugins/infra/config` | 改 `ctx.Config().Bind`（P6，18 处引用） |
| `infra/persistence`（DB/Redis 初始化） | 639 | `plugins/infra/database` | 换 `contracts.DBService`（P6） |
| `infra/persistence/migrator` | 298 | `w_schema_versions` + 每插件迁移 | 见 foundation 计划 P3（含历史 stamp 桥接） |
| `repository/logstore` | 3,286 | `plugins/domain/risk_control/logstore` | 近似分叉：合并并保留 OF 的 `AccessLogStore`/`ObservabilityStore` 接口 |
| `repository/analytics` | 2,848 | 同上（仅用户访问日志半边） | 合并半边；`node_*` 与 CH 维护上游没有，留 `server` |
| `repository`、`model` | 8,527 | 各上游域插件内 | 拆分：`w_*` 部分随 P6 归上游，`of_*` 留 `server` |
| `oauth`、`user`、`admin`、`upload`、`cap`、`config`、`health` | ~7,600 | `plugins/domain/{auth,user,admin,upload,cap,system}` | **受“关键防线”约束**，逐端点判定后合并，禁止整体替换 |
| `admin/push` | 1,776 | `plugins/domain/message_gateway` | 合并并保留 OF 的 `RegisterBuiltInEvent`/`RegisterChannelDefinition`/`SyncEvents` 钩子 |
| `admin/updater` | 832 | 上游仅 `GetUpdateStatus/ApplyUpdate` | OF 的 Prepare/Apply/Finish 三段式为业务能力，保留 |
| `openflare/`（27 子包） | 29,678 | 无 | 保留（产品逻辑）；`apiutil` 与 `integration/githubrelease` 若被多插件消费再移 `share` |
| `router`、`router/root`、`router/v1` | 1,242 | `core/extpoints` + `driver_http` | 保留为接线层，直至上游具备引擎中间件与 NoRoute 贡献点 |

## 执行顺序

1. **T1 纯风格差异**（`shared/response`、`pkg/{logger,mail,httppool,trace,cache/ram}`、
   `infra/persistence/{idgen,batchwriter}`）：改 import → 删本地副本 → 全量门禁。
2. **T2 `listener` 改事件订阅**、`testhelper` 合并。
3. **T3 `pkg/cache/disk` 断言守卫回流上游**，再删本地副本。
4. **T4 基础设施换内核**（`infra/config` → `ctx.Config()`；`infra/persistence` → `contracts.DBService`；
   `infra/task` → asynq 驱动）：与 P3/P4b 合并推进。
5. **T5 平台域逐个合并**（oauth/user/admin/upload/cap/config/health/push）：每个域先列差异
   （路由、门禁、响应体、表列），差异以 `server` 内薄封装或回流上游解决，逐域过 256 条对拍。
6. **T6 `repository`/`model` 按表所有权拆分**：`w_*` 归上游，`of_*` 留 `server`。

每步完成判据：`go build ./...` + `go test ./...` 全绿 + `plugin_parity_test` 零差异 +
`make swagger` 232 条零差异 + `golangci-lint` 0 issues + `frontend/` 零改动。

## 执行记录与阻塞点

### T1 已完成（提交 `0adbfc1a`）

删除并改用上游：`shared/response`→`pkg/response`、`pkg/{logger,mail,trace,httppool,cache/ram}`、
`infra/persistence/batchwriter`→`pkg/batchwriter`。共 7 个本地包消失，`go test` 包数 144→138。

证据：256 条路由对拍零差异；232 条 swagger 操作零增减，且**把两个随包路径改名的定义
（`…shared_response.Any`→`response.Any`、`…pkg_logger.LogEntry`→`logger.LogEntry`）归一化后，
新旧 swagger.json 深度相等**——即接口形状完全未变。`$ref` 名由 Go 包路径派生，属生成物命名变化。

### 新发现的阻塞点（勿重复踩）

1. **`pkg/idgen` 暂不能换**：上游 `pkg/idgen` 要求先 `Init(nodeID)` 否则 panic
   （`idgen: Init must be called before generating IDs`），而 OpenFlare 副本是懒初始化。
   5 个 push/user 用例直接 panic。必须与 T4（`infra/config`、`infra/persistence` 改走
   `ctx.Config()`/`contracts.DBService`）一起做，在 bootstrap 阶段显式 `Init`。
2. **`pkg/cap` 不能“复用上游”**：上游 pow 实现在 `plugins/domain/cap/pow` 里，
   跨插件 import 被架构门禁禁止。合规出路二选一：把 pow 提取为上游 `pkg/pow`（通用能力，
   按规约回流 Wavelet 后两边共用），或承认它是 server 的业务实现留在插件内。
3. **`listener` 需先有内核上下文**：`core.Subscribe[T](bus, topic, fn)` 要 `*core.EventBus` 实例，
   而调用方（`oauth/handler_callback.go`、`user/routers.go`、`admin/push/custom_events/*`）都在
   内核之外、手上没有 `*core.Context`。现在改只会把全局变量从包级切片挪到包级总线。
   待 `server` 的 `Apply`/bootstrap 拿到 ctx 后随 T4 一起做。
4. **`pkg/trace` 覆盖缺口（非本次引入）**：核实后上游与 OpenFlare 副本都是 4 个同名文件、
   都**没有**测试文件，因此删除本地副本没有降低覆盖率。要补测试应直接补在上游 `backend/pkg/trace/`。
5. **`pkg/cache/disk` 已完成**：5 处类型断言的 `ok` 守卫按规约回流上游
   （Wavelet `f3d85d5`，附 `cache_corruption_test.go`：去掉守卫即 panic、加上守卫 4 用例全过），
   上游的 `Default()` 全局实例保留（下游不使用它，不构成行为差异）；本地副本已删除。
