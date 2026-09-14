// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Command flared runs the OpenFlare tunnel client daemon.
package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"Wavelet/core"
	flaredplugin "Wavelet/openflare/plugins/flared"
	edgelogging "Wavelet/openflare/share/edge/logging"
)

// shutdownTimeout 为 frpc 子进程收敛预留的退出窗口。
const shutdownTimeout = 60 * time.Second

func main() {
	edgelogging.Setup(edgelogging.Options{})

	configPath := flag.String("config", "./flared.json", "flared config path")
	flag.Parse()

	app := core.NewApp(
		core.WithProfile(core.Profile(flaredplugin.DriverTypeFlared)),
		core.WithShutdownTimeout(shutdownTimeout),
	)
	app.Use(flaredplugin.New(*configPath))

	if err := app.Prepare(); err != nil {
		slog.Error("flared startup failed", "error", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		slog.Error("flared process exited with error", "error", err)
		os.Exit(1)
	}
}
