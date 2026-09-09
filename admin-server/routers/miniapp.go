package routers

import (
	miniappcommon "mall/admin_core/internal/miniapp/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerMiniappRoutes(api *gin.RouterGroup, db *gorm.DB) {
	commonAPI := miniappcommon.NewAPI(miniappcommon.NewService(db))

	miniapp := api.Group("/miniapp")
	miniapp.GET("/hello", commonAPI.Hello)
	miniapp.GET("/bootstrap", commonAPI.Bootstrap)
}
