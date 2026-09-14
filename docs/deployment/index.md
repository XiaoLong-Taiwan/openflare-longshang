# 部署与升级

本分区提供 OpenFlare Server、Agent、Relay 中继以及 OpenFlared 内网穿透客户端的详细部署指南、配置说明和升级维护步骤。

## 内容导航

### 快速开始
* **[快速开始](../guide/quick-start.md)**：5 分钟内使用 Docker Compose 启动 Server 和首个 Agent（推荐新用户）

### Server 部署
* **[启动 Server](./server.md)**：从源码构建前端、启动 Server、选择 SQLite 或 PostgreSQL

### Agent 部署
* **[部署 Agent](./agent.md)**：Agent 接入方式、Docker 部署、脚本安装、配置文件及故障排查

### Tunnel 内网穿透部署
* **[部署 Relay](./relay.md)**：TunnelRelay 节点的配置说明、Docker 部署与宿主机运行指南
* **[部署 OpenFlared](./openflared.md)**：内网穿透客户端配置说明、Docker 运行与自同步机制

### 升级与维护
* **[升级与维护](./upgrade.md)**：Server 与 Agent 升级步骤、数据清理策略、验证命令

### 参考资料
* **[部署说明](./deployment.md)**：部署拓扑、前置条件、Docker Compose 配置示例、多种部署方式综览
