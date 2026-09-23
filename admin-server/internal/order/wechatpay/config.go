package wechatpay

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"mall/admin_core/internal/model"
)

const configKey = "wechat_pay"
const defaultNotifyBase = "https://gt-shop.xi-soft.cn"

// Config contains merchant credentials. It must never be returned to a client.
type Config struct {
	MchID              string `json:"mch_id"`
	MerchantSerialNo   string `json:"merchant_serial_no"`
	MerchantPrivateKey string `json:"merchant_private_key"`
	PublicKeyID        string `json:"public_key_id"`
	PublicKey          string `json:"public_key"`
	APIV3Key           string `json:"api_v3_key"`
	NotifyBaseURL      string `json:"notify_base_url"`
}

type PublicConfig struct {
	MchID                        string `json:"mch_id"`
	MerchantSerialNo             string `json:"merchant_serial_no"`
	PublicKeyID                  string `json:"public_key_id"`
	NotifyBaseURL                string `json:"notify_base_url"`
	MerchantPrivateKeyConfigured bool   `json:"merchant_private_key_configured"`
	PublicKeyConfigured          bool   `json:"public_key_configured"`
	APIV3KeyConfigured           bool   `json:"api_v3_key_configured"`
}

func (c Config) Public() PublicConfig {
	return PublicConfig{MchID: c.MchID, MerchantSerialNo: c.MerchantSerialNo, PublicKeyID: c.PublicKeyID, NotifyBaseURL: c.NotifyBaseURL, MerchantPrivateKeyConfigured: c.MerchantPrivateKey != "", PublicKeyConfigured: c.PublicKey != "", APIV3KeyConfigured: c.APIV3Key != ""}
}

func (c Config) Validate() error {
	if c.MchID == "" || c.MerchantSerialNo == "" || c.MerchantPrivateKey == "" || c.PublicKeyID == "" || c.PublicKey == "" || len(c.APIV3Key) != 32 {
		return errors.New("微信支付商户号、证书序列号、商户私钥、微信支付公钥 ID、公钥和 32 字节 APIv3 密钥必须配置")
	}
	u, err := url.Parse(c.NotifyBaseURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return errors.New("回调域名必须是公网 HTTPS 域名，不含路径或参数")
	}
	return nil
}
func (c Config) PaymentNotifyURL() string {
	return c.NotifyBaseURL + "/api/miniapp/payments/wechat/notify"
}
func (c Config) RefundNotifyURL() string {
	return c.NotifyBaseURL + "/api/miniapp/payments/wechat/refund-notify"
}

type Store struct{ DB *gorm.DB }

func cipherForConfig() (cipher.AEAD, error) {
	raw := strings.TrimSpace(os.Getenv("PAYMENT_CONFIG_KEY"))
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, errors.New("PAYMENT_CONFIG_KEY 未配置：需提供 base64 编码的 32 字节加密密钥")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (s Store) Load(ctx context.Context) (Config, error) {
	var cfg Config
	if s.DB == nil {
		return cfg, errors.New("数据库不可用")
	}
	var row model.SystemConfig
	err := s.DB.WithContext(ctx).Where("config_key = ?", configKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg.NotifyBaseURL = defaultNotifyBase
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	aead, err := cipherForConfig()
	if err != nil {
		return cfg, err
	}
	var stored struct {
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.Unmarshal([]byte(row.Value), &stored); err != nil {
		return cfg, errors.New("支付配置密文损坏")
	}
	raw, err := base64.StdEncoding.DecodeString(stored.Ciphertext)
	if err != nil {
		return cfg, errors.New("支付配置密文损坏")
	}
	if len(raw) < aead.NonceSize() {
		return cfg, errors.New("支付配置密文损坏")
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte(configKey))
	if err != nil {
		return cfg, errors.New("支付配置无法解密，请检查加密密钥")
	}
	if err = json.Unmarshal(plain, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (s Store) Save(ctx context.Context, input Config) error {
	old, err := s.Load(ctx)
	if err != nil {
		return err
	}
	input.MchID = strings.TrimSpace(input.MchID)
	input.MerchantSerialNo = strings.TrimSpace(input.MerchantSerialNo)
	input.PublicKeyID = strings.TrimSpace(input.PublicKeyID)
	input.NotifyBaseURL = strings.TrimRight(strings.TrimSpace(input.NotifyBaseURL), "/")
	if input.NotifyBaseURL == "" {
		input.NotifyBaseURL = defaultNotifyBase
	}
	if input.MerchantPrivateKey == "" {
		input.MerchantPrivateKey = old.MerchantPrivateKey
	}
	if input.PublicKey == "" {
		input.PublicKey = old.PublicKey
	}
	if input.APIV3Key == "" {
		input.APIV3Key = old.APIV3Key
	}
	if err = input.Validate(); err != nil {
		return err
	}
	if _, err = clientFor(input); err != nil {
		return fmt.Errorf("支付密钥无效: %w", err)
	}
	aead, err := cipherForConfig()
	if err != nil {
		return err
	}
	plain, err := json.Marshal(input)
	if err != nil {
		return err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return err
	}
	stored, err := json.Marshal(struct {
		Ciphertext string `json:"ciphertext"`
	}{Ciphertext: base64.StdEncoding.EncodeToString(aead.Seal(nonce, nonce, plain, []byte(configKey)))})
	if err != nil {
		return err
	}
	row := model.SystemConfig{ConfigKey: configKey, Value: string(stored)}
	return s.DB.WithContext(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "config_key"}}, DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"})}).Create(&row).Error
}
