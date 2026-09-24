package auth

import (
	"errors"

	"github.com/gin-gonic/gin"

	"mall/admin_core/pkg/logger"
	"mall/admin_core/pkg/response"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// Login godoc
// @Summary Login to the admin API
// @ID login
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} response.Response[LoginResponse]
// @Router /api/auth/login [post]
func (a *API) Login(c *gin.Context) {
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Warn("admin login request bind failed",
			logger.String("ip", c.ClientIP()),
			logger.String("user_agent", c.Request.UserAgent()),
			logger.Err(err),
		)
		response.Error(c, "invalid request")
		return
	}
	ctx := WithClientInfo(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	result, err := a.service.Login(ctx, request)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserDisabled) || errors.Is(err, ErrUnavailable) {
			response.Error(c, err.Error())
			return
		}
		response.Error(c, "login failed")
		return
	}
	response.Data(c, result, 0)
}

// Me godoc
// @Summary Get the current login user, roles, and permissions
// @ID getCurrentUser
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[CurrentUserResponse]
// @Router /api/auth/me [get]
func (a *API) Me(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}
	result, err := a.service.CurrentUser(c.Request.Context(), user)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, result, 0)
}

// ChangePassword godoc
// @Summary Change the current user's password
// @ID changeCurrentUserPassword
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Change password request"
// @Success 200 {object} response.Response[any]
// @Router /api/auth/password [post]
func (a *API) ChangePassword(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}
	var request ChangePasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Warn("admin change password request bind failed",
			logger.Int64("user_id", user.ID),
			logger.String("ip", c.ClientIP()),
			logger.String("user_agent", c.Request.UserAgent()),
			logger.Err(err),
		)
		response.Error(c, "invalid request")
		return
	}
	ctx := WithClientInfo(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
	if err := a.service.ChangePassword(ctx, user, request); err != nil {
		if errors.Is(err, ErrCurrentPassword) || errors.Is(err, ErrInvalidPassword) || errors.Is(err, ErrUnavailable) {
			response.Error(c, err.Error())
			return
		}
		response.Error(c, "change password failed")
		return
	}
	response.Success(c)
}
