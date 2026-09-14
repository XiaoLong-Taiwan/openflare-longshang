// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package agent 装载 OpenFlare 边缘 agent 插件：openresty/WAF 运行时管理、
// 心跳同步、配置下发与 WebSocket 控制通道，以 Cordis 驱动形态在 profile "agent" 下运行。
package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"Wavelet/core"
	"Wavelet/openflare/plugins/agent/agent"
	"Wavelet/openflare/plugins/agent/config"
	"Wavelet/openflare/plugins/agent/geoipupdate"
	"Wavelet/openflare/plugins/agent/heartbeat"
	"Wavelet/openflare/plugins/agent/httpclient"
	"Wavelet/openflare/plugins/agent/nginx"
	"Wavelet/openflare/plugins/agent/runtimeuser"
	"Wavelet/openflare/plugins/agent/state"
	syncservice "Wavelet/openflare/plugins/agent/sync"
	"Wavelet/openflare/plugins/agent/updater"
	"Wavelet/openflare/plugins/agent/wsclient"
	"Wavelet/pkg/util"
)

// DriverTypeAgent 是边缘 agent 守护进程专属的驱动类型：
// 只有 profile 与之相等时内核才会 Start/Stop 本插件。
const DriverTypeAgent core.DriverType = "agent"

// Plugin 实现 core.Plugin 与 core.Driver，承载 agent 进程的全部装配与生命周期。
type Plugin struct {
	configPath string

	runner  *agent.Runner
	geo     *geoipupdate.Updater
	done    chan error
	started bool
}

// New 创建 agent 插件，configPath 指向其 JSON 配置文件。
func New(configPath string) *Plugin {
	return &Plugin{configPath: configPath, done: make(chan error, 1)}
}

// Name 返回插件标识。
func (p *Plugin) Name() string { return "agent" }

// Type 返回驱动类型。
func (p *Plugin) Type() core.DriverType { return DriverTypeAgent }

// Apply 加载配置、确保运行环境、装配各服务并注册驱动。
func (p *Plugin) Apply(ctx *core.Context) error {
	cfg, err := config.Load(p.configPath)
	if err != nil {
		return fmt.Errorf("load agent config: %w", err)
	}
	if err := runtimeuser.EnsureProcessUser(); err != nil {
		return fmt.Errorf("ensure runtime user: %w", err)
	}
	if err := runtimeuser.EnsurePathOwnership(
		cfg.DataDir, runtimeuser.DefaultDirPerm, runtimeuser.DefaultFilePerm,
	); err != nil {
		return fmt.Errorf("ensure data dir ownership, data_dir=%s: %w", cfg.DataDir, err)
	}

	nginxOptions := nginx.ExecutorOptions{
		NginxPath:                  cfg.OpenrestyPath,
		MainConfigPath:             cfg.MainConfigPath,
		RouteConfigPath:            cfg.RouteConfigPath,
		CertDir:                    cfg.CertDir,
		NginxCertDir:               cfg.OpenrestyCertDir,
		LuaDir:                     cfg.LuaDir,
		NginxLuaDir:                cfg.OpenrestyLuaDir,
		OpenrestyObservabilityPort: cfg.OpenrestyObservabilityPort,
	}
	cfg.ExtVersion = nginx.DetectVersion(context.Background(), nginxOptions)
	logConfigLoaded(cfg)

	runtimeManager := newRuntimeManager(cfg, nginxOptions)
	if err := runtimeManager.EnsureLuaAssets(); err != nil {
		return fmt.Errorf("ensure managed lua assets: %w", err)
	}

	client := httpclient.New(cfg.ServerURL, cfg.InitialAuthToken(), cfg.RequestTimeout.Duration())
	wsClient := wsclient.New(cfg.ServerURL, cfg.InitialAuthToken(), cfg.RequestTimeout.Duration())
	stateStore := state.NewStore(cfg.StatePath)
	observabilityBuffer := state.NewObservabilityBufferStore(cfg.ObservabilityBufferPath)
	syncService := syncservice.New(client, runtimeManager, stateStore)
	syncService.SetPagesDir(cfg.PagesDir)
	heartbeatService := heartbeat.New(client)
	updateService := updater.New()

	p.runner = &agent.Runner{
		Config:     cfg,
		StateStore: stateStore,
		HeartbeatCycle: &heartbeat.Cycle{
			Config:              cfg,
			StateStore:          stateStore,
			ObservabilityBuffer: observabilityBuffer,
			Heartbeat:           heartbeatService,
			Sync:                syncService,
			Updater:             updateService,
		},
		HeartbeatService: heartbeatService,
		SyncService:      syncService,
		RuntimeManager:   runtimeManager,
		WebSocketService: wsClient,
	}
	p.geo = newGeoIPUpdater(cfg)

	return ctx.RegisterDriver(p)
}

// Start 预热 GeoIP 库并拉起心跳/同步/控制通道主循环。
//
// runner.Run 自身阻塞至 ctx 取消，因此放到独立 goroutine 中执行，
// 由 Stop 收敛其结果——内核要求驱动 Start 不得阻塞。
func (p *Plugin) Start(ctx context.Context) error {
	if err := p.geo.EnsureInitialDatabases(ctx); err != nil {
		slog.Warn("failed to prepare GeoIP databases before agent startup", "error", err)
	}
	util.Go(func() { p.geo.Run(ctx) })

	util.Go(func() { p.done <- p.runner.Run(ctx) })
	p.started = true
	slog.Info("agent process started")
	return nil
}

// Stop 等待主循环退出并返回其结果；ctx 超时时报告超时而非静默成功。
func (p *Plugin) Stop(ctx context.Context) error {
	if !p.started {
		return nil
	}
	select {
	case err := <-p.done:
		p.started = false
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		slog.Info("agent process stopped")
		return nil
	case <-ctx.Done():
		p.started = false
		return fmt.Errorf("agent shutdown timeout: %w", ctx.Err())
	}
}

// newGeoIPUpdater 装配国家/城市两套 mmdb 的下载与周期更新。
func newGeoIPUpdater(cfg *config.Config) *geoipupdate.Updater {
	return &geoipupdate.Updater{
		MMDBPath:        cfg.MMDBPath,
		DownloadURL:     cfg.MMDBDownloadURL,
		CityMMDBPath:    cfg.CityMMDBPath,
		CityDownloadURL: cfg.CityMMDBDownloadURL,
		UpdateInterval:  cfg.MMDBUpdateInterval.Duration(),
	}
}

// newRuntimeManager 构造 openresty 配置与证书/Lua 资产的运行时管理器。
func newRuntimeManager(cfg *config.Config, nginxOptions nginx.ExecutorOptions) *nginx.Manager {
	return &nginx.Manager{
		MainConfigPath:               cfg.MainConfigPath,
		RouteConfigPath:              cfg.RouteConfigPath,
		AccessLogPath:                cfg.AccessLogPath,
		CertDir:                      cfg.CertDir,
		NginxCertDir:                 cfg.OpenrestyCertDir,
		LuaDir:                       cfg.LuaDir,
		NginxLuaDir:                  cfg.OpenrestyLuaDir,
		RuntimeConfigDir:             cfg.RuntimeConfigDir,
		MMDBPath:                     cfg.MMDBPath,
		CityMMDBPath:                 cfg.CityMMDBPath,
		PagesDir:                     cfg.PagesDir,
		OpenrestyObservabilityListen: nginx.ObservabilityListenAddress(cfg.OpenrestyObservabilityPort),
		OpenrestyObservabilityPort:   cfg.OpenrestyObservabilityPort,
		OpenrestyResolverDirective:   "",
		Executor:                     nginx.NewExecutor(nginxOptions),
	}
}

func logConfigLoaded(cfg *config.Config) {
	slog.Info("agent config loaded",
		"server", cfg.ServerURL,
		"node", cfg.NodeName,
		"ip", cfg.NodeIP,
		"heartbeat_interval", cfg.HeartbeatInterval,
		"route_config", cfg.RouteConfigPath,
		"access_log", cfg.AccessLogPath,
		"cert_dir", cfg.CertDir,
		"lua_dir", cfg.LuaDir,
		"runtime_config_dir", cfg.RuntimeConfigDir,
		"mmdb_path", cfg.MMDBPath,
		"city_mmdb_path", cfg.CityMMDBPath,
	)
}
