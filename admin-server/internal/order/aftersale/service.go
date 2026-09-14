package aftersale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/refund"
	productinventory "mall/admin_core/internal/product/inventory"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/bizno"
	"mall/admin_core/pkg/util"
)

var (
	ErrNotFound            = errors.New("售后单不存在")
	ErrOrderNotFound       = errors.New("订单不存在")
	ErrOrderNotBelong      = errors.New("无权操作该订单")
	ErrAfterSaleNotBelong  = errors.New("无权操作该售后单")
	ErrInvalidStatus       = errors.New("当前售后状态不可执行该操作")
	ErrActiveAfterSale     = errors.New("该订单已有进行中的售后申请")
	ErrAlreadyRefunded     = errors.New("该订单已完成退款")
	ErrProofFileInvalid    = errors.New("包含无效或已删除的凭证图片")
	ErrProofTooMany        = errors.New("凭证图片最多9张")
	ErrReturnAddressEmpty  = errors.New("审核退货退款必须填写完整的退货收件人、电话及地址")
	ErrTrackingEmpty       = errors.New("物流公司和运单号不能为空")
	ErrRestockQtyInvalid   = errors.New("重新入库数量不能为负且不能大于售后商品数量")
	ErrNoEligiblePayment   = errors.New("未找到有效的成功支付记录，不能退款")
	ErrRefundStatusUnknown = errors.New("支付渠道暂无该退款记录，请人工确认退款状态")
)

type Service struct {
	db             *gorm.DB
	bizNoGen       *bizno.Generator
	refundProvider refund.Provider
	invSvc         *productinventory.Service
	fileSvc        *systemfile.Service
}

func NewService(db *gorm.DB, bizNoGen *bizno.Generator, refundProvider refund.Provider) *Service {
	if bizNoGen == nil {
		var err error
		bizNoGen, err = bizno.NewGenerator(0)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize default bizno generator in aftersale: %v", err))
		}
	}
	if refundProvider == nil {
		refundProvider = refund.NewMockProvider()
	}
	return &Service{
		db:             db,
		bizNoGen:       bizNoGen,
		refundProvider: refundProvider,
		invSvc:         productinventory.NewService(db),
		fileSvc:        systemfile.NewService(db),
	}
}

// Apply creates an after-sale request for a member.
func (s *Service) Apply(ctx context.Context, memberID int64, req MiniappApplyRequest) (*model.AfterSale, error) {
	if s.db == nil {
		return nil, errors.New("数据库服务不可用")
	}
	if memberID <= 0 {
		return nil, ErrOrderNotBelong
	}
	if req.OrderID <= 0 {
		return nil, errors.New("订单ID无效")
	}
	if req.Type != model.AfterSaleTypeRefundOnly && req.Type != model.AfterSaleTypeReturnRefund {
		return nil, errors.New("不支持的售后类型")
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errors.New("售后原因不能为空")
	}
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		return nil, errors.New("请求幂等号不能为空")
	}

	// Validate & deduplicate proof file IDs
	dedupedProofIDs := make([]int64, 0, len(req.ProofFileIDs))
	seenProof := make(map[int64]bool)
	for _, pid := range req.ProofFileIDs {
		if pid <= 0 {
			return nil, ErrProofFileInvalid
		}
		if !seenProof[pid] {
			seenProof[pid] = true
			dedupedProofIDs = append(dedupedProofIDs, pid)
		}
	}
	if len(dedupedProofIDs) > 10 {
		return nil, ErrProofTooMany
	}
	if len(dedupedProofIDs) > 0 {
		var proofFiles []model.SystemFile
		if err := s.db.WithContext(ctx).Model(&model.SystemFile{}).
			Where("id IN ? AND deleted_at IS NULL", dedupedProofIDs).Find(&proofFiles).Error; err != nil {
			return nil, err
		}
		if len(proofFiles) != len(dedupedProofIDs) {
			return nil, ErrProofFileInvalid
		}
		imageCount := 0
		videoCount := 0
		for _, pf := range proofFiles {
			if pf.Scene != model.SceneAfterSaleProof {
				return nil, ErrProofFileInvalid
			}
			if pf.OwnerType != model.OwnerTypeMember || pf.OwnerID != memberID {
				return nil, ErrProofFileInvalid
			}
			if pf.FileType != model.FileTypeImage && pf.FileType != model.FileTypeVideo {
				return nil, ErrProofFileInvalid
			}
			if pf.FileType == model.FileTypeImage {
				imageCount++
			} else if pf.FileType == model.FileTypeVideo {
				videoCount++
			}
		}
		if imageCount > 9 {
			return nil, errors.New("售后凭证图片最多9张")
		}
		if videoCount > 1 {
			return nil, errors.New("售后凭证视频最多1个")
		}
	}

	var proofJSON *string
	if len(dedupedProofIDs) > 0 {
		b, _ := json.Marshal(dedupedProofIDs)
		str := string(b)
		proofJSON = &str
	}

	var createdAfterSale *model.AfterSale
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Lock order
		var order model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", req.OrderID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}

		if order.MemberID != memberID {
			return ErrOrderNotBelong
		}
		if order.PaidAt == nil {
			return errors.New("未支付订单不能申请售后")
		}

		// Check order status eligibility
		switch order.Status {
		case model.OrderStatusPendingPayment:
			return errors.New("待支付订单不能申请售后")
		case model.OrderStatusCancelled:
			return errors.New("已取消订单不能申请售后")
		case model.OrderStatusRefunded:
			return ErrAlreadyRefunded
		case model.OrderStatusPendingShipment:
			if req.Type != model.AfterSaleTypeRefundOnly {
				return errors.New("待发货订单仅支持申请仅退款")
			}
		case model.OrderStatusPendingReceipt:
			if req.Type != model.AfterSaleTypeRefundOnly && req.Type != model.AfterSaleTypeReturnRefund {
				return errors.New("待收货订单不支持该售后类型")
			}
		case model.OrderStatusCompleted:
			if req.Type != model.AfterSaleTypeReturnRefund {
				return errors.New("已完成订单仅支持申请退货退款")
			}
			if order.CompletedAt == nil {
				return errors.New("订单完成时间异常")
			}
			deadline := order.CompletedAt.Add(time.Duration(model.DefaultAfterSaleDays) * 24 * time.Hour)
			if time.Now().After(deadline) {
				return fmt.Errorf("订单已超过售后有效期(%d天)，无法申请售后", model.DefaultAfterSaleDays)
			}
		default:
			return errors.New("当前订单状态不可申请售后")
		}

		// 2. Check idempotency: request_id for member
		var existing model.AfterSale
		if err := tx.Where("member_id = ? AND request_id = ?", memberID, requestID).First(&existing).Error; err == nil {
			if existing.OrderID == order.ID {
				createdAfterSale = &existing
				return nil
			}
			return errors.New("重复的请求幂等号")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 3. Check active after-sale on this order
		var activeCount int64
		if err := tx.Model(&model.AfterSale{}).
			Where("order_id = ? AND status IN ?", order.ID, []int8{
				model.AfterSaleStatusPendingReview,
				model.AfterSaleStatusWaitingBuyerReturn,
				model.AfterSaleStatusWaitingMerchantReceive,
				model.AfterSaleStatusRefunding,
				model.AfterSaleStatusRefundFailed,
			}).Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount > 0 {
			return ErrActiveAfterSale
		}

		// 4. Check if order was already successfully refunded
		var refundedCount int64
		if err := tx.Model(&model.AfterSale{}).
			Where("order_id = ? AND status = ?", order.ID, model.AfterSaleStatusCompleted).
			Count(&refundedCount).Error; err != nil {
			return err
		}
		if refundedCount > 0 {
			return ErrAlreadyRefunded
		}

		// 5. Query order items
		var orderItems []model.OrderItem
		if err := tx.Where("order_id = ?", order.ID).Find(&orderItems).Error; err != nil {
			return err
		}
		if len(orderItems) == 0 {
			return errors.New("订单商品不存在")
		}

		// 6. Generate after-sale business number
		afterSaleNo, err := s.bizNoGen.Next(bizno.BizPrefixAfterSale)
		if err != nil {
			return fmt.Errorf("生成售后单号失败: %w", err)
		}

		now := time.Now()
		as := model.AfterSale{
			AfterSaleNo:        afterSaleNo,
			RequestID:          requestID,
			OrderID:            order.ID,
			MemberID:           memberID,
			Type:               req.Type,
			Status:             model.AfterSaleStatusPendingReview,
			OrderStatusAtApply: order.Status,
			Reason:             reason,
			Description:        strings.TrimSpace(req.Description),
			ProofFileIDs:       proofJSON,
			RefundAmount:       order.PayAmount,
			AppliedAt:          now,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := tx.Create(&as).Error; err != nil {
			return fmt.Errorf("创建售后单失败: %w", err)
		}

		// 7. Save after_sale_item snapshots
		asItems := make([]model.AfterSaleItem, 0, len(orderItems))
		for _, it := range orderItems {
			skuText := ""
			if it.SKUSpecValues != nil {
				skuText = strings.TrimSpace(*it.SKUSpecValues)
			}
			asItems = append(asItems, model.AfterSaleItem{
				AfterSaleID:     as.ID,
				OrderItemID:     it.ID,
				ProductID:       it.ProductID,
				SKUID:           it.SKUID,
				Quantity:        it.Quantity,
				RefundAmount:    it.TotalAmount,
				ProductName:     it.ProductName,
				SKUText:         skuText,
				SKUCode:         it.SKUCode,
				SKUImageURL:     it.SKUImageURL,
				RestockQuantity: 0,
				CreatedAt:       now,
			})
		}
		if err := tx.Create(&asItems).Error; err != nil {
			return fmt.Errorf("创建售后明细快照失败: %w", err)
		}

		// 8. Record audit log
		log := model.AfterSaleLog{
			AfterSaleID:  as.ID,
			OperatorType: model.AfterSaleOperatorTypeMember,
			OperatorID:   memberID,
			OperatorName: "member",
			FromStatus:   0,
			ToStatus:     model.AfterSaleStatusPendingReview,
			Action:       "apply",
			Remark:       fmt.Sprintf("买家申请售后: %s", model.AfterSaleTypeText(as.Type)),
			CreatedAt:    now,
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		createdAfterSale = &as
		return nil
	})

	if err != nil {
		return nil, err
	}
	return createdAfterSale, nil
}

// Cancel allows a member to cancel their after-sale application when status is PendingReview (10).
func (s *Service) Cancel(ctx context.Context, memberID int64, afterSaleID int64) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}
	if memberID <= 0 {
		return ErrAfterSaleNotBelong
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var as model.AfterSale
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", afterSaleID).First(&as).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if as.MemberID != memberID {
			return ErrAfterSaleNotBelong
		}
		if as.Status != model.AfterSaleStatusPendingReview {
			return errors.New("当前售后状态不可取消")
		}

		now := time.Now()
		res := tx.Model(&model.AfterSale{}).
			Where("id = ? AND status = ?", as.ID, model.AfterSaleStatusPendingReview).
			Updates(map[string]any{
				"status":       model.AfterSaleStatusCancelled,
				"cancelled_at": &now,
				"updated_at":   now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("售后状态已变更，请刷新")
		}

		log := model.AfterSaleLog{
			AfterSaleID:  as.ID,
			OperatorType: model.AfterSaleOperatorTypeMember,
			OperatorID:   memberID,
			OperatorName: "member",
			FromStatus:   model.AfterSaleStatusPendingReview,
			ToStatus:     model.AfterSaleStatusCancelled,
			Action:       "cancel",
			Remark:       "买家取消售后申请",
			CreatedAt:    now,
		}
		return tx.Create(&log).Error
	})
}

// SubmitReturnLogistics allows a member to fill in return tracking info when status is WaitingBuyerReturn (20).
func (s *Service) SubmitReturnLogistics(ctx context.Context, memberID int64, afterSaleID int64, company, trackingNo string) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}
	if memberID <= 0 {
		return ErrAfterSaleNotBelong
	}
	company = strings.TrimSpace(company)
	trackingNo = strings.TrimSpace(trackingNo)
	if company == "" || trackingNo == "" {
		return ErrTrackingEmpty
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var as model.AfterSale
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", afterSaleID).First(&as).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if as.MemberID != memberID {
			return ErrAfterSaleNotBelong
		}
		if as.Type != model.AfterSaleTypeReturnRefund {
			return errors.New("仅退款类型无需填写退货物流")
		}
		if as.Status != model.AfterSaleStatusWaitingBuyerReturn {
			return errors.New("当前售后状态不可填写退货物流")
		}

		now := time.Now()
		res := tx.Model(&model.AfterSale{}).
			Where("id = ? AND status = ?", as.ID, model.AfterSaleStatusWaitingBuyerReturn).
			Updates(map[string]any{
				"buyer_return_company": company,
				"buyer_return_no":      trackingNo,
				"buyer_returned_at":    &now,
				"status":               model.AfterSaleStatusWaitingMerchantReceive,
				"updated_at":           now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("售后状态已变更，请刷新")
		}

		log := model.AfterSaleLog{
			AfterSaleID:  as.ID,
			OperatorType: model.AfterSaleOperatorTypeMember,
			OperatorID:   memberID,
			OperatorName: "member",
			FromStatus:   model.AfterSaleStatusWaitingBuyerReturn,
			ToStatus:     model.AfterSaleStatusWaitingMerchantReceive,
			Action:       "submit_return_logistics",
			Remark:       fmt.Sprintf("买家提交退货物流: %s %s", company, trackingNo),
			CreatedAt:    now,
		}
		return tx.Create(&log).Error
	})
}

// AdminReject rejects an after-sale application when status is PendingReview (10).
func (s *Service) AdminReject(ctx context.Context, user model.SystemUser, req AdminRejectRequest) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}
	rejectReason := strings.TrimSpace(req.RejectReason)
	if rejectReason == "" {
		return errors.New("拒绝原因不能为空")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var as model.AfterSale
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", req.ID).First(&as).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if as.Status != model.AfterSaleStatusPendingReview {
			return errors.New("当前售后状态不可审核拒绝")
		}

		now := time.Now()
		res := tx.Model(&model.AfterSale{}).
			Where("id = ? AND status = ?", as.ID, model.AfterSaleStatusPendingReview).
			Updates(map[string]any{
				"status":        model.AfterSaleStatusRejected,
				"reject_reason": rejectReason,
				"audited_at":    &now,
				"updated_at":    now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("售后状态已变更，请刷新")
		}

		opName := strings.TrimSpace(user.Nickname)
		if opName == "" {
			opName = user.Username
		}

		log := model.AfterSaleLog{
			AfterSaleID:  as.ID,
			OperatorType: model.AfterSaleOperatorTypeAdmin,
			OperatorID:   user.ID,
			OperatorName: opName,
			FromStatus:   model.AfterSaleStatusPendingReview,
			ToStatus:     model.AfterSaleStatusRejected,
			Action:       "reject",
			Remark:       fmt.Sprintf("管理员拒绝: %s", rejectReason),
			CreatedAt:    now,
		}
		return tx.Create(&log).Error
	})
}

// AdminApprove approves an after-sale application.
// For ReturnRefund: transitions 10 -> 20 and saves return address snapshot.
// For RefundOnly: transitions 10 -> 40, creates order_refund in DB transaction,
// then calls RefundProvider OUTSIDE the transaction, followed by FinalizeRefundSuccess/Failed.
func (s *Service) AdminApprove(ctx context.Context, user model.SystemUser, req AdminApproveRequest) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}

	var as model.AfterSale
	var isRefundOnly bool
	var refundReq refund.RefundRequest

	// Phase 1: DB transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", req.ID).First(&as).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if as.Status != model.AfterSaleStatusPendingReview {
			return errors.New("当前售后状态不可审核通过")
		}

		opName := strings.TrimSpace(user.Nickname)
		if opName == "" {
			opName = user.Username
		}
		now := time.Now()

		if as.Type == model.AfterSaleTypeReturnRefund {
			receiver := strings.TrimSpace(req.ReturnReceiver)
			phone := strings.TrimSpace(req.ReturnPhone)
			address := strings.TrimSpace(req.ReturnAddress)
			if receiver == "" || phone == "" || address == "" {
				return ErrReturnAddressEmpty
			}

			res := tx.Model(&model.AfterSale{}).
				Where("id = ? AND status = ?", as.ID, model.AfterSaleStatusPendingReview).
				Updates(map[string]any{
					"return_receiver": receiver,
					"return_phone":    phone,
					"return_address":  address,
					"status":          model.AfterSaleStatusWaitingBuyerReturn,
					"audited_at":      &now,
					"updated_at":      now,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected != 1 {
				return errors.New("售后状态已变更，请刷新")
			}

			log := model.AfterSaleLog{
				AfterSaleID:  as.ID,
				OperatorType: model.AfterSaleOperatorTypeAdmin,
				OperatorID:   user.ID,
				OperatorName: opName,
				FromStatus:   model.AfterSaleStatusPendingReview,
				ToStatus:     model.AfterSaleStatusWaitingBuyerReturn,
				Action:       "approve_return",
				Remark:       "管理员审核通过，等待买家退货",
				CreatedAt:    now,
			}
			return tx.Create(&log).Error
		}

		// RefundOnly: 10 -> 40
		isRefundOnly = true
		refundNo, err := s.bizNoGen.Next(bizno.BizPrefixRefund)
		if err != nil {
			return fmt.Errorf("生成退款单号失败: %w", err)
		}

		var payment model.OrderPayment
		if err := tx.Where("order_id = ? AND status = ?", as.OrderID, model.PaymentStatusSuccess).
			Order("id ASC").First(&payment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoEligiblePayment
			}
			return err
		}
		if payment.ID <= 0 || payment.Status != model.PaymentStatusSuccess {
			return ErrNoEligiblePayment
		}

		orderRefund := model.OrderRefund{
			RefundNo:     refundNo,
			AfterSaleID:  as.ID,
			OrderID:      as.OrderID,
			PaymentID:    payment.ID,
			MemberID:     as.MemberID,
			RefundAmount: as.RefundAmount,
			Status:       model.RefundStatusProcessing,
			Provider:     "mock",
			RequestedAt:  &now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&orderRefund).Error; err != nil {
			return fmt.Errorf("创建退款单失败: %w", err)
		}

		res := tx.Model(&model.AfterSale{}).
			Where("id = ? AND status = ?", as.ID, model.AfterSaleStatusPendingReview).
			Updates(map[string]any{
				"status":            model.AfterSaleStatusRefunding,
				"audited_at":        &now,
				"refund_started_at": &now,
				"updated_at":        now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("售后状态已变更，请刷新")
		}

		log := model.AfterSaleLog{
			AfterSaleID:  as.ID,
			OperatorType: model.AfterSaleOperatorTypeAdmin,
			OperatorID:   user.ID,
			OperatorName: opName,
			FromStatus:   model.AfterSaleStatusPendingReview,
			ToStatus:     model.AfterSaleStatusRefunding,
			Action:       "approve_refund_only",
			Remark:       "管理员审核通过，进入退款流程",
			CreatedAt:    now,
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		txnNo := ""
		if payment.TransactionNo != nil {
			txnNo = *payment.TransactionNo
		}
		refundReq = refund.RefundRequest{
			RefundNo:      refundNo,
			PaymentNo:     payment.PaymentNo,
			TransactionNo: txnNo,
			RefundAmount:  as.RefundAmount,
			TotalAmount:   as.RefundAmount,
			Reason:        as.Reason,
		}
		return nil
	})

	if err != nil {
		return err
	}

	if !isRefundOnly {
		return nil
	}

	// Phase 2: Call RefundProvider OUTSIDE DB Transaction
	result, callErr := s.refundProvider.Refund(ctx, refundReq)

	// Phase 3: Short DB transaction to finalize success or failure
	finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer finalizeCancel()

	if callErr != nil || (result != nil && !result.Success) {
		errMsg := "支付退款调用失败"
		if callErr != nil {
			errMsg = callErr.Error()
		} else if result != nil && result.ErrorMessage != "" {
			errMsg = result.ErrorMessage
		}
		return s.FinalizeRefundFailed(finalizeCtx, as.ID, errMsg)
	}

	providerTxnNo := ""
	if result != nil {
		providerTxnNo = result.ProviderRefundNo
	}
	if finErr := s.FinalizeRefundSuccess(finalizeCtx, as.ID, providerTxnNo); finErr != nil {
		return fmt.Errorf("退款渠道已成功但本地收尾失败: %w", finErr)
	}
	return nil
}

// AdminConfirmReturnReceived confirms receipt of returned items by the merchant.
// Validates restock quantities, restores inventory atomically within the transaction,
// transitions 30 -> 40, creates order_refund, and calls the refund provider OUTSIDE the transaction.
func (s *Service) AdminConfirmReturnReceived(ctx context.Context, user model.SystemUser, req AdminConfirmReceiveRequest) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}
	if len(req.Items) == 0 {
		return errors.New("必须提供入库商品明细")
	}

	restockMap := make(map[int64]int, len(req.Items))
	for _, it := range req.Items {
		if it.RestockQuantity < 0 {
			return ErrRestockQtyInvalid
		}
		restockMap[it.AfterSaleItemID] = it.RestockQuantity
	}

	var as model.AfterSale
	var refundReq refund.RefundRequest

	// Phase 1: DB transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", req.ID).First(&as).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if as.Type != model.AfterSaleTypeReturnRefund {
			return errors.New("非退货退款售后单不可确认收货")
		}
		if as.Status != model.AfterSaleStatusWaitingMerchantReceive {
			return errors.New("当前售后状态不可确认收货")
		}

		// Lock after_sale_items
		var items []model.AfterSaleItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("after_sale_id = ?", as.ID).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("售后商品明细不存在")
		}

		// Validate restock quantity bounds
		for _, item := range items {
			qty, exists := restockMap[item.ID]
			if !exists {
				return fmt.Errorf("缺少商品明细 (ID: %d) 的入库数量", item.ID)
			}
			if qty < 0 || qty > item.Quantity {
				return fmt.Errorf("商品 %s 入库数量非法 (最大: %d, 提交: %d)", item.ProductName, item.Quantity, qty)
			}
		}

		// Sort items by sku_id ASC to prevent deadlocks
		sort.Slice(items, func(i, j int) bool {
			return items[i].SKUID < items[j].SKUID
		})

		opName := strings.TrimSpace(user.Nickname)
		if opName == "" {
			opName = user.Username
		}
		now := time.Now()

		// Restore stock for restocked items
		for _, item := range items {
			restockQty := restockMap[item.ID]
			if restockQty > 0 {
				invReqID := fmt.Sprintf("as_restock_%d_%d", as.ID, item.SKUID)
				remark := fmt.Sprintf("售后单 %s 商家确认退货重新入库", as.AfterSaleNo)
				if err := s.invSvc.AfterSaleRestoreStockTx(ctx, tx, item.ProductID, item.SKUID, restockQty, as.ID, as.AfterSaleNo, invReqID, user.ID, opName, remark); err != nil {
					return fmt.Errorf("商品 %s 恢复库存失败: %w", item.ProductName, err)
				}
			}
			// Update restock_quantity snapshot on item
			if err := tx.Model(&model.AfterSaleItem{}).Where("id = ?", item.ID).
				Update("restock_quantity", restockQty).Error; err != nil {
				return err
			}
		}

		// Generate refund number and create order_refund
		refundNo, err := s.bizNoGen.Next(bizno.BizPrefixRefund)
		if err != nil {
			return fmt.Errorf("生成退款单号失败: %w", err)
		}

		var payment model.OrderPayment
		if err := tx.Where("order_id = ? AND status = ?", as.OrderID, model.PaymentStatusSuccess).
			Order("id ASC").First(&payment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoEligiblePayment
			}
			return err
		}
		if payment.ID <= 0 || payment.Status != model.PaymentStatusSuccess {
			return ErrNoEligiblePayment
		}

		orderRefund := model.OrderRefund{
			RefundNo:     refundNo,
			AfterSaleID:  as.ID,
			OrderID:      as.OrderID,
			PaymentID:    payment.ID,
			MemberID:     as.MemberID,
			RefundAmount: as.RefundAmount,
			Status:       model.RefundStatusProcessing,
			Provider:     "mock",
			RequestedAt:  &now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&orderRefund).Error; err != nil {
			return fmt.Errorf("创建退款单失败: %w", err)
		}

		// Transition after_sale 30 -> 40
		res := tx.Model(&model.AfterSale{}).
			Where("id = ? AND status = ?", as.ID, model.AfterSaleStatusWaitingMerchantReceive).
			Updates(map[string]any{
				"merchant_received_at": &now,
				"refund_started_at":    &now,
				"status":               model.AfterSaleStatusRefunding,
				"updated_at":           now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("售后状态已变更，请刷新")
		}

		log := model.AfterSaleLog{
			AfterSaleID:  as.ID,
			OperatorType: model.AfterSaleOperatorTypeAdmin,
			OperatorID:   user.ID,
			OperatorName: opName,
			FromStatus:   model.AfterSaleStatusWaitingMerchantReceive,
			ToStatus:     model.AfterSaleStatusRefunding,
			Action:       "confirm_receive",
			Remark:       "商家确认收到退货并完成入库，进入退款流程",
			CreatedAt:    now,
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		txnNo := ""
		if payment.TransactionNo != nil {
			txnNo = *payment.TransactionNo
		}
		refundReq = refund.RefundRequest{
			RefundNo:      refundNo,
			PaymentNo:     payment.PaymentNo,
			TransactionNo: txnNo,
			RefundAmount:  as.RefundAmount,
			TotalAmount:   as.RefundAmount,
			Reason:        as.Reason,
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Phase 2: Call RefundProvider OUTSIDE DB Transaction
	result, callErr := s.refundProvider.Refund(ctx, refundReq)

	// Phase 3: Short DB transaction to finalize success or failure
	finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer finalizeCancel()

	if callErr != nil || (result != nil && !result.Success) {
		errMsg := "支付退款调用失败"
		if callErr != nil {
			errMsg = callErr.Error()
		} else if result != nil && result.ErrorMessage != "" {
			errMsg = result.ErrorMessage
		}
		return s.FinalizeRefundFailed(finalizeCtx, as.ID, errMsg)
	}

	providerTxnNo := ""
	if result != nil {
		providerTxnNo = result.ProviderRefundNo
	}
	if finErr := s.FinalizeRefundSuccess(finalizeCtx, as.ID, providerTxnNo); finErr != nil {
		return fmt.Errorf("退款渠道已成功但本地收尾失败: %w", finErr)
	}
	return nil
}

// AdminRetryRefund retries a previously failed refund.
func (s *Service) AdminRetryRefund(ctx context.Context, user model.SystemUser, req AdminRetryRefundRequest) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}

	var as model.AfterSale
	var refundReq refund.RefundRequest

	// Phase 1: DB transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", req.ID).First(&as).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if as.Status != model.AfterSaleStatusRefundFailed {
			return errors.New("当前售后状态不可重试退款")
		}

		var orderRefund model.OrderRefund
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("after_sale_id = ?", as.ID).First(&orderRefund).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("退款单不存在")
			}
			return err
		}

		if orderRefund.Status != model.RefundStatusFailed {
			return errors.New("退款单当前状态不可重试")
		}

		opName := strings.TrimSpace(user.Nickname)
		if opName == "" {
			opName = user.Username
		}
		now := time.Now()

		// Update after_sale 80 -> 40
		resAS := tx.Model(&model.AfterSale{}).
			Where("id = ? AND status = ?", as.ID, model.AfterSaleStatusRefundFailed).
			Updates(map[string]any{
				"status":     model.AfterSaleStatusRefunding,
				"updated_at": now,
			})
		if resAS.Error != nil {
			return resAS.Error
		}
		if resAS.RowsAffected != 1 {
			return errors.New("售后状态已变更，请刷新")
		}

		// Update order_refund 40 -> 20
		resRef := tx.Model(&model.OrderRefund{}).
			Where("id = ? AND status = ?", orderRefund.ID, model.RefundStatusFailed).
			Updates(map[string]any{
				"status":       model.RefundStatusProcessing,
				"requested_at": &now,
				"updated_at":   now,
			})
		if resRef.Error != nil {
			return resRef.Error
		}
		if resRef.RowsAffected != 1 {
			return errors.New("退款单状态已变更，请刷新")
		}

		log := model.AfterSaleLog{
			AfterSaleID:  as.ID,
			OperatorType: model.AfterSaleOperatorTypeAdmin,
			OperatorID:   user.ID,
			OperatorName: opName,
			FromStatus:   model.AfterSaleStatusRefundFailed,
			ToStatus:     model.AfterSaleStatusRefunding,
			Action:       "retry_refund",
			Remark:       "管理员重试退款",
			CreatedAt:    now,
		}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}

		var payment model.OrderPayment
		if err := tx.Where("id = ? AND status = ?", orderRefund.PaymentID, model.PaymentStatusSuccess).First(&payment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNoEligiblePayment
			}
			return err
		}
		if payment.ID <= 0 || payment.Status != model.PaymentStatusSuccess {
			return ErrNoEligiblePayment
		}
		txnNo := ""
		if payment.TransactionNo != nil {
			txnNo = *payment.TransactionNo
		}

		// Retry uses the SAME refund_no
		refundReq = refund.RefundRequest{
			RefundNo:      orderRefund.RefundNo,
			PaymentNo:     payment.PaymentNo,
			TransactionNo: txnNo,
			RefundAmount:  orderRefund.RefundAmount,
			TotalAmount:   orderRefund.RefundAmount,
			Reason:        as.Reason,
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Phase 2: Call RefundProvider OUTSIDE DB Transaction
	result, callErr := s.refundProvider.Refund(ctx, refundReq)

	// Phase 3: Short DB transaction to finalize success or failure
	finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer finalizeCancel()

	if callErr != nil || (result != nil && !result.Success) {
		errMsg := "支付退款调用失败"
		if callErr != nil {
			errMsg = callErr.Error()
		} else if result != nil && result.ErrorMessage != "" {
			errMsg = result.ErrorMessage
		}
		return s.FinalizeRefundFailed(finalizeCtx, as.ID, errMsg)
	}

	providerTxnNo := ""
	if result != nil {
		providerTxnNo = result.ProviderRefundNo
	}
	if finErr := s.FinalizeRefundSuccess(finalizeCtx, as.ID, providerTxnNo); finErr != nil {
		return fmt.Errorf("退款渠道已成功但本地收尾失败: %w", finErr)
	}
	return nil
}

// FinalizeRefundSuccess finalizes a successful refund.
// It is strictly idempotent: repeated calls return success without duplicating stock restoration,
// log records, or order updates.
func (s *Service) FinalizeRefundSuccess(ctx context.Context, afterSaleID int64, providerRefundNo string) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var orderRefund model.OrderRefund
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("after_sale_id = ?", afterSaleID).First(&orderRefund).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("退款单不存在")
			}
			return err
		}

		var as model.AfterSale
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", afterSaleID).First(&as).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		var order model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", as.OrderID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}

		// Idempotency: if already completed, return nil immediately
		if orderRefund.Status == model.RefundStatusSuccess && as.Status == model.AfterSaleStatusCompleted {
			return nil
		}

		now := time.Now()

		// Restock rule:
		// [Pending Shipment + Refund Only]: goods were not sent; restore entire order item quantities.
		// [Pending Receipt + Refund Only]: goods were shipped and not returned; stock += 0.
		// [Return Refund]: restocked already when merchant confirmed receipt; stock += 0 here.
		if as.Type == model.AfterSaleTypeRefundOnly && as.OrderStatusAtApply == model.OrderStatusPendingShipment {
			var items []model.AfterSaleItem
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("after_sale_id = ?", as.ID).Find(&items).Error; err != nil {
				return err
			}

			// Sort by sku_id ASC to avoid deadlocks
			sort.Slice(items, func(i, j int) bool {
				return items[i].SKUID < items[j].SKUID
			})

			for _, item := range items {
				invReqID := fmt.Sprintf("as_refund_restore_%d_%d", as.ID, item.SKUID)
				remark := fmt.Sprintf("待发货仅退款退款成功恢复库存 (售后单: %s)", as.AfterSaleNo)
				if err := s.invSvc.AfterSaleRestoreStockTx(ctx, tx, item.ProductID, item.SKUID, item.Quantity, as.ID, as.AfterSaleNo, invReqID, 0, "system", remark); err != nil {
					return fmt.Errorf("商品 %s 恢复库存失败: %w", item.ProductName, err)
				}
			}
		}

		// Update order_refund 20/40 -> 30
		refUpdates := map[string]any{
			"status":      model.RefundStatusSuccess,
			"refunded_at": &now,
			"updated_at":  now,
		}
		if providerRefundNo != "" {
			refUpdates["provider_refund_no"] = providerRefundNo
		}
		if err := tx.Model(&model.OrderRefund{}).Where("id = ?", orderRefund.ID).Updates(refUpdates).Error; err != nil {
			return err
		}

		// Update after_sale 40 -> 50
		if err := tx.Model(&model.AfterSale{}).Where("id = ?", as.ID).
			Updates(map[string]any{
				"status":       model.AfterSaleStatusCompleted,
				"completed_at": &now,
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}

		// Update order_main -> 60 (Refunded)
		if err := tx.Model(&model.OrderMain{}).Where("id = ?", order.ID).
			Updates(map[string]any{
				"status":     model.OrderStatusRefunded,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		// Record after_sale_log
		asLog := model.AfterSaleLog{
			AfterSaleID:  as.ID,
			OperatorType: model.AfterSaleOperatorTypeSystem,
			OperatorID:   0,
			OperatorName: "system",
			FromStatus:   as.Status,
			ToStatus:     model.AfterSaleStatusCompleted,
			Action:       "refund_success",
			Remark:       fmt.Sprintf("退款成功，外部单号: %s", providerRefundNo),
			CreatedAt:    now,
		}
		if err := tx.Create(&asLog).Error; err != nil {
			return err
		}

		// Record order_log
		orderLog := model.OrderLog{
			OrderID:      order.ID,
			FromStatus:   order.Status,
			ToStatus:     model.OrderStatusRefunded,
			Action:       "refunded",
			OperatorType: "system",
			OperatorID:   0,
			OperatorName: "system",
			Remark:       fmt.Sprintf("售后退款成功，订单进入已退款状态 (售后单号: %s)", as.AfterSaleNo),
			CreatedAt:    now,
		}
		return tx.Create(&orderLog).Error
	})
}

// FinalizeRefundFailed records a failed refund attempt.
func (s *Service) FinalizeRefundFailed(ctx context.Context, afterSaleID int64, lastError string) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var orderRefund model.OrderRefund
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("after_sale_id = ?", afterSaleID).First(&orderRefund).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("退款单不存在")
			}
			return err
		}

		var as model.AfterSale
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", afterSaleID).First(&as).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		// If already success, do not transition to failed
		if orderRefund.Status == model.RefundStatusSuccess {
			return nil
		}

		now := time.Now()
		if err := tx.Model(&model.OrderRefund{}).Where("id = ?", orderRefund.ID).
			Updates(map[string]any{
				"status":      model.RefundStatusFailed,
				"last_error":  lastError,
				"retry_count": orderRefund.RetryCount + 1,
				"updated_at":  now,
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.AfterSale{}).Where("id = ?", as.ID).
			Updates(map[string]any{
				"status":     model.AfterSaleStatusRefundFailed,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		log := model.AfterSaleLog{
			AfterSaleID:  as.ID,
			OperatorType: model.AfterSaleOperatorTypeSystem,
			OperatorID:   0,
			OperatorName: "system",
			FromStatus:   as.Status,
			ToStatus:     model.AfterSaleStatusRefundFailed,
			Action:       "refund_failed",
			Remark:       fmt.Sprintf("退款失败: %s", lastError),
			CreatedAt:    now,
		}
		return tx.Create(&log).Error
	})
}

// ReconcileRefund queries the payment provider for the actual refund status and reconciles local state idempotently.
// It NEVER invokes the refund operation itself, preventing duplicate refunds.
func (s *Service) ReconcileRefund(ctx context.Context, afterSaleID int64) error {
	if s.db == nil {
		return errors.New("数据库服务不可用")
	}

	var as model.AfterSale
	if err := s.db.WithContext(ctx).Where("id = ?", afterSaleID).First(&as).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	var orderRefund model.OrderRefund
	if err := s.db.WithContext(ctx).Where("after_sale_id = ?", afterSaleID).First(&orderRefund).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("退款单不存在")
		}
		return err
	}

	// If already finalized successfully, return immediately (strictly idempotent)
	if orderRefund.Status == model.RefundStatusSuccess && as.Status == model.AfterSaleStatusCompleted {
		return nil
	}

	// Query provider for current refund status
	provRefundNo := ""
	if orderRefund.ProviderRefundNo != nil {
		provRefundNo = *orderRefund.ProviderRefundNo
	}
	queryRes, err := s.refundProvider.QueryRefund(ctx, orderRefund.RefundNo, provRefundNo)
	if err != nil {
		return fmt.Errorf("查询退款状态失败: %w", err)
	}
	if queryRes == nil {
		return errors.New("退款状态查询结果为空")
	}

	finalizeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch queryRes.Status {
	case refund.RefundQueryStatusSuccess:
		txnNo := queryRes.ProviderRefundNo
		if txnNo == "" {
			txnNo = provRefundNo
		}
		return s.FinalizeRefundSuccess(finalizeCtx, afterSaleID, txnNo)
	case refund.RefundQueryStatusFailed:
		errMsg := queryRes.ErrorMessage
		if errMsg == "" {
			errMsg = "渠道查询退款失败"
		}
		return s.FinalizeRefundFailed(finalizeCtx, afterSaleID, errMsg)
	case refund.RefundQueryStatusProcessing:
		// Still processing, keep current state
		return nil
	case refund.RefundQueryStatusUnknown:
		// Unknown status from provider: do not finalize, do not restore stock, return clear error
		return ErrRefundStatusUnknown
	default:
		return fmt.Errorf("未知的退款查询状态: %s", queryRes.Status)
	}
}

// AdminList queries after-sales for admin management with filters and pagination.
func (s *Service) AdminList(ctx context.Context, req AdminAfterSaleListRequest) ([]AdminAfterSaleListRow, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("数据库服务不可用")
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	query := s.db.WithContext(ctx).Model(&model.AfterSale{})

	if req.Type != nil && *req.Type > 0 {
		query = query.Where("type = ?", *req.Type)
	}
	if req.Status != nil && *req.Status > 0 {
		query = query.Where("status = ?", *req.Status)
	}
	if asNo := strings.TrimSpace(req.AfterSaleNo); asNo != "" {
		query = query.Where("after_sale_no LIKE ?", "%"+asNo+"%")
	}
	if orderNo := strings.TrimSpace(req.OrderNo); orderNo != "" {
		query = query.Where("order_id IN (SELECT id FROM order_main WHERE order_no LIKE ?)", "%"+orderNo+"%")
	}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		query = query.Where("(after_sale_no LIKE ? OR member_id IN (SELECT id FROM member WHERE nickname LIKE ? OR mobile LIKE ?))", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			query = query.Where("created_at >= ?", t)
		} else if t, err := time.Parse("2006-01-02 15:04:05", req.StartTime); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			query = query.Where("created_at <= ?", t)
		} else if t, err := time.Parse("2006-01-02 15:04:05", req.EndTime); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []model.AfterSale
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	if len(records) == 0 {
		return []AdminAfterSaleListRow{}, total, nil
	}

	// Fetch orders & members in batch
	orderIDs := make([]int64, 0, len(records))
	memberIDs := make([]int64, 0, len(records))
	for _, r := range records {
		orderIDs = append(orderIDs, r.OrderID)
		memberIDs = append(memberIDs, r.MemberID)
	}

	orderMap := make(map[int64]model.OrderMain)
	var orders []model.OrderMain
	if err := s.db.WithContext(ctx).Where("id IN ?", orderIDs).Find(&orders).Error; err == nil {
		for _, o := range orders {
			orderMap[o.ID] = o
		}
	}

	memberMap := make(map[int64]model.Member)
	var members []model.Member
	if err := s.db.WithContext(ctx).Where("id IN ?", memberIDs).Find(&members).Error; err == nil {
		for _, m := range members {
			memberMap[m.ID] = m
		}
	}

	rows := make([]AdminAfterSaleListRow, 0, len(records))
	for _, r := range records {
		ord := orderMap[r.OrderID]
		mem := memberMap[r.MemberID]
		mobile := ""
		if mem.Mobile != nil {
			mobile = *mem.Mobile
		}
		rows = append(rows, AdminAfterSaleListRow{
			ID:             r.ID,
			AfterSaleNo:    r.AfterSaleNo,
			OrderID:        r.OrderID,
			OrderNo:        ord.OrderNo,
			MemberID:       r.MemberID,
			MemberNickname: mem.Nickname,
			MemberMobile:   mobile,
			Type:           r.Type,
			TypeText:       model.AfterSaleTypeText(r.Type),
			Status:         r.Status,
			StatusText:     model.AfterSaleStatusText(r.Status),
			RefundAmount:   normalizeAmount(r.RefundAmount),
			Reason:         r.Reason,
			AppliedAt:      r.AppliedAt,
			CreatedAt:      r.CreatedAt,
		})
	}

	return rows, total, nil
}

// AdminDetail returns full after-sale details for admin.
func (s *Service) AdminDetail(ctx context.Context, afterSaleID int64) (*AdminAfterSaleDetailResponse, error) {
	if s.db == nil {
		return nil, errors.New("数据库服务不可用")
	}

	var as model.AfterSale
	if err := s.db.WithContext(ctx).Where("id = ?", afterSaleID).First(&as).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var order model.OrderMain
	_ = s.db.WithContext(ctx).Where("id = ?", as.OrderID).First(&order).Error

	var mem model.Member
	_ = s.db.WithContext(ctx).Where("id = ?", as.MemberID).First(&mem).Error
	mobile := ""
	if mem.Mobile != nil {
		mobile = *mem.Mobile
	}

	var items []model.AfterSaleItem
	if err := s.db.WithContext(ctx).Where("after_sale_id = ?", as.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	itemRows := make([]AfterSaleItemRow, 0, len(items))
	for _, it := range items {
		itemRows = append(itemRows, AfterSaleItemRow{
			ID:              it.ID,
			AfterSaleID:     it.AfterSaleID,
			OrderItemID:     it.OrderItemID,
			ProductID:       it.ProductID,
			SKUID:           it.SKUID,
			Quantity:        it.Quantity,
			RefundAmount:    normalizeAmount(it.RefundAmount),
			ProductName:     it.ProductName,
			SKUText:         it.SKUText,
			SKUCode:         it.SKUCode,
			SKUImageURL:     it.SKUImageURL,
			RestockQuantity: it.RestockQuantity,
		})
	}

	var logs []model.AfterSaleLog
	if err := s.db.WithContext(ctx).Where("after_sale_id = ?", as.ID).Order("id ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	logRows := make([]AfterSaleLogRow, 0, len(logs))
	for _, l := range logs {
		logRows = append(logRows, AfterSaleLogRow{
			ID:               l.ID,
			OperatorType:     l.OperatorType,
			OperatorTypeText: model.AfterSaleOperatorTypeText(l.OperatorType),
			OperatorID:       l.OperatorID,
			OperatorName:     l.OperatorName,
			FromStatus:       l.FromStatus,
			FromStatusText:   model.AfterSaleStatusText(l.FromStatus),
			ToStatus:         l.ToStatus,
			ToStatusText:     model.AfterSaleStatusText(l.ToStatus),
			Action:           l.Action,
			Remark:           l.Remark,
			CreatedAt:        l.CreatedAt,
		})
	}

	var orderRefund model.OrderRefund
	var refundRow *OrderRefundRow
	if err := s.db.WithContext(ctx).Where("after_sale_id = ?", as.ID).First(&orderRefund).Error; err == nil {
		refundRow = &OrderRefundRow{
			ID:               orderRefund.ID,
			RefundNo:         orderRefund.RefundNo,
			RefundAmount:     normalizeAmount(orderRefund.RefundAmount),
			Status:           orderRefund.Status,
			StatusText:       model.RefundStatusText(orderRefund.Status),
			Provider:         orderRefund.Provider,
			ProviderRefundNo: orderRefund.ProviderRefundNo,
			RetryCount:       orderRefund.RetryCount,
			LastError:        orderRefund.LastError,
			RequestedAt:      orderRefund.RequestedAt,
			RefundedAt:       orderRefund.RefundedAt,
		}
	}

	proofIDs, proofURLs, proofFiles := s.parseAndResolveProofs(ctx, as.ProofFileIDs)

	return &AdminAfterSaleDetailResponse{
		ID:                     as.ID,
		AfterSaleNo:            as.AfterSaleNo,
		RequestID:              as.RequestID,
		OrderID:                as.OrderID,
		OrderNo:                order.OrderNo,
		MemberID:               as.MemberID,
		MemberNickname:         mem.Nickname,
		MemberMobile:           mobile,
		Type:                   as.Type,
		TypeText:               model.AfterSaleTypeText(as.Type),
		Status:                 as.Status,
		StatusText:             model.AfterSaleStatusText(as.Status),
		OrderStatusAtApply:     as.OrderStatusAtApply,
		OrderStatusAtApplyText: model.OrderStatusText(as.OrderStatusAtApply),
		Reason:                 as.Reason,
		Description:            as.Description,
		ProofFileIDs:           proofIDs,
		ProofURLs:              proofURLs,
		ProofFiles:             proofFiles,
		RefundAmount:           normalizeAmount(as.RefundAmount),
		ReturnReceiver:         as.ReturnReceiver,
		ReturnPhone:            as.ReturnPhone,
		ReturnAddress:          as.ReturnAddress,
		BuyerReturnCompany:     as.BuyerReturnCompany,
		BuyerReturnNo:          as.BuyerReturnNo,
		RejectReason:           as.RejectReason,
		AppliedAt:              as.AppliedAt,
		AuditedAt:              as.AuditedAt,
		BuyerReturnedAt:        as.BuyerReturnedAt,
		MerchantReceivedAt:     as.MerchantReceivedAt,
		RefundStartedAt:        as.RefundStartedAt,
		CompletedAt:            as.CompletedAt,
		CancelledAt:            as.CancelledAt,
		CreatedAt:              as.CreatedAt,
		UpdatedAt:              as.UpdatedAt,
		Items:                  itemRows,
		Logs:                   logRows,
		Refund:                 refundRow,
	}, nil
}

// MiniappList queries after-sales for the logged-in member.
func (s *Service) MiniappList(ctx context.Context, memberID int64, req MiniappListRequest) ([]MiniappListItemResponse, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("数据库服务不可用")
	}
	if memberID <= 0 {
		return nil, 0, ErrAfterSaleNotBelong
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	query := s.db.WithContext(ctx).Model(&model.AfterSale{}).Where("member_id = ?", memberID)
	if req.Status != nil && *req.Status > 0 {
		query = query.Where("status = ?", *req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []model.AfterSale
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	if len(records) == 0 {
		return []MiniappListItemResponse{}, total, nil
	}

	asIDs := make([]int64, 0, len(records))
	for _, r := range records {
		asIDs = append(asIDs, r.ID)
	}

	var items []model.AfterSaleItem
	itemsByAS := make(map[int64][]MiniappAfterSaleItemResponse)
	if err := s.db.WithContext(ctx).Where("after_sale_id IN ?", asIDs).Find(&items).Error; err == nil {
		for _, it := range items {
			itemsByAS[it.AfterSaleID] = append(itemsByAS[it.AfterSaleID], MiniappAfterSaleItemResponse{
				ID:           it.ID,
				ProductID:    it.ProductID,
				SKUID:        it.SKUID,
				Quantity:     it.Quantity,
				RefundAmount: normalizeAmount(it.RefundAmount),
				ProductName:  it.ProductName,
				SKUText:      it.SKUText,
				SKUCode:      it.SKUCode,
				SKUImageURL:  it.SKUImageURL,
			})
		}
	}

	rows := make([]MiniappListItemResponse, 0, len(records))
	for _, r := range records {
		rows = append(rows, MiniappListItemResponse{
			ID:           r.ID,
			AfterSaleNo:  r.AfterSaleNo,
			OrderID:      r.OrderID,
			Type:         r.Type,
			TypeText:     model.AfterSaleTypeText(r.Type),
			Status:       r.Status,
			StatusText:   model.AfterSaleStatusText(r.Status),
			RefundAmount: normalizeAmount(r.RefundAmount),
			Reason:       r.Reason,
			CreatedAt:    r.CreatedAt,
			Items:        itemsByAS[r.ID],
		})
	}

	return rows, total, nil
}

// MiniappDetail returns member-accessible after-sale details.
func (s *Service) MiniappDetail(ctx context.Context, memberID int64, afterSaleID int64) (*MiniappDetailResponse, error) {
	if s.db == nil {
		return nil, errors.New("数据库服务不可用")
	}
	if memberID <= 0 {
		return nil, ErrAfterSaleNotBelong
	}

	var as model.AfterSale
	if err := s.db.WithContext(ctx).Where("id = ? AND member_id = ?", afterSaleID, memberID).First(&as).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var items []model.AfterSaleItem
	if err := s.db.WithContext(ctx).Where("after_sale_id = ?", as.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	itemResponses := make([]MiniappAfterSaleItemResponse, 0, len(items))
	for _, it := range items {
		itemResponses = append(itemResponses, MiniappAfterSaleItemResponse{
			ID:           it.ID,
			ProductID:    it.ProductID,
			SKUID:        it.SKUID,
			Quantity:     it.Quantity,
			RefundAmount: normalizeAmount(it.RefundAmount),
			ProductName:  it.ProductName,
			SKUText:      it.SKUText,
			SKUCode:      it.SKUCode,
			SKUImageURL:  it.SKUImageURL,
		})
	}

	var logs []model.AfterSaleLog
	if err := s.db.WithContext(ctx).Where("after_sale_id = ?", as.ID).Order("id ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	logResponses := make([]MiniappAfterSaleLogRow, 0, len(logs))
	for _, l := range logs {
		logResponses = append(logResponses, MiniappAfterSaleLogRow{
			Action:         l.Action,
			OperatorType:   model.AfterSaleOperatorTypeText(l.OperatorType),
			FromStatusText: model.AfterSaleStatusText(l.FromStatus),
			ToStatusText:   model.AfterSaleStatusText(l.ToStatus),
			Remark:         l.Remark,
			CreatedAt:      l.CreatedAt,
		})
	}

	var orderRefund model.OrderRefund
	var refundInfo *MiniappAfterSaleRefundInfo
	if err := s.db.WithContext(ctx).Where("after_sale_id = ?", as.ID).First(&orderRefund).Error; err == nil {
		refundInfo = &MiniappAfterSaleRefundInfo{
			Status:       orderRefund.Status,
			StatusText:   model.RefundStatusText(orderRefund.Status),
			RefundAmount: normalizeAmount(orderRefund.RefundAmount),
			RefundedAt:   orderRefund.RefundedAt,
		}
	}

	_, proofURLs, proofFiles := s.parseAndResolveProofs(ctx, as.ProofFileIDs)

	return &MiniappDetailResponse{
		ID:                 as.ID,
		AfterSaleNo:        as.AfterSaleNo,
		OrderID:            as.OrderID,
		Type:               as.Type,
		TypeText:           model.AfterSaleTypeText(as.Type),
		Status:             as.Status,
		StatusText:         model.AfterSaleStatusText(as.Status),
		Reason:             as.Reason,
		Description:        as.Description,
		ProofURLs:          proofURLs,
		ProofFiles:         proofFiles,
		RefundAmount:       normalizeAmount(as.RefundAmount),
		ReturnReceiver:     as.ReturnReceiver,
		ReturnPhone:        as.ReturnPhone,
		ReturnAddress:      as.ReturnAddress,
		BuyerReturnCompany: as.BuyerReturnCompany,
		BuyerReturnNo:      as.BuyerReturnNo,
		RejectReason:       as.RejectReason,
		AppliedAt:          as.AppliedAt,
		AuditedAt:          as.AuditedAt,
		BuyerReturnedAt:    as.BuyerReturnedAt,
		MerchantReceivedAt: as.MerchantReceivedAt,
		CompletedAt:        as.CompletedAt,
		CancelledAt:        as.CancelledAt,
		CreatedAt:          as.CreatedAt,
		Items:              itemResponses,
		Refund:             refundInfo,
		Logs:               logResponses,
	}, nil
}

func (s *Service) parseAndResolveProofs(ctx context.Context, proofJSON *string) ([]int64, []string, []AfterSaleProofFileResponse) {
	if proofJSON == nil || *proofJSON == "" || *proofJSON == "[]" {
		return []int64{}, []string{}, []AfterSaleProofFileResponse{}
	}
	var ids []int64
	if err := json.Unmarshal([]byte(*proofJSON), &ids); err != nil || len(ids) == 0 {
		return []int64{}, []string{}, []AfterSaleProofFileResponse{}
	}
	resolved, err := s.fileSvc.ResolvePrivate(ctx, ids)
	if err != nil {
		return ids, []string{}, []AfterSaleProofFileResponse{}
	}
	resMap := make(map[int64]systemfile.FileResponse, len(resolved))
	for _, f := range resolved {
		resMap[f.ID] = f
	}
	urls := make([]string, 0, len(ids))
	files := make([]AfterSaleProofFileResponse, 0, len(ids))
	for _, id := range ids {
		if f, ok := resMap[id]; ok {
			if f.URL != "" {
				urls = append(urls, f.URL)
			}
			files = append(files, AfterSaleProofFileResponse{
				ID:           f.ID,
				FileType:     f.FileType,
				OriginalName: f.OriginalName,
				MIMEType:     f.MIMEType,
				Size:         f.Size,
				URL:          f.URL,
				Width:        f.Width,
				Height:       f.Height,
				DurationMS:   f.DurationMS,
				CoverURL:     f.CoverURL,
			})
		}
	}
	return ids, urls, files
}

func (s *Service) UploadProof(ctx context.Context, memberID int64, filename string, reader io.Reader, declaredSize int64) (systemfile.FileResponse, error) {
	return s.fileSvc.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     filename,
		GroupID:      0,
		Scene:        model.SceneAfterSaleProof,
		OwnerType:    model.OwnerTypeMember,
		OwnerID:      memberID,
		Visibility:   model.VisibilityPrivate,
		Reader:       reader,
		DeclaredSize: declaredSize,
	})
}

func normalizeAmount(s string) string {
	if norm, err := util.NormalizeDecimalYuan(s); err == nil {
		return norm
	}
	return s
}

// Request types adapter for internal Miniapp
type MiniappApplyRequest struct {
	OrderID      int64   `json:"order_id"`
	Type         int8    `json:"type"`
	Reason       string  `json:"reason"`
	Description  string  `json:"description"`
	ProofFileIDs []int64 `json:"proof_file_ids"`
	RequestID    string  `json:"request_id"`
}

type MiniappListRequest struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Status   *int8 `json:"status"`
}

type MiniappListItemResponse struct {
	ID           int64                          `json:"id"`
	AfterSaleNo  string                         `json:"after_sale_no"`
	OrderID      int64                          `json:"order_id"`
	Type         int8                           `json:"type"`
	TypeText     string                         `json:"type_text"`
	Status       int8                           `json:"status"`
	StatusText   string                         `json:"status_text"`
	RefundAmount string                         `json:"refund_amount"`
	Reason       string                         `json:"reason"`
	CreatedAt    time.Time                      `json:"created_at"`
	Items        []MiniappAfterSaleItemResponse `json:"items"`
}

type MiniappAfterSaleItemResponse struct {
	ID           int64  `json:"id"`
	ProductID    int64  `json:"product_id"`
	SKUID        int64  `json:"sku_id"`
	Quantity     int    `json:"quantity"`
	RefundAmount string `json:"refund_amount"`
	ProductName  string `json:"product_name"`
	SKUText      string `json:"sku_text"`
	SKUCode      string `json:"sku_code"`
	SKUImageURL  string `json:"sku_image_url"`
}

type MiniappAfterSaleLogRow struct {
	Action         string    `json:"action"`
	OperatorType   string    `json:"operator_type"`
	FromStatusText string    `json:"from_status_text"`
	ToStatusText   string    `json:"to_status_text"`
	Remark         string    `json:"remark"`
	CreatedAt      time.Time `json:"created_at"`
}

type MiniappAfterSaleRefundInfo struct {
	Status       int8       `json:"status"`
	StatusText   string     `json:"status_text"`
	RefundAmount string     `json:"refund_amount"`
	RefundedAt   *time.Time `json:"refunded_at"`
}

type MiniappDetailResponse struct {
	ID                 int64                          `json:"id"`
	AfterSaleNo        string                         `json:"after_sale_no"`
	OrderID            int64                          `json:"order_id"`
	Type               int8                           `json:"type"`
	TypeText           string                         `json:"type_text"`
	Status             int8                           `json:"status"`
	StatusText         string                         `json:"status_text"`
	Reason             string                         `json:"reason"`
	Description        string                         `json:"description"`
	ProofURLs          []string                       `json:"proof_urls"`
	ProofFiles         []AfterSaleProofFileResponse   `json:"proof_files"`
	RefundAmount       string                         `json:"refund_amount"`
	ReturnReceiver     string                         `json:"return_receiver"`
	ReturnPhone        string                         `json:"return_phone"`
	ReturnAddress      string                         `json:"return_address"`
	BuyerReturnCompany string                         `json:"buyer_return_company"`
	BuyerReturnNo      string                         `json:"buyer_return_no"`
	RejectReason       string                         `json:"reject_reason"`
	AppliedAt          time.Time                      `json:"applied_at"`
	AuditedAt          *time.Time                     `json:"audited_at"`
	BuyerReturnedAt    *time.Time                     `json:"buyer_returned_at"`
	MerchantReceivedAt *time.Time                     `json:"merchant_received_at"`
	CompletedAt        *time.Time                     `json:"completed_at"`
	CancelledAt        *time.Time                     `json:"cancelled_at"`
	CreatedAt          time.Time                      `json:"created_at"`
	Items              []MiniappAfterSaleItemResponse `json:"items"`
	Refund             *MiniappAfterSaleRefundInfo    `json:"refund"`
	Logs               []MiniappAfterSaleLogRow       `json:"logs"`
}
