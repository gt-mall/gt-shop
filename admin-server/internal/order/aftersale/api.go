package aftersale

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
// @Summary 获取售后单列表
// @ID listAfterSales
// @Tags order
// @Accept json
// @Produce json
// @Param request body AdminAfterSaleListRequest true "查询参数"
// @Success 200 {object} response.Response[[]AdminAfterSaleListRow]
// @Router /api/order/after-sales/list [post]
func (a *API) List(c *gin.Context) {
	var req AdminAfterSaleListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	rows, total, err := a.service.AdminList(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, rows, int(total))
}

// Detail godoc
// @Summary 获取售后单详情
// @ID getAfterSaleDetail
// @Tags order
// @Accept json
// @Produce json
// @Param request body AdminAfterSaleDetailRequest true "售后单ID"
// @Success 200 {object} response.Response[AdminAfterSaleDetailResponse]
// @Router /api/order/after-sales/detail [post]
func (a *API) Detail(c *gin.Context) {
	var req AdminAfterSaleDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	detail, err := a.service.AdminDetail(c.Request.Context(), req.ID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, detail, 0)
}

// Approve godoc
// @Summary 审核通过售后单
// @ID approveAfterSale
// @Tags order
// @Accept json
// @Produce json
// @Param request body AdminApproveRequest true "审核通过参数"
// @Success 200 {object} response.Response[any]
// @Router /api/order/after-sales/approve [post]
func (a *API) Approve(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "未登录")
		return
	}

	var req AdminApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	if err := a.service.AdminApprove(c.Request.Context(), user, req); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}

// Reject godoc
// @Summary 审核拒绝售后单
// @ID rejectAfterSale
// @Tags order
// @Accept json
// @Produce json
// @Param request body AdminRejectRequest true "审核拒绝参数"
// @Success 200 {object} response.Response[any]
// @Router /api/order/after-sales/reject [post]
func (a *API) Reject(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "未登录")
		return
	}

	var req AdminRejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	if err := a.service.AdminReject(c.Request.Context(), user, req); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}

// Receive godoc
// @Summary 商家确认收到退货
// @ID confirmReturnReceived
// @Tags order
// @Accept json
// @Produce json
// @Param request body AdminConfirmReceiveRequest true "确认收货入库参数"
// @Success 200 {object} response.Response[any]
// @Router /api/order/after-sales/receive [post]
func (a *API) Receive(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "未登录")
		return
	}

	var req AdminConfirmReceiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	if err := a.service.AdminConfirmReturnReceived(c.Request.Context(), user, req); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}

// RetryRefund godoc
// @Summary 重试售后退款
// @ID retryAfterSaleRefund
// @Tags order
// @Accept json
// @Produce json
// @Param request body AdminRetryRefundRequest true "重试退款参数"
// @Success 200 {object} response.Response[any]
// @Router /api/order/after-sales/retry-refund [post]
func (a *API) RetryRefund(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "未登录")
		return
	}

	var req AdminRetryRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败")
		return
	}

	if err := a.service.AdminRetryRefund(c.Request.Context(), user, req); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}
