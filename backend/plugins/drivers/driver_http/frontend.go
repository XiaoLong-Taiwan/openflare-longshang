// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package driver_http

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const indexFile = "index.html"

// serverOwnedPrefixes are backend-owned namespaces. A miss there must keep Gin's default
// 404 instead of silently returning the frontend shell, which would mask broken API links.
var serverOwnedPrefixes = []string{"/api/", "/f/", "/swagger/"}

// registerFrontend mounts assets as the NoRoute fallback so client-side routes resolve.
// It is a no-op when assets is nil, i.e. the binary was built without the embed_frontend tag.
func registerFrontend(engine *gin.Engine, assets fs.FS) {
	if assets == nil {
		return
	}

	engine.NoRoute(func(c *gin.Context) {
		if isServerOwned(c.Request.URL.Path) {
			return
		}

		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error_msg": "Method not allowed"})
			return
		}

		for _, candidate := range assetCandidates(c.Request.URL.Path) {
			if serveAsset(c, assets, candidate) {
				return
			}
		}
	})
}

func isServerOwned(urlPath string) bool {
	for _, prefix := range serverOwnedPrefixes {
		if strings.HasPrefix(urlPath, prefix) {
			return true
		}
	}
	return false
}

// assetCandidates maps a request path onto the static export layout: exact file,
// Next.js clean-URL siblings, then the single-page-app shell.
func assetCandidates(urlPath string) []string {
	clean := strings.TrimPrefix(path.Join("/", urlPath), "/")
	if clean == "" || clean == "." {
		return []string{indexFile}
	}

	candidates := []string{clean}
	if !strings.Contains(clean, ".") {
		candidates = append(candidates, clean+".html", path.Join(clean, indexFile))
	}
	if template := nextExportIDTemplate(clean); template != "" {
		candidates = append(candidates, template)
	}

	return append(candidates, indexFile)
}

func nextExportIDTemplate(clean string) string {
	parts := strings.Split(clean, "/")
	if len(parts) <= 1 {
		return ""
	}
	idIndex := len(parts) - 1
	suffix := ".html"
	if strings.HasPrefix(parts[idIndex], "__next.") && strings.HasSuffix(parts[idIndex], ".txt") {
		idIndex--
		suffix = ""
	} else if strings.HasSuffix(parts[idIndex], ".txt") {
		parts[idIndex] = strings.TrimSuffix(parts[idIndex], ".txt")
		suffix = ".txt"
	}
	id, err := strconv.ParseUint(parts[idIndex], 10, 64)
	if err != nil || id == 0 || idIndex == 0 {
		return ""
	}
	parts[idIndex] = "1"
	return strings.Join(parts, "/") + suffix
}

// serveAsset writes name when it exists as a regular file, reporting whether it served.
func serveAsset(c *gin.Context, assets fs.FS, name string) bool {
	file, err := assets.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		return false
	}

	seeker, ok := file.(io.ReadSeeker)
	if !ok {
		return false
	}

	http.ServeContent(c.Writer, c.Request, name, info.ModTime(), seeker)
	return true
}
