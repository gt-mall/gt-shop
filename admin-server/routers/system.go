package routers

import (
	"mall/admin_core/internal/system"
	systemdemo "mall/admin_core/internal/system/demo"
	systemhealth "mall/admin_core/internal/system/health"
	systemrole "mall/admin_core/internal/system/role"
	systemuser "mall/admin_core/internal/system/user"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerSystemRoutes(router *gin.Engine, api *gin.RouterGroup, runtime *system.Runtime, db *gorm.DB) {
	healthAPI := systemhealth.NewAPI(systemhealth.NewService(runtime))
	demoAPI := systemdemo.NewAPI(systemdemo.NewService(runtime))
	roleAPI := systemrole.NewAPI(systemrole.NewService(db))
	userAPI := systemuser.NewAPI(systemuser.NewService(db))

	router.GET("/health", healthAPI.Check)
	api.GET("/demo", demoAPI.Get)

	users := api.Group("/system/users")
	users.POST("/create", userAPI.Create)
	users.POST("/list", userAPI.List)
	users.POST("/detail", userAPI.Get)
	users.POST("/update", userAPI.Update)
	users.POST("/delete", userAPI.Delete)

	roles := api.Group("/system/roles")
	roles.POST("/create", roleAPI.Create)
	roles.POST("/list", roleAPI.List)
	roles.POST("/detail", roleAPI.Get)
	roles.POST("/update", roleAPI.Update)
	roles.POST("/delete", roleAPI.Delete)
}
