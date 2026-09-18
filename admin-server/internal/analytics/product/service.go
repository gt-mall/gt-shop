package product

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"mall/admin_core/internal/analytics"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/util"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type productSaleRawRow struct {
	ProductID      int64  `gorm:"column:product_id"`
	ProductName    string `gorm:"column:product_name"`
	SKUID          int64  `gorm:"column:sku_id"`
	SKUCode        string `gorm:"column:sku_code"`
	SKUText        string `gorm:"column:sku_text"`
	ImageURL       string `gorm:"column:image_url"`
	Quantity       int64  `gorm:"column:quantity"`
	SalesAmount    string `gorm:"column:sales_amount"`
	PaidOrderCount int64  `gorm:"column:paid_order_count"`
	BuyerCount     int64  `gorm:"column:buyer_count"`
}

type productRefundRawRow struct {
	ProductID      int64  `gorm:"column:product_id"`
	ProductName    string `gorm:"column:product_name"`
	SKUID          int64  `gorm:"column:sku_id"`
	SKUCode        string `gorm:"column:sku_code"`
	SKUText        string `gorm:"column:sku_text"`
	ImageURL       string `gorm:"column:image_url"`
	RefundQuantity int64  `gorm:"column:refund_quantity"`
	RefundAmount   string `gorm:"column:refund_amount"`
}

func (s *Service) Rankings(ctx context.Context, req ProductRankingsRequest) (ProductRankingsResponse, error) {
	if s.db == nil {
		return ProductRankingsResponse{}, errors.New("数据库服务不可用")
	}

	startTime, endTime, _, _, _, err := analytics.ParseTimeRange(req.TimeRange, req.StartTime, req.EndTime)
	if err != nil {
		return ProductRankingsResponse{}, err
	}

	// 1. Query product sales aggregates
	var salesRows []productSaleRawRow
	err = s.db.WithContext(ctx).Table("order_item AS i").
		Select(`
			i.product_id,
			MAX(i.product_name) AS product_name,
			COALESCE(MAX(i.sku_image_url), '') AS image_url,
			COALESCE(SUM(i.quantity), 0) AS quantity,
			COALESCE(SUM(i.total_amount), 0) AS sales_amount,
			COUNT(DISTINCT i.order_id) AS paid_order_count
		`).
		Where("i.order_id IN (SELECT DISTINCT order_id FROM order_payment WHERE status = ? AND paid_at >= ? AND paid_at < ?)", model.PaymentStatusSuccess, startTime, endTime).
		Group("i.product_id").
		Scan(&salesRows).Error
	if err != nil {
		return ProductRankingsResponse{}, err
	}

	// 2. Query product refund aggregates
	var refundRows []productRefundRawRow
	err = s.db.WithContext(ctx).Table("after_sale_item AS asi").
		Select(`
			asi.product_id,
			MAX(asi.product_name) AS product_name,
			COALESCE(MAX(asi.sku_image_url), '') AS image_url,
			COALESCE(SUM(asi.quantity), 0) AS refund_quantity,
			COALESCE(SUM(asi.refund_amount), 0) AS refund_amount
		`).
		Joins("INNER JOIN order_refund AS r ON r.after_sale_id = asi.after_sale_id AND r.status = ? AND r.refunded_at >= ? AND r.refunded_at < ?", model.RefundStatusSuccess, startTime, endTime).
		Group("asi.product_id").
		Scan(&refundRows).Error
	if err != nil {
		return ProductRankingsResponse{}, err
	}

	refundMap := make(map[int64]productRefundRawRow, len(refundRows))
	for _, r := range refundRows {
		refundMap[r.ProductID] = r
	}

	itemsMap := make(map[int64]*ProductRankingItem)
	for _, sRow := range salesRows {
		sCents, _ := util.ParseDecimalYuanToCent(sRow.SalesAmount)
		refRow := refundMap[sRow.ProductID]
		rCents, _ := util.ParseDecimalYuanToCent(refRow.RefundAmount)

		itemsMap[sRow.ProductID] = &ProductRankingItem{
			ProductID:      sRow.ProductID,
			ProductName:    sRow.ProductName,
			ImageURL:       sRow.ImageURL,
			SalesQuantity:  sRow.Quantity,
			SalesAmount:    util.FormatCentToYuan(sCents),
			RefundQuantity: refRow.RefundQuantity,
			RefundAmount:   util.FormatCentToYuan(rCents),
			RefundRate:     analytics.CalcRefundRate(rCents, sCents),
		}
	}

	// Also add products that had refunds but 0 sales in this period
	for _, rRow := range refundRows {
		if _, ok := itemsMap[rRow.ProductID]; !ok {
			prodName := rRow.ProductName
			imgURL := rRow.ImageURL
			if prodName == "" {
				var prod model.Product
				if err := s.db.WithContext(ctx).Table("product").Where("id = ?", rRow.ProductID).First(&prod).Error; err == nil {
					prodName = prod.Name
				}
			}
			rCents, _ := util.ParseDecimalYuanToCent(rRow.RefundAmount)
			itemsMap[rRow.ProductID] = &ProductRankingItem{
				ProductID:      rRow.ProductID,
				ProductName:    prodName,
				ImageURL:       imgURL,
				SalesQuantity:  0,
				SalesAmount:    "0.00",
				RefundQuantity: rRow.RefundQuantity,
				RefundAmount:   util.FormatCentToYuan(rCents),
				RefundRate:     analytics.CalcRefundRate(rCents, 0),
			}
		}
	}

	allList := make([]ProductRankingItem, 0, len(itemsMap))
	for _, v := range itemsMap {
		allList = append(allList, *v)
	}

	// Top Quantity Rankings
	byQuantity := make([]ProductRankingItem, len(allList))
	copy(byQuantity, allList)
	sort.Slice(byQuantity, func(i, j int) bool {
		if byQuantity[i].SalesQuantity != byQuantity[j].SalesQuantity {
			return byQuantity[i].SalesQuantity > byQuantity[j].SalesQuantity
		}
		return byQuantity[i].ProductID < byQuantity[j].ProductID
	})
	if len(byQuantity) > 10 {
		byQuantity = byQuantity[:10]
	}

	// Top Sales Amount Rankings
	byAmount := make([]ProductRankingItem, len(allList))
	copy(byAmount, allList)
	sort.Slice(byAmount, func(i, j int) bool {
		cA, _ := util.ParseDecimalYuanToCent(byAmount[i].SalesAmount)
		cB, _ := util.ParseDecimalYuanToCent(byAmount[j].SalesAmount)
		if cA != cB {
			return cA > cB
		}
		return byAmount[i].ProductID < byAmount[j].ProductID
	})
	if len(byAmount) > 10 {
		byAmount = byAmount[:10]
	}

	// Top Refund Rankings
	byRefund := make([]ProductRankingItem, len(allList))
	copy(byRefund, allList)
	sort.Slice(byRefund, func(i, j int) bool {
		rA, _ := util.ParseDecimalYuanToCent(byRefund[i].RefundAmount)
		rB, _ := util.ParseDecimalYuanToCent(byRefund[j].RefundAmount)
		if rA != rB {
			return rA > rB
		}
		if byRefund[i].RefundQuantity != byRefund[j].RefundQuantity {
			return byRefund[i].RefundQuantity > byRefund[j].RefundQuantity
		}
		return byRefund[i].ProductID < byRefund[j].ProductID
	})
	if len(byRefund) > 10 {
		byRefund = byRefund[:10]
	}

	return ProductRankingsResponse{
		SalesQuantityRankings: byQuantity,
		SalesAmountRankings:   byAmount,
		RefundRankings:        byRefund,
	}, nil
}

func (s *Service) List(ctx context.Context, req ProductListRequest) ([]ProductAnalysisRow, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("数据库服务不可用")
	}

	startTime, endTime, _, _, _, err := analytics.ParseTimeRange(req.TimeRange, req.StartTime, req.EndTime)
	if err != nil {
		return nil, 0, err
	}

	isSKUDimension := strings.ToLower(strings.TrimSpace(req.Dimension)) == "sku"

	// 1. Query sales
	var salesRows []productSaleRawRow
	if isSKUDimension {
		err = s.db.WithContext(ctx).Table("order_item AS i").
			Select(`
				i.product_id,
				MAX(i.product_name) AS product_name,
				i.sku_id,
				MAX(i.sku_code) AS sku_code,
				COALESCE(MAX(i.sku_spec_values), '') AS sku_text,
				COALESCE(MAX(i.sku_image_url), '') AS image_url,
				COALESCE(SUM(i.quantity), 0) AS quantity,
				COALESCE(SUM(i.total_amount), 0) AS sales_amount,
				COUNT(DISTINCT i.order_id) AS paid_order_count,
				COUNT(DISTINCT o.member_id) AS buyer_count
			`).
			Joins("INNER JOIN order_main AS o ON o.id = i.order_id").
			Where("i.order_id IN (SELECT DISTINCT order_id FROM order_payment WHERE status = ? AND paid_at >= ? AND paid_at < ?)", model.PaymentStatusSuccess, startTime, endTime).
			Group("i.product_id, i.sku_id").
			Scan(&salesRows).Error
	} else {
		err = s.db.WithContext(ctx).Table("order_item AS i").
			Select(`
				i.product_id,
				MAX(i.product_name) AS product_name,
				0 AS sku_id,
				'' AS sku_code,
				'' AS sku_text,
				COALESCE(MAX(i.sku_image_url), '') AS image_url,
				COALESCE(SUM(i.quantity), 0) AS quantity,
				COALESCE(SUM(i.total_amount), 0) AS sales_amount,
				COUNT(DISTINCT i.order_id) AS paid_order_count,
				COUNT(DISTINCT o.member_id) AS buyer_count
			`).
			Joins("INNER JOIN order_main AS o ON o.id = i.order_id").
			Where("i.order_id IN (SELECT DISTINCT order_id FROM order_payment WHERE status = ? AND paid_at >= ? AND paid_at < ?)", model.PaymentStatusSuccess, startTime, endTime).
			Group("i.product_id").
			Scan(&salesRows).Error
	}
	if err != nil {
		return nil, 0, err
	}

	// 2. Query refunds
	var refundRows []productRefundRawRow
	if isSKUDimension {
		err = s.db.WithContext(ctx).Table("after_sale_item AS asi").
			Select(`
				asi.product_id,
				MAX(asi.product_name) AS product_name,
				asi.sku_id,
				MAX(asi.sku_code) AS sku_code,
				COALESCE(MAX(asi.sku_text), '') AS sku_text,
				COALESCE(MAX(asi.sku_image_url), '') AS image_url,
				COALESCE(SUM(asi.quantity), 0) AS refund_quantity,
				COALESCE(SUM(asi.refund_amount), 0) AS refund_amount
			`).
			Joins("INNER JOIN order_refund AS r ON r.after_sale_id = asi.after_sale_id AND r.status = ? AND r.refunded_at >= ? AND r.refunded_at < ?", model.RefundStatusSuccess, startTime, endTime).
			Group("asi.product_id, asi.sku_id").
			Scan(&refundRows).Error
	} else {
		err = s.db.WithContext(ctx).Table("after_sale_item AS asi").
			Select(`
				asi.product_id,
				MAX(asi.product_name) AS product_name,
				0 AS sku_id,
				'' AS sku_code,
				'' AS sku_text,
				COALESCE(MAX(asi.sku_image_url), '') AS image_url,
				COALESCE(SUM(asi.quantity), 0) AS refund_quantity,
				COALESCE(SUM(asi.refund_amount), 0) AS refund_amount
			`).
			Joins("INNER JOIN order_refund AS r ON r.after_sale_id = asi.after_sale_id AND r.status = ? AND r.refunded_at >= ? AND r.refunded_at < ?", model.RefundStatusSuccess, startTime, endTime).
			Group("asi.product_id").
			Scan(&refundRows).Error
	}
	if err != nil {
		return nil, 0, err
	}

	refundMap := make(map[string]productRefundRawRow, len(refundRows))
	for _, r := range refundRows {
		k := keyFor(r.ProductID, r.SKUID, isSKUDimension)
		refundMap[k] = r
	}

	// 3. Merge rows
	mergedMap := make(map[string]*ProductAnalysisRow)
	for _, sRow := range salesRows {
		k := keyFor(sRow.ProductID, sRow.SKUID, isSKUDimension)
		refRow := refundMap[k]
		sCents, _ := util.ParseDecimalYuanToCent(sRow.SalesAmount)
		rCents, _ := util.ParseDecimalYuanToCent(refRow.RefundAmount)

		mergedMap[k] = &ProductAnalysisRow{
			ProductID:      sRow.ProductID,
			ProductName:    sRow.ProductName,
			SKUID:          sRow.SKUID,
			SKUCode:        sRow.SKUCode,
			SKUText:        sRow.SKUText,
			ImageURL:       sRow.ImageURL,
			SalesQuantity:  sRow.Quantity,
			SalesAmount:    util.FormatCentToYuan(sCents),
			PaidOrderCount: sRow.PaidOrderCount,
			BuyerCount:     sRow.BuyerCount,
			RefundQuantity: refRow.RefundQuantity,
			RefundAmount:   util.FormatCentToYuan(rCents),
			RefundRate:     analytics.CalcRefundRate(rCents, sCents),
		}
	}

	// Include items with refunds but no sales in the period
	for _, rRow := range refundRows {
		k := keyFor(rRow.ProductID, rRow.SKUID, isSKUDimension)
		if _, ok := mergedMap[k]; !ok {
			prodName := rRow.ProductName
			imgURL := rRow.ImageURL
			if prodName == "" {
				var p model.Product
				if s.db.WithContext(ctx).Table("product").Where("id = ?", rRow.ProductID).First(&p).Error == nil {
					prodName = p.Name
				}
			}

			skuCode := rRow.SKUCode
			skuText := rRow.SKUText
			if isSKUDimension && rRow.SKUID > 0 && skuCode == "" {
				var sku model.ProductSKU
				if err := s.db.WithContext(ctx).Table("product_sku").Where("id = ?", rRow.SKUID).First(&sku).Error; err == nil {
					if sku.SKUCode != nil {
						skuCode = *sku.SKUCode
					}
				}
			}

			rCents, _ := util.ParseDecimalYuanToCent(rRow.RefundAmount)
			mergedMap[k] = &ProductAnalysisRow{
				ProductID:      rRow.ProductID,
				ProductName:    prodName,
				SKUID:          rRow.SKUID,
				SKUCode:        skuCode,
				SKUText:        skuText,
				ImageURL:       imgURL,
				SalesQuantity:  0,
				SalesAmount:    "0.00",
				PaidOrderCount: 0,
				BuyerCount:     0,
				RefundQuantity: rRow.RefundQuantity,
				RefundAmount:   util.FormatCentToYuan(rCents),
				RefundRate:     analytics.CalcRefundRate(rCents, 0),
			}
		}
	}

	// 4. Filtering
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	skuCodeReq := strings.ToLower(strings.TrimSpace(req.SKUCode))

	var filtered []ProductAnalysisRow
	for _, row := range mergedMap {
		if req.ProductID != nil && *req.ProductID > 0 && row.ProductID != *req.ProductID {
			continue
		}
		if keyword != "" {
			matchName := strings.Contains(strings.ToLower(row.ProductName), keyword)
			matchCode := strings.Contains(strings.ToLower(row.SKUCode), keyword)
			if !matchName && !matchCode {
				continue
			}
		}
		if skuCodeReq != "" && !strings.Contains(strings.ToLower(row.SKUCode), skuCodeReq) {
			continue
		}
		filtered = append(filtered, *row)
	}

	// 5. Sorting
	isAsc := strings.ToLower(req.SortOrder) == "asc"
	sortBy := strings.ToLower(strings.TrimSpace(req.SortBy))
	if sortBy == "" {
		sortBy = "sales_amount"
	}

	sort.Slice(filtered, func(i, j int) bool {
		var cmp int
		switch sortBy {
		case "sales_quantity":
			if filtered[i].SalesQuantity != filtered[j].SalesQuantity {
				if filtered[i].SalesQuantity > filtered[j].SalesQuantity {
					cmp = 1
				} else {
					cmp = -1
				}
			}
		case "buyer_count":
			if filtered[i].BuyerCount != filtered[j].BuyerCount {
				if filtered[i].BuyerCount > filtered[j].BuyerCount {
					cmp = 1
				} else {
					cmp = -1
				}
			}
		case "refund_amount":
			cA, _ := util.ParseDecimalYuanToCent(filtered[i].RefundAmount)
			cB, _ := util.ParseDecimalYuanToCent(filtered[j].RefundAmount)
			if cA != cB {
				if cA > cB {
					cmp = 1
				} else {
					cmp = -1
				}
			}
		case "refund_quantity":
			if filtered[i].RefundQuantity != filtered[j].RefundQuantity {
				if filtered[i].RefundQuantity > filtered[j].RefundQuantity {
					cmp = 1
				} else {
					cmp = -1
				}
			}
		case "paid_order_count":
			if filtered[i].PaidOrderCount != filtered[j].PaidOrderCount {
				if filtered[i].PaidOrderCount > filtered[j].PaidOrderCount {
					cmp = 1
				} else {
					cmp = -1
				}
			}
		default: // sales_amount
			cA, _ := util.ParseDecimalYuanToCent(filtered[i].SalesAmount)
			cB, _ := util.ParseDecimalYuanToCent(filtered[j].SalesAmount)
			if cA != cB {
				if cA > cB {
					cmp = 1
				} else {
					cmp = -1
				}
			}
		}

		if cmp == 0 {
			if filtered[i].ProductID != filtered[j].ProductID {
				return filtered[i].ProductID < filtered[j].ProductID
			}
			return filtered[i].SKUID < filtered[j].SKUID
		}
		if isAsc {
			return cmp < 0
		}
		return cmp > 0
	})

	total := int64(len(filtered))

	// 6. Pagination
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	startIndex := (page - 1) * pageSize
	if startIndex >= len(filtered) {
		return []ProductAnalysisRow{}, total, nil
	}
	endIndex := startIndex + pageSize
	if endIndex > len(filtered) {
		endIndex = len(filtered)
	}

	return filtered[startIndex:endIndex], total, nil
}

func keyFor(productID, skuID int64, isSKU bool) string {
	if isSKU {
		return strconv.FormatInt(productID, 10) + ":" + strconv.FormatInt(skuID, 10)
	}
	return strconv.FormatInt(productID, 10)
}
