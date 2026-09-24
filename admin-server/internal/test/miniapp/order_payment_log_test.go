package miniapp_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	miniapporder "mall/admin_core/internal/miniapp/order"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/wechatpay"
	"mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/logger"
	"mall/admin_core/pkg/util"
)

func setupOrderPaymentLogDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	logCfg := logger.Config{
		Level:      "debug",
		Path:       tmpDir,
		Console:    false,
		File:       true,
		MaxSize:    10,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := logger.Init(logCfg); err != nil {
		t.Fatalf("logger init failed: %v", err)
	}
	t.Cleanup(func() {
		_ = logger.Close()
	})
	return tmpDir
}

func TestOrderCreationAndCancellationLogging(t *testing.T) {
	logDir := setupOrderPaymentLogDir(t)
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID)

	orderSvc := miniapporder.NewService(db)
	ctx := context.Background()

	// 1. Create order
	resp, err := orderSvc.Create(ctx, model.Member{ID: memberID, Nickname: "TestUser"}, miniapporder.OrderCreateRequest{
		RequestID: "log-test-order-create-001",
		AddressID: addrID,
		Items: []miniapporder.CheckoutItemRequest{
			{SKUID: skuID, Quantity: 1},
		},
		Remark: "测试下单日志",
	})
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}
	_ = logger.Sync()

	appLogFile := filepath.Join(logDir, "app.log")
	content, err := os.ReadFile(appLogFile)
	if err != nil {
		t.Fatalf("read app.log: %v", err)
	}
	logStr := string(content)

	if !strings.Contains(logStr, "user order create success") {
		t.Errorf("expected 'user order create success' in log, got:\n%s", logStr)
	}
	if !strings.Contains(logStr, resp.OrderNo) {
		t.Errorf("expected order_no %q in log", resp.OrderNo)
	}
	if !strings.Contains(logStr, "log-test-order-create-001") {
		t.Errorf("expected request_id in log")
	}

	// 2. Idempotent hit
	_, err = orderSvc.Create(ctx, model.Member{ID: memberID, Nickname: "TestUser"}, miniapporder.OrderCreateRequest{
		RequestID: "log-test-order-create-001",
		AddressID: addrID,
		Items: []miniapporder.CheckoutItemRequest{
			{SKUID: skuID, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("idempotent create failed: %v", err)
	}
	_ = logger.Sync()

	content, _ = os.ReadFile(appLogFile)
	logStr = string(content)
	if !strings.Contains(logStr, "user order create idempotent hit") {
		t.Errorf("expected 'user order create idempotent hit' in log, got:\n%s", logStr)
	}

	// 3. Cancel order
	err = orderSvc.Cancel(ctx, model.Member{ID: memberID, Nickname: "TestUser"}, resp.ID)
	if err != nil {
		t.Fatalf("cancel order failed: %v", err)
	}
	_ = logger.Sync()

	content, _ = os.ReadFile(appLogFile)
	logStr = string(content)
	if !strings.Contains(logStr, "user order cancel success") {
		t.Errorf("expected 'user order cancel success' in log, got:\n%s", logStr)
	}
}

func TestPaymentCreationAndTimeoutLogging(t *testing.T) {
	logDir := setupOrderPaymentLogDir(t)
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID)

	orderSvc := miniapporder.NewService(db)
	ctx := context.Background()

	// 1. Create order
	resp, err := orderSvc.Create(ctx, model.Member{ID: memberID, Nickname: "PaymentLogger"}, miniapporder.OrderCreateRequest{
		RequestID: "log-test-payment-001",
		AddressID: addrID,
		Items: []miniapporder.CheckoutItemRequest{
			{SKUID: skuID, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	// 2. Create Payment
	paymentSvc := miniapppayment.NewService(db)
	payment, err := paymentSvc.CreatePayment(ctx, memberID, miniapppayment.CreatePaymentRequest{
		OrderID: resp.ID,
	})
	if err != nil {
		t.Fatalf("create payment failed: %v", err)
	}
	_ = logger.Sync()

	appLogFile := filepath.Join(logDir, "app.log")
	content, err := os.ReadFile(appLogFile)
	if err != nil {
		t.Fatalf("read app.log: %v", err)
	}
	logStr := string(content)

	if !strings.Contains(logStr, "order payment created") {
		t.Errorf("expected 'order payment created' in log, got:\n%s", logStr)
	}
	if !strings.Contains(logStr, payment.PaymentNo) {
		t.Errorf("expected payment_no in log")
	}

	// 3. Timeout Close on an order without external wechat gateway call
	resp2, err := orderSvc.Create(ctx, model.Member{ID: memberID, Nickname: "PaymentLogger"}, miniapporder.OrderCreateRequest{
		RequestID: "log-test-payment-timeout-002",
		AddressID: addrID,
		Items: []miniapporder.CheckoutItemRequest{
			{SKUID: skuID, Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("create order 2 failed: %v", err)
	}

	past := time.Now().Add(-2 * time.Hour)
	if err := db.Model(&model.OrderMain{}).Where("id = ?", resp2.ID).Updates(map[string]any{
		"created_at": past,
		"expire_at":  past.Add(30 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("update order time: %v", err)
	}

	err = paymentSvc.CloseTimeoutOrder(ctx, resp2.ID)
	if err != nil {
		t.Fatalf("close timeout order failed: %v", err)
	}
	_ = logger.Sync()

	content, _ = os.ReadFile(appLogFile)
	logStr = string(content)
	if !strings.Contains(logStr, "order closed due to payment timeout") {
		t.Errorf("expected 'order closed due to payment timeout' in log, got:\n%s", logStr)
	}
}

func TestWechatPaymentConfirmationLogging(t *testing.T) {
	t.Setenv("PAYMENT_CONFIG_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")
	logDir := setupOrderPaymentLogDir(t)
	db, _, _, memberID, _, _ := setupOrderTestApp(t)
	_, skuID, addrID := createOrderTestData(t, db, memberID)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privateBytes, _ := x509.MarshalPKCS8PrivateKey(key)
	publicBytes, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	cfg := wechatpay.Config{
		MchID:              "1234567890",
		MerchantSerialNo:   "SERIAL",
		MerchantPrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes})),
		PublicKeyID:        "PUB_KEY_ID_TEST",
		PublicKey:          string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes})),
		APIV3Key:           "12345678901234567890123456789012",
		NotifyBaseURL:      "https://gt-shop.xi-soft.cn",
	}
	if err := (wechatpay.Store{DB: db}).Save(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	_ = setting.NewService(db).SaveMp(context.Background(), setting.MpConfig{Name: "test", AppID: "wx-test"})
	_ = db.Create(&model.MemberAuth{MemberID: memberID, AuthType: "wechat_miniapp", AppID: "wx-test", OpenID: "openid-log-test"}).Error

	order, err := miniapporder.NewService(db).Create(context.Background(), model.Member{ID: memberID}, miniapporder.OrderCreateRequest{
		RequestID: "real-payment-log-confirm",
		AddressID: addrID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := miniapppayment.NewService(db)
	p, err := svc.CreatePayment(context.Background(), memberID, miniapppayment.CreatePaymentRequest{OrderID: order.ID})
	if err != nil {
		t.Fatal(err)
	}
	cents, _ := util.ParseDecimalYuanToCent(p.Amount)

	// 1. Mismatched amount -> error log
	txn := &payments.Transaction{
		OutTradeNo:    core.String(p.PaymentNo),
		TradeState:    core.String("SUCCESS"),
		TransactionId: core.String("wx-txn-log-1"),
		Amount:        &payments.TransactionAmount{Total: core.Int64(cents + 999)},
		Mchid:         core.String(cfg.MchID),
		Appid:         core.String("wx-test"),
		Payer:         &payments.TransactionPayer{Openid: core.String("openid-log-test")},
	}
	_, err = svc.ConfirmWechat(context.Background(), txn)
	if err == nil {
		t.Fatal("expected amount mismatch error")
	}
	_ = logger.Sync()

	appLogFile := filepath.Join(logDir, "app.log")
	content, _ := os.ReadFile(appLogFile)
	logStr := string(content)
	if !strings.Contains(logStr, "wechat payment confirm failed: amount mismatch") {
		t.Errorf("expected amount mismatch log, got:\n%s", logStr)
	}

	// 2. Correct confirmation -> success log
	txn.Amount.Total = core.Int64(cents)
	result, err := svc.ConfirmWechat(context.Background(), txn)
	if err != nil || result.Status != model.PaymentStatusSuccess {
		t.Fatalf("confirm failed: %v", err)
	}
	_ = logger.Sync()

	content, _ = os.ReadFile(appLogFile)
	logStr = string(content)
	if !strings.Contains(logStr, "wechat payment confirmed success") {
		t.Errorf("expected 'wechat payment confirmed success' in log, got:\n%s", logStr)
	}
	if !strings.Contains(logStr, p.PaymentNo) {
		t.Errorf("expected payment_no in log")
	}
	if !strings.Contains(logStr, "wx-txn-log-1") {
		t.Errorf("expected transaction_no in log")
	}
}

