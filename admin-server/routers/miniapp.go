package routers

import (
	"time"

	"mall/admin_core/bootstrap"
	miniappauth "mall/admin_core/internal/miniapp/auth"
	miniappcommon "mall/admin_core/internal/miniapp/common"
	miniappmember "mall/admin_core/internal/miniapp/member"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerMiniappRoutes(api *gin.RouterGroup, db *gorm.DB, config bootstrap.Config) {
	commonAPI := miniappcommon.NewAPI(miniappcommon.NewService(db))
	authService := miniappauth.NewService(db, config.JWTSecret, time.Duration(config.JWTExpireHour)*time.Hour)
	authAPI := miniappauth.NewAPI(authService)
	memberService := miniappmember.NewService(db)
	memberAPI := miniappmember.NewAPI(memberService)

	miniapp := api.Group("/miniapp")
	miniapp.GET("/hello", commonAPI.Hello)
	miniapp.GET("/bootstrap", commonAPI.Bootstrap)

	authGroup := miniapp.Group("/auth")
	authGroup.POST("/wechat-login", authAPI.WechatLogin)

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
}
