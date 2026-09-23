package wechatpay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	"mall/admin_core/internal/order/refund"
	"mall/admin_core/pkg/util"
)

func clientFor(cfg Config) (*core.Client, error) {
	privateKey, err := utils.LoadPrivateKey(cfg.MerchantPrivateKey)
	if err != nil {
		return nil, errors.New("商户私钥格式错误")
	}
	publicKey, err := utils.LoadPublicKey(cfg.PublicKey)
	if err != nil {
		return nil, errors.New("微信支付公钥格式错误")
	}
	return core.NewClient(context.Background(), option.WithWechatPayPublicKeyAuthCipher(cfg.MchID, cfg.MerchantSerialNo, privateKey, cfg.PublicKeyID, publicKey))
}

func (cfg Config) NotifyHandler() (*notify.Handler, error) {
	publicKey, err := utils.LoadPublicKey(cfg.PublicKey)
	if err != nil {
		return nil, err
	}
	return notify.NewRSANotifyHandler(cfg.APIV3Key, verifiers.NewSHA256WithRSAPubkeyVerifier(cfg.PublicKeyID, *publicKey))
}

func (cfg Config) Prepay(ctx context.Context, appID, openid, paymentNo, description, amount string, expires time.Time) (*jsapi.PrepayWithRequestPaymentResponse, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cents, err := util.ParseDecimalYuanToCent(amount)
	if err != nil || cents <= 0 {
		return nil, errors.New("支付金额无效")
	}
	client, err := clientFor(cfg)
	if err != nil {
		return nil, err
	}
	api := jsapi.JsapiApiService{Client: client}
	resp, _, err := api.PrepayWithRequestPayment(ctx, jsapi.PrepayRequest{Appid: core.String(appID), Mchid: core.String(cfg.MchID), Description: core.String(description), OutTradeNo: core.String(paymentNo), TimeExpire: &expires, NotifyUrl: core.String(cfg.PaymentNotifyURL()), Amount: &jsapi.Amount{Total: core.Int64(cents), Currency: core.String("CNY")}, Payer: &jsapi.Payer{Openid: core.String(openid)}})
	return resp, err
}

func (cfg Config) QueryPayment(ctx context.Context, paymentNo string) (*payments.Transaction, error) {
	client, err := clientFor(cfg)
	if err != nil {
		return nil, err
	}
	api := jsapi.JsapiApiService{Client: client}
	txn, _, err := api.QueryOrderByOutTradeNo(ctx, jsapi.QueryOrderByOutTradeNoRequest{OutTradeNo: core.String(paymentNo), Mchid: core.String(cfg.MchID)})
	return txn, err
}

func (cfg Config) ClosePayment(ctx context.Context, paymentNo string) error {
	client, err := clientFor(cfg)
	if err != nil {
		return err
	}
	api := jsapi.JsapiApiService{Client: client}
	_, err = api.CloseOrder(ctx, jsapi.CloseOrderRequest{OutTradeNo: core.String(paymentNo), Mchid: core.String(cfg.MchID)})
	return err
}

type RefundProvider struct{ Store Store }

func (RefundProvider) Channel() string { return "wechat" }

func (p RefundProvider) Refund(ctx context.Context, req refund.RefundRequest) (*refund.RefundResult, error) {
	cfg, err := p.Store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if err = cfg.Validate(); err != nil {
		return nil, err
	}
	refundCents, err := util.ParseDecimalYuanToCent(req.RefundAmount)
	if err != nil {
		return nil, err
	}
	totalCents, err := util.ParseDecimalYuanToCent(req.TotalAmount)
	if err != nil {
		return nil, err
	}
	if refundCents <= 0 || totalCents < refundCents {
		return nil, errors.New("退款金额超出原支付金额")
	}
	client, err := clientFor(cfg)
	if err != nil {
		return nil, err
	}
	api := refunddomestic.RefundsApiService{Client: client}
	resp, _, err := api.Create(ctx, refunddomestic.CreateRequest{OutTradeNo: core.String(req.PaymentNo), OutRefundNo: core.String(req.RefundNo), Reason: core.String(req.Reason), NotifyUrl: core.String(cfg.RefundNotifyURL()), Amount: &refunddomestic.AmountReq{Refund: core.Int64(refundCents), Total: core.Int64(totalCents), Currency: core.String("CNY")}})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Status == nil {
		return nil, errors.New("微信退款响应缺少状态")
	}
	id := ""
	if resp.RefundId != nil {
		id = *resp.RefundId
	}
	return &refund.RefundResult{Success: *resp.Status == refunddomestic.STATUS_SUCCESS, Accepted: *resp.Status == refunddomestic.STATUS_PROCESSING || *resp.Status == refunddomestic.STATUS_ABNORMAL, ProviderRefundNo: id}, nil
}

func (p RefundProvider) QueryRefund(ctx context.Context, refundNo, providerRefundNo string) (*refund.RefundQueryResult, error) {
	cfg, err := p.Store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if err = cfg.Validate(); err != nil {
		return nil, err
	}
	client, err := clientFor(cfg)
	if err != nil {
		return nil, err
	}
	api := refunddomestic.RefundsApiService{Client: client}
	resp, _, err := api.QueryByOutRefundNo(ctx, refunddomestic.QueryByOutRefundNoRequest{OutRefundNo: core.String(refundNo)})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Status == nil {
		return nil, errors.New("微信退款查询响应缺少状态")
	}
	status := refund.RefundQueryStatusProcessing
	switch *resp.Status {
	case refunddomestic.STATUS_SUCCESS:
		status = refund.RefundQueryStatusSuccess
	case refunddomestic.STATUS_CLOSED:
		status = refund.RefundQueryStatusFailed
	}
	id := ""
	if resp.RefundId != nil {
		id = *resp.RefundId
	}
	return &refund.RefundQueryResult{Status: status, ProviderRefundNo: id}, nil
}

// ParseNotify verifies the WeChat signature and decrypts an authenticated notification.
func (cfg Config) ParseNotify(ctx context.Context, r *http.Request, dst any) error {
	handler, err := cfg.NotifyHandler()
	if err != nil {
		return err
	}
	_, err = handler.ParseNotifyRequest(ctx, r, dst)
	if err != nil {
		return fmt.Errorf("微信支付通知验证失败: %w", err)
	}
	return nil
}
