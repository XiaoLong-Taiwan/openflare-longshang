// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server"
	ofrouter "Wavelet/openflare/plugins/server/httpapi"
	"Wavelet/openflare/plugins/server/kernel/testhelper"
	"Wavelet/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func decodeAPIResponse(t *testing.T, rec *httptest.ResponseRecorder) response.Any {
	t.Helper()

	var resp response.Any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

func requireAPIOK(t *testing.T, rec *httptest.ResponseRecorder) response.Any {
	t.Helper()

	resp := decodeAPIResponse(t, rec)
	require.Empty(t, resp.ErrorMsg, "unexpected API error: %s", resp.ErrorMsg)
	return resp
}

func unmarshalAPIData(t *testing.T, data any, target any) {
	t.Helper()

	payload, err := json.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(payload, target))
}

func unmarshalAPIMap(t *testing.T, data any) map[string]any {
	t.Helper()

	var result map[string]any
	unmarshalAPIData(t, data, &result)
	return result
}

func unmarshalAPISlice(t *testing.T, data any) []any {
	t.Helper()

	var result []any
	unmarshalAPIData(t, data, &result)
	return result
}

// mountOpenFlareTestRoutes 复刻 driver_http 的挂载方式：先由 server 插件经内核
// 路由注册表声明路由，再把每条 (方法, 路径, 中间件+处理链) 挂到测试引擎上。
func mountOpenFlareTestRoutes(engine *gin.Engine) {
	ctx := core.NewContext(context.Background())
	core.Provide[contracts.AuthService](ctx, testhelper.StubAuth{})
	if err := server.New().Apply(ctx); err != nil {
		panic(err)
	}
	for _, rd := range ctx.Router().Routes() {
		chain := make([]gin.HandlerFunc, 0, len(rd.Middlewares)+len(rd.Handlers))
		for _, item := range append(append([]any{}, rd.Middlewares...), rd.Handlers...) {
			chain = append(chain, toGinHandler(item))
		}
		engine.Handle(rd.Method, rd.Path, chain...)
	}
}

// toGinHandler 与 driver_http 接受的处理函数形态保持一致。
func toGinHandler(item any) gin.HandlerFunc {
	switch fn := item.(type) {
	case gin.HandlerFunc:
		return fn
	case func(*gin.Context):
		return gin.HandlerFunc(fn)
	default:
		panic("unexpected handler type")
	}
}

func apiPath(subpath string) string {
	return ofrouter.V1BasePath + subpath
}

func performJSONRequest(
	t *testing.T,
	engine http.Handler,
	method, path string,
	body any,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func adminAuthHeaders(token string) map[string]string {
	return map[string]string{
		"X-Access-Token": token,
	}
}
