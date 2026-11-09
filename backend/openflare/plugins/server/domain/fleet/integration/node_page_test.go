// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/openflare/plugins/server"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"Wavelet/openflare/plugins/server/domain/fleet/node"
	"Wavelet/openflare/plugins/server/kernel/testhelper"
	"Wavelet/pkg/response"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
)

func TestNodePageAPI(t *testing.T) {
	_, seed, cleanup := setupCoreChainTest(t)
	engine := testhelper.NewTestGinEngine(sessions.Sessions("node-page-test", cookie.NewStore([]byte("node-page-test-session-secret"))))
	ctx := core.NewContext(context.Background())
	core.Provide[contracts.AuthService](ctx, nodePageAuth{StubAuth: testhelper.StubAuth{}, token: seed.Token})
	if err := server.New().Apply(ctx); err != nil {
		t.Fatal(err)
	}
	for _, route := range ctx.Router().Routes() {
		chain := make([]gin.HandlerFunc, 0)
		for _, handler := range append(append([]any{}, route.Middlewares...), route.Handlers...) {
			chain = append(chain, toGinHandler(handler))
		}
		engine.Handle(route.Method, route.Path, chain...)
	}
	t.Cleanup(cleanup)
	headers := adminAuthHeaders(seed.Token)
	created := performJSONRequest(t, engine, http.MethodPost, apiPath("/nodes"), node.Input{Name: "Searchable Edge"}, headers)
	if created.Code != http.StatusOK {
		t.Fatalf("create node status=%d body=%s", created.Code, created.Body.String())
	}
	var createdEnvelope struct {
		Data node.View `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdEnvelope); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/nodes/page", fmt.Sprintf("/nodes/%d", createdEnvelope.Data.ID)} {
		unauthorized := performJSONRequest(t, engine, http.MethodGet, apiPath(path), nil, nil)
		var denied response.Any
		if err := json.Unmarshal(unauthorized.Body.Bytes(), &denied); err != nil {
			t.Fatal(err)
		}
		if unauthorized.Code != http.StatusUnauthorized || denied.Data != nil || denied.ErrorMsg == "" {
			t.Errorf("GET %s withoutauth status=%d body=%s,want401 withoutdata", path, unauthorized.Code, unauthorized.Body.String())
		}
	}
	page := performJSONRequest(t, engine, http.MethodGet, apiPath("/nodes/page?search=searchable&status=pending&node_type=edge_node"), nil, headers)
	var envelope struct {
		ErrorMsg string          `json:"error_msg"`
		Data     node.ListResult `json:"data"`
	}
	if err := json.Unmarshal(page.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if page.Code != http.StatusOK || envelope.ErrorMsg != "" || envelope.Data.Total != 1 || len(envelope.Data.Results) != 1 {
		t.Fatalf("GET page status=%d body=%s,wantone", page.Code, page.Body.String())
	}
	detail := performJSONRequest(t, engine, http.MethodGet, apiPath(fmt.Sprintf("/nodes/%d", createdEnvelope.Data.ID)), nil, headers)
	if detail.Code != http.StatusOK {
		t.Errorf("GET detail status=%d body=%s,want200", detail.Code, detail.Body.String())
	}
	missing := performJSONRequest(t, engine, http.MethodGet, apiPath("/nodes/99999"), nil, headers)
	if missing.Code != http.StatusNotFound {
		t.Errorf("GET missing node status=%d,want404", missing.Code)
	}
	invalid := []string{"page=-1", "page=no", "page=9223372036854775807&page_size=100", "page_size=-1", "page_size=no", "status=healthy", "node_type=relay", "search=" + strings.Repeat("x", 257)}
	for _, query := range invalid {
		bad := performJSONRequest(t, engine, http.MethodGet, apiPath("/nodes/page?"+query), nil, headers)
		var body response.Any
		if err := json.Unmarshal(bad.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if bad.Code != http.StatusBadRequest || body.Data != nil || body.ErrorMsg == "" {
			t.Errorf("GET page?%s status=%d body=%s,want400 null data", query, bad.Code, bad.Body.String())
		}
	}
	legacy := performJSONRequest(t, engine, http.MethodGet, apiPath("/nodes"), nil, headers)
	var old struct {
		Data []node.View `json:"data"`
	}
	if err := json.Unmarshal(legacy.Body.Bytes(), &old); err != nil {
		t.Fatalf("legacy array JSON: %v", err)
	}
	if len(old.Data) != 1 {
		t.Errorf("legacy len=%d,want1", len(old.Data))
	}
}

type nodePageAuth struct {
	testhelper.StubAuth
	token string
}

func (auth nodePageAuth) RequireAuthMiddleware() any {
	return gin.HandlerFunc(func(c *gin.Context) {
		if c.GetHeader("X-Access-Token") != auth.token {
			response.AbortUnauthorized(c, "authentication required")
			return
		}
		c.Next()
	})
}
