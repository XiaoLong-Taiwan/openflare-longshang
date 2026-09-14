// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Command relay runs the OpenFlare relay node daemon.
package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"Wavelet/core"
	relayplugin "Wavelet/openflare/plugins/relay"
	edgelogging "Wavelet/openflare/share/edge/logging"
)

// shutdownTimeout 为 frps 子进程收敛预留的退出窗口。
const shutdownTimeout = 60 * time.Second

func main() {
	edgelogging.Setup(edgelogging.Options{})

	configPath := flag.String("config", "./relay.json", "relay config path")
	flag.Parse()

	app := core.NewApp(
		core.WithProfile(core.Profile(relayplugin.DriverTypeRelay)),
		core.WithShutdownTimeout(shutdownTimeout),
	)
	app.Use(relayplugin.New(*configPath))

	if err := app.Prepare(); err != nil {
		slog.Error("relay startup failed", "error", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		slog.Error("relay process exited with error", "error", err)
		os.Exit(1)
	}
}
