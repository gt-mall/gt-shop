package controller

import (
	"mall/admin_core/internal/models"
	"mall/admin_core/internal/service"
	"mall/admin_core/pkg/response"

	"github.com/gin-gonic/gin"
)

var _ = models.HealthData{}

type HealthController struct {
	service *service.HealthService
}

func NewHealthController(healthService *service.HealthService) *HealthController {
	return &HealthController{service: healthService}
}

// Check godoc
// @Summary Check service health
// @ID getHealth
// @Tags system
// @Produce json
// @Success 200 {object} response.Response[models.HealthData]
// @Router /health [get]
func (h *HealthController) Check(c *gin.Context) {
	response.Data(c, h.service.Check(c.Request.Context()), 0)
}
