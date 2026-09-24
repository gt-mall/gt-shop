package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/wechatpay"
	productinventory "mall/admin_core/internal/product/inventory"
	"mall/admin_core/pkg/ordertimeout"
	"mall/admin_core/pkg/util"
)

type PrepayResponse struct {
	PaymentNo string `json:"payment_no"`
	AppID     string `json:"appId"`
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

func (s *Service) Prepay(ctx context.Context, memberID, orderID int64) (*PrepayResponse, error) {
	cfg, err := (wechatpay.Store{DB: s.db}).Load(ctx)
	if err != nil {
		return nil, err
	}
	if err = cfg.Validate(); err != nil {
		return nil, err
	}
	var mp struct {
		AppID string `json:"app_id"`
	}
	// The AppID is fetched through the existing miniapp settings service.
	mpConfig, err := s.mpConfig(ctx)
	if err != nil {
		return nil, err
	}
	mp.AppID = mpConfig
	if mp.AppID == "" {
		return nil, errors.New("请先配置小程序 AppID")
	}
	var memberAuth model.MemberAuth
	if err = s.db.WithContext(ctx).Where("member_id = ? AND auth_type = ? AND app_id = ?", memberID, "wechat_miniapp", mp.AppID).First(&memberAuth).Error; err != nil {
		return nil, errors.New("请先使用微信小程序登录")
	}
	payment, err := s.CreatePayment(ctx, memberID, CreatePaymentRequest{OrderID: orderID})
	if err != nil {
		return nil, err
	}
	var order model.OrderMain
	if err = s.db.WithContext(ctx).Where("id = ?", payment.OrderID).First(&order).Error; err != nil {
		return nil, err
	}
	expires := order.CreatedAt.Add(ordertimeout.DefaultOrderExpiry)
	if order.ExpireAt != nil {
		expires = *order.ExpireAt
	}
	if !expires.After(time.Now().Add(time.Minute)) {
		return nil, ErrOrderExpired
	}
	result, err := cfg.Prepay(ctx, mp.AppID, memberAuth.OpenID, payment.PaymentNo, "商城订单 "+order.OrderNo, payment.Amount, expires)
	if err != nil {
		return nil, fmt.Errorf("微信预支付下单失败: %w", err)
	}
	var latest model.OrderMain
	if err = s.db.WithContext(ctx).Select("status").Where("id = ?", orderID).First(&latest).Error; err != nil {
		return nil, err
	}
	if latest.Status != model.OrderStatusPendingPayment {
		_ = cfg.ClosePayment(ctx, payment.PaymentNo)
		return nil, ErrOrderCancelled
	}
	if result == nil || result.Appid == nil || result.TimeStamp == nil || result.NonceStr == nil || result.Package == nil || result.SignType == nil || result.PaySign == nil {
		return nil, errors.New("微信预支付参数不完整")
	}
	return &PrepayResponse{PaymentNo: payment.PaymentNo, AppID: *result.Appid, TimeStamp: *result.TimeStamp, NonceStr: *result.NonceStr, Package: *result.Package, SignType: *result.SignType, PaySign: *result.PaySign}, nil
}

func (s *Service) mpConfig(ctx context.Context) (string, error) {
	var row model.SystemConfig
	if err := s.db.WithContext(ctx).Where("config_key = ?", "mp_basic").First(&row).Error; err != nil {
		return "", err
	}
	var cfg struct {
		AppID string `json:"app_id"`
	}
	if err := json.Unmarshal([]byte(row.Value), &cfg); err != nil {
		return "", err
	}
	return cfg.AppID, nil
}

func (s *Service) Query(ctx context.Context, memberID int64, paymentNo string) (*model.OrderPayment, error) {
	var p model.OrderPayment
	if err := s.db.WithContext(ctx).Where("payment_no = ?", paymentNo).First(&p).Error; err != nil {
		return nil, ErrPaymentNotFound
	}
	var order model.OrderMain
	if err := s.db.WithContext(ctx).Where("id = ? AND member_id = ?", p.OrderID, memberID).First(&order).Error; err != nil {
		return nil, ErrOrderNotBelong
	}
	if p.Status == model.PaymentStatusSuccess {
		return &p, nil
	}
	cfg, err := (wechatpay.Store{DB: s.db}).Load(ctx)
	if err != nil {
		return nil, err
	}
	txn, err := cfg.QueryPayment(ctx, paymentNo)
	if err != nil {
		return nil, err
	}
	if txn.TradeState != nil && *txn.TradeState == "SUCCESS" {
		return s.ConfirmWechat(ctx, txn)
	}
	return &p, nil
}

func (s *Service) ConfirmWechat(ctx context.Context, txn *payments.Transaction) (*model.OrderPayment, error) {
	if txn == nil || txn.OutTradeNo == nil || txn.TradeState == nil || *txn.TradeState != "SUCCESS" || txn.TransactionId == nil || txn.Amount == nil || txn.Amount.Total == nil || txn.Mchid == nil || txn.Appid == nil || txn.Payer == nil || txn.Payer.Openid == nil {
		return nil, errors.New("微信交易信息不完整或未成功")
	}
	cfg, err := (wechatpay.Store{DB: s.db}).Load(ctx)
	if err != nil {
		return nil, err
	}
	appID, err := s.mpConfig(ctx)
	if err != nil {
		return nil, err
	}
	if *txn.Mchid != cfg.MchID || *txn.Appid != appID {
		return nil, errors.New("微信交易商户信息不匹配")
	}
	var p model.OrderPayment
	if err = s.db.WithContext(ctx).Where("payment_no = ? AND channel = ?", *txn.OutTradeNo, "wechat").First(&p).Error; err != nil {
		return nil, ErrPaymentNotFound
	}
	amount, err := util.ParseDecimalYuanToCent(p.Amount)
	if err != nil || amount != *txn.Amount.Total {
		return nil, errors.New("微信交易金额不匹配")
	}
	var auth model.MemberAuth
	if err = s.db.WithContext(ctx).Where("member_id = ? AND auth_type = ? AND app_id = ? AND openid = ?", pMemberID(s.db, ctx, p.OrderID), "wechat_miniapp", appID, *txn.Payer.Openid).First(&auth).Error; err != nil {
		return nil, errors.New("微信交易付款人不匹配")
	}
	invSvc := productinventory.NewService(s.db)
	var affected []int64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", p.OrderID).First(&order).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", p.ID).First(&p).Error; err != nil {
			return err
		}
		if p.Status == model.PaymentStatusSuccess {
			if p.TransactionNo != nil && *p.TransactionNo == *txn.TransactionId {
				return nil
			}
			return errors.New("支付单交易号冲突")
		}
		if p.Status != model.PaymentStatusPending || order.Status != model.OrderStatusPendingPayment {
			return errors.New("订单已关闭但微信交易成功，请立即人工处理退款")
		}
		var items []model.OrderItem
		if err := tx.Where("order_id = ?", order.ID).Find(&items).Error; err != nil {
			return err
		}
		sort.Slice(items, func(i, j int) bool { return items[i].SKUID < items[j].SKUID })
		for _, it := range items {
			if err := invSvc.OrderDeductLockedStockTx(ctx, tx, it.ProductID, it.SKUID, it.Quantity, order.OrderNo, order.MemberID, "member"); err != nil {
				return err
			}
		}
		now := time.Now()
		if err := tx.Model(&model.OrderMain{}).Where("id = ? AND status = ?", order.ID, model.OrderStatusPendingPayment).Updates(map[string]any{"status": model.OrderStatusPendingShipment, "paid_at": &now, "updated_at": now}).Error; err != nil {
			return err
		}
		counts := map[int64]int{}
		for _, it := range items {
			counts[it.ProductID] += it.Quantity
		}
		for id, qty := range counts {
			if err := tx.Model(&model.Product{}).Where("id = ?", id).Update("sales_count", gorm.Expr("sales_count + ?", qty)).Error; err != nil {
				return err
			}
			affected = append(affected, id)
		}
		if err := tx.Model(&model.OrderPayment{}).Where("id = ? AND status = ?", p.ID, model.PaymentStatusPending).Updates(map[string]any{"status": model.PaymentStatusSuccess, "transaction_no": *txn.TransactionId, "paid_at": &now, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.OrderPayment{}).Where("order_id = ? AND id <> ? AND status = ?", order.ID, p.ID, model.PaymentStatusPending).Updates(map[string]any{"status": model.PaymentStatusClosed, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.OrderLog{OrderID: order.ID, FromStatus: model.OrderStatusPendingPayment, ToStatus: model.OrderStatusPendingShipment, Action: "pay", OperatorType: "system", OperatorName: "wechat", Remark: "微信支付成功", CreatedAt: now}).Error; err != nil {
			return err
		}
		p.Status = model.PaymentStatusSuccess
		p.PaidAt = &now
		p.TransactionNo = txn.TransactionId
		return nil
	})
	if err == nil {
		s.invalidateProductSalesCache(affected)
	}
	return &p, err
}

func pMemberID(db *gorm.DB, ctx context.Context, orderID int64) int64 {
	var o model.OrderMain
	_ = db.WithContext(ctx).Select("member_id").Where("id = ?", orderID).First(&o).Error
	return o.MemberID
}

// EnsureClosableTx runs while the caller holds the order row lock. The provider must
// confirm that it can no longer accept a payment before local stock is released.
func (s *Service) EnsureClosableTx(ctx context.Context, tx *gorm.DB, orderID int64) error {
	var p model.OrderPayment
	err := tx.Where("order_id = ? AND channel = ? AND status = ?", orderID, "wechat", model.PaymentStatusPending).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	cfg, err := (wechatpay.Store{DB: s.db}).Load(ctx)
	if err != nil {
		return err
	}
	providerCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	txn, err := cfg.QueryPayment(providerCtx, p.PaymentNo)
	if err == nil && txn != nil && txn.TradeState != nil && *txn.TradeState == "SUCCESS" {
		return ErrOrderAlreadyPaid
	}
	if err == nil && txn != nil && txn.TradeState != nil && *txn.TradeState == "CLOSED" {
		return nil
	}
	if err = cfg.ClosePayment(providerCtx, p.PaymentNo); err != nil {
		return fmt.Errorf("微信订单关单失败，请稍后重试: %w", err)
	}
	return nil
}
