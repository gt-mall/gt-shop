package routers

import (
	"mall/admin_core/internal/auth"
	orderaftersale "mall/admin_core/internal/order/aftersale"
	orderorder "mall/admin_core/internal/order/order"
	orderreturnaddress "mall/admin_core/internal/order/returnaddress"
	"mall/admin_core/internal/order/wechatpay"
	"mall/admin_core/pkg/cache"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerOrderRoutes(router *gin.Engine, api *gin.RouterGroup, db *gorm.DB, authorization *auth.Service, caches ...cache.Cache) {
	var c cache.Cache
	if len(caches) > 0 {
		c = caches[0]
	}
	orderService := orderorder.NewService(db)
	orderAPI := orderorder.NewAPI(orderService)

	afterSaleService := orderaftersale.NewService(db, nil, wechatpay.RefundProvider{Store: wechatpay.Store{DB: db}}, c)
	afterSaleAPI := orderaftersale.NewAPI(afterSaleService)

	returnAddressService := orderreturnaddress.NewService(db)
	returnAddressAPI := orderreturnaddress.NewAPI(returnAddressService)

	protected := api.Group("/order")
	protected.Use(authorization.Middleware())

	orders := protected.Group("/orders")
	orders.POST("/list", authorization.Permission(auth.PermissionOrderOrderList), orderAPI.List)
	orders.POST("/detail", authorization.Permission(auth.PermissionOrderOrderDetail), orderAPI.Detail)
	orders.POST("/ship", authorization.Permission(auth.PermissionOrderOrderShip), orderAPI.Ship)

	aftersales := protected.Group("/after-sales")
	aftersales.POST("/list", authorization.Permission(auth.PermissionOrderAfterSaleList), afterSaleAPI.List)
	aftersales.POST("/detail", authorization.Permission(auth.PermissionOrderAfterSaleDetail), afterSaleAPI.Detail)
	aftersales.POST("/approve", authorization.Permission(auth.PermissionOrderAfterSaleApprove), afterSaleAPI.Approve)
	aftersales.POST("/reject", authorization.Permission(auth.PermissionOrderAfterSaleReject), afterSaleAPI.Reject)
	aftersales.POST("/receive", authorization.Permission(auth.PermissionOrderAfterSaleReceive), afterSaleAPI.Receive)
	aftersales.POST("/retry-refund", authorization.Permission(auth.PermissionOrderAfterSaleRetryRefund), afterSaleAPI.RetryRefund)
	aftersales.POST("/sync-refund", authorization.Permission(auth.PermissionOrderAfterSaleRetryRefund), afterSaleAPI.SyncRefund)

	returnAddresses := protected.Group("/return-addresses")
	returnAddresses.POST("/list", returnAddressAPI.List)
	returnAddresses.POST("/detail", returnAddressAPI.Detail)
	returnAddresses.GET("/default", returnAddressAPI.GetDefault)
	returnAddresses.POST("/create", authorization.Permission(auth.PermissionSystemSettingUpdate), returnAddressAPI.Create)
	returnAddresses.POST("/update", authorization.Permission(auth.PermissionSystemSettingUpdate), returnAddressAPI.Update)
	returnAddresses.POST("/delete", authorization.Permission(auth.PermissionSystemSettingUpdate), returnAddressAPI.Delete)
	returnAddresses.POST("/set-default", authorization.Permission(auth.PermissionSystemSettingUpdate), returnAddressAPI.SetDefault)
}
