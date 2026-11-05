package httpclient

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEdgeReportsNegotiateGzip(t *testing.T) {
	for _, path := range []string{"/api/v1/relay/heartbeat", "/api/v1/tunnel/heartbeat"} {
		t.Run(path, func(t *testing.T) {
			calls := 0
			payload := map[string]string{"summary": strings.Repeat("complete summary ", 1000)}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				wantEncoding := ""
				if calls == 2 {
					wantEncoding = "gzip"
				}
				if got := r.Header.Get("Content-Encoding"); got != wantEncoding {
					t.Errorf("PostJSON(call=%d) encoding = %q, want %q", calls, got, wantEncoding)
				}
				var reader io.Reader = r.Body
				if wantEncoding == "gzip" {
					gz, err := gzip.NewReader(reader)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					defer func() { _ = gz.Close() }()
					reader = gz
				}
				var got map[string]string
				if err := json.NewDecoder(reader).Decode(&got); err != nil || got["summary"] != payload["summary"] {
					t.Errorf("PostJSON(decoded summary) error = %v, length = %d, want nil/%d", err, len(got["summary"]), len(payload["summary"]))
				}
				w.Header().Set("Accept-Encoding", "gzip")
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(server.Close)
			client := New(server.URL, "token", time.Second, "X-Agent-Token")
			for range 2 {
				if err := client.PostJSON(context.Background(), path, payload, nil); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestEncodeReportThresholdAndSavings(t *testing.T) {
	client := New("", "", time.Second, "")
	client.reportGzip.Store(true)
	random := make([]byte, 2048)
	for i := range random {
		random[i] = byte(rand.IntN(256))
	}
	for _, tc := range []struct {
		name     string
		data     []byte
		encoding string
	}{
		{name: "small", data: bytes.Repeat([]byte("a"), 1023), encoding: ""},
		{name: "threshold", data: bytes.Repeat([]byte("a"), 1024), encoding: "gzip"},
		{name: "incompressible", data: random, encoding: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, encoding, err := client.EncodeReport(tc.data)
			if err != nil || encoding != tc.encoding {
				t.Fatalf("EncodeReport(%s) = encoding %q, error %v, want %q/nil", tc.name, encoding, err, tc.encoding)
			}
			if encoding == "gzip" {
				gz, err := gzip.NewReader(bytes.NewReader(got))
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = gz.Close() }()
				got, err = io.ReadAll(gz)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(got, tc.data) {
				t.Errorf("EncodeReport(%s) round trip differs, want all original bytes", tc.name)
			}
		})
	}
}

func BenchmarkEncodeReport(b *testing.B) {
	client := New("", "", time.Second, "")
	client.reportGzip.Store(true)
	data := []byte(`{"access_logs":[` + strings.Repeat(`{"host":"example.com","path":"/index.html","bytes_sent":12345,"request_length":678},`, 999) + `{}]}`)
	compressed, _, err := client.EncodeReport(data)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := client.EncodeReport(data); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(len(compressed)), "wire-bytes")
	b.ReportMetric(100*float64(len(data)-len(compressed))/float64(len(data)), "saved-percent")
}
