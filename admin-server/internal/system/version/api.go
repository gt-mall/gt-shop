package version

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
// @Summary 获取系统版本信息
// @ID getSystemVersion
// @Tags system
// @Produce json
// @Success 200 {object} response.Response[VersionData]
// @Router /api/system/version [get]
func (a *API) Get(c *gin.Context) {
	response.Data(c, a.service.Get(), 0)
}
