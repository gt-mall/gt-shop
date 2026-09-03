package hello

import (
	"mall/admin_core/pkg/response"

	"github.com/gin-gonic/gin"
)

// Get godoc
// @Summary Get hello plugin data
// @ID getHelloPlugin
// @Tags plugins
// @Produce json
// @Success 200 {object} response.Response[HelloData]
// @Router /api/plugins/hello [get]
func Get(c *gin.Context) {
	response.Data(c, HelloData{
		Plugin:  "hello",
		Message: "Hello from plugin",
	}, 0)
}
