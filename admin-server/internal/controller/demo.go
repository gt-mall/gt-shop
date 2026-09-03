package controller

import (
	"mall/admin_core/internal/models"
	"mall/admin_core/internal/service"
	"mall/admin_core/pkg/response"

	"github.com/gin-gonic/gin"
)

var _ = models.DemoData{}

type DemoController struct {
	service *service.DemoService
}

func NewDemoController(demoService *service.DemoService) *DemoController {
	return &DemoController{service: demoService}
}

// Get godoc
// @Summary Get backend demo data
// @ID getDemo
// @Tags demo
// @Produce json
// @Success 200 {object} response.Response[models.DemoData]
// @Router /api/demo [get]
func (d *DemoController) Get(c *gin.Context) {
	response.Data(c, d.service.Get(c.Request.Context()), 0)
}
