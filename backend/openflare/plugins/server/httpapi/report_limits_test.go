package httpapi

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"Wavelet/openflare/plugins/server/domain/fleet/reporting"
	"Wavelet/pkg/response"
)

func TestLimitNodeReport(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		status int
	}{
		{name: "valid", body: `{"node_id":"n1"}`, status: http.StatusNoContent},
		{name: "unknown length oversized", body: strings.Repeat(" ", reporting.MaxPayloadBytes+1), status: http.StatusRequestEntityTooLarge},
		{name: "invalid", body: `{`, status: http.StatusBadRequest},
		{name: "nested logs", body: `{"access_logs":[` + strings.Repeat(`{},`, 599) + `{}],"buffered":[{"access_logs":[` + strings.Repeat(`{},`, 400) + `{}]}]}`, status: http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(response.ErrorHandlerMiddleware())
			router.POST("/", limitNodeReport, func(c *gin.Context) {
				var payload map[string]any
				if err := c.ShouldBindJSON(&payload); err != nil {
					t.Errorf("ShouldBindJSON(restored body) = %v, want nil", err)
				}
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tc.body))
			request.ContentLength = -1
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != tc.status {
				t.Errorf("POST(%s) status = %d, want %d", tc.name, recorder.Code, tc.status)
			}
		})
	}
}

func TestLimitNodeReportEncoding(t *testing.T) {
	var valid bytes.Buffer
	writer := gzip.NewWriter(&valid)
	if _, err := writer.Write([]byte(`{"node_id":"n1","access_logs":[{"bytes_sent":12345}],"buffered":[{"captured_at_unix":123}]} `)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	var oversized bytes.Buffer
	writer = gzip.NewWriter(&oversized)
	if _, err := io.Copy(writer, strings.NewReader(strings.Repeat(" ", reporting.MaxPayloadBytes+1))); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	corrupt := bytes.Clone(valid.Bytes())
	corrupt[len(corrupt)-8] ^= 1
	for _, tc := range []struct {
		name     string
		encoding string
		body     []byte
		status   int
	}{
		{name: "gzip", encoding: "gzip", body: valid.Bytes(), status: http.StatusNoContent},
		{name: "case insensitive", encoding: " GZip ", body: valid.Bytes(), status: http.StatusNoContent},
		{name: "identity", encoding: "identity", body: []byte(`{"node_id":"n1"}`), status: http.StatusNoContent},
		{name: "unsupported", encoding: "br", body: valid.Bytes(), status: http.StatusUnsupportedMediaType},
		{name: "stacked", encoding: "gzip, gzip", body: valid.Bytes(), status: http.StatusUnsupportedMediaType},
		{name: "invalid header", encoding: "gzip", body: []byte("bad"), status: http.StatusBadRequest},
		{name: "truncated", encoding: "gzip", body: valid.Bytes()[:valid.Len()-4], status: http.StatusBadRequest},
		{name: "checksum", encoding: "gzip", body: corrupt, status: http.StatusBadRequest},
		{name: "decoded oversized", encoding: "gzip", body: oversized.Bytes(), status: http.StatusRequestEntityTooLarge},
		{name: "wire oversized", encoding: "identity", body: []byte(strings.Repeat(" ", reporting.MaxPayloadBytes+1)), status: http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(response.ErrorHandlerMiddleware())
			router.POST("/", limitNodeReport, func(c *gin.Context) {
				if c.GetHeader("Content-Encoding") != "" {
					t.Error("limitNodeReport() retained Content-Encoding, want decoded body metadata")
				}
				var payload struct {
					NodeID string `json:"node_id"`
				}
				if err := c.ShouldBindJSON(&payload); err != nil || payload.NodeID != "n1" {
					t.Errorf("ShouldBindJSON(decoded report) = %v, %v, want n1 and nil", payload, err)
				}
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(tc.body))
			request.ContentLength = -1
			request.Header.Set("Content-Encoding", tc.encoding)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != tc.status || recorder.Header().Get("Accept-Encoding") != "gzip" {
				t.Errorf("POST(%s) = status %d encoding %q, want %d/gzip", tc.name, recorder.Code, recorder.Header().Get("Accept-Encoding"), tc.status)
			}
		})
	}
}
