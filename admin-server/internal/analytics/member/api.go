package member

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

// Member godoc
// @Summary 获取会员数据分析
// @ID getAnalyticsMember
// @Tags analytics
// @Produce json
// @Param request query MemberQueryRequest false "查询参数"
// @Success 200 {object} response.Response[MemberResponse]
// @Failure 400 {object} response.Response[any]
// @Router /api/analytics/member [get]
func (a *API) Member(c *gin.Context) {
	var req MemberQueryRequest
	_ = c.ShouldBindQuery(&req)

	data, err := a.service.Member(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, data, 0)
}
