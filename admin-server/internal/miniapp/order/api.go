package order

import (
	"strconv"

	"github.com/gin-gonic/gin"

	miniappauth "mall/admin_core/internal/miniapp/auth"
	"mall/admin_core/pkg/logger"
	"mall/admin_core/pkg/response"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// Preview godoc
// @Summary 结账预览 (计算金额、地址、库存及明细)
// @ID miniappCheckoutPreview
// @Tags order
// @Accept json
// @Produce json
// @Param request body MiniappCheckoutPreviewRequest true "结账预览参数"
// @Success 200 {object} response.Response[MiniappCheckoutPreviewResponse]
// @Failure 200 {object} response.Response[any]
// @Router /checkout/preview [post]
func (a *API) Preview(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req CheckoutPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	resp, err := a.service.Preview(c.Request.Context(), member.ID, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, resp, 0)
}

// Create godoc
// @Summary 创建订单
// @ID miniappOrderCreate
// @Tags order
// @Accept json
// @Produce json
// @Param request body MiniappOrderCreateRequest true "创建订单参数"
// @Success 200 {object} response.Response[MiniappOrderDetailResponse]
// @Failure 200 {object} response.Response[any]
// @Router /orders/create [post]
func (a *API) Create(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req OrderCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("user order create request bind failed",
			logger.Int64("member_id", member.ID),
			logger.String("ip", c.ClientIP()),
			logger.Err(err),
		)
		response.Error(c, "请求参数错误")
		return
	}

	resp, err := a.service.Create(c.Request.Context(), member, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, resp, 0)
}

// List godoc
// @Summary 获取我的订单列表
// @ID miniappOrderList
// @Tags order
// @Produce json
// @Param page query int false "页码，默认 1" default(1)
// @Param page_size query int false "每页条数，默认 10，最大 50" default(10)
// @Param status query int false "订单状态 (10待支付, 20待发货, 30待收货, 40已完成, 50已取消)"
// @Success 200 {object} response.Response[[]MiniappOrderListItemResponse]
// @Failure 200 {object} response.Response[any]
// @Router /orders [get]
func (a *API) List(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req OrderListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	orders, total, err := a.service.List(c.Request.Context(), member.ID, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, orders, int(total))
}

// Detail godoc
// @Summary 获取我的订单详情
// @ID miniappOrderDetail
// @Tags order
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response[MiniappOrderDetailResponse]
// @Failure 200 {object} response.Response[any]
// @Router /orders/{id} [get]
func (a *API) Detail(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || orderID <= 0 {
		response.Error(c, "订单ID无效")
		return
	}

	resp, err := a.service.Detail(c.Request.Context(), member.ID, orderID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, resp, 0)
}

// Cancel godoc
// @Summary 取消订单
// @ID miniappOrderCancel
// @Tags order
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response[any]
// @Failure 200 {object} response.Response[any]
// @Router /orders/{id}/cancel [post]
func (a *API) Cancel(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || orderID <= 0 {
		response.Error(c, "订单ID无效")
		return
	}

	if err := a.service.Cancel(c.Request.Context(), member, orderID); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}

// Confirm godoc
// @Summary 确认收货
// @ID miniappOrderConfirm
// @Tags order
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} response.Response[any]
// @Failure 200 {object} response.Response[any]
// @Router /orders/{id}/confirm [post]
func (a *API) Confirm(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || orderID <= 0 {
		response.Error(c, "订单ID无效")
		return
	}

	if err := a.service.Confirm(c.Request.Context(), member, orderID); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}
