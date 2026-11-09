package heartbeat

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	edgeupdater "Wavelet/openflare/share/edge/updater"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTryAutoUpdateThrottling(t *testing.T) {
	currentTime := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	requestCount := 0

	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestCount++
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v1.0.0"}`)),
			}, nil
		}),
	}

	updater := edgeupdater.New(edgeupdater.Config{
		LocalVersion:   "v1.0.0",
		AssetPrefix:    "openflare-agent",
		LogLabel:       "agent",
		CheckInterval:  6 * time.Hour,
		InitialBackoff: 1 * time.Minute,
		MaxBackoff:     3 * time.Hour,
		NowFunc:        func() time.Time { return currentTime },
		HTTPClient:     httpClient,
	})

	settings := &AutoUpdateSettings{
		AutoUpdate: true,
		UpdateRepo: "Rain-kl/OpenFlare",
	}

	opts := edgeupdater.UpdateOptions{Channel: "stable", Force: false}
	if !updater.CanCheck(opts) {
		t.Fatalf("expected CanCheck to be true initially")
	}

	TryAutoUpdate(context.Background(), updater, settings, "agent")
	if requestCount != 1 {
		t.Fatalf("expected 1 request after first check, got %d", requestCount)
	}

	if updater.CanCheck(opts) {
		t.Fatalf("expected CanCheck to be false after check")
	}

	var logBuf strings.Builder
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
	defer slog.SetDefault(prevLogger)

	TryAutoUpdate(context.Background(), updater, settings, "agent")
	if strings.Contains(logBuf.String(), "checking for agent updates") {
		t.Fatalf("expected no log message when throttled, got: %s", logBuf.String())
	}
	if requestCount != 1 {
		t.Fatalf("expected no request when throttled, got %d requests", requestCount)
	}

	forceSettings := &AutoUpdateSettings{
		AutoUpdate: true,
		UpdateNow:  true,
		UpdateRepo: "Rain-kl/OpenFlare",
	}
	logBuf.Reset()
	TryAutoUpdate(context.Background(), updater, forceSettings, "agent")
	if !strings.Contains(logBuf.String(), "checking for agent updates") {
		t.Fatalf("expected log message for force check, got: %s", logBuf.String())
	}
	if requestCount != 2 {
		t.Fatalf("expected force check to send a request, got %d requests", requestCount)
	}
}
