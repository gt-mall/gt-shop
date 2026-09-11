package payment

import (
	"github.com/gin-gonic/gin"

	miniappauth "mall/admin_core/internal/miniapp/auth"
	"mall/admin_core/pkg/response"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// CreatePayment godoc
// @Summary 创建支付单
// @ID miniappCreatePayment
// @Tags payment
// @Accept json
// @Produce json
// @Param request body CreatePaymentRequest true "创建支付单参数"
// @Success 200 {object} response.Response[PaymentResponse]
// @Failure 200 {object} response.Response[any]
// @Router /payments/create [post]
func (a *API) CreatePayment(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	p, err := a.service.CreatePayment(c.Request.Context(), member.ID, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	resp := BuildPaymentResponse(p)
	response.Data(c, resp, 0)
}

// MockPay godoc
// @Summary 模拟支付成功
// @ID miniappMockPay
// @Tags payment
// @Accept json
// @Produce json
// @Param request body MockPayRequest true "模拟支付参数"
// @Success 200 {object} response.Response[PaymentResponse]
// @Failure 200 {object} response.Response[any]
// @Router /payments/mock-pay [post]
func (a *API) MockPay(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req MockPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	p, err := a.service.MockPay(c.Request.Context(), member.ID, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	resp := BuildPaymentResponse(p)
	response.Data(c, resp, 0)
}
