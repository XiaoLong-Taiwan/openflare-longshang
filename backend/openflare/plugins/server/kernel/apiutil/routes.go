// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package apiutil

import (
	"Wavelet/core"

	"github.com/gin-gonic/gin"
)

// RegisterCollection registers a collection endpoint on both "" and "/" so requests
// work with or without a trailing slash.
//
// 尾部斜杠变体必须用 HandleRaw 注册：RouterExtension.Handle 会经 cleanPath 归一化
// 掉尾部斜杠，而部署关闭了 gin 的 RedirectTrailingSlash，缺一条即 404。
func RegisterCollection(route core.RouterExtension, method string, handlers ...gin.HandlerFunc) {
	hs := make([]any, len(handlers))
	for i, h := range handlers {
		hs[i] = h
	}
	route.Handle(method, "", hs...)
	if route.BasePath() != "" {
		route.HandleRaw(method, "/", hs...)
	}
}
