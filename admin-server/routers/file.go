package routers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"mall/admin_core/internal/auth"
	systemfile "mall/admin_core/internal/system/file"
)

func registerFileRoutes(router *gin.Engine, protected *gin.RouterGroup, db *gorm.DB, authorization *auth.Service) {
	api := systemfile.NewAPI(systemfile.NewService(db))
	router.GET("/media/*path", api.ServeLocal)
	router.HEAD("/media/*path", api.ServeLocal)
	files := protected.Group("/system/files")
	files.POST("/list", authorization.Permission(auth.PermissionSystemFileList), api.List)
	files.POST("/upload", authorization.Permission(auth.PermissionSystemFileUpload), api.Upload)
	files.POST("/rename", authorization.Permission(auth.PermissionSystemFileUpdate), api.Rename)
	files.POST("/move", authorization.Permission(auth.PermissionSystemFileUpdate), api.Move)
	files.POST("/delete", authorization.Permission(auth.PermissionSystemFileDelete), api.Delete)
	// Business pages may display existing IDs without permission to browse the library.
	files.POST("/resolve", api.Resolve)
	files.GET("/groups", authorization.Permission(auth.PermissionSystemFileList), api.Groups)
	files.GET("/groups/counts", authorization.Permission(auth.PermissionSystemFileList), api.GroupCounts)
	files.POST("/groups/save", authorization.Permission(auth.PermissionSystemFileGroup), api.SaveGroup)
	files.POST("/groups/delete", authorization.Permission(auth.PermissionSystemFileGroup), api.DeleteGroup)
	files.GET("/config", authorization.Permission(auth.PermissionSystemFileConfig), api.Config)
	files.POST("/config/save", authorization.Permission(auth.PermissionSystemFileConfig), api.SaveConfig)
}
