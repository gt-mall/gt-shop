package routers

import (
	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/system"
	systemfile "mall/admin_core/internal/system/file"
	systemjob "mall/admin_core/internal/system/job"
	"mall/admin_core/pkg/cache"
	"mall/admin_core/static"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NewRouter wires public routes and their dependencies in one place.
func NewRouter(config bootstrap.Config, db *gorm.DB, authorization *auth.Service, caches ...cache.Cache) *gin.Engine {
	return NewRouterWithJobs(config, db, authorization, nil, caches...)
}

// NewRouterWithJobs wires the job manager into the protected system API.
func NewRouterWithJobs(config bootstrap.Config, db *gorm.DB, authorization *auth.Service, jobs *systemjob.Manager, caches ...cache.Cache) *gin.Engine {
	// 路由注册不打印 Gin 的 debug 列表；请求日志仍由 gin.Logger 处理。
	if gin.Mode() != gin.TestMode {
		gin.SetMode(gin.ReleaseMode)
	}

	systemfile.SetSigningSecret(config.JWTSecret)
	var c cache.Cache = cache.NewNoop()
	if len(caches) > 0 && caches[0] != nil {
		c = caches[0]
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	runtime := system.NewRuntime(db, config.DatabaseSet)

	api := r.Group("/api")
	registerSystemRoutes(r, api, runtime, db, authorization, c, jobs)
	registerProductRoutes(r, api, db, authorization, c)
	registerMemberRoutes(r, api, db, authorization)
	registerOrderRoutes(r, api, db, authorization, c)
	registerDashboardRoutes(r, api, db, authorization)
	registerAnalyticsRoutes(r, api, db, authorization)
	registerPluginRoutes(api.Group("/plugins"))
	registerMiniappRoutes(api, db, config, c)

	registerSwaggerRoutes(r)
	static.Register(r)
	return r
}
