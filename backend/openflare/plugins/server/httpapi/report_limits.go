package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"Wavelet/openflare/plugins/server/domain/fleet/reporting"
	"Wavelet/openflare/plugins/server/domain/observability/chwriter"
	"Wavelet/openflare/share/protocol"
	"Wavelet/pkg/response"
)

func limitNodeReport(c *gin.Context) {
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, reporting.MaxPayloadBytes+1))
	if err != nil {
		response.AbortBadRequest(c, "无法读取节点上报")
		return
	}
	if len(data) > reporting.MaxPayloadBytes {
		response.AbortWithError(c, http.StatusRequestEntityTooLarge, "节点上报超过 8 MiB")
		return
	}
	var payload protocol.NodePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		response.AbortBadRequest(c, "节点上报格式无效")
		return
	}
	if !reporting.Valid(payload) {
		response.AbortWithError(c, http.StatusRequestEntityTooLarge, "节点上报批次过大")
		return
	}
	for _, stats := range chwriter.WriterStats() {
		logs := len(payload.AccessLogs)
		for _, record := range payload.Buffered {
			logs += len(record.AccessLogs)
		}
		if stats.Name == "node_access_logs" && logs > stats.Cap-stats.Depth {
			c.Header("Retry-After", "2")
			response.AbortTooManyRequests(c, "节点日志队列繁忙")
			return
		}
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(data))
}
