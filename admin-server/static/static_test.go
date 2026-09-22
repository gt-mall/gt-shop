package static_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"mall/admin_core/static"
)

func TestFrontendGzip(t *testing.T) {
	const source = "console.log('compressed frontend')"
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(source)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.NoRoute(static.Handler(fstest.MapFS{
		"index.html":       {Data: []byte(source)},
		"index.html.gz":    {Data: compressed.Bytes()},
		"assets/app.js":    {Data: []byte(source)},
		"assets/app.js.gz": {Data: compressed.Bytes()},
		"assets/plain.js":  {Data: []byte(source)},
	}))
	for _, tc := range []struct {
		method, path, accept string
		gzip                 bool
	}{
		{"GET", "/assets/app.js", "gzip, deflate, br", true},
		{"GET", "/system/users", "gzip", true},
		{"HEAD", "/assets/app.js", "gzip", true},
		{"GET", "/assets/app.js", "", false},
		{"GET", "/assets/app.js", "gzip;q=0, *;q=1", false},
		{"GET", "/assets/app.js", "gzip;q=0.5", true},
		{"GET", "/assets/app.js", "br", false},
		{"GET", "/assets/app.js", "*", true},
		{"GET", "/assets/plain.js", "gzip", false},
	} {
		t.Run(tc.method+tc.path+tc.accept, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, nil)
			request.Header.Set("Accept-Encoding", tc.accept)
			response := httptest.NewRecorder()
			r.ServeHTTP(response, request)
			if response.Code != 200 || (response.Header().Get("Content-Encoding") == "gzip") != tc.gzip {
				t.Fatalf("status=%d headers=%v", response.Code, response.Header())
			}
			if response.Header().Get("Vary") != "Accept-Encoding" {
				t.Fatal("missing Vary header")
			}
			if strings.HasSuffix(tc.path, ".js") && !strings.Contains(response.Header().Get("Content-Type"), "javascript") {
				t.Fatal("incorrect JS content type")
			}
			if tc.method == "HEAD" {
				if response.Body.Len() != 0 {
					t.Fatal("HEAD returned body")
				}
				return
			}
			var body io.Reader = response.Body
			if tc.gzip {
				reader, err := gzip.NewReader(body)
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				body = reader
			}
			data, err := io.ReadAll(body)
			if err != nil || string(data) != source {
				t.Fatalf("body=%q err=%v", data, err)
			}
		})
	}
}

func TestFrontendRoutes(t *testing.T) {
	r := gin.New()
	r.NoRoute(static.Handler(fstest.MapFS{
		"index.html":    {Data: []byte("<html>admin</html>")},
		"assets/app.js": {Data: []byte("console.log('admin')")},
	}))
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/", "<html>admin</html>", 200},
		{"GET", "/system/users", "<html>admin</html>", 200},
		{"HEAD", "/system/users", "", 200},
		{"GET", "/assets/app.js", "console.log('admin')", 200},
		{"GET", "/assets/missing.js", "", 404},
		{"GET", "/assets/", "", 404},
		{"GET", "/api/missing", "", 404},
		{"GET", "/media/missing", "", 404},
		{"GET", "/swagger/missing", "", 404},
		{"POST", "/system/users", "", 404},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status || (tc.body != "" && !strings.Contains(w.Body.String(), tc.body)) {
				t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
			}
			if tc.method == "HEAD" && w.Body.Len() != 0 {
				t.Fatal("HEAD returned a body")
			}
		})
	}
}
