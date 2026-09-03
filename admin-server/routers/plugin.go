package routers

import (
	"mall/admin_core/internal/plugin/hello"

	"github.com/gin-gonic/gin"
)

func registerPluginRoutes(group *gin.RouterGroup) {
	group.GET("/hello", hello.Get)
}
