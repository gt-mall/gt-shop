package order

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mall/admin_core/internal/model"
	productfreight "mall/admin_core/internal/product/freight"
	productinventory "mall/admin_core/internal/product/inventory"
	"mall/admin_core/internal/region"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/bizno"
	"mall/admin_core/pkg/util"
)

var (
	ErrNotFound       = errors.New("订单不存在")
	ErrInvalidStatus  = errors.New("订单状态不符合操作要求")
	ErrAddressMissing = errors.New("请选择有效的收货地址")
	ErrItemsEmpty     = errors.New("订单商品不能为空")
)

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if mysqlErr.Number == 1062 {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique")
}

type Service struct {
	db        *gorm.DB
	bizNoGen  *bizno.Generator
	regionSvc *region.Service
}

func NewService(db *gorm.DB, bizNoGen ...*bizno.Generator) *Service {
	var gen *bizno.Generator
	if len(bizNoGen) > 0 && bizNoGen[0] != nil {
		gen = bizNoGen[0]
	} else {
		var err error
		gen, err = bizno.NewGenerator(0)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize default bizno generator: %v", err))
		}
	}
	rs, _ := region.NewService()
	return &Service{db: db, bizNoGen: gen, regionSvc: rs}
}

func (s *Service) SetRegionService(rs *region.Service) {
	s.regionSvc = rs
}

func (s *Service) SetBizNoGenerator(gen *bizno.Generator) {
	if gen != nil {
		s.bizNoGen = gen
	}
}

func normalizeAmount(s string) string {
	if norm, err := util.NormalizeDecimalYuan(s); err == nil {
		return norm
	}
	return s
}

// Preview calculates order totals and validates items & address for checkout.
func (s *Service) Preview(ctx context.Context, memberID int64, req CheckoutPreviewRequest) (CheckoutPreviewResponse, error) {
	if len(req.Items) == 0 {
		return CheckoutPreviewResponse{}, ErrItemsEmpty
	}

	// 1. Resolve Address
	var chosenAddress *CheckoutAddressResponse
	if req.AddressID != nil && *req.AddressID > 0 {
		var addr model.MemberAddress
		if err := s.db.WithContext(ctx).Where("id = ? AND member_id = ? AND deleted_at IS NULL", *req.AddressID, memberID).First(&addr).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return CheckoutPreviewResponse{}, errors.New("收货地址不存在或已删除")
			}
			return CheckoutPreviewResponse{}, err
		}
		chosenAddress = &CheckoutAddressResponse{
			ID:           addr.ID,
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
			IsDefault:    addr.IsDefault,
		}
	} else {
		// 没有提供 address_id：自动寻找默认地址，没有默认地址再取第一条有效地址
		var defaultAddr model.MemberAddress
		if err := s.db.WithContext(ctx).Where("member_id = ? AND is_default = 1 AND deleted_at IS NULL", memberID).First(&defaultAddr).Error; err == nil {
			chosenAddress = &CheckoutAddressResponse{
				ID:           defaultAddr.ID,
				Name:         defaultAddr.Name,
				Mobile:       defaultAddr.Mobile,
				ProvinceCode: defaultAddr.ProvinceCode,
				Province:     defaultAddr.Province,
				CityCode:     defaultAddr.CityCode,
				City:         defaultAddr.City,
				DistrictCode: defaultAddr.DistrictCode,
				District:     defaultAddr.District,
				Address:      defaultAddr.Address,
				PostalCode:   defaultAddr.PostalCode,
				IsDefault:    defaultAddr.IsDefault,
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return CheckoutPreviewResponse{}, err
		} else {
			var firstAddr model.MemberAddress
			if err := s.db.WithContext(ctx).Where("member_id = ? AND deleted_at IS NULL", memberID).Order("id ASC").First(&firstAddr).Error; err == nil {
				chosenAddress = &CheckoutAddressResponse{
					ID:           firstAddr.ID,
					Name:         firstAddr.Name,
					Mobile:       firstAddr.Mobile,
					ProvinceCode: firstAddr.ProvinceCode,
					Province:     firstAddr.Province,
					CityCode:     firstAddr.CityCode,
					City:         firstAddr.City,
					DistrictCode: firstAddr.DistrictCode,
					District:     firstAddr.District,
					Address:      firstAddr.Address,
					PostalCode:   firstAddr.PostalCode,
					IsDefault:    firstAddr.IsDefault,
				}
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return CheckoutPreviewResponse{}, err
			}
		}
	}

	// 2. Resolve Items & Calculate (Batch query without N+1)
	type mergedPreviewItem struct {
		skuID int64
		qty   int
	}
	itemMap := make(map[int64]int)
	orderedSKUIDs := make([]int64, 0, len(req.Items))
	for _, it := range req.Items {
		if it.Quantity <= 0 {
			return CheckoutPreviewResponse{}, errors.New("购买数量必须大于0")
		}
		if it.SKUID <= 0 {
			return CheckoutPreviewResponse{}, errors.New("商品规格无效")
		}
		if _, exists := itemMap[it.SKUID]; !exists {
			orderedSKUIDs = append(orderedSKUIDs, it.SKUID)
		}
		itemMap[it.SKUID] += it.Quantity
	}
	mergedItems := make([]mergedPreviewItem, 0, len(orderedSKUIDs))
	for _, skuID := range orderedSKUIDs {
		mergedItems = append(mergedItems, mergedPreviewItem{
			skuID: skuID,
			qty:   itemMap[skuID],
		})
	}

	// Batch query SKUs
	var skus []model.ProductSKU
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", orderedSKUIDs).Find(&skus).Error; err != nil {
		return CheckoutPreviewResponse{}, err
	}
	skuMap := make(map[int64]model.ProductSKU, len(skus))
	for _, sku := range skus {
		skuMap[sku.ID] = sku
	}
	for _, it := range mergedItems {
		sku, ok := skuMap[it.skuID]
		if !ok {
			return CheckoutPreviewResponse{}, fmt.Errorf("商品规格不存在 (ID: %d)", it.skuID)
		}
		if sku.Status != 1 {
			return CheckoutPreviewResponse{}, fmt.Errorf("商品规格已停售 (SKU ID: %d)", sku.ID)
		}
	}

	// Batch query Products
	prodIDMap := make(map[int64]struct{})
	for _, sku := range skus {
		prodIDMap[sku.ProductID] = struct{}{}
	}
	prodIDs := make([]int64, 0, len(prodIDMap))
	for pid := range prodIDMap {
		prodIDs = append(prodIDs, pid)
	}
	var prods []model.Product
	if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", prodIDs).Find(&prods).Error; err != nil {
		return CheckoutPreviewResponse{}, err
	}
	prodMap := make(map[int64]model.Product, len(prods))
	for _, p := range prods {
		prodMap[p.ID] = p
	}
	for _, sku := range skus {
		prod, ok := prodMap[sku.ProductID]
		if !ok {
			return CheckoutPreviewResponse{}, fmt.Errorf("商品不存在 (ID: %d)", sku.ProductID)
		}
		if prod.Status != 1 {
			return CheckoutPreviewResponse{}, fmt.Errorf("商品未上架: %s", prod.Name)
		}
	}

	// Batch query Inventories
	var invs []model.ProductSkuInventory
	if err := s.db.WithContext(ctx).Where("sku_id IN ?", orderedSKUIDs).Find(&invs).Error; err != nil {
		return CheckoutPreviewResponse{}, err
	}
	invMap := make(map[int64]model.ProductSkuInventory, len(invs))
	for _, inv := range invs {
		invMap[inv.SKUID] = inv
	}
	for _, it := range mergedItems {
		inv, ok := invMap[it.skuID]
		sku := skuMap[it.skuID]
		prod := prodMap[sku.ProductID]
		if !ok {
			return CheckoutPreviewResponse{}, fmt.Errorf("商品库存未初始化: %s", prod.Name)
		}
		availStock := inv.Stock - inv.LockedStock
		if availStock < it.qty {
			return CheckoutPreviewResponse{}, fmt.Errorf("商品库存不足: %s (可用: %d, 需要: %d)", prod.Name, availStock, it.qty)
		}
	}

	// Batch collect image IDs
	imageIDs := make([]int64, 0, len(skus)*2)
	for _, sku := range skus {
		if sku.ImageID != nil && *sku.ImageID > 0 {
			imageIDs = append(imageIDs, *sku.ImageID)
		}
		prod := prodMap[sku.ProductID]
		if prod.MainImageID != nil && *prod.MainImageID > 0 {
			imageIDs = append(imageIDs, *prod.MainImageID)
		}
	}
	urlMap := s.resolveImages(ctx, s.db, imageIDs)

	// Query freight templates for products with freight_type = 3
	templateIDs := make([]int64, 0)
	for _, p := range prods {
		if p.FreightType == 3 && p.FreightTemplateID != nil && *p.FreightTemplateID > 0 {
			templateIDs = append(templateIDs, *p.FreightTemplateID)
		}
	}
	tmplMap := make(map[int64]*model.FreightTemplate)
	if len(templateIDs) > 0 {
		var tmpls []model.FreightTemplate
		if err := s.db.WithContext(ctx).Where("id IN ? AND deleted_at IS NULL", templateIDs).Find(&tmpls).Error; err != nil {
			return CheckoutPreviewResponse{}, err
		}
		for i := range tmpls {
			tmplMap[tmpls[i].ID] = &tmpls[i]
		}
	}

	// Calculate freight per distinct product
	type prodFreightInfo struct {
		feeCents    int64
		deliverable bool
		reason      string
	}
	prodFreightMap := make(map[int64]prodFreightInfo, len(prodMap))
	var totalFreightCents int64 = 0
	overallDeliverable := true
	overallReason := ""

	var provinceCode, cityCode string
	if chosenAddress != nil {
		if s.regionSvc != nil {
			resolved, err := s.regionSvc.ValidateAndResolveAddress(
				chosenAddress.ProvinceCode, chosenAddress.Province,
				chosenAddress.CityCode, chosenAddress.City,
				chosenAddress.DistrictCode, chosenAddress.District,
			)
			if err != nil {
				return CheckoutPreviewResponse{}, fmt.Errorf("收货地址行政区划不合规: %w", err)
			}
			provinceCode = resolved.ProvinceCode
			cityCode = resolved.CityCode
		} else {
			provinceCode = chosenAddress.ProvinceCode
			cityCode = chosenAddress.CityCode
		}
	}

	for pid, p := range prodMap {
		if chosenAddress == nil {
			prodFreightMap[pid] = prodFreightInfo{
				feeCents:    0,
				deliverable: true,
				reason:      "",
			}
			continue
		}

		var tmpl *model.FreightTemplate
		if p.FreightTemplateID != nil {
			tmpl = tmplMap[*p.FreightTemplateID]
		}

		feeCents, deliverable, err := productfreight.CalculateProductFreight(&p, tmpl, provinceCode, cityCode)
		if err != nil {
			prodFreightMap[pid] = prodFreightInfo{
				feeCents:    0,
				deliverable: false,
				reason:      err.Error(),
			}
			overallDeliverable = false
			if overallReason == "" {
				overallReason = err.Error()
			}
		} else if !deliverable {
			prodFreightMap[pid] = prodFreightInfo{
				feeCents:    0,
				deliverable: false,
				reason:      "当前收货地区暂不配送",
			}
			overallDeliverable = false
			if overallReason == "" {
				overallReason = "部分商品不支持配送到当前地区"
			}
		} else {
			prodFreightMap[pid] = prodFreightInfo{
				feeCents:    feeCents,
				deliverable: true,
				reason:      "",
			}
			totalFreightCents += feeCents
		}
	}

	totalQuantity := 0
	var totalCents int64 = 0
	previewItems := make([]CheckoutItemResponse, 0, len(mergedItems))

	for _, d := range mergedItems {
		sku := skuMap[d.skuID]
		prod := prodMap[sku.ProductID]
		inv := invMap[d.skuID]

		cents, err := util.ParseDecimalYuanToCent(sku.Price)
		if err != nil {
			return CheckoutPreviewResponse{}, fmt.Errorf("解析商品价格失败: %w", err)
		}
		itemTotalCents := cents * int64(d.qty)
		totalCents += itemTotalCents
		totalQuantity += d.qty

		imgURL := ""
		if sku.ImageID != nil && urlMap[*sku.ImageID] != "" {
			imgURL = urlMap[*sku.ImageID]
		} else if prod.MainImageID != nil && urlMap[*prod.MainImageID] != "" {
			imgURL = urlMap[*prod.MainImageID]
		}

		skuCode := ""
		if sku.SKUCode != nil {
			skuCode = *sku.SKUCode
		}

		pf := prodFreightMap[prod.ID]

		previewItems = append(previewItems, CheckoutItemResponse{
			SKUID:               sku.ID,
			ProductID:           prod.ID,
			ProductName:         prod.Name,
			SKUCode:             skuCode,
			SKUSpecValues:       util.EnrichSKUSpecValues(sku.SpecValues, prod.Specs),
			ImageURL:            imgURL,
			UnitPrice:           sku.Price,
			Quantity:            d.qty,
			TotalAmount:         util.FormatCentToYuan(itemTotalCents),
			Stock:               inv.Stock - inv.LockedStock,
			Deliverable:         pf.deliverable,
			UndeliverableReason: pf.reason,
		})
	}

	totalAmountStr := util.FormatCentToYuan(totalCents)
	freightAmountStr := util.FormatCentToYuan(totalFreightCents)
	payAmountStr := util.FormatCentToYuan(totalCents + totalFreightCents)

	return CheckoutPreviewResponse{
		Address:             chosenAddress,
		Items:               previewItems,
		TotalQuantity:       totalQuantity,
		TotalAmount:         totalAmountStr,
		FreightAmount:       freightAmountStr,
		DiscountAmount:      "0.00",
		PayAmount:           payAmountStr,
		Deliverable:         overallDeliverable,
		UndeliverableReason: overallReason,
	}, nil
}

// Create creates an order with inventory deduction and idempotency support.
func (s *Service) Create(ctx context.Context, member model.Member, req OrderCreateRequest) (OrderDetailResponse, error) {
	reqID := strings.TrimSpace(req.RequestID)
	if reqID == "" {
		return OrderDetailResponse{}, errors.New("request_id 必填")
	}
	if len(reqID) > 64 {
		return OrderDetailResponse{}, errors.New("request_id 长度不能超过64")
	}
	if len(req.Items) == 0 {
		return OrderDetailResponse{}, ErrItemsEmpty
	}
	if req.AddressID <= 0 {
		return OrderDetailResponse{}, ErrAddressMissing
	}

	// 1. Check idempotency outside tx
	var existingOrder model.OrderMain
	if err := s.db.WithContext(ctx).
		Where("member_id = ? AND request_id = ? AND deleted_at IS NULL", member.ID, reqID).
		First(&existingOrder).Error; err == nil {
		return s.Detail(ctx, member.ID, existingOrder.ID)
	}

	// 2. Validate Address
	var addr model.MemberAddress
	if err := s.db.WithContext(ctx).
		Where("id = ? AND member_id = ? AND deleted_at IS NULL", req.AddressID, member.ID).
		First(&addr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return OrderDetailResponse{}, errors.New("收货地址不存在或已删除")
		}
		return OrderDetailResponse{}, err
	}

	// 3. Normalize items and sort by sku_id ASC to prevent deadlock
	// Consolidate duplicate sku_ids if any
	type mergedItem struct {
		skuID int64
		qty   int
	}
	itemMap := make(map[int64]int)
	for _, it := range req.Items {
		if it.SKUID <= 0 || it.Quantity <= 0 {
			return OrderDetailResponse{}, errors.New("商品规格或数量无效")
		}
		itemMap[it.SKUID] += it.Quantity
	}
	sortedItems := make([]mergedItem, 0, len(itemMap))
	for skuID, qty := range itemMap {
		sortedItems = append(sortedItems, mergedItem{skuID: skuID, qty: qty})
	}
	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].skuID < sortedItems[j].skuID
	})

	orderNo, err := s.bizNoGen.Next(bizno.BizPrefixOrder)
	if err != nil {
		return OrderDetailResponse{}, fmt.Errorf("生成订单号失败: %w", err)
	}
	invSvc := productinventory.NewService(s.db)

	var createdOrderID int64
	now := time.Now()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Idempotency check inside transaction
		var inTxExisting model.OrderMain
		if err := tx.Where("member_id = ? AND request_id = ?", member.ID, reqID).First(&inTxExisting).Error; err == nil {
			createdOrderID = inTxExisting.ID
			return nil
		}

		var totalCents int64 = 0
		type preparedItem struct {
			productID   int64
			skuID       int64
			productName string
			skuCode     string
			specValues  string
			imageID     *int64
			unitPrice   string
			quantity    int
			itemTotal   string
		}
		prepared := make([]preparedItem, 0, len(sortedItems))
		imageIDs := make([]int64, 0, len(sortedItems)*2)

		operatorName := strings.TrimSpace(member.Nickname)
		if operatorName == "" && member.Mobile != nil {
			operatorName = *member.Mobile
		}
		if operatorName == "" {
			operatorName = "member"
		}

		for _, item := range sortedItems {
			// Find SKU & Product
			var sku model.ProductSKU
			if err := tx.Where("id = ? AND deleted_at IS NULL", item.skuID).First(&sku).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("商品规格不存在 (SKU ID: %d)", item.skuID)
				}
				return err
			}
			if sku.Status != 1 {
				return fmt.Errorf("商品规格已停售 (SKU ID: %d)", sku.ID)
			}

			var prod model.Product
			if err := tx.Where("id = ? AND deleted_at IS NULL", sku.ProductID).First(&prod).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("商品不存在 (ID: %d)", sku.ProductID)
				}
				return err
			}
			if prod.Status != 1 {
				return fmt.Errorf("商品未上架: %s", prod.Name)
			}

			// Atomic inventory deduction

			if err := invSvc.OrderDeductStockTx(ctx, tx, prod.ID, sku.ID, item.qty, orderNo, member.ID, operatorName); err != nil {
				return err
			}

			cents, err := util.ParseDecimalYuanToCent(sku.Price)
			if err != nil {
				return fmt.Errorf("解析商品单价失败: %w", err)
			}
			itemCents := cents * int64(item.qty)
			totalCents += itemCents

			skuCode := ""
			if sku.SKUCode != nil {
				skuCode = *sku.SKUCode
			}

			var imgID *int64
			if sku.ImageID != nil && *sku.ImageID > 0 {
				imgID = sku.ImageID
				imageIDs = append(imageIDs, *sku.ImageID)
			} else if prod.MainImageID != nil && *prod.MainImageID > 0 {
				imgID = prod.MainImageID
				imageIDs = append(imageIDs, *prod.MainImageID)
			}

			prepared = append(prepared, preparedItem{
				productID:   prod.ID,
				skuID:       sku.ID,
				productName: prod.Name,
				skuCode:     skuCode,
				specValues:  util.EnrichSKUSpecValues(sku.SpecValues, prod.Specs),
				imageID:     imgID,
				unitPrice:   sku.Price,
				quantity:    item.qty,
				itemTotal:   util.FormatCentToYuan(itemCents),
			})
		}

		// Consolidate distinct products for freight calculation
		distinctProds := make(map[int64]model.Product)
		templateIDs := make([]int64, 0)
		for _, p := range prepared {
			if _, exists := distinctProds[p.productID]; !exists {
				var prod model.Product
				if err := tx.Where("id = ? AND deleted_at IS NULL", p.productID).First(&prod).Error; err != nil {
					return err
				}
				distinctProds[p.productID] = prod
				if prod.FreightType == 3 && prod.FreightTemplateID != nil && *prod.FreightTemplateID > 0 {
					templateIDs = append(templateIDs, *prod.FreightTemplateID)
				}
			}
		}

		tmplMap := make(map[int64]*model.FreightTemplate)
		if len(templateIDs) > 0 {
			var tmpls []model.FreightTemplate
			if err := tx.Where("id IN ? AND deleted_at IS NULL", templateIDs).Find(&tmpls).Error; err != nil {
				return err
			}
			for i := range tmpls {
				tmplMap[tmpls[i].ID] = &tmpls[i]
			}
		}

		calcProvinceCode := addr.ProvinceCode
		calcCityCode := addr.CityCode
		if s.regionSvc != nil {
			resolved, err := s.regionSvc.ValidateAndResolveAddress(
				addr.ProvinceCode, addr.Province,
				addr.CityCode, addr.City,
				addr.DistrictCode, addr.District,
			)
			if err != nil {
				return fmt.Errorf("收货地址行政区划不合规: %w", err)
			}
			calcProvinceCode = resolved.ProvinceCode
			calcCityCode = resolved.CityCode
		}

		var totalFreightCents int64 = 0
		for _, prod := range distinctProds {
			var tmpl *model.FreightTemplate
			if prod.FreightTemplateID != nil {
				tmpl = tmplMap[*prod.FreightTemplateID]
			}
			feeCents, deliverable, err := productfreight.CalculateProductFreight(&prod, tmpl, calcProvinceCode, calcCityCode)
			if err != nil {
				return fmt.Errorf("商品 [%s] 运费计算失败: %w", prod.Name, err)
			}
			if !deliverable {
				return fmt.Errorf("商品 [%s] 不支持配送到当前收货地址", prod.Name)
			}
			totalFreightCents += feeCents
		}

		totalAmountStr := util.FormatCentToYuan(totalCents)
		freightAmountStr := util.FormatCentToYuan(totalFreightCents)
		payAmountStr := util.FormatCentToYuan(totalCents + totalFreightCents)

		// Create OrderMain
		orderMain := model.OrderMain{
			OrderNo:        orderNo,
			MemberID:       member.ID,
			RequestID:      reqID,
			Status:         model.OrderStatusPendingShipment, // Direct to 20 since no real payment
			TotalAmount:    totalAmountStr,
			PayAmount:      payAmountStr,
			FreightAmount:  freightAmountStr,
			DiscountAmount: "0.00",
			Remark:         strings.TrimSpace(req.Remark),
			PaidAt:         nil, // 修复：必须为 NULL，无真实支付不能伪造 paid_at
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := tx.Create(&orderMain).Error; err != nil {
			return err
		}
		createdOrderID = orderMain.ID

		// Resolve image URLs
		urlMap := s.resolveImages(ctx, tx, imageIDs)

		// Insert OrderItems
		for _, p := range prepared {
			imgURL := ""
			if p.imageID != nil && urlMap[*p.imageID] != "" {
				imgURL = urlMap[*p.imageID]
			}

			var specVal *string
			if p.specValues != "" {
				specVal = &p.specValues
			}

			orderItem := model.OrderItem{
				OrderID:       orderMain.ID,
				ProductID:     p.productID,
				SKUID:         p.skuID,
				ProductName:   p.productName,
				SKUCode:       p.skuCode,
				SKUSpecValues: specVal,
				SKUImageID:    p.imageID,
				SKUImageURL:   imgURL,
				UnitPrice:     p.unitPrice,
				Quantity:      p.quantity,
				TotalAmount:   p.itemTotal,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if err := tx.Create(&orderItem).Error; err != nil {
				return err
			}
		}

		// Insert OrderAddress (immutable snapshot)
		orderAddr := model.OrderAddress{
			OrderID:      orderMain.ID,
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
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&orderAddr).Error; err != nil {
			return err
		}

		// Insert OrderLog
		orderLog := model.OrderLog{
			OrderID:      orderMain.ID,
			FromStatus:   0,
			ToStatus:     model.OrderStatusPendingShipment,
			Action:       "create",
			OperatorType: "member",
			OperatorID:   member.ID,
			OperatorName: operatorName,
			Remark:       "创建订单",
			CreatedAt:    now,
		}
		return tx.Create(&orderLog).Error
	})

	if err != nil {
		// If duplicate key error occurred due to concurrent request with same request_id
		if isDuplicateKeyError(err) {
			var recheck model.OrderMain
			if qErr := s.db.WithContext(ctx).
				Where("member_id = ? AND request_id = ? AND deleted_at IS NULL", member.ID, reqID).
				First(&recheck).Error; qErr == nil {
				return s.Detail(ctx, member.ID, recheck.ID)
			}
		}
		return OrderDetailResponse{}, err
	}

	return s.Detail(ctx, member.ID, createdOrderID)
}

// List returns a paginated list of orders for the member.
func (s *Service) List(ctx context.Context, memberID int64, req OrderListRequest) ([]OrderListItemResponse, int64, error) {
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

	query := s.db.WithContext(ctx).Model(&model.OrderMain{}).
		Where("member_id = ? AND deleted_at IS NULL", memberID)

	if req.Status != nil && *req.Status > 0 {
		query = query.Where("status = ?", *req.Status)
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
		return []OrderListItemResponse{}, total, nil
	}

	orderIDs := make([]int64, 0, len(orders))
	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
	}

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
	urlMap := s.resolveImages(ctx, s.db, imageIDs)

	prodIDs := make([]int64, 0, len(items))
	for _, it := range items {
		if it.SKUSpecValues != nil && *it.SKUSpecValues != "" && *it.SKUSpecValues != "[]" && !strings.Contains(*it.SKUSpecValues, "value_name") {
			prodIDs = append(prodIDs, it.ProductID)
		}
	}
	prodSpecsMap := s.resolveProductSpecs(ctx, prodIDs)

	itemsByOrder := make(map[int64][]OrderItemResponse)
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
		itemsByOrder[it.OrderID] = append(itemsByOrder[it.OrderID], OrderItemResponse{
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

	result := make([]OrderListItemResponse, 0, len(orders))
	for _, o := range orders {
		orderItems := itemsByOrder[o.ID]
		totalQty := 0
		for _, it := range orderItems {
			totalQty += it.Quantity
		}

		result = append(result, OrderListItemResponse{
			ID:             o.ID,
			OrderNo:        o.OrderNo,
			Status:         o.Status,
			StatusText:     model.OrderStatusText(o.Status),
			TotalAmount:    normalizeAmount(o.TotalAmount),
			PayAmount:      normalizeAmount(o.PayAmount),
			FreightAmount:  normalizeAmount(o.FreightAmount),
			TotalQuantity:  totalQty,
			CreatedAt:      o.CreatedAt,
			ExpressCompany: o.ExpressCompany,
			ExpressNo:      o.ExpressNo,
			Items:          orderItems,
		})
	}

	return result, total, nil
}

// Detail retrieves complete order details for a member.
func (s *Service) Detail(ctx context.Context, memberID int64, orderID int64) (OrderDetailResponse, error) {
	var o model.OrderMain
	if err := s.db.WithContext(ctx).
		Where("id = ? AND member_id = ? AND deleted_at IS NULL", orderID, memberID).
		First(&o).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return OrderDetailResponse{}, ErrNotFound
		}
		return OrderDetailResponse{}, err
	}

	var addr model.OrderAddress
	var addrResp *OrderAddressResponse
	if err := s.db.WithContext(ctx).Where("order_id = ?", o.ID).First(&addr).Error; err == nil {
		addrResp = &OrderAddressResponse{
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
	urlMap := s.resolveImages(ctx, s.db, imageIDs)

	prodIDs := make([]int64, 0, len(items))
	for _, it := range items {
		if it.SKUSpecValues != nil && *it.SKUSpecValues != "" && *it.SKUSpecValues != "[]" && !strings.Contains(*it.SKUSpecValues, "value_name") {
			prodIDs = append(prodIDs, it.ProductID)
		}
	}
	prodSpecsMap := s.resolveProductSpecs(ctx, prodIDs)

	itemResponses := make([]OrderItemResponse, 0, len(items))
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
		itemResponses = append(itemResponses, OrderItemResponse{
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
	logResponses := make([]OrderLogResponse, 0, len(logs))
	for _, l := range logs {
		logResponses = append(logResponses, OrderLogResponse{
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
		Address:        addrResp,
		Items:          itemResponses,
		Logs:           logResponses,
	}, nil
}

// Cancel allows member to cancel an order and restores inventory atomically.
func (s *Service) Cancel(ctx context.Context, member model.Member, orderID int64) error {
	invSvc := productinventory.NewService(s.db)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var o model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND member_id = ? AND deleted_at IS NULL", orderID, member.ID).
			First(&o).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if o.Status != model.OrderStatusPendingShipment && o.Status != model.OrderStatusPendingPayment {
			return fmt.Errorf("订单当前状态(%s)不可取消", model.OrderStatusText(o.Status))
		}

		var items []model.OrderItem
		if err := tx.Where("order_id = ?", o.ID).Find(&items).Error; err != nil {
			return err
		}

		// Sort by sku_id ASC to prevent deadlocks
		sort.Slice(items, func(i, j int) bool {
			return items[i].SKUID < items[j].SKUID
		})

		operatorName := strings.TrimSpace(member.Nickname)
		if operatorName == "" && member.Mobile != nil {
			operatorName = *member.Mobile
		}
		if operatorName == "" {
			operatorName = "member"
		}

		// Restore stock for all items
		for _, it := range items {
			if err := invSvc.OrderRestoreStockTx(ctx, tx, it.ProductID, it.SKUID, it.Quantity, o.OrderNo, member.ID, operatorName); err != nil {
				return err
			}
		}

		now := time.Now()
		res := tx.Model(&model.OrderMain{}).
			Where("id = ? AND status = ?", o.ID, o.Status).
			Updates(map[string]any{
				"status":       model.OrderStatusCancelled,
				"cancelled_at": &now,
				"updated_at":   now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("订单状态已变更，请刷新")
		}

		orderLog := model.OrderLog{
			OrderID:      o.ID,
			FromStatus:   o.Status,
			ToStatus:     model.OrderStatusCancelled,
			Action:       "cancel",
			OperatorType: "member",
			OperatorID:   member.ID,
			OperatorName: operatorName,
			Remark:       "用户取消订单",
			CreatedAt:    now,
		}
		return tx.Create(&orderLog).Error
	})
}

// Confirm allows member to confirm receipt of shipped order.
func (s *Service) Confirm(ctx context.Context, member model.Member, orderID int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var o model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND member_id = ? AND deleted_at IS NULL", orderID, member.ID).
			First(&o).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		if o.Status != model.OrderStatusPendingReceipt {
			return fmt.Errorf("订单当前状态(%s)不可确认收货", model.OrderStatusText(o.Status))
		}

		now := time.Now()
		res := tx.Model(&model.OrderMain{}).
			Where("id = ? AND status = ?", o.ID, o.Status).
			Updates(map[string]any{
				"status":       model.OrderStatusCompleted,
				"completed_at": &now,
				"updated_at":   now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("订单状态已变更，请刷新")
		}

		operatorName := strings.TrimSpace(member.Nickname)
		if operatorName == "" && member.Mobile != nil {
			operatorName = *member.Mobile
		}
		if operatorName == "" {
			operatorName = "member"
		}

		orderLog := model.OrderLog{
			OrderID:      o.ID,
			FromStatus:   model.OrderStatusPendingReceipt,
			ToStatus:     model.OrderStatusCompleted,
			Action:       "confirm",
			OperatorType: "member",
			OperatorID:   member.ID,
			OperatorName: operatorName,
			Remark:       "用户确认收货",
			CreatedAt:    now,
		}
		return tx.Create(&orderLog).Error
	})
}

func (s *Service) resolveImages(ctx context.Context, db *gorm.DB, imageIDs []int64) map[int64]string {
	result := make(map[int64]string)
	if len(imageIDs) == 0 {
		return result
	}
	if db == nil {
		db = s.db
	}
	fileSvc := systemfile.NewService(db)
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

