package routers

import (
	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/system"

	"github.com/gin-gonic/gin"
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
	registerMemberRoutes(r, api, db, authorization)
	registerOrderRoutes(r, api, db, authorization)
	registerPluginRoutes(api.Group("/plugins"))
	registerMiniappRoutes(api, db, config)

	registerSwaggerRoutes(r)
	return r
}
