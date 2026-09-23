package miniapp_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	miniapporder "mall/admin_core/internal/miniapp/order"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/wechatpay"
	"mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/util"
)

func TestWechatPaymentConfirmationChecksIdentityAmountAndIdempotency(t *testing.T) {
	t.Setenv("PAYMENT_CONFIG_KEY", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID)
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
	cfg := wechatpay.Config{MchID: "1234567890", MerchantSerialNo: "SERIAL", MerchantPrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes})), PublicKeyID: "PUB_KEY_ID_TEST", PublicKey: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes})), APIV3Key: "12345678901234567890123456789012", NotifyBaseURL: "https://gt-shop.xi-soft.cn"}
	if err := (wechatpay.Store{DB: db}).Save(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if err := setting.NewService(db).SaveMp(context.Background(), setting.MpConfig{Name: "test", AppID: "wx-test"}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.MemberAuth{MemberID: memberID, AuthType: "wechat_miniapp", AppID: "wx-test", OpenID: "openid-1"}).Error; err != nil {
		t.Fatal(err)
	}
	order, err := miniapporder.NewService(db).Create(context.Background(), model.Member{ID: memberID}, miniapporder.OrderCreateRequest{RequestID: "real-payment-confirm", AddressID: addrID, Items: []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	svc := miniapppayment.NewService(db)
	p, err := svc.CreatePayment(context.Background(), memberID, miniapppayment.CreatePaymentRequest{OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	cents, err := util.ParseDecimalYuanToCent(p.Amount)
	if err != nil {
		t.Fatal(err)
	}
	txn := &payments.Transaction{OutTradeNo: core.String(p.PaymentNo), TradeState: core.String("SUCCESS"), TransactionId: core.String("wx-transaction-1"), Amount: &payments.TransactionAmount{Total: core.Int64(cents)}, Mchid: core.String(cfg.MchID), Appid: core.String("wx-test"), Payer: &payments.TransactionPayer{Openid: core.String("openid-1")}}
	txn.Amount.Total = core.Int64(cents + 1)
	if _, err := svc.ConfirmWechat(context.Background(), txn); err == nil {
		t.Fatal("mismatched amount accepted")
	}
	txn.Amount.Total = core.Int64(cents)
	txn.Payer.Openid = core.String("wrong-openid")
	if _, err := svc.ConfirmWechat(context.Background(), txn); err == nil {
		t.Fatal("mismatched payer accepted")
	}
	txn.Payer.Openid = core.String("openid-1")
	for i := 0; i < 2; i++ {
		result, err := svc.ConfirmWechat(context.Background(), txn)
		if err != nil || result.Status != model.PaymentStatusSuccess {
			t.Fatalf("valid or repeated callback failed: %v", err)
		}
	}
	var saved model.OrderMain
	if err := db.First(&saved, order.ID).Error; err != nil || saved.Status != model.OrderStatusPendingShipment {
		t.Fatalf("order not paid: %v, %d", err, saved.Status)
	}
	var inv model.ProductSkuInventory
	if err := db.Where("sku_id = ?", skuID).First(&inv).Error; err != nil || inv.Stock != 49 || inv.LockedStock != 0 {
		t.Fatalf("stock deducted incorrectly: %v, %+v", err, inv)
	}
	var logs int64
	if err := db.Model(&model.OrderLog{}).Where("order_id = ? AND action = ?", order.ID, "pay").Count(&logs).Error; err != nil || logs != 1 {
		t.Fatalf("duplicate payment log: %v, %d", err, logs)
	}
}
