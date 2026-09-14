---
layout: home

hero:
  name: OpenFlare
  text: 开源 CDN 编排与边缘安全平台
  tagline: 支持反向代理、集中式配置同步、Pages 静态托管、内网穿透（Tunnels）、动态 WAF 防护与人机防 CC 挑战。
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/quick-start
    - theme: alt
      text: 设计边界
      link: /design/
    - theme: alt
      text: GitHub
      link: https://github.com/XiaoLong-Taiwan/openflare-longshang

features:
  - icon: 🛰️
    title: 集中式配置同步
    details: 通过 WebSocket 与心跳实现全网节点配置秒级同步下发与热生效，状态即时回收。
  - icon: 🌐
    title: 分布式 CDN 编排
    details: 将独立的 OpenResty 编排为高度协同的分布式 CDN 舰队，支持源站多负载均衡。
  - icon: 📄
    title: Pages 静态托管
    details: 直接上传前端打包 zip 资产，由边缘节点拉取解压并提供高性能本地服务与 API 代理。
  - icon: 🚇
    title: 安全内网穿透 (Tunnels)
    details: 对标 Cloudflare Tunnels，无须公网 IP 或暴露入向端口，安全穿透本地服务至公网。
  - icon: 🛡️
    title: 边缘 WAF 安全防护
    details: IP 组成员差分同步写入 Lua 共享内存，实现免 Nginx 重载的 WAF 热更新与 GeoIP 过滤。
  - icon: 🧩
    title: 防 CC 与人机挑战 (PoW)
    details: 内置高性能客户端 Proof of Work 密码学挑战，网关边缘秒级拦截阻断僵尸网络与爬虫。
---
