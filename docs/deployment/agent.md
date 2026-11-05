# 接入 Agent

你会学到：Agent 的职责、两种接入 Token 的区别、安装脚本参数、`agent.json` 配置方式，以及如何确认节点已经上线。

OpenFlare Agent 运行在代理节点侧。它不会接收远程 shell 指令，而是通过 Agent API 拉取控制面发布的配置版本，在本地写入 OpenResty 文件、执行配置校验、reload，并在失败时尝试回滚到可运行配置。

## 上报与重送

Agent 的采集与上报间隔至少为 3 秒；每批总计最多 1000 条访问日志、4 个缓冲窗口，完整 JSON 最多 1 MiB，低于 Server 的 500 个缓冲记录与 8 MiB 限制。尚未确认的同一分钟数据也会参与重送，成功后只移除本批实际发送的记录，其余数据保留到后续批次或配置的重放保留期限届满。

当前 Server 的 WebSocket 没有状态上报确认消息，因此 WebSocket 连接期间仍通过 HTTP 上报并等待成功响应，WebSocket 继续接收配置推送并定期发送 pong 保活。只有 HTTP 200、有效的非空 data 与空 error_msg 才确认本批数据；写入 WebSocket 成功不代表日志已被接受。

HTTP 413 后保留数据，并在后续周期将字节、日志及窗口批次上限减半，最低为 64 KiB、1 条日志及 1 个窗口；达到全部下限仍被拒绝时记录错误并等待 60 秒重试，避免缩小到无法发送基础信息。HTTP 429 后按 3、6、12 秒递增退避，退避上限为 60 秒，同时遵守更长的 Retry-After（秒数或 HTTP 日期）。退避期间不采集、不发送，成功后重置退避。所有心跳准备出口（包括无缓冲存储与写盘失败）都检查完整 JSON 限额，超限记录错误并阻止发送。

访问日志每轮最多读取 1 MiB，使用 16 KiB 有界行缓冲，单条编码后的 JSON 也不得超过 16 KiB。超长原始行与超限的新日志会记录警告并丢弃；跨轮、跨重启保存超长行的丢弃状态，直到换行才恢复解析。旧缓冲中的超限日志保留到重放期限届满并输出警告，不阻止后续窗口发送。基础节点信息或窗口元数据无法装入批次时输出错误，需要排查异常字段。ACK 按完整日志值及数量、完整指标与健康快照确认，保留送出后新增或同时间戳更新的数据；本地缓冲并非永久归档。

## 接入方式

Agent 啟動時會先檢查本地 OpenResty；若已有設定但服務未正常運行，會在連線控制面前嘗試以既有設定恢復服務。沒有本地設定時仍需完成首次同步；設定無效或本地執行環境故障時，恢復可能失敗並記錄不健康狀態。

設定套用失敗後，同一 `version + checksum` 依 1、2、4 分鐘及後續 5 分鐘的間隔重試，等待期間保留可用的舊設定或安全兜底。下一個同步週期到達重試期限後才重新套用；「強制同步」或收到不同版本會清除原有退避，成功套用後重設重試狀態。

| 方式 | 适用场景 |
| --- | --- |
| `discovery_token` | 首次自动注册节点，由 Server 置换为节点专属凭证 |
| `agent_token` | 已在管理端创建或分配节点，直接使用节点专属凭证接入 |

`agent_token` 与 `discovery_token` 至少填写一个。

### 凭证获取路径

- **`discovery_token`（自动注册凭证）**：登录管理端后台，导航至「系统设置」->「自动注册」，在页面中可直接生成、查看和复制全局的自动注册凭证。
- **`agent_token`（节点专属凭证）**：登录管理端后台，导航至「节点管理」->「新增节点」，填写节点基本信息保存后，在节点详情页面即可直接复制该节点专属的接入 Token。

## 一键安装

### 交互式安装（推荐）

如果在不传递任何参数的情况下运行安装脚本，脚本将进入交互模式。您将可以通过向导选择安装方式（本地运行 / Docker 容器运行），并配置 Server 地址与认证 Token（若选择 Docker 方式且本地没有 Docker，脚本还会询问并智能安装 Docker）：

```bash
curl -fsSL https://raw.githubusercontent.com/XiaoLong-Taiwan/openflare-longshang/main/scripts/install-agent.sh | bash
```

### 自动化（非交互式）安装

如果在执行脚本时附加了任何参数，脚本将进入自动化安装模式，不需要任何交互。

使用 `discovery_token` 进行本地安装：

```bash
curl -fsSL https://raw.githubusercontent.com/XiaoLong-Taiwan/openflare-longshang/main/scripts/install-agent.sh | bash -s -- \
  --server-url http://your-server:3000 \
  --discovery-token YOUR_DISCOVERY_TOKEN
```

使用节点专属 `agent_token` 进行本地安装：

```bash
curl -fsSL https://raw.githubusercontent.com/XiaoLong-Taiwan/openflare-longshang/main/scripts/install-agent.sh | bash -s -- \
  --server-url http://your-server:3000 \
  --agent-token YOUR_AGENT_TOKEN
```

使用 Docker 容器自动化安装：

```bash
curl -fsSL https://raw.githubusercontent.com/XiaoLong-Taiwan/openflare-longshang/main/scripts/install-agent.sh | bash -s -- \
  --server-url http://your-server:3000 \
  --discovery-token YOUR_DISCOVERY_TOKEN \
  --docker
```

安装脚本在本地安装模式下会下载最新 Agent，默认写入 `/opt/openflare-agent`，生成 `agent.json`，自动检测并创建低权限系统账号 `openflare`（将整个安装目录赋权给该用户），并在 Linux + systemd 环境创建 `openflare-agent.service` 服务。该服务将以 `openflare` 普通用户运行，并通过 Linux Capabilities（`CAP_NET_BIND_SERVICE`）保障其监听特权端口（如 80、443）的能力。

支持参数：

| 参数 | 说明 |
| --- | --- |
| `--server-url` | Server 地址 |
| `--discovery-token` | 首次自动注册 Token |
| `--agent-token` | 节点专属 Token |
| `--install-dir` | 安装目录，默认 `/opt/openflare-agent`（仅本地安装生效） |
| `--openresty-path` | OpenResty 二进制路径，未传时自动查找 `openresty`（仅本地安装生效） |
| `--repo` | 下载 Agent 的 GitHub 仓库，默认 `XiaoLong-Taiwan/openflare-longshang` |
| `--no-service` | 不创建 systemd 服务（仅本地安装生效） |
| `--docker` | 使用 Docker 容器方式安装 |
| `--method` | 安装方式，可选 `local` 或 `docker`（默认 `local`） |

## 配置文件

默认配置文件路径：

```text
/opt/openflare-agent/agent.json
```

本地配置示例：

```json
{
  "server_url": "http://127.0.0.1:3000",
  "agent_token": "replace-with-node-auth-token",
  "data_dir": "./data",
  "openresty_path": "openresty",
  "openresty_observability_port": 18081,
  "observability_replay_minutes": 60,
  "heartbeat_interval": 3000,
  "request_timeout": 10000
}
```

自定义 OpenResty 路径示例：

```json
{
  "server_url": "http://127.0.0.1:3000",
  "agent_token": "replace-with-node-auth-token",
  "data_dir": "/var/lib/openflare-agent",
  "openresty_path": "/usr/local/openresty/nginx/sbin/openresty",
  "main_config_path": "/var/lib/openflare-agent/etc/nginx/nginx.conf",
  "route_config_path": "/var/lib/openflare-agent/etc/nginx/conf.d/openflare_routes.conf",
  "access_log_path": "/var/lib/openflare-agent/var/log/openflare/access.log",
  "cert_dir": "/var/lib/openflare-agent/etc/nginx/certs",
  "lua_dir": "/var/lib/openflare-agent/etc/nginx/lua",
  "runtime_config_dir": "/var/lib/openflare-agent/etc/openflare",
  "heartbeat_interval": 3000,
  "request_timeout": 10000
}
```

如果不配置 `openresty_path`，Agent 默认调用 `openresty`。完整字段见 [配置项参考](../reference/configuration.md#agent-命令行参数与配置字段)。

## Docker 运行

Docker 部署时直接运行内置 OpenResty 的 Agent 镜像：

```bash
docker pull ghcr.io/XiaoLong-Taiwan/openflare-longshang-agent:latest
docker rm -f openflare-agent 2>/dev/null || true
docker run -d --name openflare-agent --restart unless-stopped \
  -p 80:80 -p 443:443/tcp -p 443:443/udp \
  -v openflare-agent-pages:/data/var/lib/openflare/pages \
  -e OPENFLARE_SERVER_URL=http://your-server:3000 \
  -e OPENFLARE_AGENT_TOKEN=YOUR_AGENT_TOKEN \
  ghcr.io/XiaoLong-Taiwan/openflare-longshang-agent:latest
```

> [!NOTE]
> **Pages 持久化**
> 默认将 Pages 部署目录挂载到 Docker 命名卷 `openflare-agent-pages`（容器内路径 `/data/var/lib/openflare/pages`）。重建或升级 Agent 容器时无需重新拉取静态站点包。

## 卸载

### 交互式卸载（推荐）

如果在不传递任何参数的情况下运行卸载脚本，脚本将进入交互模式。您可以通过提示菜单选择卸载方式（本地卸载 / Docker 容器卸载）：

```bash
curl -fsSL https://raw.githubusercontent.com/XiaoLong-Taiwan/openflare-longshang/main/scripts/uninstall-agent.sh | bash
```

### Docker 容器卸载

停止并删除 `openflare-agent` 容器即可

## 常见问题

| 现象 | 处理步骤                                                                                                    |
| --- |---------------------------------------------------------------------------------------------------------|
| `agent_token 和 discovery_token 不能同时为空` | 检查 `agent.json` 至少配置了一个 Token                                                                           |
| 节点一直离线 | 在 Agent 节点执行 `curl -I http://your-server:3000`，确认 Server 地址可达                                           |
| 发布后重复失败 | 檢查套用錯誤；Agent 會依 1、2、4 分鐘及後續 5 分鐘退避重試，在節點詳情頁點擊「強制同步」可立即重試，發布新版本會重設退避 |
