package role

import (
	"errors"

	"mall/admin_core/internal/auth"
	"mall/admin_core/pkg/response"

	"github.com/gin-gonic/gin"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// Create godoc
// @Summary Create a system role
// @ID createSystemRole
// @Tags system-role
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRequest true "Create system role request"
// @Success 200 {object} response.Response[RoleResponse]
// @Router /api/system/roles/create [post]
func (a *API) Create(c *gin.Context) {
	var request CreateRequest
	if !bindJSON(c, &request) {
		return
	}

	result, err := a.service.Create(c.Request.Context(), request)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// List godoc
// @Summary List system roles
// @ID listSystemRoles
// @Tags system-role
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ListRequest true "List system roles request"
// @Success 200 {object} response.Response[[]RoleResponse]
// @Router /api/system/roles/list [post]
func (a *API) List(c *gin.Context) {
	var request ListRequest
	if !bindJSON(c, &request) {
		return
	}

	result, total, err := a.service.List(c.Request.Context(), request)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, total)
}

// Get godoc
// @Summary Get a system role
// @ID getSystemRole
// @Tags system-role
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body IDRequest true "Get system role request"
// @Success 200 {object} response.Response[RoleResponse]
// @Router /api/system/roles/detail [post]
func (a *API) Get(c *gin.Context) {
	var request IDRequest
	if !bindJSON(c, &request) || !validID(c, request.ID) {
		return
	}

	result, err := a.service.Get(c.Request.Context(), request.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// Update godoc
// @Summary Update a system role
// @ID updateSystemRole
// @Tags system-role
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateRequest true "Update system role request"
// @Success 200 {object} response.Response[RoleResponse]
// @Router /api/system/roles/update [post]
func (a *API) Update(c *gin.Context) {
	var request UpdateRequest
	if !bindJSON(c, &request) || !validID(c, request.ID) {
		return
	}

	result, err := a.service.Update(c.Request.Context(), request.ID, request)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// Delete godoc
// @Summary Delete a system role
// @ID deleteSystemRole
// @Tags system-role
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body IDRequest true "Delete system role request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/roles/delete [post]
func (a *API) Delete(c *gin.Context) {
	var request IDRequest
	if !bindJSON(c, &request) || !validID(c, request.ID) {
		return
	}

	if err := a.service.Delete(c.Request.Context(), request.ID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Permissions godoc
// @Summary Get permissions assigned to a role
// @ID getSystemRolePermissions
// @Tags system-role
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PermissionRequest true "Role permissions request"
// @Success 200 {object} response.Response[[]string]
// @Router /api/system/roles/permissions [post]
func (a *API) Permissions(c *gin.Context) {
	var request PermissionRequest
	if !bindJSON(c, &request) || !validID(c, request.ID) {
		return
	}
	result, err := a.service.Permissions(c.Request.Context(), request.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, len(result))
}

// SavePermissions godoc
// @Summary Replace all permissions assigned to a role
// @ID saveSystemRolePermissions
// @Tags system-role
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SavePermissionRequest true "Save role permissions request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/roles/permissions/save [post]
func (a *API) SavePermissions(c *gin.Context) {
	var request SavePermissionRequest
	if !bindJSON(c, &request) || !validID(c, request.ID) {
		return
	}
	if err := a.service.SavePermissions(c.Request.Context(), request.ID, request.Permissions); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

func bindJSON(c *gin.Context, request any) bool {
	if err := c.ShouldBindJSON(request); err != nil {
		response.Error(c, "invalid request")
		return false
	}
	return true
}

func validID(c *gin.Context, id int64) bool {
	if id <= 0 {
		response.Error(c, "invalid role id")
		return false
	}
	return true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput),
		errors.Is(err, ErrRoleNotFound),
		errors.Is(err, ErrRoleCodeExists),
		errors.Is(err, ErrDatabaseUnavailable),
		errors.Is(err, ErrBuiltInRole),
		errors.Is(err, auth.ErrUnavailable),
		errors.Is(err, auth.ErrRoleNotFound),
		errors.Is(err, auth.ErrInvalidPermission),
		errors.Is(err, auth.ErrSuperAdminRole):
		response.Error(c, err.Error())
	default:
		response.Error(c, "database error")
	}
}
