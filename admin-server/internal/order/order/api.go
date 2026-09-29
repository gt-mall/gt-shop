package order

import (
	"github.com/gin-gonic/gin"

	"mall/admin_core/internal/auth"
	"mall/admin_core/pkg/response"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// List godoc
// @Summary 获取订单列表
// @ID listOrders
// @Tags order
// @Accept json
// @Produce json
// @Param request body OrderListRequest true "查询参数"
// @Success 200 {object} response.Response[[]OrderListRow]
// @Router /api/order/orders/list [post]
func (a *API) List(c *gin.Context) {
	var req OrderListRequest
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

// Detail godoc
// @Summary 获取订单详情
// @ID getOrderDetail
// @Tags order
// @Accept json
// @Produce json
// @Param request body OrderDetailRequest true "订单ID"
// @Success 200 {object} response.Response[OrderDetailResponse]
// @Router /api/order/orders/detail [post]
func (a *API) Detail(c *gin.Context) {
	var req OrderDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	detail, err := a.service.Detail(c.Request.Context(), req.ID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, detail, 0)
}

// Ship godoc
// @Summary 订单发货
// @ID shipOrder
// @Tags order
// @Accept json
// @Produce json
// @Param request body OrderShipRequest true "发货参数"
// @Success 200 {object} response.Response[any]
// @Router /api/order/orders/ship [post]
func (a *API) Ship(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "未登录")
		return
	}

	var req OrderShipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	if err := a.service.Ship(c.Request.Context(), user, req); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}

// TabCounts godoc
// @Summary 获取订单状态统计
// @ID getOrderTabCounts
// @Tags order
// @Accept json
// @Produce json
// @Success 200 {object} response.Response[OrderTabCountsResponse]
// @Router /api/order/orders/tab-counts [post]
func (a *API) TabCounts(c *gin.Context) {
	counts, err := a.service.TabCounts(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, counts, 0)
}
