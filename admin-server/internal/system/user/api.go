package user

import (
	"errors"

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
// @Summary Create a system user
// @ID createSystemUser
// @Tags system-user
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Create system user request"
// @Success 200 {object} response.Response[UserResponse]
// @Router /api/system/users/create [post]
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
// @Summary List system users
// @ID listSystemUsers
// @Tags system-user
// @Accept json
// @Produce json
// @Param request body ListRequest true "List system users request"
// @Success 200 {object} response.Response[[]UserResponse]
// @Router /api/system/users/list [post]
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
// @Summary Get a system user
// @ID getSystemUser
// @Tags system-user
// @Accept json
// @Produce json
// @Param request body IDRequest true "Get system user request"
// @Success 200 {object} response.Response[UserResponse]
// @Router /api/system/users/detail [post]
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
// @Summary Update a system user
// @ID updateSystemUser
// @Tags system-user
// @Accept json
// @Produce json
// @Param request body UpdateRequest true "Update system user request"
// @Success 200 {object} response.Response[UserResponse]
// @Router /api/system/users/update [post]
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
// @Summary Delete a system user
// @ID deleteSystemUser
// @Tags system-user
// @Accept json
// @Produce json
// @Param request body IDRequest true "Delete system user request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/users/delete [post]
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

func bindJSON(c *gin.Context, request any) bool {
	if err := c.ShouldBindJSON(request); err != nil {
		response.Error(c, "invalid request")
		return false
	}
	return true
}

func validID(c *gin.Context, id int64) bool {
	if id <= 0 {
		response.Error(c, "invalid user id")
		return false
	}
	return true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput),
		errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrUsernameExists),
		errors.Is(err, ErrDatabaseUnavailable):
		response.Error(c, err.Error())
	default:
		response.Error(c, "database error")
	}
}
