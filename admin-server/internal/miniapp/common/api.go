package common

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

// Hello godoc
// @Summary Miniapp hello world
// @ID miniappHello
// @Tags common
// @Produce json
// @Success 200 {object} response.Response[HelloResponse]
// @Router /hello [get]
func (a *API) Hello(c *gin.Context) {
	response.Data(c, a.service.Hello(), 0)
}

// Bootstrap godoc
// @Summary 获取小程序公共启动配置
// @ID miniappBootstrap
// @Tags common
// @Produce json
// @Success 200 {object} response.Response[BootstrapResponse]
// @Router /bootstrap [get]
func (a *API) Bootstrap(c *gin.Context) {
	result, err := a.service.Bootstrap(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, result, 0)
}
