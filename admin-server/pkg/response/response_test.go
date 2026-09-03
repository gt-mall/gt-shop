package response

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDataUsesUnifiedEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/demo", func(c *gin.Context) {
		Data(c, map[string]string{"name": "mall"}, 1)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/demo", nil)
	r.ServeHTTP(recorder, request)

	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Body.String(); got != `{"code":1,"message":"ok","total":1,"data":{"name":"mall"}}` {
		t.Fatalf("body = %q", got)
	}
}
