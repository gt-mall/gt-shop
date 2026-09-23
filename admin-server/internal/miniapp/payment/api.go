package payment

import (
	"github.com/gin-gonic/gin"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/aftersale"
	"mall/admin_core/internal/order/wechatpay"

	miniappauth "mall/admin_core/internal/miniapp/auth"
	"mall/admin_core/pkg/response"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// Prepay godoc
// @Summary 微信小程序预支付下单
// @ID miniappWechatPrepay
// @Tags payment
// @Accept json
// @Produce json
// @Param request body PrepayRequest true "订单"
// @Success 200 {object} response.Response[PrepayResponse]
// @Router /payments/prepay [post]
func (a *API) Prepay(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}
	var req PrepayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "订单参数错误")
		return
	}
	p, err := a.service.Prepay(c.Request.Context(), member.ID, req.OrderID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, p, 0)
}

// Query godoc
// @Summary 查询微信支付状态并同步订单
// @ID miniappWechatPaymentQuery
// @Tags payment
// @Produce json
// @Param payment_no path string true "商户支付单号"
// @Success 200 {object} response.Response[PaymentResponse]
// @Router /payments/{payment_no} [get]
func (a *API) Query(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}
	p, err := a.service.Query(c.Request.Context(), member.ID, c.Param("payment_no"))
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, BuildPaymentResponse(p), 0)
}

// WechatNotify godoc
// @Summary 微信支付异步通知
// @ID miniappWechatPaymentNotify
// @Tags payment
// @Router /payments/wechat/notify [post]
func (a *API) WechatNotify(c *gin.Context) {
	cfg, err := (wechatpay.Store{DB: a.service.db}).Load(c.Request.Context())
	if err != nil {
		c.Status(500)
		return
	}
	var txn payments.Transaction
	if err = cfg.ParseNotify(c.Request.Context(), c.Request, &txn); err != nil {
		c.Status(401)
		return
	}
	if _, err = a.service.ConfirmWechat(c.Request.Context(), &txn); err != nil {
		c.Status(500)
		return
	}
	c.JSON(200, gin.H{"code": "SUCCESS", "message": "成功"})
}

// WechatRefundNotify godoc
// @Summary 微信退款异步通知
// @ID miniappWechatRefundNotify
// @Tags payment
// @Router /payments/wechat/refund-notify [post]
func (a *API) WechatRefundNotify(c *gin.Context) {
	cfg, err := (wechatpay.Store{DB: a.service.db}).Load(c.Request.Context())
	if err != nil {
		c.Status(500)
		return
	}
	var data struct {
		OutRefundNo string `json:"out_refund_no"`
		Status      string `json:"refund_status"`
	}
	if err = cfg.ParseNotify(c.Request.Context(), c.Request, &data); err != nil {
		c.Status(401)
		return
	}
	var row model.OrderRefund
	if err = a.service.db.WithContext(c.Request.Context()).Where("refund_no = ?", data.OutRefundNo).First(&row).Error; err != nil {
		c.Status(500)
		return
	}
	svc := aftersale.NewService(a.service.db, nil, wechatpay.RefundProvider{Store: wechatpay.Store{DB: a.service.db}})
	if err = svc.ReconcileRefund(c.Request.Context(), row.AfterSaleID); err != nil {
		c.Status(500)
		return
	}
	c.JSON(200, gin.H{"code": "SUCCESS", "message": "成功"})
}
