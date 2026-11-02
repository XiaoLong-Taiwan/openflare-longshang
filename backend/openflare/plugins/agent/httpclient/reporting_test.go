package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
