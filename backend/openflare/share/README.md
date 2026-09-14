# share — 跨插件共享资源层

本目录存放被 **两个及以上插件共同消费**、且无处安放的可复用资源：

- 不能放 `Wavelet/pkg/`：该层是上游通用库，禁止包含业务语义。
- 不能放进某个插件内部：Cordis 规则禁止插件之间互相 import 实现包。

因此 `share/` 是内核（`core/`）之外唯一允许被任意插件直接 import 的共享层。
本目录内的包**禁止** import `Wavelet/openflare/...`（下游业务）与 `Wavelet/plugins/...`（具体插件实现），
只允许 import 标准库、第三方库与 `Wavelet/core`、`Wavelet/pkg`。

## 当前内容

| 包 | 内容 | 消费方 |
| :-- | :-- | :-- |
| `share/protocol` | server 与边缘三进制的控制消息线格式 | `server`、`agent`、`relay`、`flared` |
| `share/geoip` | GeoIP 解析与 IP 工具（含 `iputil` 子包） | `server`、`agent` |
| `share/edge/logging` | 边缘守护进程统一日志初始化 | `agent`、`relay`、`flared` |
| `share/githubrelease` | GitHub Release 资产解析、校验与下载 | `server`（Pages）、边缘更新器 |

## 所有权与上游同步

**本目录由 OpenFlare 拥有，不属于上游 merge 范围**：`git merge wavelet/main` 吸收
`backend/{core,pkg,plugins}` 等上游变更，从不覆盖 `share/` 与 `openflare/`。

计划迁入的候选（随插件化进度）：`wsclient`（需先合并 `pkg/util` 的仅存符号）、
`render`（openresty 配置渲染）、`pagesarchive`（Pages 产物解包）。
