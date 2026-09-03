package hello

import (
	"mall/admin_core/internal/models"
	"mall/admin_core/pkg/response"

	"github.com/gin-gonic/gin"
)

type Plugin struct{}

func (Plugin) Name() string {
	return "hello"
}

func (Plugin) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/hello", Get)
}

// Get godoc
// @Summary Get hello plugin data
// @ID getHelloPlugin
// @Tags plugins
// @Produce json
// @Success 200 {object} response.Response[models.HelloData]
// @Router /api/plugins/hello [get]
func Get(c *gin.Context) {
	response.Data(c, models.HelloData{
		Plugin:  "hello",
		Message: "Hello from plugin",
	}, 0)
}
