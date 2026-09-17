package overview

import (
	"github.com/gin-gonic/gin"

	"mall/admin_core/pkg/response"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// Overview godoc
// @Summary 获取经营概览数据
// @ID getAnalyticsOverview
// @Tags analytics
// @Produce json
// @Param request query OverviewQueryRequest false "查询参数"
// @Success 200 {object} response.Response[OverviewResponse]
// @Failure 400 {object} response.Response[any]
// @Router /api/analytics/overview [get]
func (a *API) Overview(c *gin.Context) {
	var req OverviewQueryRequest
	_ = c.ShouldBindQuery(&req)

	data, err := a.service.Overview(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, data, 0)
}
