package routers

import (
	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/controller"
	"mall/admin_core/internal/dao"
	"mall/admin_core/internal/plugin"
	"mall/admin_core/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewRouter wires public routes and their dependencies in one place.
func NewRouter(config bootstrap.Config, pool *pgxpool.Pool) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	runtimeDAO := dao.NewRuntimeDAO(pool, config.DatabaseSet)
	healthController := controller.NewHealthController(service.NewHealthService(runtimeDAO))
	demoController := controller.NewDemoController(service.NewDemoService(runtimeDAO))

	r.GET("/health", healthController.Check)

	api := r.Group("/api")
	api.GET("/demo", demoController.Get)
	plugin.RegisterAll(api.Group("/plugins"))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}
