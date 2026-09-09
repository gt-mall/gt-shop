package auth

import (
	"mall/admin_core/pkg/response"

	"github.com/gin-gonic/gin"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// WechatLogin godoc
// @Summary 微信小程序登录
// @ID miniappWechatLogin
// @Tags auth
// @Accept json
// @Produce json
// @Param request body WechatLoginRequest true "微信登录请求"
// @Success 200 {object} response.Response[AuthResponse]
// @Router /auth/wechat-login [post]
func (a *API) WechatLogin(c *gin.Context) {
	var req WechatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	result, err := a.service.WechatLogin(c.Request.Context(), req.Code)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, result, 0)
}

// Register godoc
// @Summary 会员注册 (手机号+密码)
// @ID miniappRegister
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册请求"
// @Success 200 {object} response.Response[AuthResponse]
// @Router /auth/register [post]
func (a *API) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	result, err := a.service.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, result, 0)
}

// Login godoc
// @Summary 会员登录 (手机号+密码)
// @ID miniappLogin
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} response.Response[AuthResponse]
// @Router /auth/login [post]
func (a *API) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	result, err := a.service.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, result, 0)
}
