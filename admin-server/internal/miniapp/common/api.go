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
