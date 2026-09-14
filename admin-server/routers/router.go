package routers

import (
	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/system"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NewRouter wires public routes and their dependencies in one place.
func NewRouter(config bootstrap.Config, db *gorm.DB, authorization *auth.Service, caches ...cache.Cache) *gin.Engine {
	systemfile.SetSigningSecret(config.JWTSecret)
	var c cache.Cache = cache.NewNoop()
	if len(caches) > 0 && caches[0] != nil {
		c = caches[0]
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	runtime := system.NewRuntime(db, config.DatabaseSet)

	api := r.Group("/api")
	registerSystemRoutes(r, api, runtime, db, authorization, c)
	registerProductRoutes(r, api, db, authorization, c)
	registerMemberRoutes(r, api, db, authorization)
	registerOrderRoutes(r, api, db, authorization)
	registerPluginRoutes(api.Group("/plugins"))
	registerMiniappRoutes(api, db, config, c)

	registerSwaggerRoutes(r)
	return r
}
