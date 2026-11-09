package cmd

import (
	"strings"

	"Wavelet/core/extpoints"
)

func resolveTraceProtocol(src extpoints.ConfigSource, configured string) string {
	for _, key := range []string{"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "OTEL_EXPORTER_OTLP_PROTOCOL"} {
		if value, ok := src.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return strings.ToLower(strings.TrimSpace(configured))
}
