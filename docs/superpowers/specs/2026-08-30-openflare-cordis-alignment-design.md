# OpenFlare Cordis 对齐设计

- **文档状态**: 已敲定 (Approved)
- **版本**: v1.0.0 (2026-08-30)
- **范围**: 后端。禁止修改前端源码。禁止修改金标准树。
- **取代**: `2026-08-29-openflare-cordis-refactor-design.md` 中尚未落地或与本文冲突的部分（装配旁路、server 内平台副本、rsync 长期同步、在 server 上挂 health/self/cap 别名）。
- **Wavelet**: `/Users/ryan/Code/Go/Wavelet`
- **OpenFlare Cordis 工作树**: `/Users/ryan/Code/Go/OpenFlare-cordis`（分支 `cordis`）
- **金标准（未改造旧框架）**: `/Users/ryan/Code/Go/OpenFlare` @ `9f79fb99`（v3.5.4，与工作树 `main` 同 SHA）

---

## 1. 目标

OpenFlare 是 Wavelet 的下游产品。框架层与 Wavelet 一致，便于此后用 `git merge wavelet/main` 吸收上游。OpenFlare 只做 OpenFlare 业务，不再承担与 Wavelet 同类的职责。

Wavelet 提供框架与平台能力（微内核、infra、drivers、auth/admin/user/cap/upload/message_gateway/risk_control/system）。缺能力时改 Wavelet，且必须通用，禁止把 OpenFlare 业务写进框架。

---

## 2. 决策记录

| # | 决策 | 理由 |
| :--- | :--- | :--- |
| D1 | 前端 HTTP 契约冻结 | 禁止改 `frontend/`。路径可超集，不可删改金标准已有 path+method 的响应形状。 |
| D2 | 同一轮删除全部平台副本 | `cap` / `admin` / `infra` / `oauth` / `user` / `upload` / `config` / `health` / `admin/push` / `w_*` model+repository。留两套会双注册路由。 |
| D3 | 完整 stamp 桥接 | 现网 `goose_db_version` 76 链不重跑；新装走 Wavelet 每插件迁移 + `of_*` initial。 |
| D4 | 兼容靠 Wavelet 补通用能力，不靠 server 包装同类 HTTP | health / user/self / cap 路径差由拥有该功能的 Wavelet 插件多挂入口。 |
| D5 | Wavelet 作为 git 上游，merge 不改写历史 | 接线后废弃 rsync。禁止 rebase / filter-repo / force-push OpenFlare。 |
| D6 | OpenFlare 不承担同类职责 | 同一张 `w_*` 表、同一类鉴权/用户/验证码/上传/管理端/推送/任务/配置/健康检查/HTTP 引擎/关系库 migrator，均属 Wavelet。 |
| D7 | 金标准树只读 | `/Users/ryan/Code/Go/OpenFlare` @ `9f79fb99` 零改动。升级与对拍以它现跑现导为准。 |
| D8 | 先对齐代码，再第一次 unrelated merge | 半成品 merge 会把分叉和 Wavelet 前端搅进 OpenFlare。 |

---

## 3. 职责铁律

判定「同类职责」：Wavelet 已经覆盖的平台能力，OpenFlare **不得再实现一份**，也不得在 `server` 里用别名/包装再挂一套同类 HTTP。缺口（路径、中间件、贡献点）→ 改 Wavelet。

OpenFlare 只拥有：`of_*`、边缘协议、节点/Pages/WAF/TLS/Cloudflare 等产品逻辑；以及通过契约 **注册** 自己的事件与公共配置贡献（不是再写一套推送或配置服务）。

---

## 4. 目标结构与装配根

单 module `Wavelet`，与上游同构：

```
backend/
├── core/  pkg/  plugins/     【Wavelet】merge 进来，OpenFlare 不留私补丁
├── cmd/                      【Wavelet 骨架 + OpenFlare 持久 diff】
│   ├── app.go                即 Wavelet newWaveletApp，多 app.Use(server.New())
│   ├── api.go / worker.go / scheduler.go / all.go / root.go
│   └── {agent,relay,flared}/ OpenFlare 额外守护进程
├── main.go
└── OpenFlare/                占据上游 downstream/ 的位置
    ├── plugins/server        只留 OpenFlare 业务
    ├── plugins/{agent,relay,flared}
    └── share/
```

控制面 `newOpenFlareApp(profile)` 的 `app.Use` 顺序与 Wavelet `newWaveletApp` 相同：

1. infra：`database`、`logger`、`storage`
2. `cache` / `cache_memory`，`asynq` / `inproc` worker+cron（配置门控不变）
3. domain：`admin`、`user`、`auth`、`message_gateway`、`risk_control`、`upload`、`cap`、`system`
4. `OpenFlare/plugins/server.New()`
5. `SetMigrationEngine`（Wavelet 那份 `w_schema_versions` goose 引擎，不复制第二套）
6. `core.WithMigrationBaseline(stampFn)`：stamp 函数本体在 `OpenFlare/` 内，装配根只传入回调。与 `server.New()` 同属 cmd 允许的持久 diff。
7. `driver_http.New()`，**不再** `WithEngine(router.BuildEngine())`

`api` / `worker` / `scheduler` / `all` 全部 `runProfileApp(profile)`。删除：

- `bootstrap.RegisterAPI` / `Init` 进程级副作用链
- Cobra `PreRun` 本地 `migrator.Migrate()`
- `all.go` 自行拉起 `worker.StartWorker` / `scheduler.StartScheduler`
- 全局 `infra/config.Config` 单例；host 配置抄 Wavelet `root.go` 的 `config.NewSource()` + `hostConfig`

`cmd` 允许的持久 diff 仅：`app.Use(server.New())`、`WithMigrationBaseline(stampFn)`、agent/relay/flared 子命令、swagger 标题。禁止在 cmd 里自己拉起 DB/worker/migrator，禁止在 cmd 里实现 stamp 细节。

agent / relay / flared 入口已经是 `core.NewApp` + 单插件，本轮不改 CLI 形态。

---

## 5. Wavelet 通用扩展点（W1–W9）

先在 `/Users/ryan/Code/Go/Wavelet` 落地并进入其主线，再进入 OpenFlare。接口与实现禁止出现 OpenFlare / `of_` / 边缘节点等业务词。

### W1. `HandleRaw`

`Handle()` 经 `cleanPath` 会剥尾斜杠。需要 `HandleRaw`（保留尾斜杠）与 `BasePath()`。本地补丁已在 OpenFlare `upstream-patches.md` 与 Wavelet 分支 `feat/cordis-router-raw-routes`。合并进 Wavelet main 后清空补丁清单。

### W2. `contracts.CaptchaService`

Wavelet 已有 `cap.login_enabled`，但 `user` 的 login/register/send-email-code 未挂验证码。跨插件禁止 import `cap`。

```go
type CaptchaService interface {
    VerifyMiddleware(scope string) any // gin.HandlerFunc
    ChallengeHandler() any
    RedeemHandler() any
}
```

`cap.Apply` 中 `Provide`；`user` 在上述三路由上 `Inject`，有实现则包一层，没有则保持裸路由。`cap` 同时再挂 `POST /api/cap/{challenge,redeem}`（原 `/api/v1/cap/*` 保留）。pow 留在 `cap` 内，不抽到 `pkg/pow`。

### W3. 可选 `PublicConfigProvider`

`GET /api/v1/config/public` 路径相同。Wavelet 默认 `{configs, app}`；金标准前端消费扁平 `map[string]string`。

```go
type PublicConfigProvider interface {
    PublicConfig(ctx context.Context) (any, error)
}
```

`system` 在 handler 里 `Inject`：有 provider 用其 payload，没有走默认。OpenFlare `server` Provide 扁平 map。默认 Wavelet 行为不变。

### W4. `driver_http` 引擎选项

已有 Recovery / CORS / session / otelgin / 错误处理，以及 `embed_frontend` 的 SPA `NoRoute`。再增加配置项控制 `RedirectTrailingSlash`（默认 `true`，Wavelet 行为不变）。OpenFlare 通过配置关闭尾斜杠重定向。前端静态资源走 Wavelet `embed_frontend`（Makefile 拷贝目标改到 `driver_http/dist`，不改 `frontend/` 源码）。去掉 `WithEngine` 旁路。

### W5. upload 补挂已有 handler

`ListMyFiles` / `UpdateMyFile` / `DownloadFile` 已实现，测试已用 `GET /my`，但 `plugin.go` 未全部挂到用户组。在 `/api/v1/upload` 用户组补上 `GET /my`、`PUT /:id`、`GET /download/:id`、`POST /download/batch`（只增不改）。原 `GET ""`、`POST /batch-download` 保留。

### W6. `contracts.PushRegistry`

`RegisterBuiltInEvent` / `SyncEvents` 已在 `message_gateway` 存在，但是包级函数。收到契约上，下游只 `Inject` 注册自己的事件，禁止 import 域插件。

### W7. `system` 增加 `GET /api/health`

与现有 `/healthz`、`/api/healthz` 同一 handler。金标准调用 `/api/health`。响应保持金标准 `{error_msg, data: null}` 信封（`response.OKNil()`），不得只返回 `{status: ok}` 却让金标准路径变成另一种 JSON。若 `/healthz` 维持 Wavelet 原 JSON，`/api/health` 必须是金标准形状。

### W8. `user` 增加 `GET /api/v1/user/self`

内部 `AuthService.GetCurrentUser`。鉴权中间件与其它需登录的 user 路由相同。

### W9. Migration Baseline 钩子

`MigrationEngine` 在创建 `w_schema_versions` 之后、`Up` 之前调用可选 Baseline 回调（`core.WithMigrationBaseline`）。任何有历史库的下游可用。引擎接口禁止出现产品名。OpenFlare 的 stamp 实现放在 `OpenFlare/` 内，由 cmd 传入。并发保护（PG advisory lock / sqlite 单写者）加在 Wavelet 引擎上，不在 OpenFlare 再写迁移框架。

---

## 6. 清理后的 `server` 边界

### 删除（改走上游）

| 现路径 | 改走 |
| :--- | :--- |
| `admin/`（除 updater） | `plugins/domain/admin` |
| `oauth/` | `plugins/domain/auth` |
| `cap/`、`pkg/cap/` | `plugins/domain/cap` + `CaptchaService` |
| `user/` | `plugins/domain/user` |
| `upload/` | `plugins/domain/upload` |
| `config/` | `system` + `PublicConfigProvider` |
| `health/` | Wavelet `system` 的 `GET /api/health` |
| `admin/push/`、`pkg/push/` | `message_gateway` + `PushRegistry` |
| `infra/config`、`persistence`、`task`、`objectstore`、`diskcache`、`idgen` | 上游 infra / drivers / `pkg/idgen` |
| `infra/persistence/migrator`（关系库引擎与 76 链执行） | Wavelet 引擎 + stamp |
| `model` / `repository` 中所有 `w_*` | 对应上游域插件 |
| `listener/` | `ctx.Events().On` |
| `platform/bootstrap` 的任务/推送/进程初始化 | 各插件 `Apply` + 上游驱动 |
| ClickHouse 链中的 `w_user_access_logs` | `risk_control` |

删除后禁止再 import 这些包。业务只通过 `core/contracts`、`ctx.Config()`、`ctx.Events()`、`ctx.Task()`、`ctx.Router()` 协作。

### 保留

- `openflare/`（zone、node、origin、pages、waf、tls、cloudflare、agent/relay/flared 接入、websocket、dashboard、option 等）
- `of_*` 的 model / repository（含节点访问日志、可观测性、`chwriter`）
- `admin/updater`（三段式 Prepare/Apply/Finish；上游只有 GetUpdateStatus/ApplyUpdate）
- `integration/githubrelease`（仅 server 使用则留 server；多插件再用再进 `share/`）
- ClickHouse `of_node_*` 链（不并入 `w_schema_versions`）
- `share/`（protocol、wsclient、geoip、edge；本轮不搬）

`server.Apply` **不**注册 `/api/health`、`/api/v1/user/self`、`/api/cap/*`。这些由 Wavelet 插件声明。

`Apply` 职责：绑定 OpenFlare 自有配置；注册 `of_*` 迁移；Provide `PublicConfigProvider`；Inject 契约并注册 `of_*` 业务路由/任务/设置；经 `PushRegistry` 注册 OpenFlare 自己的事件。拆掉 `router/v1` 里把平台路由和业务编在一起的接线。SPA 兜底交给 `driver_http` + `embed_frontend`。

---

## 7. 迁移与 stamp

OpenFlare **不再自建关系库 migrator**。迁移即 Wavelet `cmd/app.go` 的 `w_schema_versions` + 每插件 goose。

### 表所有权

| 表 | 谁建、谁改 |
| :--- | :--- |
| `w_*` | Wavelet 各 domain `migrations/` |
| `of_*` | `OpenFlare/plugins/server/migrations/{postgres,sqlite}` |
| `w_schema_versions` | Wavelet 引擎 |
| ClickHouse `of_node_*` | `server` 自有 CH 链 |
| ClickHouse `w_user_access_logs` | Wavelet `risk_control` |

76 条历史 SQL 混着 `w_*` 与 `of_*`，**停止执行**。仅暂时留作与金标准 schema 对拍的夹具；门禁通过后可靠 git 历史，不再注册到引擎。

### 新装

1. Wavelet 各插件 `00001_initial`（`CREATE IF NOT EXISTS` / seed `ON CONFLICT DO NOTHING`）建 `w_*`，并补金标准没有的 `w_message_channels` / `w_message_bindings` / `w_message_pairing_codes`。
2. `server` 只含 `of_*` 的 `00001_initial`（当前 schema 压缩版，postgres/sqlite 文件名与版本号对齐），全部 `IF NOT EXISTS`。
3. 禁止 `DROP TABLE` / `DROP COLUMN` / `TRUNCATE`。

### 升级（金标准库：已有 `goose_db_version`，最大版本 `202608090003`）

Baseline 回调（幂等，`ON CONFLICT DO NOTHING`）：

1. 按与 Wavelet 逐字相同的 DDL 确保 `w_schema_versions` 存在。
2. 写入 `(openflare/legacy, 0)` 与 `(openflare/legacy, MAX(goose_db_version))`，76 条历史不再跑。
3. 写入 `(server, 1)`，压缩版 `of_*` initial 视为已应用。
4. **不 stamp Wavelet 插件**，让其 `00001` 以 `IF NOT EXISTS` 补缺口表。
5. 仅当老库版本落在 `[202607120002, 202607130001)` 时调用 `zone.ImportLegacyTx`。金标准已在 `202608090003`，此步为空操作。
6. ClickHouse 仍走 `goose_clickhouse_version`，只保留 `of_node_*`。

删除 `infra/persistence/migrator` 整包（关系库 `Migrate()`、Cobra `PreRun`、进程级 goose）。CH 升级保留为 `server` 内小函数，不是第二套通用 migrator。

---

## 8. Git 上游

两边目前没有共同祖先（OpenFlare 从 `init` 起 1064 提交；Wavelet 根提交为历史压缩）。禁止 rebase / filter-repo / force-push OpenFlare。

| 路径 | 所有权 | merge |
| :--- | :--- | :--- |
| `backend/{core,pkg,plugins}` | Wavelet | 三路合并，无私补丁 |
| `backend/cmd`、`backend/main.go` | Wavelet 骨架 + 第 4 节所述 diff | 三路合并 |
| `backend/OpenFlare/` | OpenFlare | `merge=ours` |
| `frontend/` | OpenFlare | `merge=ours` |
| `docs/changelog`、`docs/superpowers` | OpenFlare | `merge=ours` |

`.gitattributes`：

```gitattributes
backend/OpenFlare/**    merge=ours
frontend/**             merge=ours
docs/changelog/**       merge=ours
docs/superpowers/**     merge=ours
```

接线（对齐完成且第 9 节三层门禁全绿之后）：

```text
git remote add wavelet <Wavelet.git>
git fetch wavelet
git merge --allow-unrelated-histories wavelet/main   # 仅此一次
# 框架目录以 Wavelet 为准；ours 路径保持 OpenFlare
```

此后：`git fetch wavelet && git merge wavelet/main`。删除 `scripts/sync-upstream.sh`。接线前为导入 W1–W9 可使用一次 rsync。接线后 OpenFlare 不得再改 `core/pkg/plugins`。

成功判据：`git merge-base HEAD wavelet/main` 非空；`git log --first-parent` 仍是 OpenFlare 提交链；`frontend/` 与 `backend/OpenFlare/` 的历史 SHA 不被重写。

---

## 9. 验证门禁

金标准目录 `/Users/ryan/Code/Go/OpenFlare` @ `9f79fb99`，本轮零改动。Cordis 仓只读它。

### L1 — 金标准自身绿

在金标准树：`go test ./...` 全过。失败则停。

### L2 — Cordis 仓绿

在 `/Users/ryan/Code/Go/OpenFlare-cordis`：

- `go test ./...`
- Wavelet 仓 `go test ./...`（W1–W9 落地后）
- `make swagger`：金标准 swagger 的 path+method 一条不删、形状不改；允许超集
- 路由：金标准引擎表 ⊆ 完整 `newOpenFlareApp` 路由表。`plugin_parity_test` 必须对完整装配断言子集，不再只 `server.New().Apply`，不再要求精确相等
- `git diff --name-only -- frontend/` 为空

金标准必须包含且不得丢失的路径：`GET /api/health`、`GET /api/v1/user/self`、`POST /api/cap/challenge`、`POST /api/cap/redeem`、`GET /api/v1/config/public`（扁平 map）。

`docs/superpowers/specs/baseline/*` 必须与金标准二进制 **现跑现导** 一致；不一致以金标准为准改夹具。

### L3 — 从金标准库升级

用金标准二进制在空库跑完 76 链得库 A（sqlite 必做，Postgres 必做）：

1. 记录 `goose_db_version` 最大版本（`202608090003`）、表清单、`of_*` / `w_*` 行数抽样。
2. 同一份库文件启动 Cordis `api`。
3. 断言：已有表不丢列、不丢行；76 链不重跑；`w_schema_versions` 含 `openflare/legacy` 与 `server`；允许补 `w_message_*`；对升级后进程请求金标准关键 API，HTTP 与 JSON 形状与金标准一致。

| 路径 | 断言 |
| :--- | :--- |
| A 金标准二进制 + 76 链 | 与现导 schema 一致 |
| B Cordis 空库新装 | `of_*` 与 A 一致；`w_*` 允许 Wavelet 多表，禁止少列、禁止改已有列 |
| C 恢复 A 再启 Cordis | 数据仍在；关系库无重放历史 |

升级路径必须是 `go test`（夹具来自金标准迁移产物或测试内生成），禁止只手测一次。

B/C 失败即改造未完成。只测空库新装、未测金标准升级，算失败。改了金标准树，算失败。

---

## 10. 落地顺序

1. Wavelet 落地 W1–W9，`go test ./...` 绿。
2. OpenFlare-cordis 同步 `core/pkg/plugins`（接线前可用一次 `sync-upstream.sh`），与 Wavelet 零差，清空 `upstream-patches.md`。
3. 装配根同构：`runProfileApp` + `server.New()`；删除 bootstrap / 本地 migrator / `WithEngine`。
4. 删除平台副本；`server` 收敛到第 6 节边界。
5. stamp + `of_*` 压缩 initial；删除关系库 `migrator` 包。
6. 第 9 节 L1/L2/L3 全绿。
7. 第一次 `git merge --allow-unrelated-histories wavelet/main`；此后只 merge。

---

## 11. 明确不做

- 修改 `/Users/ryan/Code/Go/OpenFlare`
- 修改 `frontend/` 源码
- 在 OpenFlare 实现鉴权/用户/验证码/上传/管理端（updater 除外）/推送实现/健康检查/HTTP 引擎/关系库 migrator
- 在 `server` 再挂 `/api/health`、`/api/v1/user/self`、`/api/cap/*`
- 把 pow 抽到 `pkg/pow`
- 把 OpenFlare 业务写进 Wavelet
- rebase / filter-repo / force-push 改写 OpenFlare 历史
- 本轮改变 agent/relay/flared 的 CLI 形态
- 把 ClickHouse `of_node_*` 并进 `w_schema_versions`

---

## 12. 完成定义

以下同时成立才算完成：

- 金标准库能升到 Cordis，数据还在，76 链不重跑
- 金标准路由 ⊆ Cordis 路由；`GET /api/v1/config/public` 仍为扁平 map
- 金标准 `/Users/ryan/Code/Go/OpenFlare` 上 `go test ./...` 绿
- Cordis `go test ./...` 绿；Wavelet `go test ./...` 绿
- `frontend/` 无 diff
- 第 7 步之后 `git merge-base HEAD wavelet/main` 非空；`git log --first-parent` 仍是 OpenFlare 提交
