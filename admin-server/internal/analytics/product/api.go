package product

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

// Rankings godoc
// @Summary 获取商品销售与退款排行
// @ID getAnalyticsProductRankings
// @Tags analytics
// @Produce json
// @Param request query ProductRankingsRequest false "查询参数"
// @Success 200 {object} response.Response[ProductRankingsResponse]
// @Failure 400 {object} response.Response[any]
// @Router /api/analytics/product/rankings [get]
func (a *API) Rankings(c *gin.Context) {
	var req ProductRankingsRequest
	_ = c.ShouldBindQuery(&req)

	data, err := a.service.Rankings(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, data, 0)
}

// List godoc
// @Summary 分页查询商品/SKU数据分析
// @ID listAnalyticsProducts
// @Tags analytics
// @Accept json
// @Produce json
// @Param request body ProductListRequest true "查询与分页参数"
// @Success 200 {object} response.Response[[]ProductAnalysisRow]
// @Failure 400 {object} response.Response[any]
// @Router /api/analytics/product/list [post]
func (a *API) List(c *gin.Context) {
	var req ProductListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	rows, total, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, rows, int(total))
}
