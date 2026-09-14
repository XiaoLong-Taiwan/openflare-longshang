# Ideas backlog (代码质量)

## 已尝试并收尾（2026-08-16 会话，14 个实验，108→8）

- 生产代码 golangci 扩展集 13 类 linter 全量清理（modernize/perfsprint/
  errorlint/canonicalheader/usestdlibvars/intrange/wastedassign/errname/
  forcetypeassert/prealloc/gosec/recvcheck/exhaustive），剩余 8 处全部为
  有据可查的刻意保留项（telegram %v、3 处嵌套 struct omitempty、
  3 处 not-found 惯例、1 处 encoding/json 接收者混合）。
- 测试代码质量维度（testifylint/usetesting/thelper）25→0。
- 前端 eslint/tsc 0。
- 修复中积累的工具经验：golangci-lint v2 `--fix` 的 import 管理不可靠，
  跑完必须 `goimports -w`；`--max-issues-per-linter=0` 才能拿到全量清单
  （默认 50 + max-same-issues=3 会掩盖重复模式）；cyclop 与 exhaustive
  有张力（显式 case 计入复杂度）。

## 未来可深化方向（均经评估）

- 测试可运行性修复：`go test ./internal/...` 目前在 main 上就有失败
  （无本地 redis、frpc 进程测试 flaky）。修复这些环境问题后，可以把
  `go test` 加入 checks.sh，解锁 paralleltest/tparallel 维度
  （t.Parallel 提速 + 正确性，目前因共享状态+不可运行而放弃）。
- frontend biome 格式漂移（76 文件）：一次性 `make format` 提交，
  与质量修复分开做，不进基准。
- fieldalignment：结构体内存布局优化，但会改变 JSON key 顺序且有
  位置字面量风险 —— 若做，需按文件人工核对，不进自动基准。
- Go 1.26 新特性扫描：`go vet` 新分析器、golangci-lint 新 linter
  （如 recvcheck 之后的 new receivers 检查）随版本跟进。
- 文档/示例代码（docs/、scripts/）质量：目前不在 golangci 范围（tests:false
  之外还有 scripts 目录），可用同一扩展集扫 scripts/ 下的 main.go。

## 会话收尾（2026-08-16，run #23 后）

- 已确认收敛：基准 5 维全下限、-race 全仓清零、双端测试全绿、发布构建可复现、
  config.example.yaml ↔ model.go 同步无漂移、无 flaky 测试。
- 明确评估为不值得做的方向：paralleltest/tparallel（共享全局状态风险）、
  fieldalignment（JSON key 顺序变化）、biome 格式漂移（纯噪声）、
  frpc/frps 慢测试注入 backoff（为省 ~40s 改生产时序逻辑，不值）。
- 未来如继续：可周期跑 `go test -race ./...` 全量（frpc/frps 慢套件）；
  或前端 a11y 用 axe 做浏览器级审计（超出 eslint 静态规则）。

## 本会话新增（runs #39-#43）

已修复：
- agent auth_cache negative 缓存无上限 → 10k 上限+过期清理（DoS 防护）
- relay/flared 与 agent 三份重复 authenticateAccessToken → 共享 agent 版（负缓存共享，DB 压力下降）
- websocket 三 hub：runWritePump 抽取、wsClientCore 嵌入（close/enqueue 单份）、broadcastAgent 合并
- frps/frpc TOML 注入 → pkg/protocol/toml.go TOMLQuote 转义全部插值

评估后不修/暂缓：
- cloudflare listMemberItems、config_version snapshot 证书循环的 N+1：管理端小 N 低频，
  加批量 repo API 属投机优化；若未来组员数量变大再做 ListZoneDomainsByIDs。
- fatcontext ×3（oauth/upload/auth_source cache listener）：别名赋值误报，非嵌套包装。
- objectstore newOSSBackend/newWebDAVBackend 恒 nil error：跨后端工厂签名统一，刻意设计。
- edge/updater assetNameForGOOSGOARCH 恒 "linux"：跨平台预留参数，刻意泛化。
- agent ResolverDirective explicitResolvers 原样插入 nginx conf：管理员配置属可信输入；
  若未来开放给低权限角色需加格式校验（IP 解析）。
- pkg/render/openresty 管理端旋钮（ClientMaxBodySize 等）原样插值：管理员权限范围内。
- frontend/settings/profile.tsx（858 行）超 AGENTS.md ~600 行指引：存量组件，拆分属
  纯重构无质量增益，暂缓；若后续要改该页面功能时顺手拆 components/。

## Run #44（全仓 -race 扫描）

- 发现并修复 upload/cache 监听器 DATA RACE：goroutine 读可变全局 db.Redis vs
  testhelper 清理置 nil。根因修复=启动时捕获 redisClient（oauth×2/repository×2
  同型监听器一并加固），StopUploadMetaCacheListener 补 done 等待。
- 教训：testhelper 不能 import upload/cache（循环依赖）；"捕获替代全局读"是
  无环的根因修法。
- 全仓 -race 现为 0 竞争（internal/... + pkg/...）；建议周期性重跑。

## LIKE 转义（本轮已修日志搜索 4 站点；同类遗留）

- 已修：analytics/node_access_log_filter.go、analytics/access_log_filter.go、
  logstore/postgres_store.go×2（PG/SQLite 加 ESCAPE '\'，CH 用默认反斜杠转义）。
  新助手 pkg/util/like.go EscapeLike + 单测。
- Run #47 已收尾全部 GORM 站点：upload.go keyword、user.go:73/76/188/229
  （含 OAuth uniqueUsername base 转义——外部输入含 _ 曾误报用户名冲突）、
  task_execution.go task_type 前缀。均加显式 ESCAPE '\'。
- 刻意保留：upload.go:199 `image/%`（系统常量）、config_version.go:65（系统生成）。

## Run #48（后台 goroutine panic 防护，55db1c01）

- 全仓 20 处裸 go func() 零 recover → 新增 pkg/util/goroutine.go `Go(fn)`（recover +
  slog + debug.Stack，runtime.Caller 自动记录调用点无需手写名字），22 个站点全部收口
  （oauth/upload/system_config/auth_source 的嵌套 ctx-done watcher 也含）。
- 教训：脚本括号深度匹配首轮会跳过嵌套内层 goroutine，需跑两轮；新 Go 文件必须先跑
  scripts/update_go_license.sh（license-check 会拦）。
- 已过期记录：go test ./internal/... ./pkg/... 现全过（94 ok）——"main 上测试失败"
  不再成立。scripts/、docs/ 下 Go 文件用扩展 linter 扫过：0 issues。

## Run #50（发现型 linter 扫描，全证伪——勿重跑这些维度）

- errchkjson 12 处：全部为不可能失败的 json.Marshal（纯 string/int/[]string
  结构体；admin/logs/routers.go:131 与 waf/ip_group_sync.go:255 的 "unsafe type"
  是传递性保守标记，RawMessage/time.Time 内容来自必然成功的 marshal）。
- spancheck 1 处（pkg/trace/trace.go:61）：误报，helper 正常返回 span，
  唯一调用方 internal/infra/task/executor.go:242 有 defer span.End()。
- unparam ×2（objectstore oss/webdav 恒 nil error）：已在 #43 前评估为跨后端工厂签名统一。
- 性能排查：正则全部包级编译（无函数内 MustCompile）；包级 map 全为有界静态注册表；
  task AppendLog 走 DB 非内存累积；push escapeJSONString 用法正确。
- 结论：Go 静态可发现的低垂果实已穷尽。剩余方向：frontend axe a11y 浏览器级审计、
  周期性 -race 重跑（上次 #49 干净）、运维类增长审查。

## Run #54（认证页 axe a11y 审计+修复，451ce525）

已修（复扫验证生效）：
- 布局级全局：sidebar 折叠按钮 aria-label、Sidebar role=navigation（region 18 节点/页清零）、
  header Kbd 对比度 text-foreground/70、空态/错误/加载 h3→p（heading-order 清零）。
- 页面级：dashboard 4 个 Progress aria-label、users 分页 prev/next aria-label、
  admin/system 无内容 Tabs→aria-pressed 按钮组（aria-valid-attr-value critical 清零）。
- / 与 /admin/system 现 axe 0 违规。

后续可做（页面级批量，工作量大）：
- admin 数据表格行内操作图标按钮（编辑/删除）与 Switch 开关无 aria-label —— 每张管理表逐个补；
- muted 文本对比度（card description、radix tabs trigger、primary 按钮文字）—— shadcn 默认色在浅色主题下 axe 判 fail，改主题变量影响面大需设计确认。
- 审计环境复用：后端 :3100 + CONFIG_PATH=/tmp/of-audit/config.yaml（sqlite）、docker redis --network host、
  pnpm dev --port 3002 WAVELET_BACKEND_URL=:3100；admin 密码 reset-passwd 重置。注意 :3000 是生产实例勿动。

## Run #54-#55（认证页 a11y 审计，两轮 keep）

已修复（浏览器 axe 复扫验证）：
- 全局布局：sidebar 折叠按钮 aria-label、Sidebar role=navigation、header Kbd 对比度、
  dashboard Progress aria-label、分页 prev/next、空态/加载 h3→p、admin/system Tabs→aria-pressed。
- 主题级根因：--primary indigo-500(#6366f1) 白字对比度仅 4.27(AA 需 4.5) → indigo-600
  oklch(51.1% 0.262 276.966) ≈6.8，一处修复全站 contrast 清零。
- 控件名：access-analytics 刷新、events-tab Switch/编辑/删除、openflare-ops Switch/Select/
  Input(htmlFor)/Textarea、table-browser/sql-console SelectTrigger；heading-order：眉题
  h4→p(cache-manager/user-detail-sheet)、卡片题 h3→p(task-manager/file-manager)。
- 结果：dashboard、admin/system、admin/settings、admin/logs、admin/push、admin/tasks、
  admin/database、files 共 8 页 axe 0 违规。

审计方法（可复用）：后端 :3100（CONFIG_PATH=/tmp/of-audit/config.yaml，sqlite，
api_prefix 必须显式 /api）+ docker redis --network host（本机 bridge NAT 坏）+
pnpm dev --port 3002 WAVELET_BACKEND_URL=:3100 + admin 密码经 reset-passwd 重置。
axe 注入：eval 建 CDN script → Promise 轮询 window.axe → axe.run。
教训：表单页异步渲染，须 wait≥5s 再扫否则漏报 label 规则；Radix SelectValue
value='' 时 placeholder 不显示，combobox 无名需 aria-label 兜底。

## 剩余可做

- 抽查其余页面（websites/[zoneId]、origins/detail、responses 编辑器等富交互页）
  ——contrast 已由主题修复覆盖，预期只剩个别控件名。
- 周期性 go test -race ./... 全量重跑（上次干净为 run #49 后）。

## Run #56（富交互页抽查，keep，63e3b852）

- 扫描 11 页：websites/origins/proxy-routes/certificates/dns-accounts 直接 0 违规
  （indigo-600 主题修复已覆盖全站 contrast）。
- 修复 3 处并复扫归零：
  1. cloudflare/components/sync-tasks-panel.tsx 状态筛选 SelectTrigger 加 aria-label
     （Radix SelectValue value='' 时 placeholder 不渲染，combobox 无名）。
  2. components/common/settings/access-token.tsx 安全提示 text-amber-600→amber-700
     （12px 小字对比度不足）。
  3. settings/notifications 面包屑页缺 h1 → sr-only h1。教训：h1 不能作为
     BreadcrumbList 子元素（axe list 规则报 list 语义破坏），须放 <Breadcrumb> 外；
     BreadcrumbPage 无 asChild 支持。
- a11y 维度至此穷尽：累计 14 页 axe 全部 0 违规。

## Run #59（-shuffle=on 测试顺序随机化扫描，keep，b56f2763）

- 新维度：`go test -shuffle=on` 抓到 config_version 包测试顺序依赖——
  TestBuildOpenRestyConfigSnapshotOriginErrorPageDefaults 在 shuffle 下命中
  Custom 用例留在进程级 RAM 配置缓存的值（GetSystemConfigByGroup 未命中时
  ram.Set 回填，TTL 跨测试存活；:memory: DB + SetDB 换库不使缓存失效）。
- 修复：setupOriginErrorPageSnapshotDB / setupConfigVersionTestDB 换 DB 前后
  接入既有 ram.ResetForTest()。包内 shuffle×8 + 全仓 shuffle 复扫全过。
- 教训：默认源码顺序掩盖顺序依赖；-shuffle=on 是低成本周期扫描手段。
  全仓 -race（#58 后）同样干净。其余用 SetDB 的测试包如后续 shuffle 复发，
  同法接入 ResetForTest 即可。
