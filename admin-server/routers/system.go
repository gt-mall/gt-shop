package routers

import (
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/system"
	systemdemo "mall/admin_core/internal/system/demo"
	systemhealth "mall/admin_core/internal/system/health"
	systempermission "mall/admin_core/internal/system/permission"
	systemrole "mall/admin_core/internal/system/role"
	systemsetting "mall/admin_core/internal/system/setting"
	systemuser "mall/admin_core/internal/system/user"
	"mall/admin_core/pkg/cache"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerSystemRoutes(router *gin.Engine, api *gin.RouterGroup, runtime *system.Runtime, db *gorm.DB, authorization *auth.Service, c cache.Cache) {
	healthAPI := systemhealth.NewAPI(systemhealth.NewService(runtime))
	demoAPI := systemdemo.NewAPI(systemdemo.NewService(runtime))
	roleAPI := systemrole.NewAPI(systemrole.NewService(db, authorization))
	userAPI := systemuser.NewAPI(systemuser.NewService(db, authorization))
	settingAPI := systemsetting.NewAPI(systemsetting.NewService(db, c))
	authAPI := auth.NewAPI(authorization)

	router.GET("/health", healthAPI.Check)
	api.GET("/demo", demoAPI.Get)
	api.POST("/auth/login", authAPI.Login)

	protected := api.Group("")
	protected.Use(authorization.Middleware())
	protected.GET("/auth/me", authAPI.Me)
	registerFileRoutes(router, protected, db, authorization)

	users := protected.Group("/system/users")
	users.POST("/create", authorization.Permission(auth.PermissionSystemUserCreate), userAPI.Create)
	users.POST("/list", authorization.Permission(auth.PermissionSystemUserList), userAPI.List)
	users.POST("/detail", authorization.Permission(auth.PermissionSystemUserList), userAPI.Get)
	users.POST("/update", authorization.Permission(auth.PermissionSystemUserUpdate), userAPI.Update)
	users.POST("/delete", authorization.Permission(auth.PermissionSystemUserDelete), userAPI.Delete)
	users.POST("/roles", authorization.Permission(auth.PermissionSystemUserRole), userAPI.Roles)
	users.POST("/roles/save", authorization.Permission(auth.PermissionSystemUserRole), userAPI.SaveRoles)

	roles := protected.Group("/system/roles")
	roles.POST("/create", authorization.Permission(auth.PermissionSystemRoleCreate), roleAPI.Create)
	roles.POST("/list", authorization.Permission(auth.PermissionSystemRoleList), roleAPI.List)
	roles.POST("/detail", authorization.Permission(auth.PermissionSystemRoleList), roleAPI.Get)
	roles.POST("/update", authorization.Permission(auth.PermissionSystemRoleUpdate), roleAPI.Update)
	roles.POST("/delete", authorization.Permission(auth.PermissionSystemRoleDelete), roleAPI.Delete)
	roles.POST("/permissions", authorization.Permission(auth.PermissionSystemRolePermission), roleAPI.Permissions)
	roles.POST("/permissions/save", authorization.Permission(auth.PermissionSystemRolePermission), roleAPI.SavePermissions)

	permissions := protected.Group("/system/permissions")
	permissions.GET("/list", authorization.Permission(auth.PermissionSystemRolePermission), systempermission.List)

	settings := protected.Group("/system/settings")
	settings.GET("/shop", settingAPI.GetShop)
	settings.POST("/shop/save", authorization.Permission(auth.PermissionSystemSettingUpdate), settingAPI.SaveShop)
	settings.GET("/site", settingAPI.GetSite)
	settings.POST("/site/save", authorization.Permission(auth.PermissionSystemSettingUpdate), settingAPI.SaveSite)
	settings.GET("/mp", settingAPI.GetMp)
	settings.POST("/mp/save", authorization.Permission(auth.PermissionSystemSettingUpdate), settingAPI.SaveMp)
}
