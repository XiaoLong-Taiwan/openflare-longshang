// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package flared 装载 OpenFlare 隧道客户端插件：frpc 进程管理、配置同步与心跳上报，
// 以 Cordis 驱动形态在 profile "flared" 下运行。
package flared

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"Wavelet/core"
	"Wavelet/openflare/plugins/flared/config"
	flaredrunner "Wavelet/openflare/plugins/flared/flared"
	"Wavelet/openflare/plugins/flared/frpc"
	"Wavelet/openflare/plugins/flared/heartbeat"
	"Wavelet/openflare/plugins/flared/httpclient"
	"Wavelet/openflare/plugins/flared/sync"
	"Wavelet/openflare/plugins/flared/wsclient"
	"Wavelet/pkg/util"
)

// DriverTypeFlared 是隧道客户端守护进程专属的驱动类型。
const DriverTypeFlared core.DriverType = "flared"

// Plugin 实现 core.Plugin 与 core.Driver。
type Plugin struct {
	configPath string

	runner  *flaredrunner.Runner
	done    chan error
	started bool
}

// New 创建 flared 插件，configPath 指向其 JSON 配置文件。
func New(configPath string) *Plugin {
	return &Plugin{configPath: configPath, done: make(chan error, 1)}
}

// Name 返回插件标识。
func (p *Plugin) Name() string { return "flared" }

// Type 返回驱动类型。
func (p *Plugin) Type() core.DriverType { return DriverTypeFlared }

// Apply 加载配置、恢复 frpc 状态并装配各服务，然后注册驱动。
func (p *Plugin) Apply(ctx *core.Context) error {
	cfg, err := config.Load(p.configPath)
	if err != nil {
		return fmt.Errorf("load flared config: %w", err)
	}
	slog.Info("flared config loaded",
		"server", cfg.ServerURL,
		"frpc_path", cfg.FrpcPath,
		"data_dir", cfg.DataDir,
		"heartbeat_interval", cfg.HeartbeatInterval,
		"sync_interval", cfg.SyncInterval,
	)

	frpcManager := frpc.NewManager(cfg)
	_ = frpcManager.LoadState()
	slog.Info("detected frpc version", "version", frpcManager.GetVersion(context.Background()))

	httpClient := httpclient.New(cfg.ServerURL, cfg.InitialAuthToken(), cfg.RequestTimeout.Duration())
	wsClient := wsclient.New(cfg.ServerURL, cfg.InitialAuthToken(), cfg.RequestTimeout.Duration())

	p.runner = &flaredrunner.Runner{
		Config:           cfg,
		FrpcManager:      frpcManager,
		HTTPClient:       httpClient,
		WebSocketService: wsClient,
		HeartbeatService: heartbeat.New(httpClient, frpcManager, cfg),
		SyncService:      sync.New(httpClient, frpcManager, cfg),
	}

	return ctx.RegisterDriver(p)
}

// Start 拉起隧道主循环；runner.Run 阻塞至 ctx 取消，故置于独立 goroutine。
func (p *Plugin) Start(ctx context.Context) error {
	util.Go(func() { p.done <- p.runner.Run(ctx) })
	p.started = true
	slog.Info("flared process started")
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
		slog.Info("flared process stopped")
		return nil
	case <-ctx.Done():
		p.started = false
		return fmt.Errorf("flared shutdown timeout: %w", ctx.Err())
	}
}
