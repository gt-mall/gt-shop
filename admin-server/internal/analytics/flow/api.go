package flow

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

// List godoc
// @Summary 分页查询资金流水
// @ID listAnalyticsFlows
// @Tags analytics
// @Accept json
// @Produce json
// @Param request body FlowListRequest true "查询参数"
// @Success 200 {object} response.Response[FlowListResponse]
// @Failure 400 {object} response.Response[any]
// @Router /api/analytics/flows/list [post]
func (a *API) List(c *gin.Context) {
	var req FlowListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	res, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, res, 0)
}
