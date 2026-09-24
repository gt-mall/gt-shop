package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/util"
)

var (
	ErrNotFound       = errors.New("订单不存在")
	ErrInvalidStatus  = errors.New("订单当前状态不可发货")
	ErrExpressMissing = errors.New("物流公司和运单号不能为空")
)

func normalizeAmount(s string) string {
	if norm, err := util.NormalizeDecimalYuan(s); err == nil {
		return norm
	}
	return s
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// List queries orders with filters and keyword searching.
func (s *Service) List(ctx context.Context, req OrderListRequest) ([]OrderListRow, int64, error) {
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

	query := s.db.WithContext(ctx).Model(&model.OrderMain{}).Where("deleted_at IS NULL")

	if req.Status != nil && *req.Status > 0 {
		query = query.Where("status = ?", *req.Status)
	}
	if req.MemberID != nil && *req.MemberID > 0 {
		query = query.Where("member_id = ?", *req.MemberID)
	}

	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("(order_no LIKE ? OR id IN (SELECT order_id FROM order_address WHERE name LIKE ? OR mobile LIKE ?))", kw, kw, kw)
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

	var orders []model.OrderMain
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	if len(orders) == 0 {
		return []OrderListRow{}, total, nil
	}

	orderIDs := make([]int64, 0, len(orders))
	memberIDs := make([]int64, 0, len(orders))
	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
		memberIDs = append(memberIDs, o.MemberID)
	}

	// Batch query members
	var members []model.Member
	if err := s.db.WithContext(ctx).Where("id IN (?)", memberIDs).Find(&members).Error; err != nil {
		return nil, 0, err
	}
	memberMap := make(map[int64]model.Member)
	for _, m := range members {
		memberMap[m.ID] = m
	}

	// Batch query addresses
	var addrs []model.OrderAddress
	if err := s.db.WithContext(ctx).Where("order_id IN (?)", orderIDs).Find(&addrs).Error; err != nil {
		return nil, 0, err
	}
	addrMap := make(map[int64]OrderAddressRow)
	for _, a := range addrs {
		addrMap[a.OrderID] = OrderAddressRow{
			Name:         a.Name,
			Mobile:       a.Mobile,
			ProvinceCode: a.ProvinceCode,
			Province:     a.Province,
			CityCode:     a.CityCode,
			City:         a.City,
			DistrictCode: a.DistrictCode,
			District:     a.District,
			Address:      a.Address,
			PostalCode:   a.PostalCode,
		}
	}

	// Batch query items
	var items []model.OrderItem
	if err := s.db.WithContext(ctx).Where("order_id IN (?)", orderIDs).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	imageIDs := make([]int64, 0, len(items))
	for _, it := range items {
		if it.SKUImageID != nil && *it.SKUImageID > 0 {
			imageIDs = append(imageIDs, *it.SKUImageID)
		}
	}
	urlMap := s.resolveImages(ctx, imageIDs)

	prodIDs := make([]int64, 0, len(items))
	for _, it := range items {
		if it.SKUSpecValues != nil && *it.SKUSpecValues != "" && *it.SKUSpecValues != "[]" && !strings.Contains(*it.SKUSpecValues, "value_name") {
			prodIDs = append(prodIDs, it.ProductID)
		}
	}
	prodSpecsMap := s.resolveProductSpecs(ctx, prodIDs)

	itemsMap := make(map[int64][]OrderItemRow)
	for _, it := range items {
		imgURL := it.SKUImageURL
		if it.SKUImageID != nil && *it.SKUImageID > 0 {
			if u, ok := urlMap[*it.SKUImageID]; ok && strings.TrimSpace(u) != "" {
				imgURL = u
			}
		}
		specVal := it.SKUSpecValues
		if specVal != nil && *specVal != "" && *specVal != "[]" && !strings.Contains(*specVal, "value_name") {
			if pSpecs, ok := prodSpecsMap[it.ProductID]; ok {
				enriched := util.EnrichSKUSpecValues(*specVal, pSpecs)
				specVal = &enriched
			}
		}
		itemsMap[it.OrderID] = append(itemsMap[it.OrderID], OrderItemRow{
			ID:            it.ID,
			ProductID:     it.ProductID,
			SKUID:         it.SKUID,
			ProductName:   it.ProductName,
			SKUCode:       it.SKUCode,
			SKUSpecValues: specVal,
			ImageURL:      imgURL,
			UnitPrice:     normalizeAmount(it.UnitPrice),
			Quantity:      it.Quantity,
			TotalAmount:   normalizeAmount(it.TotalAmount),
		})
	}

	result := make([]OrderListRow, 0, len(orders))
	for _, o := range orders {
		mem := memberMap[o.MemberID]
		nickname := mem.Nickname
		mobile := ""
		if mem.Mobile != nil {
			mobile = *mem.Mobile
		}

		var addrRow *OrderAddressRow
		if a, ok := addrMap[o.ID]; ok {
			addrRow = &a
		}

		orderItems := itemsMap[o.ID]
		totalQty := 0
		for _, it := range orderItems {
			totalQty += it.Quantity
		}

		result = append(result, OrderListRow{
			ID:             o.ID,
			OrderNo:        o.OrderNo,
			MemberID:       o.MemberID,
			MemberNickname: nickname,
			MemberMobile:   mobile,
			Status:         o.Status,
			StatusText:     model.OrderStatusText(o.Status),
			TotalAmount:    normalizeAmount(o.TotalAmount),
			PayAmount:      normalizeAmount(o.PayAmount),
			FreightAmount:  normalizeAmount(o.FreightAmount),
			TotalQuantity:  totalQty,
			ExpressCompany: o.ExpressCompany,
			ExpressNo:      o.ExpressNo,
			PaidAt:         o.PaidAt,
			ShippedAt:      o.ShippedAt,
			CompletedAt:    o.CompletedAt,
			CancelledAt:    o.CancelledAt,
			CreatedAt:      o.CreatedAt,
			Address:        addrRow,
			Items:          orderItems,
		})
	}

	return result, total, nil
}

// Detail retrieves complete order details for admin.
func (s *Service) Detail(ctx context.Context, id int64) (OrderDetailResponse, error) {
	var o model.OrderMain
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&o).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return OrderDetailResponse{}, ErrNotFound
		}
		return OrderDetailResponse{}, err
	}

	var mem model.Member
	if err := s.db.WithContext(ctx).First(&mem, o.MemberID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return OrderDetailResponse{}, err
	}
	mobile := ""
	if mem.Mobile != nil {
		mobile = *mem.Mobile
	}

	var addr model.OrderAddress
	var addrRow *OrderAddressRow
	if err := s.db.WithContext(ctx).Where("order_id = ?", o.ID).First(&addr).Error; err == nil {
		addrRow = &OrderAddressRow{
			Name:         addr.Name,
			Mobile:       addr.Mobile,
			ProvinceCode: addr.ProvinceCode,
			Province:     addr.Province,
			CityCode:     addr.CityCode,
			City:         addr.City,
			DistrictCode: addr.DistrictCode,
			District:     addr.District,
			Address:      addr.Address,
			PostalCode:   addr.PostalCode,
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return OrderDetailResponse{}, err
	}

	var items []model.OrderItem
	if err := s.db.WithContext(ctx).Where("order_id = ?", o.ID).Find(&items).Error; err != nil {
		return OrderDetailResponse{}, err
	}

	imageIDs := make([]int64, 0, len(items))
	for _, it := range items {
		if it.SKUImageID != nil && *it.SKUImageID > 0 {
			imageIDs = append(imageIDs, *it.SKUImageID)
		}
	}
	urlMap := s.resolveImages(ctx, imageIDs)

	prodIDs := make([]int64, 0, len(items))
	for _, it := range items {
		if it.SKUSpecValues != nil && *it.SKUSpecValues != "" && *it.SKUSpecValues != "[]" && !strings.Contains(*it.SKUSpecValues, "value_name") {
			prodIDs = append(prodIDs, it.ProductID)
		}
	}
	prodSpecsMap := s.resolveProductSpecs(ctx, prodIDs)

	itemRows := make([]OrderItemRow, 0, len(items))
	for _, it := range items {
		imgURL := it.SKUImageURL
		if it.SKUImageID != nil && *it.SKUImageID > 0 {
			if u, ok := urlMap[*it.SKUImageID]; ok && strings.TrimSpace(u) != "" {
				imgURL = u
			}
		}
		specVal := it.SKUSpecValues
		if specVal != nil && *specVal != "" && *specVal != "[]" && !strings.Contains(*specVal, "value_name") {
			if pSpecs, ok := prodSpecsMap[it.ProductID]; ok {
				enriched := util.EnrichSKUSpecValues(*specVal, pSpecs)
				specVal = &enriched
			}
		}
		itemRows = append(itemRows, OrderItemRow{
			ID:            it.ID,
			ProductID:     it.ProductID,
			SKUID:         it.SKUID,
			ProductName:   it.ProductName,
			SKUCode:       it.SKUCode,
			SKUSpecValues: specVal,
			ImageURL:      imgURL,
			UnitPrice:     normalizeAmount(it.UnitPrice),
			Quantity:      it.Quantity,
			TotalAmount:   normalizeAmount(it.TotalAmount),
		})
	}

	var logs []model.OrderLog
	if err := s.db.WithContext(ctx).Where("order_id = ?", o.ID).Order("id ASC").Find(&logs).Error; err != nil {
		return OrderDetailResponse{}, err
	}
	logRows := make([]OrderLogRow, 0, len(logs))
	for _, l := range logs {
		logRows = append(logRows, OrderLogRow{
			ID:             l.ID,
			Action:         l.Action,
			OperatorType:   l.OperatorType,
			OperatorName:   l.OperatorName,
			FromStatus:     l.FromStatus,
			FromStatusText: model.OrderStatusText(l.FromStatus),
			ToStatus:       l.ToStatus,
			ToStatusText:   model.OrderStatusText(l.ToStatus),
			Remark:         l.Remark,
			CreatedAt:      l.CreatedAt,
		})
	}

	return OrderDetailResponse{
		ID:             o.ID,
		OrderNo:        o.OrderNo,
		MemberID:       o.MemberID,
		MemberNickname: mem.Nickname,
		MemberMobile:   mobile,
		Status:         o.Status,
		StatusText:     model.OrderStatusText(o.Status),
		TotalAmount:    normalizeAmount(o.TotalAmount),
		PayAmount:      normalizeAmount(o.PayAmount),
		FreightAmount:  normalizeAmount(o.FreightAmount),
		DiscountAmount: normalizeAmount(o.DiscountAmount),
		ExpressCompany: o.ExpressCompany,
		ExpressNo:      o.ExpressNo,
		Remark:         o.Remark,
		PaidAt:         o.PaidAt,
		ShippedAt:      o.ShippedAt,
		CompletedAt:    o.CompletedAt,
		CancelledAt:    o.CancelledAt,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
		Address:        addrRow,
		Items:          itemRows,
		Logs:           logRows,
	}, nil
}

// Ship updates order status to pending_receipt and stores shipping details.
func (s *Service) Ship(ctx context.Context, user model.SystemUser, req OrderShipRequest) error {
	company := strings.TrimSpace(req.ExpressCompany)
	no := strings.TrimSpace(req.ExpressNo)
	if company == "" || no == "" {
		return ErrExpressMissing
	}

	tradeCfg, _ := systemsetting.NewService(s.db).GetTrade(ctx)
	autoReceiveDays := tradeCfg.AutoReceiveDays
	if autoReceiveDays <= 0 {
		autoReceiveDays = 7
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var o model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", req.ID).
			First(&o).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if o.Status != model.OrderStatusPendingShipment {
			return fmt.Errorf("订单当前状态(%s)不可发货", model.OrderStatusText(o.Status))
		}

		if tx.Migrator().HasTable(&model.AfterSale{}) {
			var activeAfterSaleCount int64
			if err := tx.Model(&model.AfterSale{}).
				Where("order_id = ? AND status IN ?", o.ID, []int8{
					model.AfterSaleStatusPendingReview,
					model.AfterSaleStatusWaitingBuyerReturn,
					model.AfterSaleStatusWaitingMerchantReceive,
					model.AfterSaleStatusRefunding,
					model.AfterSaleStatusRefundFailed,
				}).Count(&activeAfterSaleCount).Error; err != nil {
				return err
			}
			if activeAfterSaleCount > 0 {
				return errors.New("订单存在进行中的售后申请，暂不可发货")
			}
		}

		now := time.Now()
		autoReceiveAt := now.Add(time.Duration(autoReceiveDays) * 24 * time.Hour)

		res := tx.Model(&model.OrderMain{}).
			Where("id = ? AND status = ?", o.ID, o.Status).
			Updates(map[string]any{
				"status":          model.OrderStatusPendingReceipt,
				"express_company": company,
				"express_no":      no,
				"shipped_at":      &now,
				"auto_receive_at": &autoReceiveAt,
				"updated_at":      now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("订单状态已变更，请刷新")
		}

		opName := strings.TrimSpace(user.Nickname)
		if opName == "" {
			opName = user.Username
		}

		orderLog := model.OrderLog{
			OrderID:      o.ID,
			FromStatus:   model.OrderStatusPendingShipment,
			ToStatus:     model.OrderStatusPendingReceipt,
			Action:       "ship",
			OperatorType: "admin",
			OperatorID:   user.ID,
			OperatorName: opName,
			Remark:       fmt.Sprintf("订单发货: %s %s", company, no),
			CreatedAt:    now,
		}
		return tx.Create(&orderLog).Error
	})
}

func (s *Service) resolveImages(ctx context.Context, imageIDs []int64) map[int64]string {
	result := make(map[int64]string)
	if len(imageIDs) == 0 {
		return result
	}
	fileSvc := systemfile.NewService(s.db)
	resolved, err := fileSvc.Resolve(ctx, systemfile.IDsRequest{IDs: imageIDs})
	if err == nil {
		for _, f := range resolved {
			result[f.ID] = f.URL
		}
	}
	return result
}

func (s *Service) resolveProductSpecs(ctx context.Context, productIDs []int64) map[int64]*string {
	result := make(map[int64]*string)
	if len(productIDs) == 0 {
		return result
	}
	var prods []model.Product
	if err := s.db.WithContext(ctx).Select("id, specs").Where("id IN ?", productIDs).Find(&prods).Error; err == nil {
		for _, p := range prods {
			result[p.ID] = p.Specs
		}
	}
	return result
}
