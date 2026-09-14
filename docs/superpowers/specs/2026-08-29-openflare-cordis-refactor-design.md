# OpenFlare Cordis 架构改造设计

- **文档类型**: 架构设计 / 改造规范（Spec）
- **日期**: 2026-08-29
- **范围**: OpenFlare 后端全量迁移到 Wavelet Cordis 插件化架构
- **上游参照**: `/Users/ryan/Code/Go/Wavelet`（`backend/`，module `Wavelet`）
- **改造基线**: OpenFlare `main` @ `9f79fb99`（v3.5.4），工作分支 `cordis`

---

## 1. 目标与不可协商约束

| # | 约束 | 验证方式 |
| :-- | :-- | :-- |
| G1 | OpenFlare 改造为 Cordis 架构，上游包名为 `Wavelet` | `backend/Wavelet/` 与上游逐字节一致（`share/` 除外）；`go build ./...` 通过 |
| G2 | OpenFlare 功能收敛为 **4 个插件**：`server`、`agent`、`relay`、`flared` | `backend/plugins/` 下仅此 4 个目录，均实现 `core.Plugin` |
| G3 | 在上游树内创建 `share` 包承载跨插件共享资源 | `backend/Wavelet/share/` 存在且被 4 个插件至少两处 import |
| G4 | **不修改前端** | `git diff --name-only main...cordis -- frontend/` 为空 |
| G5 | 数据库迁移谨慎：已部署库不重跑历史、不丢数据、零破坏性变更 | 三方 schema 一致性门禁（§5.4） |
| G6 | 既有 HTTP 契约不变（前端与 API 消费者零感知） | 路由清单 diff + swagger 对拍 |
| G7 | 边缘三进制的 CLI 形态不变（`-config` 默认路径、退出码、日志格式） | 启动冒烟 + 现有测试保持绿 |

## 1.5 布局修正（v1.1，取代 §3 与 §4 中的路径描述）

初版设计提出的「`backend/` + `backend/Wavelet/` 双 module、OpenFlare 插件放 `backend/plugins/`」已被否定。
**采用与 Wavelet 完全同构的单 module 布局**（已落地，提交 `9bf0b2de`）：

```
backend/                              module Wavelet（与上游同名，上游 import 路径逐字一致）
├── main.go  cmd/                     控制面装配根（server）
├── core/                             【上游】微内核 + contracts + extpoints
├── pkg/                              【上游】通用库（禁止 import plugins）
├── plugins/{drivers,infra,domain}/   【上游】平台插件
├── docs/                             swaggo 产物（json/yaml 复制回根 docs/ 供站点消费）
├── share/                            【新增 G3】跨插件共享资源层
└── OpenFlare/                        【下游】占据上游 backend/downstream 的位置
    ├── cmd/{agent,relay,flared}/     三个 daemon 入口（package main）
    └── plugins/{server,agent,relay,flared}/   【G2】4 个业务插件
```

约束更新：
- 上游同步 = 覆盖 `backend/{core,pkg,plugins}`，零 import 改写；`OpenFlare/` 与 `share/` 永不被覆盖。
- OpenFlare 现有代码（apps/repository/model/infra/router/shared/…）暂整体驻留 `backend/OpenFlare/**`，由 P4/P5 逐个搬入 `backend/OpenFlare/plugins/<插件>`；搬完后 `backend/OpenFlare` 下不再保留平铺的遗留层。
- 装配根在 `backend/cmd`（模块内非 internal 路径），因此下游树不得使用 `internal/`（已在本次落地时展平）。
- 平台表由上游 `plugins/domain/*` 拥有，业务表 `of_*` 由 `OpenFlare/plugins/server` 拥有；§5 迁移方案不变，仅历史链落点改为 `backend/OpenFlare/plugins/server/migrations/{postgres,sqlite}`。

落地验证（实测，非推断）：`go build ./...` 通过；`go test ./...` exit 0 且 142 个包 ok（含上游插件测试）；`make swagger` exit 0 且 232 条 API 操作与改造前基线逐条一致；`make build-all` 产出 4 个二进制；`git status --short -- frontend/` 为空。
上游 `plugins`、`core` 暂从 swaggo 扫描中排除（其包名 `model`、`disk` 与下游遗留注解的裸名引用冲突），P4 挂载上游路由时随 `OpenFlare/model` 展平一并解除。

## 2. 已核实的事实基线（设计前提，非推测）

1. **OpenFlare 是 Cordis 之前的 Wavelet 分叉**：`go.mod` module 路径即 `github.com/Rain-kl/Wavelet`；`internal/apps/` 同时含继承自 Wavelet 的平台应用（admin/oauth/user/upload/cap/config/health）与 OpenFlare 自有业务（openflare 29.6k 行、agent 7.4k、edge 1.5k、relay 1.0k、flared 0.9k）。Go 代码总量 151,473 行，209 个测试文件。
2. **表前缀双轨**：平台表沿用 `w_*`（OpenFlare 的 76 迁移已在其上分叉），业务表 `of_*`（28 张）。上游 `w_*` 表宇宙 17 张，且上游含 `w_message_channels`/`w_message_bindings`/`w_message_pairing_codes`（OpenFlare 无）→ **双向分叉**。
3. **内核支持非 HTTP 形态**（`Wavelet/backend/core`）：
   - `normalizeProfile`（app.go:684）对未知 profile 字符串原样透传；`matchesProfile`（app.go:668）`default` 分支做字符串相等比较 → `core.Profile("agent")` 仅启动 `DriverType("agent")` 的驱动。
   - `App.Run`（app.go:600-613）**始终**阻塞在 `signal.NotifyContext`；即使零驱动匹配也会阻塞 → 插件自身禁止阻塞。
   - `Profile` 枚举仅 api/worker/schedule/all；驱动仅在 `Start` 中按 profile 过滤（app.go:487-494），插件 `Apply`、IoC、事件、配置在各 profile 下行为一致。
   - 长生命周期服务标准写法（`plugins/drivers/driver_inproc_worker/plugin.go:118-151`）：`Apply` 内保存 coreCtx → `core.Provide` → `ctx.OnDispose(...)` → `ctx.RegisterDriver(p)`；`Stop` 逆序（LIFO）调用并受 `shutdownTimeout` 约束。
   - `RunMigrations()`（app.go:414-433）在无 `MigrationEngine` 且容器无该服务时返回 nil → 无 DB 的 daemon 合法。
   - DI 真实 API 为包级函数 `core.Provide[T](ctx, svc)` / `core.Inject[T](ctx)` / `core.Using[T](ctx, fn)` / `core.When[T](ctx, fn)`；`Context` **没有** `Provide/DB/Cache/Logger/Storage/DistLock` 方法（ cookbook 中的 `ctx.DB()` 写法为文档性简化，且内核无 DistLock 能力）。
4. **迁移子系统现状**：
   - `w_schema_versions(plugin_id, version_id, applied_at)` 的 DDL 来自 Go（`Wavelet/backend/cmd/app.go:142-153`），**没有** SQL 文件创建它；插入使用 `ON CONFLICT DO NOTHING`。
   - `sharedStore` 未实现 `TableExists` → goose 回退探测失败后会给每个 plugin_id 插入哨兵 `version_id = 0` 行（幂等、良性，但桥接必须知道）。
   - 版本号解析：goose `NumericComponent` 取文件名首个 `_` 之前部分按十进制 int64 解析，须 ≥1；`2026MMDDNNNN` 与 `00001` 均合法；同一 plugin_id 内重复版本号致命。
   - `findMigrationFS`（app.go:309-358）按目录名 `postgres`/`sqlite` 探测，**注册时传入的 `Dir` 参数被忽略** → 目录基名必须是 `postgres`/`sqlite`。
   - 方言选择：`ctx.Config().Bool("database.enabled", false)` → postgres，否则 sqlite3；无 `WithSessionLocker` → **多节点并发迁移无保护**（上游未解）。
   - 上游 `risk_control/logstore/migrations-clickhouse/00001_initial.sql` 是**未被 embed、无处应用**的孤儿文件。
   - **上游不存在任何历史库基线/stamp 能力**（无 migrate 命令、无 `--baseline`），桥接为 OpenFlare 自有产物。
5. **`backend/` 化的路径耦合点**（Phase 1 必须一并处理）：
   - `internal/router/root/frontend.go:18` `//go:embed all:dist`（前端导出物被 Go 包内嵌）；
   - `.github/workflows/build-release.yml` 三处 `-X 'github.com/Rain-kl/Wavelet/internal/apps/{agent,relay,flared}/config.Version'` 与 `./cmd/*/main.go` 构建路径；
   - `Makefile` 的 `build-embedded` / `build-backend` / `cross-build` / `swagger` / `code-check` / `dev-*`；
   - `docker/` 四个镜像的构建上下文与 COPY 路径；`config.example.yaml`、`config/clickhouse`、`.env.example` 的读取路径。

## 3. 目标仓库布局

```
OpenFlare/
├── backend/                              # Go 代码根，module OpenFlare
│   ├── go.mod                            # require Wavelet + replace Wavelet => ./Wavelet
│   ├── go.work                           # 本地一体构建（use . ./Wavelet）
│   ├── Wavelet/                          # 【上游】从 Wavelet/backend 原样拷入
│   │   ├── go.mod                        # module Wavelet
│   │   ├── core/                         # 微内核（零业务）
│   │   ├── core/contracts/               # 跨插件契约（纯 Interface + DTO）
│   │   ├── core/extpoints/               # 扩展点（Router/Task/Schedule/Migration/Setting/Config/Driver）
│   │   ├── plugins/drivers/              # driver_http / asynq_* / inproc_*
│   │   ├── plugins/infra/                # database / cache / cache_memory / config / logger / storage
│   │   ├── plugins/domain/               # admin / user / auth / message_gateway / risk_control / upload / cap / system
│   │   ├── pkg/                          # 上游通用库（禁止 import plugins）
│   │   ├── scripts/check_cordis_architecture.sh
│   │   └── share/                        # 【G3 新增】跨插件共享资源层，OpenFlare 所有
│   ├── plugins/                          # 【G2】OpenFlare 的 4 个插件
│   │   ├── server/                       # 控制面：of_* 业务域 + 边缘接入 API
│   │   ├── agent/                        # 边缘 nginx/WAF 代理守护
│   │   ├── relay/                        # frps 中继守护
│   │   └── flared/                       # frpc 隧道客户端守护
│   ├── cmd/                              # 装配根：root/server/agent/relay/flared/bridge
│   ├── main.go                           # server 入口（swagger 注释保留原位）
│   └── internal/                         # 仅保留非业务基础设施（迁移桥接、构建信息）
├── frontend/                             # 【不修改】
├── docs/                                 # VitePress + swagger 产物路径不变
├── docker/  Makefile  .github/           # 路径改写至 backend/
└── config.example.yaml                   # 位置不变（新增 plugins.* 节点）
```

### 3.1 双 module 而非单 module 的理由

上游树内 import 保持 `Wavelet/core`、`Wavelet/plugins/...` 逐字节等于上游仓库，同步 = `rsync --delete` + 一条 `replace` 指令，无需任何 import 改写。单 module 方案（`OpenFlare/Wavelet/core`）每次同步都要全量重写 import，同步成本随上游演进线性上升，且必然产生漂移。以 Go module 依赖引用上游（不拷代码）被否，因为 OpenFlare 已分叉平台代码，需先在 `backend/Wavelet/` 内落地再回流。

### 3.2 `share` 的落点与所有权

`share` 建在上游 module 内（`Wavelet/share/...`），因为：`Wavelet/pkg/` 禁止 import 业务代码，而 `protocol`、`wsclient`、edge 日志/状态机、`geoip` 等资源**同时被 server 与三个 daemon 消费**；Cordis 规则禁止插件互相 import，故必须存在一个双方都可见的中立层。

**所有权声明**（写入 `backend/Wavelet/share/README.md`）：`share/` 由 OpenFlare 拥有，同步脚本 `scripts/sync-upstream.sh` 显式排除该目录；上游同步仅覆盖 `core/`、`plugins/`、`pkg/`、`scripts/`。此决定的代价是 `share/` 不能从上游获得更新——这是刻意选择，且是唯一需要否定的子决策（备选：落点 `backend/share/`，import `OpenFlare/share/...`，其余设计不变）。

### 3.3 模块与命名

- OpenFlare module 路径：`OpenFlare`（对齐上游 `module Wavelet` 的简化命名惯例）。
- 插件 ID 与目录名同名：`server` / `agent` / `relay` / `flared`。
- 表所有权：`of_*` 全部归 `server`；`w_*` 归各上游平台插件；`w_schema_versions` 归装配根。
- 插件内部分层：一律采用模式 2（`plugin.go` + `handler/ service/ repository/ model/ errs/ migrations/`），子包内文件按业务实体命名，禁止 `handlers_*` 平铺。

## 4. 插件设计

### 4.1 `server`

装配 profile：`api` / `worker` / `schedule` / `all`（与上游语义一致）。挂载上游 infra（database、cache 或 cache_memory、logger、storage）、8 个上游 domain 插件、`driver_http` + asynq/inproc 驱动对，再挂 `server` 插件。

`server` 插件职责：
- 路由：`ctx.Router().Group("/api/v1/...")` 自包含注册 OpenFlare 业务路由；免鉴权路径（边缘节点接入、健康探针、OAuth 回调）由本插件 `RegisterWhitelist` 主动声明，杜绝误拦截。
- 任务/调度：`ctx.Task().Register` / `ctx.Schedule().RegisterCron`（迁移现有 `internal/platform/bootstrap` 的显式装配）。
- 设置：`ctx.Settings().Register`（迁移现有动态设置声明）。
- 契约：向平台暴露 `contracts.*` 之外不新增跨插件事务；OpenFlare 自有能力经 `server` 单入口暴露。
- 迁移：`of_*` 全量 + 历史链（§5）。

内部按子域分包：`zone / cloudflare / pages / waf / node / proxy_route / dns / acme / tls / edge_control / config_version / health`，每子域内部遵循 handler→service→repository→model 单向依赖。

### 4.2 `agent` / `relay` / `flared`

同一模式，各自实现 `core.Plugin` + `core.Driver`：

```go
func (p *Plugin) Name() string { return "agent" }
func (p *Plugin) Type() core.DriverType { return core.DriverType("agent") }
func (p *Plugin) Apply(ctx *core.Context) error {
    if err := ctx.Config().Bind("agent", &p.cfg); err != nil { return err }   // JSON 配置经 ConfigSource 适配
    core.Provide[contracts.LoggerService](ctx, lg)
    ctx.OnDispose(func() error { return p.stop() })
    return ctx.RegisterDriver(p)
}
func (p *Plugin) Start(ctx context.Context) error { /* util.Go 启动 heartbeat/sync/updater，select ctx.Done() */ }
func (p *Plugin) Stop(ctx context.Context) error  { /* 收敛 frpc/frps 子进程与 ws 连接 */ }
```

- 入口 `cmd/agent/main.go` → `core.NewApp(core.WithProfile(core.Profile("agent")), core.WithConfigSource(jsonSource))` + `app.Use(agent.New())`；**不挂** http/database/cache/asynq 驱动。
- JSON 配置文件（`agent.json`/`relay.json`/`flared.json`）通过实现 `extpoints.ConfigSource`（`Lookup/LookupEnv/Describe`）接入，优先级：env 覆盖 → 文件 → default，与上游 `plugins/infra/config` 的解析优先级一致；`-config` 旗标与默认路径不变。
- 三者共享的 protocol / wsclient / edge logging / state 迁入 `Wavelet/share/`；`internal/apps/edge/` 消失（其内容分别归入 `share` 与 daemon 插件）。
- 裸 `go func()` 一律经 `util.Go`（架构门禁禁止裸 go）。

## 5. 数据库迁移方案（G5）

### 5.1 历史链保真

`goose/postgres`（76）与 `goose/sqlite`（76）文件**逐字节原样**迁入 `backend/plugins/server/migrations/{postgres,sqlite}/`，文件名、序号、内容不改；ClickHouse 链（14，含 `goose_clickhouse_version` 版本表）与其现有 `MigrateClickHouse` 路径**完全保持不动**，不并入 `w_schema_versions`（语义不同、且上游 CH 文件为孤儿，不引入其语义）。

### 5.2 插件初建与对齐

- 上游平台插件的 `00001_initial.sql` 保留：全部 `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS` / seed `ON CONFLICT DO NOTHING` → 在 OpenFlare 老库上只补建上游新增表（如 `w_message_channels`），已存在表自动跳过。
- 若上游模型需要 OpenFlare 现库缺失的列，**只允许**在对应插件的 `migrations/{postgres,sqlite}/00002_openflare_align.sql` 追加（`ADD COLUMN` / `CREATE INDEX IF NOT EXISTS`）。全链路禁止 `DROP TABLE` / `DROP COLUMN` / `TRUNCATE`。
- 双方言目录必须文件名与版本号一一对应，否则 PG 与 sqlite 的 `version_id` 漂移。

### 5.3 一次性桥接（`cmd/bridge`，启动前置）

1. 以与上游 `cmd/app.go:142-153` **逐字节一致**的 DDL 建 `w_schema_versions`（保证上游 `CreateVersionTable` 的 `IF NOT EXISTS` 成为 no-op）。
2. 取 PG advisory lock（`pg_advisory_lock(key)`）/ sqlite 依赖单写者，补齐上游缺失的并发保护。
3. 若存在老库版本表（`goose_db_version` / OpenFlare 现名）且 `w_schema_versions` 无 `openflare/legacy` 记录：写入 `(openflare/legacy, 0)` 哨兵与 `(openflare/legacy, MAX(老库已应用版本))`，全部 `ON CONFLICT DO NOTHING`。
4. 桥接幂等：重复执行零变更；老库此后**永不重跑**历史链，新库正常跑完整链后收敛到同一 schema。
5. 桥接仅由 server 装配根在 `app.RunMigrations()` 之前调用一次；agent/relay/flared 不涉及（无 DB）。

### 5.4 三方一致性门禁（不可跳过）

| 路径 | 构造方式 | 断言 |
| :-- | :-- | :-- |
| A（基线） | 改造前 `main` 二进制在全新 sqlite/PG 上跑完 76 链 | dump `information_schema` + seed 行数 |
| B（新架构全新安装） | 桥接 + 历史链 + 各插件 initial/align | 与 A **逐项 diff 为空** |
| C（升级路径） | 恢复 A 的库文件后再启动一次新架构 | schema diff 为空，且本次启动**零 SQL 变更**（迁移结果集为空） |

工具：`backend/cmd/migrate-audit`（导 dump 与 diff，只读、不写源码树，临时目录用 `t.TempDir()`）。任一路径失败即视为改造未完成，禁止合并。

## 6. 前端与 API 契约（G4/G6）

`frontend/` 零改动（`git diff --name-only main...cordis -- frontend/` 必须为空）。内嵌物路径由 `Makefile` 的 `build-embedded` 负责：前端导出物仍拷到 `backend/internal/router/root/dist`（或迁移后对应包目录），`//go:embed all:dist` 随包位置同步。

服务端 HTTP 契约保持：路径、`{error_msg,data}` 信封、分页 `{total,results}`、成功 200 / 失败 `Abort*` 语义、swagger 产物路径（`docs/swagger.json|yaml`）。验收：改造前后各跑一次 `swag init`，对拍 `paths` 键集合必须一致（新增路径允许，删改不允许）。

## 7. 工具链、CI 与文档

- `Makefile`：Go 目标切到 `backend/`；新增 `sync-upstream`、`migrate-audit`、`arch-check`。
- `.github/workflows`：4 个 build-image 工作流的构建上下文/`Dockerfile` COPY 路径、`build-release.yml` 的 `./cmd/*/main.go` 与三处 `-X` ldflags 模块路径（改为 `OpenFlare/plugins/{agent,relay,flared}/config.Version`）。
- 门禁移植：`backend/Wavelet/scripts/check_cordis_architecture.sh`（core 不 import gin/gorm/asynq；pkg 不 import plugins；插件间不互相 import；禁 AutoMigrate；禁裸 go）在 OpenFlare `backend/` 下纳入 `make code-check`。
- 文档：中文文档同步（不同步英文）；`docs/changelog/index.md` 的 `[Unreleased]` 记录用户可读变更。

## 8. 分期执行计划（每期终点：`go build ./...` 通过 + 209 测试文件绿）

| 期 | 交付 | 验证 |
| :-- | :-- | :-- |
| P0 | 基线测量：构建/测试记录、A 路 schema dump、路由清单快照 | 存档于 `docs/superpowers/specs/baseline/` |
| P1 | Go 树 `git mv` 入 `backend/`，module 改 `OpenFlare`，import 全量改写，Makefile/CI/Docker 路径 | `go build ./...`、四二进制产出、swagger diff 为空 |
| P2 | 上游 vendoring 成 `backend/Wavelet/`（第二 module）+ `share/` 骨架 + 同步脚本 | `go work sync`、`go build ./...`、架构门禁通过 |
| P3 | 迁移子系统：`w_schema_versions` 引擎接入、桥接、历史链归属、align 迁移、`migrate-audit` | **§5.4 三方 diff 全空** |
| P4 | `server` 插件化：`apps/openflare` + health/config/edge 控制面 → 插件子包 + 扩展点注册 | 路由对拍 + 业务测试绿 |
| P5 | `agent`/`relay`/`flared` 插件化 + `share` 落地（protocol/wsclient/edge） | 三 daemon 实跑冒烟 + JSON 配置兼容 |
| P6 | 平台分叉审计与回流：OpenFlare 对 admin/oauth/user/upload/cap/push 的定制逐项移植进 `backend/Wavelet/` | 分叉清单逐项关闭，无静默丢功能 |
| P7 | 工具链/CI/文档/changelog | `make code-check`、`make format` |
| P8 | 加固：`go test -race ./...`、`-shuffle=on`、四 profile 实跑、老库升级演练 | 全绿方可声明完成 |

依赖：P3 依赖 P2；P4/P5 依赖 P2、P3；P6 可与 P4 并行但须在同一期收口。

## 9. 风险与对策

1. **fresh 安装与升级路径 schema 漂移**（桥接最大风险，`CREATE TABLE IF NOT EXISTS` 会静默保留旧表）→ §5.4 三方 diff 门禁 + `migrate-audit` 常驻 CI。
2. **平台分叉被静默丢弃**（OpenFlare 的 admin/oauth/push 定制）→ P6 强制逐项清单，禁止以"上游没有"为由删除行为。
3. **多节点同时迁移竞争**（上游无 session locker）→ 桥接与迁移阶段 advisory lock。
4. **29.6k 行业务塞进单插件导致不可维护**→ `server` 内部按子域物理分包（§4.1），插件数保持 4 不变。
5. **CI/Docker 路径漏改导致发布链断裂**（release 工作流含硬编码 ldflags 模块路径）→ P1 一次性收口并跑 dry-run 构建。
6. **规模**：151k 行、166 迁移文件，上游自身耗时 100+ 提交 → 分期增量、每期可编译可测；不以"部分完成"冒充目标达成。

## 10. 已定决策（可在复审时否定）

- 布局：镜像 `backend/`，上游为第二 module（用户选定）。
- 插件边界：每二进制 = 1 插件，四者全部走内核（用户选定）。
- 迁移：保留历史 + 版本 stamp 桥接（用户选定）。
- `share` 落点：`backend/Wavelet/share/`，OpenFlare 所有、同步排除（本设计采纳，备选见 §3.2）。
- 桥接粒度：stamp 至 `MAX(已应用版本)` 而非逐条 76 行（版本号语义为「≤ max 即已应用」，逐条无额外收益且放大漂移面）。
- ClickHouse：保持既有 `goose_clickhouse_version` 独立链，不并入插件版本表。
