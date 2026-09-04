package permission

import (
	"github.com/gin-gonic/gin"

	"mall/admin_core/internal/auth"
	"mall/admin_core/pkg/response"
)

// List godoc
// @Summary List all permission definitions
// @ID listAllPermissions
// @Tags system-permission
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[[]auth.PermissionDefinition]
// @Router /api/system/permissions/list [get]
func List(c *gin.Context) {
	permissions := auth.AllPermissions()
	response.Data(c, permissions, len(permissions))
}
