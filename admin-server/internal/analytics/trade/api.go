package trade

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

// Trade godoc
// @Summary 获取交易分析数据
// @ID getAnalyticsTrade
// @Tags analytics
// @Produce json
// @Param request query TradeQueryRequest false "查询参数"
// @Success 200 {object} response.Response[TradeResponse]
// @Failure 400 {object} response.Response[any]
// @Router /api/analytics/trade [get]
func (a *API) Trade(c *gin.Context) {
	var req TradeQueryRequest
	_ = c.ShouldBindQuery(&req)

	data, err := a.service.Trade(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, data, 0)
}
