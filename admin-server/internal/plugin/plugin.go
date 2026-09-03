package plugin

import (
	"mall/admin_core/internal/plugin/hello"

	"github.com/gin-gonic/gin"
)

// Plugin is the compile-time extension point for admin modules.
type Plugin interface {
	Name() string
	RegisterRoutes(*gin.RouterGroup)
}

// RegisterAll is intentionally explicit. Future plugins can be added here
// without introducing dynamic loading or a micro-frontend runtime.
func RegisterAll(group *gin.RouterGroup) {
	plugins := []Plugin{hello.Plugin{}}
	for _, item := range plugins {
		item.RegisterRoutes(group)
	}
}
