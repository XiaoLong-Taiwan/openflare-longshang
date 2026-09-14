# OpenFlare API 收口与 server 目录整理

- **文档状态**: 已敲定 (Approved)
- **版本**: v1.0.0 (2026-08-31)
- **范围**: Wavelet 平台路由（cap/health）、OpenFlare 前端请求路径、OpenFlare `server` 目录与压缩迁移（含种子）
- **承接**: `2026-08-30-openflare-cordis-alignment-design.md`。本文件**有意修改**此前冻结的 `/api/cap/*` 与 `/api/health` 契约。

---

## 1. 目标

1. 功能相同的双路径只留 **`/api/v1/...` 形态**（health 例外：只留 `GET /api/healthz`）。前端跟着改。
2. `OpenFlare/plugins/server` 目录干净：一个迁移包、updater 不再套空的 `admin/`，删除已停用的 76 条历史 SQL。
3. 压缩版 `00001` **既有建表也有预制插入**。新装与金标准 v3.5.4 跑完 76 链后的 **OF 种子行**一致（key/task_type 集合），升级库不重插、不覆盖已有值。

---

## 2. 决策记录

| # | 决策 | 理由 |
| :--- | :--- | :--- |
| D1 | cap 只留 `/api/v1/cap/*`，删除 `/api/cap/*` | 同一套 Challenge/Redeem，前端改为 v1 |
| D2 | health 只留 `GET /api/healthz`，删除 `/healthz` 与 `/api/health` | 用户指定；JSON 维持 `{status: ok}` |
| D3 | 改 Wavelet 插件再同步进 OpenFlare | 路由属于 cap/system，禁止在 OF server 再挂别名 |
| D4 | 76 历史 SQL **从仓库删除** | git 历史可查；基线夹具已在 `docs/superpowers/specs/baseline/` |
| D5 | `00001` = of_* DDL + OF 种子 INSERT | 当前压缩版只有 CREATE，新装会缺默认配置和定时任务 |
| D6 | 种子只含 Wavelet 未提供的 key/任务 | `w_*` 表仍归 Wavelet；OF 只插产品默认行 |
| D7 | `admin/` 去掉，updater 提到 `server/updater/` | admin 包只剩 updater，无独立意义 |
| D8 | stamp + of_* SQL + CH 收入 `server/migrate/` | 结束三个迁移包并列 |

---

## 3. API 收口

### 3.1 Wavelet（`feat/cordis-alignment` 或其后继，禁止 OpenFlare 字样）

**cap**（`plugins/domain/cap/plugin.go`）

- 保留 `Group("/api/v1/cap")`：`GET/POST /challenge`、`POST /redeem`。
- 删除 `Group("/api/cap")` 及对其的 `RegisterWhitelist`。
- Swagger `@Router` 一律 `/api/v1/cap/...`。
- 白名单只保留 `/api/v1/cap/challenge`、`/api/v1/cap/redeem`（及现有 v1 GET）。

**system**（`plugins/domain/system/plugin.go`）

- 保留 `GET /api/healthz`，handler 仍返回 `{"status":"ok"}`。
- 删除 `GET /healthz`、`GET /api/health`（`Health`/`OKNil` 信封不再对外）。
- 白名单只保留 `/api/healthz`（若探针无需登录；现 `/healthz` 白名单一并改到此路径）。

测试：`plugin_captcha_contract_test` 不再要求 `/api/cap`；`health_route_test` 只断言 `/api/healthz`，且不得再存在 `/api/health`、`/healthz`。

### 3.2 OpenFlare

| 位置 | 改动 |
| :--- | :--- |
| `frontend/lib/cap-solver.ts` | `fetch('/api/v1/cap/challenge')`、`fetch('/api/v1/cap/redeem')` |
| README / README_zh 健康检查 | `http://localhost:8000/api/healthz` |
| `backend/cmd/app_test.go`、parity | 断言 `POST /api/v1/cap/challenge`、`GET /api/healthz`；删除对 `/api/cap`、`/api/health` 的硬性要求 |
| swagger | 再生后无 `/api/cap`、`/api/health`、`/healthz` |

**不动**：`/api/v1/d/*`、agent/relay/flared 协议、`/api/v1/user/self`、`/api/v1/upload/my` 等已是 v1 且前端在用的路径。

同步：Wavelet 改完后 `git merge wavelet/<alignment-branch>`（或 sync 若尚未只走 merge）。禁止在 `backend/{core,pkg,plugins}` 留私补丁。

---

## 4. `server` 目标目录

```
backend/OpenFlare/plugins/server/
├── plugin.go
├── plugin_tasks.go
├── updater/                 # 原 admin/updater
├── migrate/                 # 唯一迁移包
│   ├── stamp.go
│   ├── stamp_test.go
│   ├── clickhouse.go
│   ├── clickhouse_test.go
│   ├── postgres/00001_initial.sql
│   ├── sqlite/00001_initial.sql
│   └── clickhouse/*.sql     # of_node_*，无用户访问日志表
├── publicconfig/
├── ofevents/
├── openflare/
├── router/v1/openflare/
├── model/                   # of_* 为主
├── repository/
├── integration/githubrelease/
├── task/
├── runtimeconfig/
├── shared/
└── testhelper/
```

**删除**

- `admin/`（文件迁到 `updater/`）
- `migrator/` 整树（76×2 历史 SQL）
- 独立 `stamp/`、`chmigrate/`、`migrations/`
- 无引用残留（如 `router/root/frontend_static.go`）

`plugin.go`：`//go:embed migrate/postgres/*.sql migrate/sqlite/*.sql`；`ctx.Migrations().Register("server", fs)`；CH 调 `migrate.UpClickHouse()`。目录基名必须是 `postgres`/`sqlite`。`register_updater` import 改为 `…/updater`。

`openflare/` 业务子域本轮不重组。

---

## 5. 压缩迁移：建表 + 种子

当前 `migrations/*/00001_initial.sql` **零 INSERT**。历史链里的预制数据必须收进来。

### 5.1 建表

保留现有 `of_*` 的 `CREATE TABLE IF NOT EXISTS` 与索引（sqlite/postgres 双方言对齐）。禁止 `DROP TABLE` / `DROP COLUMN` / `TRUNCATE`。

`of_*` 业务表无必填种子行（节点/站点运行时写入）。

### 5.2 定时任务种子（`w_schedules`）

`INSERT … ON CONFLICT DO NOTHING` / sqlite `INSERT OR IGNORE`：

| id | name | task_type | cron |
| :--- | :--- | :--- | :--- |
| 101 | OpenFlare SSL 自动续期 | `of_ssl_renew` | `0 0 * * *` |
| 103 | OpenFlare WAF IP 组同步 | `of_waf_ip_group_sync` | `*/5 * * * *` |
| 104 | OpenFlare Uptime Kuma 同步 | `of_uptime_kuma_sync` | `* * * * *` |
| （不指定 id，用 NOT EXISTS） | OpenFlare Pages 部署源扫描 | `of_pages_source_scan` | `0 0 * * *` |

**禁止**再插入 `of_database_auto_cleanup`（历史链末尾已删）。**禁止**再改 `system_cleanup`（Wavelet 已是 `0 3 * * *`）。

### 5.3 系统配置种子（`w_system_configs`）

只插入 **Wavelet `admin` `00001` 种子里没有的 key**。来源：

1. `202606220004_migrate_of_options_to_system_configs.sql` 迁入的 agent/geoip/uptime/openresty 等（用该文件与 `create_of_options` 的**最终默认值**，不要 `SELECT FROM of_options`，新装没有这张表）。
2. 其后独立文件补的 key：Pages、OpenResty 限流、源站错误页、SW offline、日志/指标保留等（`202607170001`、`202607190001`、`202607200001`、`20260806*`、`20260807*`、`202608080002`、`202608090001` 等）。

全部 `ON CONFLICT (key) DO NOTHING`（sqlite：`INSERT OR IGNORE`）。

**核对方法（实现时必须跑，禁止手抄漏 key）**

1. 金标准二进制空库跑完 76 链，导出 `w_system_configs.key` 集合 G。
2. Wavelet 新库（仅平台插件）导出集合 W。
3. OF `00001` 插入的 key 集合 S 必须满足 `S = G \ W`（或 `S ⊇ G \ W` 且不含 W 中的 key）。
4. `w_schedules.task_type`：金标准减去 Wavelet 的 `system_cleanup` 等，等于 5.2 四条（无 `of_database_auto_cleanup`）。

### 5.4 新装 vs 升级

| 路径 | `00001` |
| :--- | :--- |
| 新装 | 建 of_* 表 + 插入 5.2/5.3 种子 |
| 升级（已 stamp `server=1`） | 不跑 `00001`，现网数据不动 |

ClickHouse：现有 `of_node_*` DDL 迁入 `migrate/clickhouse/`，无业务种子行；不并入 `w_schema_versions`。

---

## 6. 验证

- Wavelet：`go test ./plugins/domain/cap/ ./plugins/domain/system/`；全仓 `go test ./...`。
- OpenFlare：`go test ./cmd/ ./OpenFlare/...`；parity 含新路径、不含已删路径。
- 前端：`cap-solver.ts` 仅请求 `/api/v1/cap/*`；`git diff` 允许 `frontend/` 中与 cap/文档探针相关的文件。
- 新装 sqlite：`of_*` 表齐全；`w_schedules` 含 5.2；`w_system_configs` 差集核对通过。
- 金标准库升级：stamp 后无 76 链重放；种子行不被覆盖。
- `rg 'package migrator|admin/updater|/api/cap[^\w]|GET /api/health[^\w]'` 在 OF server 与前端 cap-solver 中无旧引用（测试夹具除外）。

---

## 7. 明确不做

- 重组 `openflare/` 业务子域。
- 把 OF 种子写进 Wavelet `admin` 迁移。
- 恢复 76 文件或第二套 migrator。
- 改 agent/relay/flared CLI。
- 把 `/api/healthz` 改成 `response.OKNil()` 信封。
- 本轮不处理 `/websites/:id` 嵌入回落。

---

## 8. 完成定义

- 浏览器/前端只打 `/api/v1/cap/{challenge,redeem}`；探针文档只写 `/api/healthz`。
- 进程路由表无 `/api/cap/*`、`/api/health`、`/healthz`。
- `server/` 无 `admin/`、`migrator/`、独立 `stamp/`/`chmigrate/`/`migrations/`。
- 新装同时具备 of_* 表与 OF 种子；升级不丢数据。
