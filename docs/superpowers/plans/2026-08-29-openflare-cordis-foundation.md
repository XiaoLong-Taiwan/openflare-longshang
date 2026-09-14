# OpenFlare Cordis 基础改造实施计划（P0–P3）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 OpenFlare 后端迁入 `backend/` 双 module 结构、以 vendoring 方式引入 Wavelet Cordis 上游与 `share` 共享层，并在不改动任何已部署数据库语义的前提下完成 per-plugin 迁移与版本 stamp 桥接。

**Architecture:** `backend/`（module `OpenFlare`）持有 4 个业务插件与装配根；`backend/Wavelet/`（module `Wavelet`）为上游原样拷贝，`replace` + `go.work` 实现单命令构建。迁移引擎自持 `w_schema_versions(plugin_id, version_id)`，历史 76 链以 plugin id `openflare/legacy` 保真注册，桥接逐行搬运 `goose_db_version` 状态，并保留 Go 侧 zone 导入钩子。

**Tech Stack:** Go 1.25、Gin、GORM、PostgreSQL/SQLite、ClickHouse（独立链，不动）、goose v3、Cobra、Viper。

**规格来源:** `docs/superpowers/specs/2026-08-29-openflare-cordis-refactor-design.md`

**后续计划:** P4（server 插件）、P5（agent/relay/flared 插件）、P6（平台分叉回流）、P7/P8（工具链与加固）各出独立 plan，依赖本 plan 产出的骨架与门禁。

---

## 进度与布局修正（2026-08-29 执行中更新）

**已完成**
- Task 1 基线：`docs/superpowers/specs/baseline/{schema-A.sql,versions-A.txt,routes.txt}`，由 `backend/OpenFlare/infra/persistence/migrator/legacy_dump_test.go` 产出（`OF_DUMP_SCHEMA` / `OF_DUMP_VERSIONS` 触发，A 路 43 表、版本止于 `202608090003`）。
- Task 2 布局：Go 树已入 `backend/`（提交 `f6dd7988`）。

**Task 3 / Task 4 作废原命令，改为**：与 Wavelet **同构的单 module**（提交 `9bf0b2de`）——`backend/{core,pkg,plugins}` 为上游拷贝，下游业务整体位于 `backend/OpenFlare/`（占据上游 `backend/downstream` 的位置），模块名保持 `Wavelet`。因此不再有 `go.work`、`replace`、第二 go.mod。

- Task 3 剩余动作：仅需 `scripts/sync-upstream.sh`（覆盖 `backend/{core,pkg,plugins}`，排除 `share`、`OpenFlare`）与上游 `scripts/check_cordis_architecture.sh` 接入。
- Task 4 改为：新建 `backend/share/`（import `Wavelet/share/...`）并写入所有权声明；把跨插件共享资源 `pkg/{protocol,geoip,wsclient}`、`apps/edge/logging` 移入；退役与上游重复的 `OpenFlare/pkg/{logger,trace,mail,httppool}` 与 `OpenFlare/{buildinfo,testhelper}`（已核实上游覆盖除 `util` 与 `testhelper.SetupLogStoresForTest` 外的全部符号）；`OpenFlare/pkg/util` 的 8 个仅存函数与 `IdentifiableTimeRecord`/`VersionInfo` 合并进上游 `pkg/util`。
- 已实测门禁：`go build ./...` 通过、`go test ./...` exit 0（142 包 ok）、`make swagger` exit 0（232 条操作与基线逐条一致）、`make build-all` 四进制产出、`frontend/` 零改动。
- 遗留待办：swaggo 暂 `--exclude plugins,core`，Task 4/P4 需连带解除该排除。

## 关键事实（执行前必读，均已核实）

- 历史链：`internal/infra/persistence/migrator/goose/{postgres,sqlite}` 各 76 个文件，终版 `202608090003_add_log_indexes.sql`；版本表为 goose 默认 `goose_db_version`。ClickHouse 链独立：`goose/clickhouse` 14 文件 + `goose_clickhouse_version`。
- **历史链含 Go 侧数据迁移**：`migrator.Migrate()`（`internal/infra/persistence/migrator/migrator.go:91-103`）先 `goose.UpTo(202607120002)`，再在版本窗口 `[202607120002, 202607130001)` 内执行 `zone.ImportLegacyTx`，最后 `goose.Up()`。改造后该钩子必须仍然生效，且判断依据改为 `w_schema_versions` 中 `openflare/legacy` 的 max version。
- 另有 PG 序列修复 `resyncGooseVersionSequence`（同文件 `:158-183`）与迁移后 `clearSystemConfigCache`（`:185-189`），两者必须保留。
- 上游迁移引擎实现位于 `Wavelet/backend/cmd/app.go:112-360`（`sharedStore` + `gooseEngine`），`core.MigrationEngine` 为公开接口，OpenFlare 需自持一份实现（放 `OpenFlare/internal/platform/migration`），不可依赖上游 `cmd` 包内未导出符号。
- `MigrationEntry.Dir` 被上游引擎忽略；目录基名必须为 `postgres`/`sqlite`。goose 会给每个 plugin_id 插入哨兵 `version_id=0`。
- 内核：`core.Profile("agent")` 合法（未知 profile 原样透传）；`App.Run` 始终阻塞在信号；长生命周期服务用 `ctx.RegisterDriver` + `ctx.OnDispose`。
- 全仓 476 个 `.go` 文件引用 `github.com/Rain-kl/Wavelet`，且**无任何非 import 出现**（已验证），模块路径替换可全局进行。
- 前端 `frontend/` 禁止改动；`//go:embed all:dist` 位于 `internal/router/root/frontend.go:18`，dist 由 `make build-embedded` 拷入。

## 文件结构（本计划涉及）

| 动作 | 路径 | 职责 |
| :-- | :-- | :-- |
| Create | `backend/`（git mv 而来） | Go 代码根 |
| Modify | `backend/go.mod` | `module OpenFlare` + `require/replace Wavelet` |
| Create | `backend/go.work` | 本地双 module 一体构建 |
| Create | `backend/Wavelet/**` | 上游拷贝（`core`、`plugins`、`pkg`、`scripts`、`go.mod`） |
| Create | `backend/Wavelet/share/{protocol,wsclient,geoip,edge}/` | 跨插件共享资源（OpenFlare 所有，同步排除） |
| Create | `backend/Wavelet/share/README.md` | 所有权与同步排除声明 |
| Create | `backend/internal/platform/migration/{store,engine,bridge,lock}.go` | 迁移引擎、桥接、锁 |
| Create | `backend/plugins/server/migrations/{postgres,sqlite}/*.sql` | 76×2 历史链原样保真 |
| Create | `backend/cmd/migrate-audit/main.go` | 三方 schema 一致性门禁工具 |
| Create | `scripts/sync-upstream.sh` | 上游同步（排除 `share/`） |
| Modify | `Makefile`、`docker/Dockerfile*`、`.github/workflows/*` | 路径切至 `backend/`、ldflags 模块路径 |

---

## Task 1: 基线快照（P0）

**Files:**
- Create: `docs/superpowers/specs/baseline/routes.txt`
- Create: `docs/superpowers/specs/baseline/go-test.txt`

- [ ] **Step 1: 导出改造前路由与测试基线**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis
mkdir -p docs/superpowers/specs/baseline
python3 -c "import json;d=json.load(open('docs/swagger.json'));print('\n'.join(sorted(f'{m.upper()} {p}' for p,v in d.get('paths',{}).items() for m in v)))" > docs/superpowers/specs/baseline/routes.txt
go test ./... > docs/superpowers/specs/baseline/go-test.txt 2>&1; echo "exit=$?"
```
Expected: `exit=0`，`routes.txt` 非空。

- [ ] **Step 2: 生成 A 路 schema 基线（改造前二进制，供 Task 9 对拍）**

```bash
go build -o /tmp/of-baseline-server .
OF_DB_PATH=/tmp/of-baseline-a.db timeout 25 /tmp/of-baseline-server api >/tmp/of-a.log 2>&1 || true
sqlite3 /tmp/of-baseline-a.db "SELECT name,sql FROM sqlite_master WHERE type='table' ORDER BY name;" > docs/superpowers/specs/baseline/schema-A.sql
sqlite3 /tmp/of-baseline-a.db "SELECT version FROM goose_db_version WHERE is_applied=1 ORDER BY version;" > docs/superpowers/specs/baseline/versions-A.txt
```
Expected: `schema-A.sql` 含 `w_users`、`of_zones`；`versions-A.txt` 末行为 `202608090003`。

- [ ] **Step 3: 提交基线**

```bash
git add docs/superpowers/specs/baseline && git commit -m "chore(cordis): 记录改造前 schema 与路由基线"
```

## Task 2: Go 树迁入 `backend/` 并改 module 名（P1）

**Files:**
- Modify: `backend/**`（全部 Go 文件）、`Makefile`、`docker/*`、`.github/workflows/*`

- [ ] **Step 1: 移动文件（保留 git 历史）**

```bash
mkdir -p backend
git mv cmd internal pkg main.go go.mod go.sum backend/
```

- [ ] **Step 2: 改 module 名并重写全部 import 路径**

```bash
perl -pi -e 's{^module github\.com/Rain-kl/Wavelet$}{module OpenFlare}' backend/go.mod
grep -rl 'github.com/Rain-kl/Wavelet' --include='*.go' backend | xargs perl -pi -e 's{\bgithub\.com/Rain-kl/Wavelet\b}{OpenFlare}g'
grep -rl 'github.com/Rain-kl/Wavelet' --include='*.yml' --include='Dockerfile*' Makefile .github docker | xargs perl -pi -e 's{github\.com/Rain-kl/Wavelet}{OpenFlare}g'
```

- [ ] **Step 3: Makefile 全部 Go 目标切到 `backend/`（对齐 Wavelet 惯例）**

`MODULE := $(shell cd backend && go list -m)`；每个含 `go ` 命令的目标前缀 `cd backend &&`；`swagger` 输出目录仍为仓库根 `docs/`（`swag init -g backend/main.go -o docs --dir backend`），确保 `frontend` 消费路径不变。

- [ ] **Step 4: 修正内嵌与 ldflags 路径**

`build-embedded` 中前端导出物拷贝目标改为 `backend/internal/router/root/dist`；`.github/workflows/build-release.yml` 三处 `-X 'OpenFlare/plugins/{agent,relay,flared}/config.Version=...'` 在 Task 12 前暂以 `OpenFlare/internal/apps/...` 为准（P5 再随插件落位改一次），并全部加 `working-directory: backend`。

- [ ] **Step 5: 验证编译与测试未回归**

```bash
cd backend && go build ./... && go test ./... 2>&1 | tail -5
```
Expected: build 无输出；test 与 `docs/superpowers/specs/baseline/go-test.txt` 同结果（exit 0）。

- [ ] **Step 6: 验证 A 路 schema 仍一致（移动未改变行为）**

重跑 Task 1 Step 2（输出改 `schema-A2.sql`、`versions-A2.txt`）并 `diff -u` 两个 schema 文件。
Expected: diff 为空。

- [ ] **Step 7: 提交**

```bash
git add -A && git commit -m "refactor(layout): Go 代码迁入 backend/ 并将模块名简化为 OpenFlare"
```

## Task 3: Vendoring 上游为第二 module（P2）

**Files:**
- Create: `backend/Wavelet/**`、`backend/Wavelet/go.mod`、`backend/go.work`、`scripts/sync-upstream.sh`

- [ ] **Step 1: 拷入上游内核与插件（排除上游装配根与运行期产物）**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend
mkdir -p Wavelet
rsync -a --delete --exclude data --exclude uploads --exclude docs --exclude cmd --exclude main.go --exclude downstream \
  /Users/ryan/Code/Go/Wavelet/backend/ Wavelet/
```
Expected: `Wavelet/core`、`Wavelet/plugins/{drivers,infra,domain}`、`Wavelet/pkg`、`Wavelet/scripts`、`Wavelet/go.mod` 存在。

- [ ] **Step 2: 建立双 module 关联**

```bash
printf 'go 1.25.7\n\nuse (\n\t.\n\t./Wavelet\n)\n' > go.work
perl -pi -e 's{^go \d.*$}{go 1.25.7\n\nrequire Wavelet v0.0.0\n\nreplace Wavelet => ./Wavelet}' go.mod
go work sync && go build ./... && cd Wavelet && go build ./... && cd ..
```
Expected: 两次 `go build ./...` 均无输出。若 `replace` 插入位置与 go.mod 现有语句冲突，手工将 `require Wavelet v0.0.0` / `replace Wavelet => ./Wavelet` 置于 `go` 指令之后。

- [ ] **Step 3: 写同步脚本（`share/` 永不覆盖）**

```bash
cat > ../scripts/sync-upstream.sh <<'SH'
#!/usr/bin/env bash
# 从 Wavelet 上游同步内核与插件；share/ 由 OpenFlare 拥有，显式排除。
set -euo pipefail
SRC="${1:-/Users/ryan/Code/Go/Wavelet/backend}"
DST="$(cd "$(dirname "$0")/../backend/Wavelet" && pwd)"
rsync -a --delete \
  --exclude 'data' --exclude 'uploads' --exclude 'docs' --exclude 'cmd' \
  --exclude 'main.go' --exclude 'downstream' --exclude 'share' \
  "$SRC/" "$DST/"
echo "synced upstream -> $DST (share/ preserved)"
SH
chmod +x ../scripts/sync-upstream.sh
```

- [ ] **Step 4: 验证 `go.mod` 中上游依赖不丢失**

`go build ./...` 若报 `missing go.sum entry`，执行 `go mod tidy && cd Wavelet && go mod tidy && cd .. && go build ./...`。
Expected: 构建通过。

- [ ] **Step 5: 提交**

```bash
git add -A && git commit -m "feat(cordis): 以第二 module 形态 vendoring Wavelet 内核与平台插件"
```

## Task 4: `share` 共享层（P2，G3）

**Files:**
- Create: `backend/Wavelet/share/{protocol,wsclient,geoip,edge}/`、`backend/Wavelet/share/README.md`

- [ ] **Step 1: 建立所有权声明**

```bash
mkdir -p Wavelet/share
cat > Wavelet/share/README.md <<'MD'
# share — 跨插件共享资源层

本目录由 **OpenFlare 拥有**，不属于 Wavelet 上游同步范围（`scripts/sync-upstream.sh` 显式排除）。

用途：被 `server` 与 `agent`/`relay`/`flared` 中两个及以上插件共同消费、且无法放入
`Wavelet/pkg/`（禁止 import 业务代码）或插件内部（禁止插件互相 import）的资源。
MD
```

- [ ] **Step 2: 迁移四个真正跨插件的资源包**

```bash
mkdir -p Wavelet/share/{protocol,wsclient,geoip,edge}
git mv pkg/protocol  Wavelet/share/protocol
git mv pkg/wsclient  Wavelet/share/wsclient
git mv pkg/geoip     Wavelet/share/geoip
git mv internal/apps/edge/logging Wavelet/share/edge/logging
grep -rl 'OpenFlare/pkg/protocol\|OpenFlare/pkg/wsclient\|OpenFlare/pkg/geoip\|OpenFlare/internal/apps/edge/logging' --include='*.go' . \
  | xargs perl -pi -e 's{OpenFlare/pkg/(protocol|wsclient|geoip)}{Wavelet/share/$1}g; s{OpenFlare/internal/apps/edge/logging}{Wavelet/share/edge/logging}g'
go build ./...
```
Expected: 构建通过。

- [ ] **Step 3: 消除 `pkg` 与上游 `Wavelet/pkg` 的重复**

对每个 OpenFlare 保留包，先比对符号集，再决定归属：

```bash
for p in util logger trace cache mail response idgen ginutil httppool buildinfo testhelper; do
  echo "=== $p: openflare-only symbols ==="
  comm -23 <(grep -rhoE '^func [A-Z][A-Za-z0-9_]*' OpenFlare/pkg/$p/*.go 2>/dev/null | awk '{print $2}' | sort -u) \
           <(grep -rhoE '^func [A-Z][A-Za-z0-9_]*' Wavelet/pkg/$p/*.go 2>/dev/null | awk '{print $2}' | sort -u)
done
```
- 上游同名包**已覆盖**的符号 → 删除 OpenFlare 副本，import 改指 `Wavelet/pkg/...`。
- 仅 OpenFlare 需要的符号（如 `util.Go`、`util.EscapeLike`、`util.DummyCheckPassword`）→ 追加到 `Wavelet/pkg/<p>/openflare.go`（OpenFlare 拥有该 fork，P6 回流上游），再删除 OpenFlare 副本。
- `pkg/cap`、`pkg/push`、`pkg/render`、`pkg/pagesarchive` → 与上游有交集者归 `Wavelet/pkg`，纯边缘共享者归 `Wavelet/share`。
每迁一包后 `go build ./... && go test ./...`，全部完成后提交。

- [ ] **Step 4: 验证共享层被真实使用并提交**

```bash
grep -rn 'Wavelet/share' --include='*.go' plugins internal cmd 2>/dev/null | head -5
go build ./... && git add -A && git commit -m "feat(cordis): 新增 Wavelet/share 跨插件共享层并收敛重复 pkg 实现"
```
Expected: 至少 server 与一个 daemon 路径 import `Wavelet/share/...`。

## Task 5: 迁移引擎骨架（P3）

**Files:**
- Create: `backend/internal/platform/migration/store.go`（自持 `sharedStore`）
- Create: `backend/internal/platform/migration/engine.go`（实现 `core.MigrationEngine`）
- Test: `backend/internal/platform/migration/store_test.go`

- [ ] **Step 1: 写失败测试——版本表 DDL 与上游逐字节一致**

```go
func TestSharedStoreDDLMatchesUpstream(t *testing.T) {
	for _, dialect := range []string{"sqlite3", "postgres"} {
		s := &sharedStore{pluginID: "t", dialect: dialect}
		db, err := sql.Open("sqlite", ":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		require.NoError(t, s.CreateVersionTable(context.Background(), db))
		var got string
		require.NoError(t, db.QueryRow("SELECT sql FROM sqlite_master WHERE name='w_schema_versions'").Scan(&got))
		require.Contains(t, got, "plugin_id")
		require.Contains(t, got, "version_id")
		require.Contains(t, got, "PRIMARY KEY (plugin_id, version_id)")
	}
}
```

- [ ] **Step 2: 运行验证失败**

Run: `cd backend && go test ./internal/platform/migration/ -run TestSharedStoreDDLMatchesUpstream -v`
Expected: FAIL（`sharedStore` 未定义）。

- [ ] **Step 3: 从上游移植 `sharedStore`**

将 `Wavelet/backend/cmd/app.go:142-240` 的 `sharedStore`（`Tablename`/`CreateVersionTable`/`Insert`/`Delete`/`GetMigration`/`GetLatestVersion`/`ListMigrations`/`placeholder`）整段复制到 `store.go`，仅改包名与注释；DDL 字符串**一字不动**。
Run: `go test ./internal/platform/migration/ -run TestSharedStoreDDLMatchesUpstream`
Expected: PASS。

- [ ] **Step 4: 写失败测试——历史链按 plugin id 独立计数**

```go
func TestLegacyPluginResumesFromStampedMax(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "t.db"))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	s := &sharedStore{pluginID: LegacyPluginID, dialect: "sqlite3"}
	ctx := context.Background()
	require.NoError(t, s.CreateVersionTable(ctx, db))
	require.NoError(t, s.Insert(ctx, db, goosedb.InsertRequest{Version: 202608090003}))
	got, err := s.GetLatestVersion(ctx, db)
	require.NoError(t, err)
	require.Equal(t, int64(202608090003), got)
}
```
Run: `go test ./internal/platform/migration/ -run TestLegacyPluginResumesFromStampedMax`
Expected: FAIL（`LegacyPluginID` 未定义）。

- [ ] **Step 5: 实现引擎**

```go
// LegacyPluginID 是 OpenFlare 76 个历史 goose 文件的归属标识。
const LegacyPluginID = "openflare/legacy"

const (
	zoneImportSQLVersion     int64 = 202607120002
	zoneDropLegacySQLVersion int64 = 202607130001
)
```
`engine.go` 以上游 `gooseEngine.Migrate`（`cmd/app.go:243-306`）为基线，差异仅两处：
1. 遍历 `entries` 时，`entry.PluginID == LegacyPluginID` 走三段式：`provider.Up` 到 `zoneImportSQLVersion` → 若 `store.GetLatestVersion` 落在 `[zoneImportSQLVersion, zoneDropLegacySQLVersion)` 则在事务内执行 `zone.ImportLegacyTx` → 继续 `provider.Up` 至末尾；
2. 引擎入口先调用 `resyncGooseVersionSequence(sqlDB)`（自 `migrator.go:158-183` 原样移植，改收 `*sql.DB` 与方言参数），并在全部 entry 完成后调用 `clearSystemConfigCache`。

Run: `go test ./internal/platform/migration/ -v`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add backend/internal/platform/migration && git commit -m "feat(migration): 自持 w_schema_versions 引擎并保留 zone 导入钩子"
```

## Task 6: 历史链归属与注册（P3）

**Files:**
- Create: `backend/internal/platform/migration/legacy/migrations/{postgres,sqlite}`（76×2 文件）
- Modify: `backend/plugins/server/plugin.go`（`ctx.Migrations().Register(LegacyPluginID, legacyFS)`）

- [ ] **Step 1: 原样保真移动 SQL**

```bash
mkdir -p internal/platform/migration/legacy/migrations
git mv internal/infra/persistence/migrator/goose/postgres internal/platform/migration/legacy/migrations/postgres
git mv internal/infra/persistence/migrator/goose/sqlite   internal/platform/migration/legacy/migrations/sqlite
git mv internal/infra/persistence/migrator/goose/clickhouse internal/platform/migration/legacy/migrations-clickhouse
```
Expected: `git diff --stat -M HEAD~1 -- '*/*.sql'` 显示纯 rename，零内容变更。

- [ ] **Step 2: embed 并注册**

```go
//go:embed legacy/migrations/postgres/*.sql legacy/migrations/sqlite/*.sql
var legacyFS embed.FS
```
ClickHouse 目录**不放**在 `migrations/` 下（避免被 `findMigrationFS` 的 `postgres|sqlite` 目录探测命中），继续由既有 `MigrateClickHouse` + `goose_clickhouse_version` 驱动，行为与改造前完全一致。

- [ ] **Step 3: 校验双方言文件集合一致**

```bash
diff <(ls internal/platform/migration/legacy/migrations/postgres) \
     <(ls internal/platform/migration/legacy/migrations/sqlite)
```
Expected: 无差异（否则 `version_id` 会跨方言漂移）。

- [ ] **Step 4: 构建并提交**

`go build ./... && go test ./...` → PASS 后
```bash
git add -A && git commit -m "refactor(migration): 历史 goose 链原样归入 openflare/legacy 插件标识"
```

## Task 7: 版本 stamp 桥接（P3）

**Files:**
- Create: `backend/internal/platform/migration/bridge.go`
- Create: `backend/internal/platform/migration/lock.go`
- Test: `backend/internal/platform/migration/bridge_test.go`

- [ ] **Step 1: 写失败测试——老库逐行搬运且不重复执行**

```go
func TestBridgeCopiesLegacyVersions(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "old.db"))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	_, err = db.Exec(`CREATE TABLE goose_db_version (id INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER NOT NULL, is_applied INTEGER NOT NULL, timestamp DATETIME)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO goose_db_version (version_id,is_applied) VALUES
		(202606090001,1),(202607120002,1),(202608090003,1),(202608100001,0)`)
	require.NoError(t, err)

	n, err := Bridge(context.Background(), db, "sqlite3")
	require.NoError(t, err)
	require.Equal(t, 3, n) // 仅 is_applied=1 行被搬运

	again, err := Bridge(context.Background(), db, "sqlite3")
	require.NoError(t, err)
	require.Equal(t, 0, again) // 幂等

	var maxV int64
	require.NoError(t, db.QueryRow(
		"SELECT MAX(version_id) FROM w_schema_versions WHERE plugin_id='openflare/legacy'").Scan(&maxV))
	require.Equal(t, int64(202608090003), maxV)
}
```
Run: `go test ./internal/platform/migration/ -run TestBridgeCopiesLegacyVersions`
Expected: FAIL（`Bridge` 未定义）。

- [ ] **Step 2: 实现 `Bridge`**

要点：以 `SELECT version_id FROM goose_db_version WHERE is_applied=1` 为源（表不存在则直接返回 0，视为新库）；`INSERT INTO w_schema_versions (plugin_id, version_id) VALUES (?, ?) ON CONFLICT (plugin_id, version_id) DO NOTHING` 逐行写入 `openflare/legacy`，并以 `RowsAffected` 汇总真实新增数；同时补写哨兵 `0`。占位符沿用 `sharedStore.placeholder`。**禁止**读写任何业务表。

- [ ] **Step 3: 写失败测试——迁移期互斥锁**

```go
func TestMigrationLockIsExclusive(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "l.db"))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	unlock, err := LockMigration(context.Background(), db, "sqlite3")
	require.NoError(t, err)
	_, err2 := db.Exec("INSERT INTO w_schema_versions VALUES ('x',1,'2026-01-01')")
	require.Error(t, err2) // 锁持有期间其他写者被阻塞/失败
	require.NoError(t, unlock())
	_, err3 := db.Exec("INSERT INTO w_schema_versions VALUES ('x',1,'2026-01-01')")
	require.NoError(t, err3)
}
```
Expected: 先 FAIL（`LockMigration` 未定义），实现后 PASS。

- [ ] **Step 4: 实现 `LockMigration`**

PG：`SELECT pg_advisory_lock($1)` / `pg_advisory_unlock($1)`（固定 key 常量）；SQLite：`PRAGMA busy_timeout` + 单写者事务（`BEGIN IMMEDIATE`）。返回 `func() error` 解锁，上游无 session locker，此处补齐多节点竞争保护。

- [ ] **Step 5: 接入装配根**

`Engine.Migrate` 开头：`unlock, err := LockMigration(...)`、`defer unlock()`；`Bridge(...)` 紧随 `CreateVersionTable` 之后、遍历 `entries` 之前。

- [ ] **Step 6: 全量测试并提交**

```bash
go test ./internal/platform/migration/ -race && go test ./... && git add -A && git commit -m "feat(migration): 一次性 goose_db_version 到 w_schema_versions 桥接与迁移互斥锁"
```

## Task 8: 上游插件初建与对齐（P3）

**Files:**
- Create: `backend/Wavelet/plugins/domain/*/migrations/{postgres,sqlite}/00002_openflare_align.sql`（仅在确有缺口时）
- Test: `backend/internal/platform/migration/parity_test.go`

- [ ] **Step 1: 列出对齐缺口（只读探测，不写库）**

```sql
-- 对 A 路基线库执行，逐表比对上游模型期望列
SELECT table_name, column_name FROM information_schema.columns
WHERE table_name LIKE 'w\_%' ORDER BY table_name, column_name;
```
Expected: 产出「上游需要但现库缺失」的列/表清单；`w_message_channels`、`w_message_bindings`、`w_message_pairing_codes` 应出现在“缺失表”中（OpenFlare 无 bot 渠道表）。

- [ ] **Step 2: 缺口只以追加式迁移补齐**

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE w_users ADD COLUMN IF NOT EXISTS need_change_password BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
```
`Down` 一律写成 no-op（禁止 DROP，避免回滚破坏现网数据）。sqlite 方言无 `ADD COLUMN IF NOT EXISTS`，需拆为独立文件并按上游惯例手写 sqlite 版本（不接受条件 SQL）。

- [ ] **Step 3: 一致性检查（B 路 vs A 路，复用 Task 1 的 dump harness）**

`legacydump_test.go` 通过环境变量 `OF_DUMP_SCHEMA=<输出路径>` 触发，把「当前 embed 内的全部 SQL」应用到 `t.TempDir()` 的临时 sqlite 库并导出表/索引定义。B 路 = 迁移文件已全部就位后的 dump：

```bash
cd backend
OF_DUMP_SCHEMA=/tmp/of-b.sql go test ./internal/platform/migration/legacy -run DumpFreshSchema
diff -u docs/superpowers/specs/baseline/schema-A.sql /tmp/of-b.sql
```
Run: 上述 diff 命令
Expected: 初期 FAIL 并逐表列出漂移（缺失的 `ADD COLUMN`、被 align 迁移改动的列）；补齐 align 迁移后 diff 为空。

- [ ] **Step 4: 提交**

```bash
git add -A && git commit -m "fix(migration): 追加式 align 迁移使全新安装与升级路径 schema 收敛"
```

## Task 9: 三方一致性门禁工具（P3 验收）

**Files:**
- Create: `backend/cmd/migrate-audit/main.go`
- Modify: `Makefile`（`migrate-audit` 目标）

- [ ] **Step 1: 实现只读 dump 与 diff CLI**

```go
// dump: sqlite  -> SELECT name,sql FROM sqlite_master WHERE type IN ('table','index') ORDER BY name;
//        postgres-> information_schema.columns + pg_indexes ORDER BY 1,2
// mode: A=基线文件  B=全新生成  C=恢复 A 后再跑一次
func main() {
	mode := flag.String("mode", "B", "A|B|C")
	dialect := flag.String("dialect", "sqlite3", "sqlite3|postgres")
	out := flag.String("out", "", "dump 输出路径")
	flag.Parse()
	// 1) 临时目录一律 os.MkdirTemp("", "of-audit-")，禁止写源码树
	// 2) B: 空库 -> bridge(no-op) -> Engine.Migrate(entries)
	// 3) C: 复制 A 的 db 文件 -> bridge(stamp) -> Engine.Migrate -> 断言 results 为空
	// 4) 输出规范化 dump 到 --out，交由 Makefile 的 diff 判定
}
```

- [ ] **Step 2: Makefile 目标与期望输出**

```make
migrate-audit:
	cd backend && go run ./cmd/migrate-audit --mode B --out /tmp/of-b.sql
	cd backend && go run ./cmd/migrate-audit --mode C --base docs/superpowers/specs/baseline/schema-A.sql --out /tmp/of-c.sql
	diff -u docs/superpowers/specs/baseline/schema-A.sql /tmp/of-b.sql
	@sqlite3 /tmp/of-c.db "SELECT count(*) FROM w_schema_versions" >/dev/null
	@grep -q 'no-op' /tmp/of-c.log && echo "MIGRATION PARITY OK"
```
Expected: `make migrate-audit` 两个 diff 全空并打印 `MIGRATION PARITY OK`。

- [ ] **Step 3: 提交并在 changelog 记录**

`docs/changelog/index.md` 的 `[Unreleased]` 增补：迁移改为按插件版本表 `w_schema_versions` 管理，已部署库通过一次性桥接保持不重跑。

## Task 10: 架构门禁与 CI 收口（P3 收尾）

- [ ] **Step 1: 接入上游架构检查脚本**

`make arch-check` → `bash backend/Wavelet/scripts/check_cordis_architecture.sh`，并按 OpenFlare 布局增补 `backend/plugins` 与 `backend/Wavelet/share` 两条扫描根。
Expected: 首次运行对 P3 范围（尚无 4 插件）通过。

- [ ] **Step 2: 四二进制构建回归 + code-check + format**

```bash
make build-backend && make code-check && make format
cd backend && go test -race ./... && go test -shuffle=on ./...
```
Expected: 全部 exit 0；`-shuffle=on` 无顺序依赖失败。

- [ ] **Step 3: 前端零改动断言与提交**

```bash
git diff --name-only main...HEAD -- frontend/ | tee /tmp/fe.diff
test ! -s /tmp/fe.diff && echo "FRONTEND UNTOUCHED"
git add -A && git commit -m "chore(ci): cordis 布局路径门禁与迁移一致性检查接入"
```
Expected: 打印 `FRONTEND UNTOUCHED`。

---

## 完成定义（本计划）

- `backend/` 双 module 构建通过；`Wavelet/` 与上游差异仅 `share/` 与被显式记录的回流项。
- `make migrate-audit` 三方 diff 为空；已部署库升级演练零 SQL 变更。
- `go test -race ./...`、`-shuffle=on`、`make code-check` 全绿；`frontend/` 零改动。
- 4 个插件与 daemon 内核化由 P4/P5 承接，本计划不含业务路由迁移。

---

## 附录 A：server 插件接入内核的可行路径（P4 前置调查结论）

调查上游 `core` 与 `plugins/drivers/driver_http` 后确认：`ctx.Router().Use` **不会**作用到
gin engine（只把中间件摊平进 `RouteDefinition`），且以下 OpenFlare 现有 engine 级行为
在内核中没有插件级贡献点：

| 需求 | 现状 | 出处 |
| :-- | :-- | :-- |
| `RedirectTrailingSlash = false` | 内核无处设置（全仓 0 命中） | `router/router.go:42-43` |
| 自有 CORS / logger 分级 / session cookie 语义与 fatal 策略 | 硬编码在 `driver_http/engine.go` | `router.go:45-79` |
| Next.js 静态导出 NoRoute（含动态页 shell 回退） | 被 `registerFrontend` 占用且缺该回退 | `router/root/frontend.go:104-111` |
| `http.Server` 读写字节超时 / TLS | 仅 `ReadHeaderTimeout` | `router.go:83-87` |
| 绑定成功后回调 `onStarted` | 无驱动级回调 | `router.go:89-95` |
| 路由前缀取自 `app.api_prefix` | `ctx.Router()` 只收字面量 | `config.example.yaml:18` |

**结论：不改上游即可闭合——`driver_http.New(driver_http.WithEngine(自建 engine))` 已存在。**
装配根（`backend/cmd`）负责构造 gin engine（保留 Recovery/CORS/session/otelgin/
RedirectTrailingSlash/NoRoute 与超时），把 engine 交给 http 驱动；`server` 插件的
`Apply` 只做 `ctx.Router().Group(...)` 声明式路由注册。其余两点：

- `onStarted` 改由 `ctx.Events().On("app:ready")` + `ctx.Driver(core.DriverTypeHTTP).Addr()` 提供；
  启动横幅在 `app.Run()` 前后打印，不再依赖回调。
- 免鉴权路径：**保留 OpenFlare 自有 `oauth.LoginRequired()`（不查白名单）**，语义与今天一致。
  注意内核 `ctx.Router().RegisterWhitelist` 目前是**空转**（`driver_http.SetWhitelist`/
  `IsPathWhitelisted` 无非测试调用方），不得依赖它放行——否则会 401。

待办：以上三项（engine 中间件贡献点、NoRoute 贡献点、白名单真正生效）应回流上游 Wavelet，
届时可去掉 `WithEngine` 例外。契约校验仍以 `docs/superpowers/specs/baseline/routes.txt`
的 232 条 **操作**（方法×路径，`.Any()` 计 7）为准，与代码内 215 个注册调用点不矛盾。

## 附录 B：server 插件化已就位件与剩余步骤

**已落地并验证**
- `router.BuildEngine()` 从 `Serve()` 中抽出（仅构造引擎与中间件，不监听），`Serve()` 复用它，行为不变。
- `router/routes_dump_test.go`：以 `OF_DUMP_ROUTES=<path>` 导出现存注册路径的 **(方法 路径) 全集**，
  已固化基线 `docs/superpowers/specs/baseline/routes-engine.txt`：**256 条**（含 20 条尾部斜杠变体，
  全为 `GET /api/v1/d/*`）。这是 server 插件化唯一的路由对拍基准（232 是 swagger 操作数，
  二者差异来自斜杠变体与 `Any` 展开，不是矛盾）。
- 内核补丁：`RouterExtension` 新增 `HandleRaw`（保留尾部斜杠）与 `BasePath`，作用域包装器同步实现并登记反注册；
  用例 `core/extpoints/router_raw_test.go` 覆盖；补丁登记在 `backend/OpenFlare/upstream-patches.md`，
  `sync-upstream.sh` 同步后会打印需确认的补丁文件。

**剩余步骤（按序，每步都要过对拍）**
1. `openflare/apiutil.RegisterCollection` 改用 `route.Handle(method,"")` + `route.HandleRaw(method,"/")`
   （取代 `route.BasePath()` 的 gin 用法与 `[]gin.HandlerFunc`→`[]any` 转换）。
2. `router/root`：`RegisterDefaultRootRoutes`/`RegisterCustomRootRoutes` 参数改 `core.RouterExtension`；
   `RegisterFrontend`（`NoRoute` + 静态资源，含 Next.js 动态 shell 回退）保留在引擎层，由 `BuildEngine()` 调用。
3. 33 个 `*gin.RouterGroup` 注册函数改 `core.RouterExtension`（本轮已验证方法面只需
   GET/POST/PUT/DELETE/Group/Use，且无人接收返回值），随后删除 `registerRoutes()`。
4. 新增 `OpenFlare/plugins/server/plugin.go`：`Name()="server"`，`Apply(ctx)` 内以
   `ctx.Router().Group(config.Config.App.APIPrefix)` 复刻 `registerRoutes` 的树
   （`/api` → `/v1` → v1/user/admin/openflare；根级 4 条走注册表绝对路径）。
5. 装配根切换：`cmd/{api,all}.go` 改为 `core.NewApp(core.WithProfile(...))` +
   `app.Use(server.New())` + `app.Use(driver_http.New(driver_http.WithEngine(router.BuildEngine())))`，
   启动横幅改挂 `ctx.Events().On("app:ready")` 并用 `httpDrv.Addr()` 取实际监听地址；
   `otel_trace.Shutdown`/`bootstrap.Stop` 迁入 `ctx.OnDispose`，删除 `Serve()` 自带的信号循环。
6. 对拍：新测试把插件注册表产出的 (方法 路径) 集合与 `routes-engine.txt` 逐条比对，
   差异必须为空；再实跑 server（sqlite）`curl` 健康检查、一个免鉴权端点与一个需鉴权端点（期望 401）。
