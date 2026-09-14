package routers

import (
	"log"
	"time"

	"mall/admin_core/bootstrap"
	miniappaftersale "mall/admin_core/internal/miniapp/aftersale"
	miniappauth "mall/admin_core/internal/miniapp/auth"
	miniappcart "mall/admin_core/internal/miniapp/cart"
	miniappcommon "mall/admin_core/internal/miniapp/common"
	miniappmember "mall/admin_core/internal/miniapp/member"
	miniapporder "mall/admin_core/internal/miniapp/order"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	miniappproduct "mall/admin_core/internal/miniapp/product"
	miniappregion "mall/admin_core/internal/miniapp/region"
	orderaftersale "mall/admin_core/internal/order/aftersale"
	commonregion "mall/admin_core/internal/region"
	"mall/admin_core/pkg/bizno"
	"mall/admin_core/pkg/cache"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerMiniappRoutes(api *gin.RouterGroup, db *gorm.DB, config bootstrap.Config, c cache.Cache) {
	commonAPI := miniappcommon.NewAPI(miniappcommon.NewService(db, c))
	authService := miniappauth.NewService(db, config.JWTSecret, time.Duration(config.JWTExpireHour)*time.Hour)
	authAPI := miniappauth.NewAPI(authService)
	memberService := miniappmember.NewService(db)
	memberAPI := miniappmember.NewAPI(memberService)
	productService := miniappproduct.NewService(db, c)
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

	bizNoGen, err := bizno.NewGenerator(config.BizNoWorkerID)
	if err != nil {
		log.Fatalf("failed to initialize bizno generator: %v", err)
	}
	orderService := miniapporder.NewService(db, bizNoGen)
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

	paymentService := miniapppayment.NewService(db, bizNoGen).WithCache(c)
	paymentAPI := miniapppayment.NewAPI(paymentService)

	paymentsGroup := miniapp.Group("/payments")
	paymentsGroup.Use(authService.Middleware())
	paymentsGroup.POST("/create", paymentAPI.CreatePayment)
	paymentsGroup.POST("/mock-pay", paymentAPI.MockPay)

	afterSaleCoreService := orderaftersale.NewService(db, bizNoGen, nil, c)
	afterSaleService := miniappaftersale.NewService(afterSaleCoreService)
	afterSaleAPI := miniappaftersale.NewAPI(afterSaleService)

	afterSaleGroup := miniapp.Group("/after-sales")
	afterSaleGroup.Use(authService.Middleware())
	afterSaleGroup.POST("/apply", afterSaleAPI.Apply)
	afterSaleGroup.POST("/upload", afterSaleAPI.UploadProof)
	afterSaleGroup.POST("/list", afterSaleAPI.List)
	afterSaleGroup.POST("/detail", afterSaleAPI.Detail)
	afterSaleGroup.POST("/cancel", afterSaleAPI.Cancel)
	afterSaleGroup.POST("/return-logistics", afterSaleAPI.ReturnLogistics)
}
