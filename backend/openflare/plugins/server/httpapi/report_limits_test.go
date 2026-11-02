package httpapi

import (
	"bytes"
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
