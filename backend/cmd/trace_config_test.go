package cmd

import (
	"testing"

	"Wavelet/core"
	"Wavelet/core/extpoints"
)

func TestOTLPProtocolPrecedence(t *testing.T) {
	for _, tt := range []struct{ name, signal, general, configured, want string }{
		{name: "default", want: "grpc"},
		{name: "yaml", configured: "http/protobuf", want: "http/protobuf"},
		{name: "normalized_yaml", configured: " HTTP/PROTOBUF ", want: "http/protobuf"},
		{name: "invalid_signal_preserved", signal: "http/json", general: "grpc", want: "http/json"},
		{name: "general", general: "http/protobuf", configured: "grpc", want: "http/protobuf"},
		{name: "signal", signal: " HTTP/PROTOBUF ", general: "grpc", want: "http/protobuf"},
		{name: "empty_signal", signal: " ", general: "grpc", configured: "http/protobuf", want: "grpc"},
		{name: "empty_env", signal: " ", general: " ", configured: "http/protobuf", want: "http/protobuf"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]any{}
			if tt.configured != "" {
				values["otel.export_protocol"] = tt.configured
			}
			src := core.NewMapSource(values)
			src.SetEnv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", tt.signal)
			src.SetEnv("OTEL_EXPORTER_OTLP_PROTOCOL", tt.general)
			var cfg hostConfig
			reg := extpoints.NewConfigRegistry(src)
			if err := reg.Declare("host", extpoints.ConfigBinding{Target: &cfg}); err != nil {
				t.Fatal(err)
			}
			if err := reg.Resolve(); err != nil {
				t.Fatal(err)
			}
			if err := reg.Bind("", &cfg); err != nil {
				t.Fatal(err)
			}
			if err := reg.Bind("otel", &cfg.OTel); err != nil {
				t.Fatal(err)
			}
			if got := resolveTraceProtocol(src, cfg.OTel.ExportProtocol); got != tt.want {
				t.Errorf("resolveTraceProtocol(%q) = %q, want %q", tt.configured, got, tt.want)
			}
		})
	}
}

func TestOTLPHostConfigBinding(t *testing.T) {
	src := core.NewMapSource(map[string]any{
		"otel.export_protocol": "http/protobuf",
		"otel.sampling_rate":   0.25,
		"otel.tracer_name":     "yaml-tracer",
	})
	src.SetEnv("OTEL_SAMPLING_RATE", "0.75")
	src.SetEnv("OTEL_TRACER_NAME", "env-tracer")
	var cfg hostConfig
	reg := extpoints.NewConfigRegistry(src)
	if err := reg.Declare("host", extpoints.ConfigBinding{Target: &cfg}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Resolve(); err != nil {
		t.Fatal(err)
	}
	if err := reg.Bind("", &cfg); err != nil {
		t.Fatal(err)
	}
	if err := reg.Bind("otel", &cfg.OTel); err != nil {
		t.Fatal(err)
	}
	if got := cfg.OTel.ExportProtocol; got != "http/protobuf" {
		t.Errorf("Bind(otel).ExportProtocol = %q, want http/protobuf", got)
	}
	if got := cfg.OTel.SamplingRate; got != 0.75 {
		t.Errorf("Bind(otel).SamplingRate = %v, want 0.75", got)
	}
	if got := cfg.OTel.TracerName; got != "env-tracer" {
		t.Errorf("Bind(otel).TracerName = %q, want env-tracer", got)
	}
}
