package httpapi

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"Wavelet/openflare/plugins/server/domain/fleet/reporting"
	"Wavelet/openflare/plugins/server/domain/observability/chwriter"
	"Wavelet/openflare/share/protocol"
	"Wavelet/pkg/response"
)

func limitNodeReport(c *gin.Context) {
	c.Header("Accept-Encoding", "gzip")
	encoding := strings.TrimSpace(c.GetHeader("Content-Encoding"))
	if encoding != "" && !strings.EqualFold(encoding, "identity") && !strings.EqualFold(encoding, "gzip") {
		response.AbortWithError(c, http.StatusUnsupportedMediaType, "不支持的节点上报编码")
		return
	}
	body := http.MaxBytesReader(c.Writer, c.Request.Body, reporting.MaxPayloadBytes)
	defer func() { _ = body.Close() }()
	var reader io.Reader = body
	if strings.EqualFold(encoding, "gzip") {
		compressed, err := gzip.NewReader(body)
		if err != nil {
			response.AbortBadRequest(c, "节点上报 gzip 格式无效")
			return
		}
		defer func() { _ = compressed.Close() }()
		reader = compressed
	}
	data, err := io.ReadAll(io.LimitReader(reader, reporting.MaxPayloadBytes+1))
	var tooLarge *http.MaxBytesError
	if len(data) > reporting.MaxPayloadBytes || errors.As(err, &tooLarge) {
		response.AbortWithError(c, http.StatusRequestEntityTooLarge, "节点上报超过 8 MiB")
		return
	}
	if err != nil {
		response.AbortBadRequest(c, "无法读取节点上报")
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
	c.Request.ContentLength = int64(len(data))
	c.Request.Header.Del("Content-Encoding")
}
