package routers

import (
	"mall/admin_core/internal/auth"
	orderorder "mall/admin_core/internal/order/order"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerOrderRoutes(router *gin.Engine, api *gin.RouterGroup, db *gorm.DB, authorization *auth.Service) {
	orderService := orderorder.NewService(db)
	orderAPI := orderorder.NewAPI(orderService)

	protected := api.Group("/order")
	protected.Use(authorization.Middleware())

	orders := protected.Group("/orders")
	orders.POST("/list", authorization.Permission(auth.PermissionOrderOrderList), orderAPI.List)
	orders.POST("/detail", authorization.Permission(auth.PermissionOrderOrderDetail), orderAPI.Detail)
	orders.POST("/ship", authorization.Permission(auth.PermissionOrderOrderShip), orderAPI.Ship)
}
