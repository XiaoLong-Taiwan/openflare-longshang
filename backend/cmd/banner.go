// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"Wavelet/pkg/buildinfo"
	"fmt"
	"runtime"
	"strings"
)

type startupState struct {
	mode           string
	listensForHTTP bool
	env            string
	addr           string
}

func printStartupBanner(state startupState) {
	fmt.Println(formatStartupBanner(state))
}

func formatStartupBanner(state startupState) string {
	env := state.env
	if env == "" {
		env = "production"
	}
	addr := state.addr
	if addr == "" {
		addr = "127.0.0.1:3000"
	}

	lines := []string{
		"",
		"   ____                   ________               ",
		"  / __ \\____  ___  ____  / ____/ /___ _________  ",
		" / / / / __ \\/ _ \\/ __ \\/ /_  / / __ `/ ___/ _ \\",
		"/ /_/ / /_/ /  __/ / / / __/ / / /_/ / /  /  __/",
		"\\____/ .___/\\___/_/ /_/_/   /_/\\__,_/_/   \\___/ ",
		"      /_/                                         ",
		fmt.Sprintf(" OpenFlare %s", buildinfo.Version),
		"",
		fmt.Sprintf(" Environment: %s", env),
		fmt.Sprintf(" Runtime:     %s/%s (%s)", runtime.GOOS, runtime.GOARCH, runtime.Version()),
		fmt.Sprintf(" Build time:  %s", buildTime()),
	}
	if state.listensForHTTP {
		lines = append(lines, fmt.Sprintf(" Listening:   http://%s", addr))
	}
	lines = append(lines, fmt.Sprintf(" Mode:        %s", state.mode), "")
	return strings.Join(lines, "\n")
}

func buildTime() string {
	if buildinfo.BuildTime == "" {
		return "development build"
	}
	return buildinfo.BuildTime
}
