package order_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"
	orderaftersale "mall/admin_core/internal/order/aftersale"
	"mall/admin_core/internal/order/refund"
	"mall/admin_core/internal/order/wechatpay"
	"mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/util"
)

// The test refund stub shares MockProvider's controllable behavior while matching
// the real WeChat payment channel. It is never wired into production routes.
type wechatRefundStub struct{ *refund.MockProvider }

func (wechatRefundStub) Channel() string { return "wechat" }

type orderWechatPaymentFixture struct {
	service  *payment.Service
	memberID int64
	mchID    string
}

func newOrderWechatPaymentFixture(t *testing.T, env *testAfterSaleEnv) *orderWechatPaymentFixture {
	t.Helper()
	t.Setenv("PAYMENT_CONFIG_KEY", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	if err := env.db.AutoMigrate(&model.SystemConfig{}, &model.MemberAuth{}); err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privateBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	cfg := wechatpay.Config{
		MchID: "1234567890", MerchantSerialNo: "SERIAL",
		MerchantPrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes})),
		PublicKeyID:        "PUB_KEY_ID_TEST",
		PublicKey:          string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes})),
		APIV3Key:           "12345678901234567890123456789012",
		NotifyBaseURL:      "https://gt-shop.xi-soft.cn",
	}
	if err := (wechatpay.Store{DB: env.db}).Save(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if err := setting.NewService(env.db).SaveMp(context.Background(), setting.MpConfig{Name: "test", AppID: "wx-test"}); err != nil {
		t.Fatal(err)
	}
	if err := env.db.Create(&model.MemberAuth{MemberID: env.memberID, AuthType: "wechat_miniapp", AppID: "wx-test", OpenID: "openid-test"}).Error; err != nil {
		t.Fatal(err)
	}
	env.asService = orderaftersale.NewService(env.db, env.bizNoGen, wechatRefundStub{env.refundProvider})
	return &orderWechatPaymentFixture{service: payment.NewService(env.db, env.bizNoGen), memberID: env.memberID, mchID: cfg.MchID}
}

func (f *orderWechatPaymentFixture) pay(t *testing.T, orderID int64) (*model.OrderPayment, *payments.Transaction) {
	t.Helper()
	ctx := context.Background()
	p, err := f.service.CreatePayment(ctx, f.memberID, payment.CreatePaymentRequest{OrderID: orderID})
	if err != nil {
		t.Fatal(err)
	}
	cents, err := util.ParseDecimalYuanToCent(p.Amount)
	if err != nil {
		t.Fatal(err)
	}
	txn := &payments.Transaction{
		OutTradeNo: core.String(p.PaymentNo), TradeState: core.String("SUCCESS"),
		TransactionId: core.String(fmt.Sprintf("wx-test-transaction-%d", p.ID)),
		Amount:        &payments.TransactionAmount{Total: core.Int64(cents)},
		Mchid:         core.String(f.mchID), Appid: core.String("wx-test"),
		Payer: &payments.TransactionPayer{Openid: core.String("openid-test")},
	}
	result, err := f.service.ConfirmWechat(ctx, txn)
	if err != nil {
		t.Fatal(err)
	}
	return result, txn
}
