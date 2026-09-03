package demo

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

// Get godoc
// @Summary Get backend demo data
// @ID getDemo
// @Tags demo
// @Produce json
// @Success 200 {object} response.Response[DemoData]
// @Router /api/demo [get]
func (a *API) Get(c *gin.Context) {
	response.Data(c, a.service.Get(c.Request.Context()), 0)
}
