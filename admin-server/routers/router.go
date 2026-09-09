package routers

import (
	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/system"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// NewRouter wires public routes and their dependencies in one place.
func NewRouter(config bootstrap.Config, db *gorm.DB, authorization *auth.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	runtime := system.NewRuntime(db, config.DatabaseSet)

	api := r.Group("/api")
	registerSystemRoutes(r, api, runtime, db, authorization)
	registerProductRoutes(r, api, db, authorization)
	registerPluginRoutes(api.Group("/plugins"))
	registerMiniappRoutes(api)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}
