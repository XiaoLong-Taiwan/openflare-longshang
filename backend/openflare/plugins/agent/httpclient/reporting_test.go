package httpclient

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"Wavelet/openflare/plugins/agent/protocol"
)

func TestHeartbeatReportingResponses(t *testing.T) {
	for _, status := range []int{200, 413, 429, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Agent-Token") != "updated" {
					t.Errorf("Heartbeat token = %q, want updated", r.Header.Get("X-Agent-Token"))
				}
				w.Header().Set("Retry-After", "7")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error_msg":"","data":{}}`))
			}))
			t.Cleanup(server.Close)
			client := New(server.URL, "old", time.Second)
			client.SetToken("updated")
			_, err := client.Heartbeat(context.Background(), protocol.NodePayload{SchemaVersion: 2})
			if status == 200 {
				if err != nil {
					t.Fatalf("Heartbeat(200) = %v, want nil", err)
				}
				return
			}
			var reportErr *reportError
			if !errors.As(err, &reportErr) || reportErr.StatusCode() != status || reportErr.RetryAfter() != 7*time.Second {
				t.Fatalf("Heartbeat(%d) = %v, want status and 7s retry preserved", status, err)
			}
		})
	}
}

func TestHeartbeatNegotiatesGzipWithoutChangingPayload(t *testing.T) {
	payload := protocol.NodePayload{
		SchemaVersion: 2,
		NodeID:        "node-1",
		AccessLogs: []protocol.NodeAccessLog{{
			Path: strings.Repeat("/path", 500), BytesSent: 12345, RequestLength: 678,
		}},
		Buffered: []protocol.BufferedObservabilityRecord{{
			CapturedAtUnix: 123, AccessLogs: []protocol.NodeAccessLog{{BytesSent: 999}},
		}},
	}
	for _, supported := range []bool{false, true} {
		t.Run(map[bool]string{false: "old server", true: "gzip server"}[supported], func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				wantEncoding := ""
				if supported && calls == 2 {
					wantEncoding = "gzip"
				}
				if got := r.Header.Get("Content-Encoding"); got != wantEncoding {
					t.Errorf("Heartbeat(call=%d) encoding = %q, want %q", calls, got, wantEncoding)
				}
				var reader io.Reader = r.Body
				if r.Header.Get("Content-Encoding") == "gzip" {
					gz, err := gzip.NewReader(r.Body)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					defer func() { _ = gz.Close() }()
					reader = gz
				}
				var got protocol.NodePayload
				if err := json.NewDecoder(reader).Decode(&got); err != nil || !reflect.DeepEqual(got, payload) {
					t.Errorf("Heartbeat(decoded payload) = %+v, %v, want %+v/nil", got, err, payload)
				}
				if supported && calls == 1 {
					w.Header().Set("Accept-Encoding", "gzip")
				}
				_, _ = w.Write([]byte(`{"error_msg":"","data":{}}`))
			}))
			t.Cleanup(server.Close)
			client := New(server.URL, "token", time.Second)
			for range 3 {
				if _, err := client.Heartbeat(context.Background(), payload); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 3 {
				t.Errorf("Heartbeat requests = %d, want 3 without automatic replay", calls)
			}
		})
	}
}

func TestHeartbeatRejectsInvalidAcknowledgement(t *testing.T) {
	for _, body := range []string{`{"error_msg":"rejected","data":null}`, `{`, `{}`, `null`, `{"error_msg":"","data":null}`, `{"data":{}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
		client := New(server.URL, "token", time.Second)
		_, err := client.Heartbeat(context.Background(), protocol.NodePayload{})
		server.Close()
		if err == nil {
			t.Errorf("Heartbeat(%q) = nil, want error", body)
		}
	}
}
