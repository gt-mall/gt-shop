package routers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"mall/admin_core/internal/analytics/flow"
	"mall/admin_core/internal/analytics/member"
	"mall/admin_core/internal/analytics/overview"
	"mall/admin_core/internal/analytics/product"
	"mall/admin_core/internal/analytics/trade"
	"mall/admin_core/internal/auth"
)

func registerAnalyticsRoutes(router *gin.Engine, api *gin.RouterGroup, db *gorm.DB, authorization *auth.Service) {
	overviewAPI := overview.NewAPI(overview.NewService(db))
	tradeAPI := trade.NewAPI(trade.NewService(db))
	productAPI := product.NewAPI(product.NewService(db))
	memberAPI := member.NewAPI(member.NewService(db))
	flowAPI := flow.NewAPI(flow.NewService(db))

	protected := api.Group("/analytics")
	protected.Use(authorization.Middleware())

	protected.GET("/overview", authorization.Permission(auth.PermissionAnalyticsOverview), overviewAPI.Overview)
	protected.GET("/trade", authorization.Permission(auth.PermissionAnalyticsTrade), tradeAPI.Trade)
	protected.GET("/product/rankings", authorization.Permission(auth.PermissionAnalyticsProduct), productAPI.Rankings)
	protected.POST("/product/list", authorization.Permission(auth.PermissionAnalyticsProduct), productAPI.List)
	protected.GET("/member", authorization.Permission(auth.PermissionAnalyticsMember), memberAPI.Member)
	protected.POST("/flows/list", authorization.Permission(auth.PermissionAnalyticsFlow), flowAPI.List)
}
