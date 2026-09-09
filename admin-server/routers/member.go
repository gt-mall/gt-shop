package routers

import (
	"mall/admin_core/internal/auth"
	membermember "mall/admin_core/internal/member/member"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerMemberRoutes(router *gin.Engine, api *gin.RouterGroup, db *gorm.DB, authorization *auth.Service) {
	memberService := membermember.NewService(db)
	memberAPI := membermember.NewAPI(memberService)

	protected := api.Group("/member")
	protected.Use(authorization.Middleware())

	members := protected.Group("/members")
	members.POST("/list", authorization.Permission(auth.PermissionMemberMemberList), memberAPI.List)
	members.POST("/detail", authorization.Permission(auth.PermissionMemberMemberList), memberAPI.Detail)
}
