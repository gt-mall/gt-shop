package dashboard

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
// @Summary 获取数据看板概览数据
// @ID getDashboardOverview
// @Tags dashboard
// @Produce json
// @Success 200 {object} response.Response[DashboardOverviewResponse]
// @Failure 400 {object} response.Response[any]
// @Router /api/dashboard/overview [get]
func (a *API) Overview(c *gin.Context) {
	data, err := a.service.Overview(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, data, 0)
}

// Trend godoc
// @Summary 获取销售与订单趋势
// @ID getDashboardTrend
// @Tags dashboard
// @Produce json
// @Param days query int false "趋势天数 (7 或 30，默认 7)"
// @Success 200 {object} response.Response[DashboardTrendResponse]
// @Failure 400 {object} response.Response[any]
// @Router /api/dashboard/trend [get]
func (a *API) Trend(c *gin.Context) {
	var req DashboardTrendRequest
	_ = c.ShouldBindQuery(&req)
	data, err := a.service.Trend(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, data, 0)
}
