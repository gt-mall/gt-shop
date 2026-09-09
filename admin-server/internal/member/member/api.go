package member

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

// List godoc
// @Summary List members
// @ID listMembers
// @Tags member
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MemberListRequest true "List members request"
// @Success 200 {object} response.Response[[]MemberRow]
// @Router /api/member/members/list [post]
func (a *API) List(c *gin.Context) {
	var req MemberListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	result, total, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, result, int(total))
}

// Detail godoc
// @Summary Get member detail
// @ID getMemberDetail
// @Tags member
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MemberIDRequest true "Get member detail request"
// @Success 200 {object} response.Response[MemberDetailResponse]
// @Router /api/member/members/detail [post]
func (a *API) Detail(c *gin.Context) {
	var req MemberIDRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID <= 0 {
		response.Error(c, "请求参数无效")
		return
	}

	result, err := a.service.Detail(c.Request.Context(), req.ID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, result, 0)
}
