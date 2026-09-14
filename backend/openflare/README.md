# OpenFlare 下游树

本目录占据上游 `backend/downstream/` 的位置，存放 OpenFlare 的全部业务，
按功能职责拆为 **4 个插件 + 1 个共享层**。上游目录
（`backend/{core,pkg,plugins}`）通过 `git fetch wavelet && git merge wavelet/main`
吸收（第一次接线已 merge `wavelet/feat/cordis-alignment`，待该分支合入上游 main
后改走 `wavelet/main`）。本目录、`frontend/` 与 `backend/cmd` 由本仓库持有。
合并前请 `git config include.path ../.gitconfig`（或 worktree 安全写法
`git config include.path "$(git rev-parse --show-toplevel)/.gitconfig"`），
以启用 `.gitattributes` 的 `merge=ours`。

```
backend/openflare/
├── plugins/
│   ├── server/     # 控制面插件：站点/区域/Cloudflare/Pages/WAF/节点/回源/健康/配置版本
│   │   ├── openflare/  admin/  oauth/  user/  upload/  cap/  config/  health/   # 业务域
│   │   ├── repository/  model/  infra/  shared/  pkg/                           # 支撑层
│   │   └── router/  platform/  listener/  integration/  testhelper/             # 装配与接线
│   ├── agent/      # 边缘 nginx/WAF 代理守护进程插件
│   ├── relay/      # frps 中继守护进程插件
│   └── flared/     # frpc 隧道客户端守护进程插件
└── share/          # 插件间共享资源（见 share/README.md）
```

装配根在 `backend/cmd`（与上游同构）：`main.go` + `cmd/*.go` 为 server 的
api/worker/schedule/all profile 入口，`cmd/{agent,relay,flared}/main.go` 为三个
守护进程入口。

## 依赖规则

1. `plugins/<A>` 与 `plugins/<B>` 之间禁止互相 import；需要协作时走 `core/contracts`
   或 `ctx.Events()`。
2. 插件只允许 import 本插件内部包、`Wavelet/core`、`Wavelet/core/contracts`、
   `Wavelet/pkg` 与 `Wavelet/openflare/share`。
3. `share/` 禁止 import 任何插件实现与下游业务包。
4. 表单一所有者：`of_*` 全部由 `server` 插件建表与读写；`w_*` 由上游平台插件拥有，
   下游只能经契约或事件访问。

## 收敛路线

- 已完成：`agent`/`relay`/`flared` 各有 `plugin.go` 实现 `core.Plugin` + `core.Driver`
  （`DriverTypeAgent`/`DriverTypeRelay`/`DriverTypeFlared`），入口 `backend/cmd/{agent,relay,flared}/main.go`
  已改为 `core.NewApp(core.WithProfile(...))` + `app.Run()` 装配，`-config` 旗标、
  默认路径、退出码与启动日志保持原样；JSON 配置由各插件 `Apply` 自行加载。
- 已完成：`server/plugin.go` 实现 `core.Plugin`，`Apply` 以 `ctx.Router().Group(api_prefix)`
  声明根级与 `/v1` 全部路由；`router.Serve` 已删除，装配根改为
  `core.App` + `driver_http.New(WithEngine(router.BuildEngine()))`，监听与优雅退出归内核。
  路由保真由 `plugin_parity_test.go` 对拍 `baseline/routes-engine.txt`（256 条方法+路径）保证。
- 待办：`platform/bootstrap` 的任务、设置与迁移注册迁入 `Apply`；各业务域按插件标准
  分层规范收敛到 `handler/ service/ repository/ model/ errs/ migrations/`（模式 2）；
  引擎级中间件、NoRoute 前端兜底与白名单生效三项能力回流上游后，去掉
  `BuildEngine` 交给 `WithEngine` 这一例外。
