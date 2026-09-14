// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Command agent runs the OpenFlare edge agent daemon.
package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"Wavelet/core"
	agentplugin "Wavelet/openflare/plugins/agent"
	"Wavelet/openflare/plugins/agent/logging"
)

// shutdownTimeout 为 openresty 收敛与在途配置同步预留的退出窗口。
const shutdownTimeout = 60 * time.Second

func main() {
	logging.Setup()

	configPath := flag.String("config", "./agent.json", "agent config path")
	flag.Parse()

	app := core.NewApp(
		core.WithProfile(core.Profile(agentplugin.DriverTypeAgent)),
		core.WithShutdownTimeout(shutdownTimeout),
	)
	app.Use(agentplugin.New(*configPath))

	if err := app.Prepare(); err != nil {
		slog.Error("agent startup failed", "error", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		slog.Error("agent process exited with error", "error", err)
		os.Exit(1)
	}
}
