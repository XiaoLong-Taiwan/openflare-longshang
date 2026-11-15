# 節點 Docker Compose 與 Linux Host Network

節點詳情的 Docker 安裝指令會產生可直接保存為 `compose.yaml` 的設定。Linux 節點使用 `network_mode: host`，讓 Agent、Relay 或 Flared 使用宿主網路命名空間，適合需要直接使用宿主監聽埠或觀察宿主網路路徑的部署。

## 限制

- `network_mode: host` 僅適用 Linux Docker Engine；Docker Desktop 的 Linux VM、Windows 與 macOS 不提供相同的宿主網路語義。
- Host network 不會自動列出宿主網卡、IP 或監聽埠。這些資訊仍須由 Agent 明確採集並上報，否則管理端只能顯示既有健康快照與存取紀錄。
- 使用 host network 後不應再設定 `ports`；容器直接共用宿主埠，若埠已被其他程序使用，容器會啟動失敗。
- 這個模式不等同於 `privileged: true`，不會授予完整主機裝置或核心權限。只有在應用程式確實需要額外核心能力時才評估 capabilities，避免擴大權限範圍。
- Host network 會降低網路隔離程度。請以主機防火牆限制對外埠，並確保 Token 只透過受保護的環境變數或部署秘密注入。

## 驗證

```bash
docker compose config
docker compose up -d
docker compose ps
```

在 Linux 上確認容器實際使用宿主網路命名空間：

```bash
docker inspect -f '{{.HostConfig.NetworkMode}}' openflare-agent
ss -lntup
```
