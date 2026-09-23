package miniapp_test

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
	"gorm.io/gorm"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/wechatpay"
	"mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/util"
)

type wechatPaymentFixture struct {
	service  *miniapppayment.Service
	db       *gorm.DB
	memberID int64
	mchID    string
	appID    string
	openID   string
}

func newWechatPaymentFixture(t *testing.T, db *gorm.DB, memberID int64) *wechatPaymentFixture {
	t.Helper()
	t.Setenv("PAYMENT_CONFIG_KEY", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
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
	if err := (wechatpay.Store{DB: db}).Save(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	const appID, openID = "wx-test", "openid-test"
	if err := setting.NewService(db).SaveMp(context.Background(), setting.MpConfig{Name: "test", AppID: appID}); err != nil {
		t.Fatal(err)
	}
	if err := db.Where("member_id = ? AND auth_type = ? AND app_id = ?", memberID, "wechat_miniapp", appID).
		FirstOrCreate(&model.MemberAuth{MemberID: memberID, AuthType: "wechat_miniapp", AppID: appID, OpenID: openID}).Error; err != nil {
		t.Fatal(err)
	}
	return &wechatPaymentFixture{service: miniapppayment.NewService(db), db: db, memberID: memberID, mchID: cfg.MchID, appID: appID, openID: openID}
}

func (f *wechatPaymentFixture) transaction(t *testing.T, payment *model.OrderPayment) *payments.Transaction {
	t.Helper()
	cents, err := util.ParseDecimalYuanToCent(payment.Amount)
	if err != nil {
		t.Fatal(err)
	}
	return &payments.Transaction{
		OutTradeNo: core.String(payment.PaymentNo), TradeState: core.String("SUCCESS"),
		TransactionId: core.String(fmt.Sprintf("wx-test-transaction-%d", payment.ID)),
		Amount:        &payments.TransactionAmount{Total: core.Int64(cents)},
		Mchid:         core.String(f.mchID), Appid: core.String(f.appID),
		Payer: &payments.TransactionPayer{Openid: core.String(f.openID)},
	}
}

func (f *wechatPaymentFixture) pay(t *testing.T, orderID int64) (*model.OrderPayment, *payments.Transaction) {
	t.Helper()
	p, err := f.service.CreatePayment(context.Background(), f.memberID, miniapppayment.CreatePaymentRequest{OrderID: orderID})
	if err != nil {
		t.Fatal(err)
	}
	txn := f.transaction(t, p)
	result, err := f.service.ConfirmWechat(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	return result, txn
}
