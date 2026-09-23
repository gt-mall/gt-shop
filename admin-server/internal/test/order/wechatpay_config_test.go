package order_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/wechatpay"
)

func TestWechatPayConfigEncryptedAndWriteOnly(t *testing.T) {
	t.Setenv("PAYMENT_CONFIG_KEY", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	defer sqlDB.Close()
	if err := db.AutoMigrate(&model.SystemConfig{}); err != nil {
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
	private := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateBytes}))
	public := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes}))
	store := wechatpay.Store{DB: db}
	cfg := wechatpay.Config{MchID: "1234567890", MerchantSerialNo: "SERIAL", MerchantPrivateKey: private, PublicKeyID: "PUB_KEY_ID_TEST", PublicKey: public, APIV3Key: "12345678901234567890123456789012", NotifyBaseURL: "https://gt-shop.xi-soft.cn"}
	if err := store.Save(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	var row model.SystemConfig
	if err := db.Where("config_key = ?", "wechat_pay").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(row.Value, "PRIVATE KEY") || strings.Contains(row.Value, cfg.APIV3Key) {
		t.Fatal("merchant secrets stored as plaintext")
	}
	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.MerchantPrivateKey != private || loaded.APIV3Key != cfg.APIV3Key {
		t.Fatal("encrypted config roundtrip failed")
	}
	masked := loaded.Public()
	if !masked.MerchantPrivateKeyConfigured || !masked.APIV3KeyConfigured {
		t.Fatal("configured flags missing")
	}
	if strings.Contains(masked.NotifyBaseURL, cfg.APIV3Key) {
		t.Fatal("secret leaked")
	}
	cfg.MerchantPrivateKey = ""
	cfg.PublicKey = ""
	cfg.APIV3Key = ""
	if err := store.Save(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Load(context.Background())
	if err != nil || loaded.APIV3Key != "12345678901234567890123456789012" {
		t.Fatal("blank secret did not preserve stored value")
	}
	t.Setenv("PAYMENT_CONFIG_KEY", base64.StdEncoding.EncodeToString([]byte("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")))
	if _, err = store.Load(context.Background()); err == nil {
		t.Fatal("wrong encryption key decrypted payment config")
	}
}
