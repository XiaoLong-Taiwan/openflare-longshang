// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package apiutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"Wavelet/openflare/plugins/server/kernel/testhelper"
	"Wavelet/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAdminMiddlewaresRunsAuthThenAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var order []string
	auth := sequentialAuth{
		auth: func(c *gin.Context) {
			order = append(order, "auth")
			c.Next()
		},
		admin: func(c *gin.Context) {
			order = append(order, "admin")
			c.Next()
		},
	}

	engine := testhelper.NewTestGinEngine()
	group := engine.Group("/protected", ginHandlerMiddlewares(AdminMiddlewares(auth)...)...)
	group.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, response.OKNil())
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []string{"auth", "admin"}, order)
}

func ginHandlerMiddlewares(middlewares ...any) []gin.HandlerFunc {
	handlers := make([]gin.HandlerFunc, 0, len(middlewares))
	for _, m := range middlewares {
		switch h := m.(type) {
		case gin.HandlerFunc:
			handlers = append(handlers, h)
		case func(*gin.Context):
			handlers = append(handlers, gin.HandlerFunc(h))
		default:
			panic("unexpected middleware type")
		}
	}
	return handlers
}

type sequentialAuth struct {
	testhelper.StubAuth
	auth, admin gin.HandlerFunc
}

func (s sequentialAuth) RequireAuthMiddleware() any  { return s.auth }
func (s sequentialAuth) RequireAdminMiddleware() any { return s.admin }
