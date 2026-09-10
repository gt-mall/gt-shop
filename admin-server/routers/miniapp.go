package routers

import (
	"log"
	"time"

	"mall/admin_core/bootstrap"
	miniappauth "mall/admin_core/internal/miniapp/auth"
	miniappcommon "mall/admin_core/internal/miniapp/common"
	miniappmember "mall/admin_core/internal/miniapp/member"
	miniappregion "mall/admin_core/internal/miniapp/region"
	miniappproduct "mall/admin_core/internal/miniapp/product"
	miniapporder "mall/admin_core/internal/miniapp/order"
	miniappcart "mall/admin_core/internal/miniapp/cart"
	commonregion "mall/admin_core/internal/region"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerMiniappRoutes(api *gin.RouterGroup, db *gorm.DB, config bootstrap.Config) {
	commonAPI := miniappcommon.NewAPI(miniappcommon.NewService(db))
	authService := miniappauth.NewService(db, config.JWTSecret, time.Duration(config.JWTExpireHour)*time.Hour)
	authAPI := miniappauth.NewAPI(authService)
	memberService := miniappmember.NewService(db)
	memberAPI := miniappmember.NewAPI(memberService)
	productService := miniappproduct.NewService(db)
	productAPI := miniappproduct.NewAPI(productService)

	regionService, err := commonregion.NewService()
	if err != nil {
		log.Printf("initialize region service error: %v", err)
	}
	regionAPI := miniappregion.NewAPI(regionService)

	miniapp := api.Group("/miniapp")
	miniapp.GET("/hello", commonAPI.Hello)
	miniapp.GET("/bootstrap", commonAPI.Bootstrap)
	miniapp.GET("/regions", regionAPI.Regions)
	miniapp.HEAD("/regions", regionAPI.Regions)
	miniapp.GET("/products", productAPI.List)
	miniapp.GET("/products/:id", productAPI.Detail)

	authGroup := miniapp.Group("/auth")
	authGroup.POST("/wechat-login", authAPI.WechatLogin)
	authGroup.POST("/register", authAPI.Register)
	authGroup.POST("/login", authAPI.Login)

	orderService := miniapporder.NewService(db)
	orderAPI := miniapporder.NewAPI(orderService)

	cartService := miniappcart.NewService(db)
	cartAPI := miniappcart.NewAPI(cartService)

	memberGroup := miniapp.Group("/member")
	memberGroup.Use(authService.Middleware())
	memberGroup.GET("/me", memberAPI.Me)
	memberGroup.POST("/profile", memberAPI.Profile)
	memberGroup.POST("/avatar", memberAPI.Avatar)
	memberGroup.GET("/addresses", memberAPI.Addresses)
	memberGroup.POST("/addresses/create", memberAPI.CreateAddress)
	memberGroup.POST("/addresses/update", memberAPI.UpdateAddress)
	memberGroup.POST("/addresses/delete", memberAPI.DeleteAddress)
	memberGroup.POST("/addresses/default", memberAPI.SetDefaultAddress)

	cartGroup := miniapp.Group("/cart")
	cartGroup.Use(authService.Middleware())
	cartGroup.GET("", cartAPI.List)
	cartGroup.POST("/add", cartAPI.Add)
	cartGroup.POST("/quantity", cartAPI.UpdateQuantity)
	cartGroup.POST("/delete", cartAPI.Delete)

	checkoutGroup := miniapp.Group("/checkout")
	checkoutGroup.Use(authService.Middleware())
	checkoutGroup.POST("/preview", orderAPI.Preview)

	ordersGroup := miniapp.Group("/orders")
	ordersGroup.Use(authService.Middleware())
	ordersGroup.POST("/create", orderAPI.Create)
	ordersGroup.GET("", orderAPI.List)
	ordersGroup.GET("/:id", orderAPI.Detail)
	ordersGroup.POST("/:id/cancel", orderAPI.Cancel)
	ordersGroup.POST("/:id/confirm", orderAPI.Confirm)
}
