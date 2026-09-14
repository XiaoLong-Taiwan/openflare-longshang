// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package relay 装载 OpenFlare 中继节点插件：frps 进程管理与心跳上报，
// 以 Cordis 驱动形态在 profile "relay" 下运行。
package relay

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"Wavelet/core"
	"Wavelet/openflare/plugins/relay/config"
	"Wavelet/openflare/plugins/relay/frps"
	"Wavelet/openflare/plugins/relay/heartbeat"
	"Wavelet/openflare/plugins/relay/httpclient"
	relayrunner "Wavelet/openflare/plugins/relay/relay"
	"Wavelet/openflare/plugins/relay/state"
	"Wavelet/openflare/plugins/relay/wsclient"
	"Wavelet/pkg/util"
)

// DriverTypeRelay 是中继守护进程专属的驱动类型。
const DriverTypeRelay core.DriverType = "relay"

// Plugin 实现 core.Plugin 与 core.Driver。
type Plugin struct {
	configPath string

	runner  *relayrunner.Runner
	done    chan error
	started bool
}

// New 创建 relay 插件，configPath 指向其 JSON 配置文件。
func New(configPath string) *Plugin {
	return &Plugin{configPath: configPath, done: make(chan error, 1)}
}

// Name 返回插件标识。
func (p *Plugin) Name() string { return "relay" }

// Type 返回驱动类型。
func (p *Plugin) Type() core.DriverType { return DriverTypeRelay }

// Apply 加载配置、装配 frps 管理器与各客户端，并注册驱动。
func (p *Plugin) Apply(ctx *core.Context) error {
	cfg, err := config.Load(p.configPath)
	if err != nil {
		return fmt.Errorf("load relay config: %w", err)
	}
	slog.Info("relay config loaded",
		"server", cfg.ServerURL,
		"node", cfg.NodeName,
		"ip", cfg.NodeIP,
		"frps_path", cfg.FrpsPath,
		"data_dir", cfg.DataDir,
		"heartbeat_interval", cfg.HeartbeatInterval,
	)

	stateStore := state.NewStore(cfg.StatePath)
	frpsManager := frps.NewManager(cfg.FrpsPath, cfg.DataDir, cfg.InitialAuthToken())
	slog.Info("detected frps version", "version", frpsManager.GetVersion(context.Background()))

	httpClient := httpclient.New(cfg.ServerURL, cfg.InitialAuthToken(), cfg.RequestTimeout.Duration())
	wsClient := wsclient.New(cfg.ServerURL, cfg.InitialAuthToken(), cfg.RequestTimeout.Duration())

	p.runner = &relayrunner.Runner{
		Config:           cfg,
		StateStore:       stateStore,
		FrpsManager:      frpsManager,
		HTTPClient:       httpClient,
		WebSocketService: wsClient,
		HeartbeatService: heartbeat.New(httpClient, frpsManager, cfg, stateStore),
	}

	return ctx.RegisterDriver(p)
}

// Start 拉起中继主循环；runner.Run 阻塞至 ctx 取消，故置于独立 goroutine。
func (p *Plugin) Start(ctx context.Context) error {
	util.Go(func() { p.done <- p.runner.Run(ctx) })
	p.started = true
	slog.Info("relay process started")
	return nil
}

// Stop 等待主循环退出并回传其结果。
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
		slog.Info("relay process stopped")
		return nil
	case <-ctx.Done():
		p.started = false
		return fmt.Errorf("relay shutdown timeout: %w", ctx.Err())
	}
}
