package health

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

// Check godoc
// @Summary Check service health
// @ID getHealth
// @Tags system
// @Produce json
// @Success 200 {object} response.Response[HealthData]
// @Router /health [get]
func (a *API) Check(c *gin.Context) {
	response.Data(c, a.service.Check(c.Request.Context()), 0)
}
