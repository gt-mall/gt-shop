package routers

import (
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/dashboard"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerDashboardRoutes(router *gin.Engine, api *gin.RouterGroup, db *gorm.DB, authorization *auth.Service) {
	dashboardService := dashboard.NewService(db)
	dashboardAPI := dashboard.NewAPI(dashboardService)

	protected := api.Group("/dashboard")
	protected.Use(authorization.Middleware())

	protected.GET("/overview", authorization.Permission(auth.PermissionDashboardView), dashboardAPI.Overview)
	protected.GET("/trend", authorization.Permission(auth.PermissionDashboardView), dashboardAPI.Trend)
}
