# API 收口与 server 目录整理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 去掉 cap/health 双路径，前端改打 `/api/v1/cap` 与 `/api/healthz`；把 OpenFlare `server` 收成单一 `migrate/` + `updater/`，压缩 `00001` 含 of_* 建表与 OF 种子。

**Architecture:** 先在 Wavelet `feat/cordis-alignment` 删别名路由，再 merge 进 OpenFlare；然后改前端与测试；最后用金标准差集生成种子、合并迁移包并删除 76 文件。

**Tech Stack:** Go 1.25、Gin、goose、Next.js 前端 `cap-solver.ts`。

**规格来源:** `docs/superpowers/specs/2026-08-31-api-unify-and-server-layout-design.md`

## Global Constraints

- cap 只留 `/api/v1/cap/*`；删除 `/api/cap/*`。
- health 只留 `GET /api/healthz`（JSON `{status: ok}`）；删除 `/healthz` 与 `/api/health`。
- 路由改在 Wavelet 插件，禁止在 OF `server` 再挂别名。
- `00001` = of_* DDL + OF 种子；种子 key = 金标准 `w_system_configs` 减 Wavelet 平台种子。
- 禁止把 OF 种子写进 Wavelet admin 迁移。
- 禁止 `DROP TABLE` / `DROP COLUMN` / `TRUNCATE`。
- 76 历史 SQL 从仓库删除。
- 不重组 `openflare/` 业务子域；不改 agent CLI。
- 金标准 `/Users/ryan/Code/Go/OpenFlare` 只读。

---

## 文件结构

| 仓库 | 路径 | 职责 |
| :--- | :--- | :--- |
| Wavelet | `backend/plugins/domain/cap/plugin.go`、`handlers.go`、测试 | 去掉 `/api/cap` |
| Wavelet | `backend/plugins/domain/system/plugin.go`、`health_route_test.go` | 只留 `/api/healthz` |
| Wavelet | `backend/plugins/domain/auth/plugin.go` 白名单 | 与 cap 路径一致 |
| OpenFlare | `frontend/lib/cap-solver.ts`、README | 请求 v1 cap / healthz |
| OpenFlare | `backend/cmd/app_test.go`、parity | 新路径断言 |
| OpenFlare | `backend/OpenFlare/plugins/server/migrate/` | stamp + SQL + CH |
| OpenFlare | `backend/OpenFlare/plugins/server/updater/` | 原 admin/updater |

---

### Task 1: Wavelet cap 只留 `/api/v1/cap`

**Files:**
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/cap/plugin.go`（约 82–93 行）
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/cap/handlers.go`（`@Router`）
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/cap/plugin_captcha_contract_test.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/auth/plugin.go` 白名单（若仍列 `/api/cap`）

**Interfaces:**
- Consumes: 现有 `Challenge`/`Redeem`、`CaptchaService`
- Produces: 路由仅 `GET/POST /api/v1/cap/challenge`、`POST /api/v1/cap/redeem`

- [ ] **Step 1: 改测试为「必须有 v1、禁止 legacy」**

在 `plugin_captcha_contract_test.go` 的路由集合里：

```go
want := map[string]bool{
    "GET /api/v1/cap/challenge":  false,
    "POST /api/v1/cap/challenge": false,
    "POST /api/v1/cap/redeem":    false,
}
// after scanning routes, all want values true
for _, rd := range ctx.Router().Routes() {
    key := rd.Method + " " + rd.Path
    if key == "POST /api/cap/challenge" || key == "POST /api/cap/redeem" {
        t.Errorf("legacy route must not exist: %s", key)
    }
}
```

- [ ] **Step 2: 跑测试确认仍看到 legacy（或改完测试后 FAIL 在「缺少禁止断言」之前）**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/cap/ -run TestApplyRegistersUnversionedCapRoutes -count=1
```

Expected: FAIL（测试改为禁止 legacy 后，现实现仍注册 `/api/cap`）。

- [ ] **Step 3: 删除 legacy 组**

`plugin.go` 删除：

```go
legacy := ctx.Router().Group("/api/cap")
legacy.POST("/challenge", Challenge)
legacy.POST("/redeem", Redeem)
ctx.Router().RegisterWhitelist("/api/cap/challenge", "/api/cap/redeem")
```

改为：

```go
ctx.Router().RegisterWhitelist("/api/v1/cap/challenge", "/api/v1/cap/redeem")
```

（若 Group 已能白名单则只保留 v1 组 + RegisterWhitelist v1 路径。）

`handlers.go`：`@Router /api/v1/cap/challenge [post]`、`@Router /api/v1/cap/redeem [post]`（GET challenge 同前缀）。

`auth/plugin.go` 的 `publicEndpoints` 去掉任何 `/api/cap` 非 v1 项。

- [ ] **Step 4: 测试通过**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/cap/ ./plugins/domain/auth/ -count=1
```

Expected: PASS。

- [ ] **Step 5: 提交（Wavelet `feat/cordis-alignment`，不要提交 main）**

```bash
cd /Users/ryan/Code/Go/Wavelet
git add backend/plugins/domain/cap backend/plugins/domain/auth
git commit -m "fix(cap): keep only /api/v1/cap routes"
```

---

### Task 2: Wavelet health 只留 `/api/healthz`

**Files:**
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/system/plugin.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/system/health_route_test.go`
- Modify: auth 白名单中的 `/healthz`（改为 `/api/healthz`）

**Interfaces:**
- Produces: 唯一健康路由 `GET /api/healthz` → `gin.H{"status":"ok"}`

- [ ] **Step 1: 改 health_route_test**

```go
func TestHealthzIsTheOnlyHealthRoute(t *testing.T) {
    ctx := core.NewContext(context.Background())
    if err := New().Apply(ctx); err != nil {
        t.Fatal(err)
    }
    var hasHealthz bool
    for _, rd := range ctx.Router().Routes() {
        key := rd.Method + " " + rd.Path
        switch key {
        case "GET /api/healthz":
            hasHealthz = true
        case "GET /healthz", "GET /api/health":
            t.Errorf("removed health route still registered: %s", key)
        }
    }
    if !hasHealthz {
        t.Fatal("GET /api/healthz missing")
    }
    if !ctx.Router().IsWhitelisted("/api/healthz") {
        t.Fatal("GET /api/healthz not whitelisted")
    }
}
```

用 httptest 调 `/api/healthz`，body 含 `"status":"ok"`。删除对 `Health`/`OKNil`/`/api/health` 的旧测试。

- [ ] **Step 2: 跑测试确认 FAIL**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/system/ -run TestHealth -count=1
```

Expected: FAIL（仍注册 `/healthz` 或 `/api/health`）。

- [ ] **Step 3: 实现**

`plugin.go` 健康检查块改为：

```go
ctx.Router().GET("/api/healthz", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
})
ctx.Router().RegisterWhitelist("/api/healthz")
```

删除 `/healthz`、`/api/health` 注册。若 `Health` 函数不再被引用则删除（含 swagger 注释）。

auth 白名单：`/healthz` → `/api/healthz`。

- [ ] **Step 4: 测试**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/system/ ./plugins/domain/auth/ ./plugins/domain/cap/ -count=1
go test ./...
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd /Users/ryan/Code/Go/Wavelet
git add backend/plugins/domain/system backend/plugins/domain/auth
git commit -m "fix(system): expose only GET /api/healthz"
```

---

### Task 3: Merge Wavelet 路由进 OpenFlare

**Files:**
- Modify: OpenFlare `backend/{core,pkg,plugins}` via merge

**Interfaces:**
- Consumes: Wavelet `feat/cordis-alignment` HEAD（含 Task 1–2）
- Produces: OF `plugins/domain/cap` 与 `system` 与 Wavelet 零差

- [ ] **Step 1: merge**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis
git fetch wavelet
git merge wavelet/feat/cordis-alignment
```

冲突：`backend/{core,pkg,plugins}` 以 Wavelet 为准；`frontend/`、`backend/OpenFlare/`、`docs/superpowers/` ours。不要 rebase。若尚未配置 `include.path`，用 `git -c merge.ours.driver=true merge ...`。

- [ ] **Step 2: 确认路由来自 Wavelet**

```bash
rg -n 'Group\("/api/cap"\)|GET\("/api/health"|GET\("/healthz"' backend/plugins/domain
```

Expected: 无匹配。

- [ ] **Step 3: `cd backend && go test ./plugins/domain/cap/ ./plugins/domain/system/ -count=1`**

Expected: PASS。`./cmd/` 可能因旧断言 FAIL，Task 5 修。

- [ ] **Step 4: 提交 merge**（若 merge 已产生 commit 则无需空提交）

---

### Task 4: 前端 cap 与文档探针

**Files:**
- Modify: `frontend/lib/cap-solver.ts`（约 138、174 行）
- Modify: `README.md`、`README_zh.md`（健康检查 URL）
- Modify: 其它文档中的 `/api/health`、`/api/cap`（`rg` 列出后改产品文档，不改 `docs/superpowers` 历史 spec 原文）

**Interfaces:**
- Produces: 浏览器只请求 `/api/v1/cap/challenge` 与 `/api/v1/cap/redeem`

- [ ] **Step 1: 改 cap-solver.ts**

```ts
const challengeRes = await fetch('/api/v1/cap/challenge', {
```

```ts
const redeemRes = await fetch('/api/v1/cap/redeem', {
```

- [ ] **Step 2: 健康检查文档**

把 `http://localhost:8000/api/health` 换成 `http://localhost:8000/api/healthz`。

```bash
rg -n '/api/cap|/api/health[^\w]|/healthz' frontend README.md README_zh.md docs --glob '!docs/superpowers/**'
```

产品文档中的旧路径改完。`docs/superpowers` 历史设计稿保留原文。

- [ ] **Step 3: 确认 cap-solver 无旧路径**

```bash
rg -n '/api/cap/' frontend/lib/cap-solver.ts
```

Expected: 无输出。

- [ ] **Step 4: 提交**

```bash
git add frontend/lib/cap-solver.ts README.md README_zh.md
git commit -m "fix(frontend): call /api/v1/cap and document /api/healthz"
```

---

### Task 5: OpenFlare 测试与 swagger 跟随新路径

**Files:**
- Modify: `backend/cmd/app_test.go`
- Modify: parity 测试（`backend/cmd/parity_test.go` 或现文件）若硬编码 `/api/health`、`/api/cap`
- Modify: `docs/swagger.json` / yaml 经 `make swagger`

**Interfaces:**
- Consumes: Task 3 的 Wavelet 路由
- Produces: 断言 `POST /api/v1/cap/challenge`、`GET /api/healthz`；禁止旧路径

- [ ] **Step 1: 改 app_test 必有路径列表**

```go
must := []string{
    "GET /api/healthz",
    "POST /api/v1/cap/challenge",
}
forbidden := []string{
    "GET /api/health",
    "GET /healthz",
    "POST /api/cap/challenge",
}
```

parity：金标准子集里去掉已废弃路径，或对这三条做 forbidden 覆盖（不要因为金标准 `routes-engine.txt` 仍含 `/api/health` 而失败——该夹具是 v3.5.4 历史，本任务**有意改契约**。改 parity：加载 baseline 后 `delete` 这几条再做 ⊆ 断言，并额外要求新路径存在）。

- [ ] **Step 2: `cd backend && go test ./cmd/ -count=1`**

Expected: PASS。

- [ ] **Step 3: `make swagger`，确认 paths 无 `/api/cap`、`/api/health`、`/healthz`**

```bash
python3 -c "import json;d=json.load(open('docs/swagger.json'));
print('\n'.join(p for p in sorted(d.get('paths',{})) if 'cap' in p or 'health' in p))"
```

Expected: 只有 `/api/v1/cap/...` 与 `/api/healthz`（若 swagger 扫到 healthz）。

- [ ] **Step 4: 提交**

```bash
git add backend/cmd docs/swagger.json docs/swagger.yaml backend/docs
git commit -m "test(cmd): require v1 cap and /api/healthz only"
```

---

### Task 6: 生成 OF 种子差集并写入 `00001`

**Files:**
- Modify: `backend/OpenFlare/plugins/server/migrations/sqlite/00001_initial.sql`
- Modify: `backend/OpenFlare/plugins/server/migrations/postgres/00001_initial.sql`
- Create: `backend/OpenFlare/plugins/server/migrate/seed_diff_test.go`（或 Task 7 搬家后再放 `migrate/`；本任务可先写在 `migrations` 旁的测试，Task 7 一起搬）

**Interfaces:**
- Consumes: 金标准 `/Users/ryan/Code/Go/OpenFlare` @ `9f79fb99`；Wavelet admin `00001` 种子
- Produces: `00001` 含 DDL + `w_schedules` 四条 + `w_system_configs` 差集

- [ ] **Step 1: 导出差集（只读金标准，临时目录）**

金标准库 G：

```bash
GOLD=/Users/ryan/Code/Go/OpenFlare
TMP=$(mktemp -d)
# 按 Task 11/15 已验证方式：拷 config 到 TMP，SQLITE 指向 $TMP/gold.db，跑 gold api 直到 goose 202608090003
sqlite3 "$TMP/gold.db" "SELECT key FROM w_system_configs ORDER BY key;" > /tmp/keys-G.txt
sqlite3 "$TMP/gold.db" "SELECT task_type FROM w_schedules ORDER BY task_type;" > /tmp/sched-G.txt
```

Wavelet 种子 W：从 `Wavelet/backend/plugins/domain/admin/migrations/sqlite/00001_initial.sql` 解析 `INSERT INTO w_system_configs` 的 key 列表（或空库只跑 Wavelet 插件）。`G \ W` 写入计划检查清单。

定时任务：G 中应有 `of_ssl_renew`、`of_waf_ip_group_sync`、`of_uptime_kuma_sync`、`of_pages_source_scan`；**无** `of_database_auto_cleanup`。`system_cleanup` 属于 W，不要插入。

- [ ] **Step 2: 写失败测试**（新装 sqlite：跑 server 迁移后断言种子）

最小：解析 `00001_initial.sql` 文本：

```go
func TestInitialSQLContainsScheduleSeeds(t *testing.T) {
    b, err := os.ReadFile("sqlite/00001_initial.sql") // 相对测试包
    if err != nil { t.Fatal(err) }
    s := string(b)
    for _, needle := range []string{"of_ssl_renew", "of_waf_ip_group_sync", "of_uptime_kuma_sync", "of_pages_source_scan"} {
        if !strings.Contains(s, needle) {
            t.Errorf("00001 missing schedule seed %s", needle)
        }
    }
    if strings.Contains(s, "of_database_auto_cleanup") {
        t.Error("must not seed of_database_auto_cleanup")
    }
}
```

另测：SQL 含 `INSERT` 且含至少一个差集 key（例如 `agent_discovery_token`、`geoip_provider`）。

- [ ] **Step 3: 跑测试确认 FAIL（当前 00001 无 INSERT）**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./OpenFlare/plugins/server/migrations/ -count=1
```

若测试放在 stamp 包，改路径。Expected: FAIL missing `of_ssl_renew`。

- [ ] **Step 4: 在 sqlite 与 postgres `00001` 末尾追加种子**

sqlite 示例：

```sql
INSERT OR IGNORE INTO w_schedules (id, name, task_type, cron, payload, is_active, created_at, updated_at)
VALUES
    (101, 'OpenFlare SSL 自动续期', 'of_ssl_renew', '0 0 * * *', '{}', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (103, 'OpenFlare WAF IP 组同步', 'of_waf_ip_group_sync', '*/5 * * * *', '{}', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (104, 'OpenFlare Uptime Kuma 同步', 'of_uptime_kuma_sync', '* * * * *', '{}', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO w_schedules (name, task_type, cron, payload, is_active, created_at, updated_at)
SELECT 'OpenFlare Pages 部署源扫描', 'of_pages_source_scan', '0 0 * * *', '{}', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM w_schedules WHERE task_type = 'of_pages_source_scan');
```

postgres 用 `INSERT … ON CONFLICT (id) DO NOTHING`；pages scan 用 `WHERE NOT EXISTS`。

`w_system_configs`：对每个 `G \ W` 的 key，用历史链**最终** default（`create_of_options` + 后续 `202608*` 文件的字面量，**不要** `SELECT FROM of_options`）。`INSERT OR IGNORE` / `ON CONFLICT (key) DO NOTHING`。

禁止插入 Wavelet 已有 key（`cap_login_enabled`、`smtp_host`、`storage_config` 等）。禁止 `DROP`。

把 Step 1 的 `G \ W` 全写进 SQL。测试增加：每个差集 key 都出现在 sqlite 与 postgres 两个文件中。

- [ ] **Step 5: 测试通过并提交**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./OpenFlare/plugins/server/... -count=1
git add backend/OpenFlare/plugins/server/migrations
git commit -m "fix(server): seed OpenFlare schedules and configs in 00001"
```

---

### Task 7: 合并 `migrate/`、`updater/`，删除历史链

**Files:**
- Create: `backend/OpenFlare/plugins/server/migrate/{stamp.go,stamp_test.go,clickhouse.go,clickhouse_test.go,postgres/,sqlite/,clickhouse/}`
- Create: `backend/OpenFlare/plugins/server/updater/*`（从 `admin/updater` git mv）
- Modify: `plugin.go` embed 与 import
- Modify: `router/v1/openflare/register_updater.go` import
- Delete: `admin/`、`migrator/`、`stamp/`、`chmigrate/`、`migrations/`（内容已迁走后）
- Delete: `router/root/frontend_static.go` 及无引用测试（若仍存在）

**Interfaces:**
- Consumes: Task 6 的 `00001` 文本
- Produces: `package migrate` 导出 `Legacy(ctx *core.Context) error`（现 `stamp.Legacy`）与 `UpClickHouse() error`

- [ ] **Step 1: git mv**

```bash
cd backend/OpenFlare/plugins/server
git mv stamp migrate/stamp_pkg_tmp  # 或直接 mv 文件进 migrate/ 并改 package 名为 migrate
```

推荐：把 `stamp/*.go` 的 `package stamp` 改为 `package migrate`，函数名保持 `Legacy`。`cmd` 里 `stamp.Legacy` 改为 `migrate.Legacy`。

```bash
git mv chmigrate/*.go migrate/
# CH sql
mkdir -p migrate/clickhouse
git mv chmigrate/goose/clickhouse/*.sql migrate/clickhouse/
git mv migrations/postgres migrate/postgres
git mv migrations/sqlite migrate/sqlite
git mv admin/updater updater
```

改所有 import。`plugin.go`：

```go
//go:embed migrate/postgres/*.sql migrate/sqlite/*.sql
var serverMigrations embed.FS

ctx.Migrations().Register("server", serverMigrations)
if err := migrate.UpClickHouse(); err != nil {
    return err
}
```

`cmd/app.go`：`core.WithMigrationBaseline(migrate.Legacy)`。

- [ ] **Step 2: 删除空壳与 76 文件**

```bash
git rm -r migrator admin stamp chmigrate migrations
```

确认无 `package migrator`、无 `admin/updater` import。

- [ ] **Step 3: 编译测试**

```bash
cd backend && go test ./cmd/ ./OpenFlare/... -count=1
rg -n 'package migrator|admin/updater|OpenFlare/plugins/server/stamp"' .
```

Expected: 测试 PASS；rg 无产品引用。

- [ ] **Step 4: 提交**

```bash
git add -A backend/OpenFlare backend/cmd
git commit -m "refactor(server): merge migrate package and drop unused admin/migrator trees"
```

---

### Task 8: 全量门禁

**Files:** 无新功能；修到绿

- [ ] **Step 1:** `cd backend && go test ./...`

Expected: PASS。

- [ ] **Step 2:** 新装冒烟（可选 sqlite）：空库 `newOpenFlareApp` Prepare 后

```sql
SELECT COUNT(*) FROM sqlite_master WHERE name LIKE 'of_%';
SELECT task_type FROM w_schedules WHERE task_type LIKE 'of_%' ORDER BY 1;
```

Expected: 含 5.2 四条 task_type；无 `of_database_auto_cleanup`。

- [ ] **Step 3:** 升级测试仍过（`go test ./cmd/ -run TestUpgradeFromGolden -count=1 -timeout 120s`）

Expected: PASS（stamp 跳过 `00001`，数据仍在）。

- [ ] **Step 4:** `git diff --name-only` 确认 frontend 只有 cap/文档；金标准 `git status --short` 空。

有失败则修并提交 `fix(cordis): satisfy api unify gates`。

---

## 自检（对照 spec）

| Spec | Task |
| :--- | :--- |
| D1 cap 只 v1 | 1、4、5 |
| D2 health 只 `/api/healthz` | 2、4、5 |
| D3 改 Wavelet 再进 OF | 1–3 |
| D4 删 76 文件 | 7 |
| D5/D6 建表+差集种子 | 6 |
| D7 updater 路径 | 7 |
| D8 单一 migrate/ | 7 |
| 前端 cap | 4 |
| 新装/升级 | 6、8 |

无 TBD。`migrate.Legacy` / `UpClickHouse` 名称在 Task 7 定义，cmd 同步改 import。
