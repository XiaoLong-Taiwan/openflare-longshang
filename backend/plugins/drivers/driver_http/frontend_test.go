package driver_http

import (
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestFrontendStaticExport(t *testing.T) {
	assets := fstest.MapFS{
		"index.html":                          {Data: []byte("shell")},
		"cloudflare/groups/1.html":            {Data: []byte("group")},
		"cloudflare/groups/1.txt":             {Data: []byte("group-rsc")},
		"cloudflare/groups/1/__next.page.txt": {Data: []byte("group-segment")},
		"cloudflare/groups/7.html":            {Data: []byte("exact-group")},
		"websites/1.html":                     {Data: []byte("website")},
		"websites/1.txt":                      {Data: []byte("website-rsc")},
		"websites/1/__next.page.txt":          {Data: []byte("website-segment")},
		"orders/1.html":                       {Data: []byte("generic")},
	}
	engine := gin.New()
	registerFrontend(engine, assets)
	tests := []struct {
		method string
		url    string
		status int
		body   string
	}{
		{method: "GET", url: "/cloudflare/groups/42", status: 200, body: "group"},
		{method: "GET", url: "/cloudflare/groups/42/", status: 200, body: "group"},
		{method: "GET", url: "/cloudflare/groups/42.txt?_rsc=abc", status: 200, body: "group-rsc"},
		{method: "GET", url: "/cloudflare/groups/42/__next.page.txt", status: 200, body: "group-segment"},
		{method: "GET", url: "/cloudflare/groups/7", status: 200, body: "exact-group"},
		{method: "HEAD", url: "/cloudflare/groups/42", status: 200, body: ""},
		{method: "GET", url: "/websites/42", status: 200, body: "website"},
		{method: "GET", url: "/websites/42.txt", status: 200, body: "website-rsc"},
		{method: "GET", url: "/websites/42/__next.page.txt", status: 200, body: "website-segment"},
		{method: "GET", url: "/orders/42", status: 200, body: "generic"},
		{method: "GET", url: "/unknown/42", status: 200, body: "shell"},
		{method: "GET", url: "/cloudflare/groups/abc", status: 200, body: "shell"},
		{method: "GET", url: "/cloudflare/groups/0", status: 200, body: "shell"},
		{method: "GET", url: "/cloudflare/groups/42/extra.txt", status: 200, body: "shell"},
		{method: "GET", url: "/api/missing", status: 404, body: "404 page not found"},
		{method: "GET", url: "/f/missing", status: 404, body: "404 page not found"},
		{method: "GET", url: "/swagger/missing", status: 404, body: "404 page not found"},
	}
	for _, tt := range tests {
		t.Run(tt.method+tt.url, func(t *testing.T) {
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(tt.method, tt.url, nil))
			if w.Code != tt.status || w.Body.String() != tt.body {
				t.Errorf("frontend(%s %s) = (%d, %q), want (%d, %q)", tt.method, tt.url, w.Code, w.Body.String(), tt.status, tt.body)
			}
		})
	}
}
