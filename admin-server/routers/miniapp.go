package routers

import (
	miniappcommon "mall/admin_core/internal/miniapp/common"

	"github.com/gin-gonic/gin"
)

func registerMiniappRoutes(api *gin.RouterGroup) {
	commonAPI := miniappcommon.NewAPI(miniappcommon.NewService())

	miniapp := api.Group("/miniapp")
	miniapp.GET("/hello", commonAPI.Hello)
}
