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
// @Success 200 {object} response.Response[WechatLoginResponse]
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
