// Package static serves the compiled administration frontend embedded in the binary.
package static

import (
	"bytes"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var content embed.FS

// Register keeps API/media misses as 404 and supports browser history routing.
// Without a frontend build, the backend remains usable on its own.
func Register(r *gin.Engine) {
	files, err := fs.Sub(content, "dist")
	if err != nil {
		return
	}
	if _, err := fs.Stat(files, "index.html"); err != nil {
		return
	}
	r.NoRoute(Handler(files))
}

// Handler serves public build files only, without directory listings.
func Handler(files fs.FS) gin.HandlerFunc {
	server := http.FileServer(http.FS(files))
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+c.Request.URL.Path), "/")
		for _, prefix := range []string{"api", "media", "swagger", "health"} {
			if name == prefix || strings.HasPrefix(name, prefix+"/") {
				c.Status(http.StatusNotFound)
				return
			}
		}
		info, err := fs.Stat(files, name)
		if name == "" || err != nil || info.IsDir() {
			if path.Ext(name) != "" || name == "assets" || strings.HasPrefix(name, "assets/") {
				c.Status(http.StatusNotFound)
				return
			}
			c.Request.URL.Path = "/"
			name = "index.html"
		}
		c.Header("Cache-Control", "no-cache")
		c.Writer.Header().Add("Vary", "Accept-Encoding")
		if acceptsGzip(c.GetHeader("Accept-Encoding")) {
			if compressed, err := fs.ReadFile(files, name+".gz"); err == nil {
				c.Header("Content-Encoding", "gzip")
				if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
					c.Header("Content-Type", contentType)
				}
				c.Status(http.StatusOK)
				http.ServeContent(c.Writer, c.Request, name, time.Time{}, bytes.NewReader(compressed))
				return
			}
		}
		c.Status(http.StatusOK)
		server.ServeHTTP(c.Writer, c.Request)
	}
}

func acceptsGzip(header string) bool {
	wildcard := false
	for _, entry := range strings.Split(header, ",") {
		parts := strings.Split(entry, ";")
		coding := strings.TrimSpace(parts[0])
		quality := 1.0
		for _, parameter := range parts[1:] {
			key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if ok && strings.EqualFold(strings.TrimSpace(key), "q") {
				parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				if err != nil || parsed < 0 || parsed > 1 {
					quality = 0
				} else {
					quality = parsed
				}
			}
		}
		if strings.EqualFold(coding, "gzip") {
			return quality > 0
		}
		if coding == "*" {
			wildcard = quality > 0
		}
	}
	return wildcard
}
