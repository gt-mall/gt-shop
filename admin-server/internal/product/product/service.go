package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mall/admin_core/internal/model"
	productinventory "mall/admin_core/internal/product/inventory"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"
	"mall/admin_core/pkg/util"
)

var (
	ErrDatabaseUnavailable   = errors.New("数据库不可用")
	ErrProductNotFound       = errors.New("商品不存在")
	ErrInvalidProductName    = errors.New("商品名称不能为空且最多255个字符")
	ErrInvalidSubtitle       = errors.New("商品副标题最多255个字符")
	ErrInvalidStatus         = errors.New("商品状态无效(仅支持0下架/1上架)")
	ErrInvalidVirtualSales   = errors.New("虚拟销量不能为负数")
	ErrInvalidWeight         = errors.New("重量不能为负数")
	ErrTooManyGalleryImages  = errors.New("商品轮播图最多支持10张")
	ErrImageNotFound         = errors.New("指定的图片素材不存在或已被删除")
	ErrCategoryNotFound      = errors.New("商品分类不存在或已被删除")
	ErrCategoryDisabled      = errors.New("上架商品所属分类必须启用")
	ErrMainImageRequired     = errors.New("上架商品必须设置有效的主图")
	ErrEnabledSKURequired    = errors.New("上架商品必须包含至少一个启用的SKU")
	ErrEnabledSKUZeroPrice   = errors.New("上架商品所有启用的SKU价格必须大于0")
	ErrSKUNotBelongToProduct = errors.New("SKU不存在或不属于该商品")
	ErrDuplicateSKUID        = errors.New("提交了重复的SKU ID")
	ErrSKUIdentityChanged    = errors.New("已有SKU规格身份不可更改")
	ErrMissingExistingSKU    = errors.New("编辑商品必须提交全部已有SKU")
	ErrInventoryNotInitialized = errors.New("SKU库存未初始化")
	ErrProductHasStock         = errors.New("商品存在未清零的库存，禁止删除")
	ErrProductHasUnfinishedOrders = errors.New("该商品存在未完成订单（待支付/待发货/待收货），禁止删除")
	ErrInvalidParamName        = errors.New("商品参数名不能为空且最多100个字符")
	ErrInvalidParamValue       = errors.New("商品参数值最多500个字符")
	ErrTooManyParams           = errors.New("商品参数最多支持100个")
)

func ValidateParams(params []ProductParam) ([]ProductParam, error) {
	if len(params) == 0 {
		return nil, nil
	}
	if len(params) > 100 {
		return nil, ErrTooManyParams
	}
	cleaned := make([]ProductParam, 0, len(params))
	for _, p := range params {
		name := strings.TrimSpace(p.Name)
		val := strings.TrimSpace(p.Value)
		if name == "" && val == "" {
			continue
		}
		if name == "" || utf8.RuneCountInString(name) > 100 {
			return nil, ErrInvalidParamName
		}
		if utf8.RuneCountInString(val) > 500 {
			return nil, ErrInvalidParamValue
		}
		cleaned = append(cleaned, ProductParam{
			Name:  name,
			Value: val,
		})
	}
	return cleaned, nil
}

type Service struct {
	db    *gorm.DB
	cache cache.Cache
}

func NewService(db *gorm.DB, caches ...cache.Cache) *Service {
	var c cache.Cache = cache.NewNoop()
	if len(caches) > 0 && caches[0] != nil {
		c = caches[0]
	}
	return &Service{db: db, cache: c}
}

func (s *Service) invalidateProductCache(productID int64, op string) {
	if s.cache == nil || s.cache.IsDisabled() {
		return
	}
	bgCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if _, err := cache.BumpProductDetailGeneration(bgCtx, s.cache, productID); err != nil {
		log.Printf("[Cache] product cache invalidation failed product_id=%d operation=%s err=%v", productID, op, err)
	}
	if _, err := cache.BumpProductListGeneration(bgCtx, s.cache); err != nil {
		log.Printf("[Cache] product list cache invalidation failed product_id=%d operation=%s err=%v", productID, op, err)
	}
}

type processedSKU struct {
	ID          int64
	SpecValues  string
	SpecHash    string
	SKUCode     *string
	Barcode     *string
	ImageID     *int64
	Price       string
	MarketPrice string
	CostPrice   string
	Unit        *string
	WeightGram  *int
	Sort        int
	Status      int8
	IsExisting  bool
	IsRestore   bool
}

func (s *Service) Create(ctx context.Context, req ProductCreateRequest) (ProductIDResponse, error) {
	if s.db == nil {
		return ProductIDResponse{}, ErrDatabaseUnavailable
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return ProductIDResponse{}, ErrInvalidProductName
	}
	subtitle := strings.TrimSpace(req.Subtitle)
	if utf8.RuneCountInString(subtitle) > 255 {
		return ProductIDResponse{}, ErrInvalidSubtitle
	}

	if err := ValidateSpecs(req.Specs); err != nil {
		return ProductIDResponse{}, err
	}

	validParams, err := ValidateParams(req.Params)
	if err != nil {
		return ProductIDResponse{}, err
	}

	if len(req.SKUs) == 0 {
		return ProductIDResponse{}, ErrEmptySKUs
	}
	if len(req.SKUs) > 200 {
		return ProductIDResponse{}, ErrTooManySKUs
	}
	if len(req.Specs) == 0 && len(req.SKUs) != 1 {
		return ProductIDResponse{}, ErrDefaultSKURequired
	}

	targetStatus := int8(0)
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			return ProductIDResponse{}, ErrInvalidStatus
		}
		targetStatus = *req.Status
	}

	sortVal := 0
	if req.Sort != nil {
		sortVal = *req.Sort
	}
	virtualSales := 0
	if req.VirtualSales != nil {
		if *req.VirtualSales < 0 {
			return ProductIDResponse{}, ErrInvalidVirtualSales
		}
		virtualSales = *req.VirtualSales
	}

	// Validate freight
	targetFreightType, normFreightFee, freightTmplID, err := validateProductFreight(ctx, s.db, req.FreightType, req.FreightFee, req.FreightTemplateID, targetStatus, nil)
	if err != nil {
		return ProductIDResponse{}, err
	}

	// Validate category
	var cat model.ProductCategory
	if req.CategoryID > 0 {
		if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.CategoryID).First(&cat).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ProductIDResponse{}, ErrCategoryNotFound
			}
			return ProductIDResponse{}, err
		}
	} else {
		req.CategoryID = 0
	}

	// Deduplicate gallery image IDs
	galleryIDs := deduplicatePositiveIDs(req.GalleryFileIDs)
	if len(galleryIDs) > 10 {
		return ProductIDResponse{}, ErrTooManyGalleryImages
	}
	for _, gid := range galleryIDs {
		if err := validateImage(s.db.WithContext(ctx), gid); err != nil {
			return ProductIDResponse{}, err
		}
	}

	// Validate main image
	var mainImageID *int64
	if req.MainImageID != nil && *req.MainImageID > 0 {
		if err := validateImage(s.db.WithContext(ctx), *req.MainImageID); err != nil {
			return ProductIDResponse{}, err
		}
		id := *req.MainImageID
		mainImageID = &id
	}

	// Process SKUs
	processed, err := s.processSKUsForCreate(ctx, req.Specs, req.SKUs)
	if err != nil {
		return ProductIDResponse{}, err
	}

	// Online validation if targetStatus == 1
	if targetStatus == 1 {
		if req.CategoryID == 0 || cat.ID == 0 {
			return ProductIDResponse{}, ErrCategoryNotFound
		}
		if cat.Status != 1 {
			return ProductIDResponse{}, ErrCategoryDisabled
		}
		if mainImageID == nil {
			return ProductIDResponse{}, ErrMainImageRequired
		}
		hasEnabled := false
		for _, sku := range processed {
			if sku.Status == 1 {
				hasEnabled = true
				cents, _ := util.ParseDecimalYuanToCent(sku.Price)
				if cents <= 0 {
					return ProductIDResponse{}, ErrEnabledSKUZeroPrice
				}
			}
		}
		if !hasEnabled {
			return ProductIDResponse{}, ErrEnabledSKURequired
		}
	}

	detailHTML := util.CleanHTML(req.Detail)

	specsJSON, err := json.Marshal(req.Specs)
	if err != nil {
		return ProductIDResponse{}, err
	}
	specsStr := string(specsJSON)

	var paramsStr *string
	if len(validParams) > 0 {
		paramsJSON, err := json.Marshal(validParams)
		if err != nil {
			return ProductIDResponse{}, err
		}
		s := string(paramsJSON)
		paramsStr = &s
	}

	galleryJSON, err := json.Marshal(galleryIDs)
	if err != nil {
		return ProductIDResponse{}, err
	}
	galleryStr := string(galleryJSON)

	var createdID int64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p := model.Product{
			CategoryID:     req.CategoryID,
			Name:           name,
			Subtitle:       subtitle,
			MainImageID:    mainImageID,
			GalleryFileIDs: &galleryStr,
			Specs:          &specsStr,
			Params:         paramsStr,
			Detail:            &detailHTML,
			FreightType:       targetFreightType,
			FreightFee:        normFreightFee,
			FreightTemplateID: freightTmplID,
			Status:            targetStatus,
			Sort:              sortVal,
			SalesCount:        0,
			VirtualSales:      virtualSales,
		}
		if err := tx.Create(&p).Error; err != nil {
			return fmt.Errorf("create product: %w", err)
		}
		createdID = p.ID

		for _, sku := range processed {
			skuModel := model.ProductSKU{
				ProductID:   p.ID,
				SpecValues:  sku.SpecValues,
				SpecHash:    sku.SpecHash,
				SKUCode:     sku.SKUCode,
				Barcode:     sku.Barcode,
				ImageID:     sku.ImageID,
				Price:       sku.Price,
				MarketPrice: sku.MarketPrice,
				CostPrice:   sku.CostPrice,
				Unit:        sku.Unit,
				WeightGram:  sku.WeightGram,
				Sort:        sku.Sort,
				Status:      sku.Status,
			}
			if err := tx.Create(&skuModel).Error; err != nil {
				return fmt.Errorf("create sku: %w", err)
			}
			if err := productinventory.NewService(s.db).InitZeroInventoryTx(ctx, tx, p.ID, skuModel.ID); err != nil {
				return fmt.Errorf("init sku inventory: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		return ProductIDResponse{}, err
	}

	s.invalidateProductCache(createdID, "create")

	return ProductIDResponse{ID: createdID}, nil
}

func (s *Service) Update(ctx context.Context, req ProductUpdateRequest) (ProductIDResponse, error) {
	if s.db == nil {
		return ProductIDResponse{}, ErrDatabaseUnavailable
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 255 {
		return ProductIDResponse{}, ErrInvalidProductName
	}
	subtitle := strings.TrimSpace(req.Subtitle)
	if utf8.RuneCountInString(subtitle) > 255 {
		return ProductIDResponse{}, ErrInvalidSubtitle
	}

	if err := ValidateSpecs(req.Specs); err != nil {
		return ProductIDResponse{}, err
	}

	validParams, err := ValidateParams(req.Params)
	if err != nil {
		return ProductIDResponse{}, err
	}

	if len(req.SKUs) == 0 {
		return ProductIDResponse{}, ErrEmptySKUs
	}
	if len(req.SKUs) > 200 {
		return ProductIDResponse{}, ErrTooManySKUs
	}
	if len(req.Specs) == 0 && len(req.SKUs) != 1 {
		return ProductIDResponse{}, ErrDefaultSKURequired
	}

	var updatedID int64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p model.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", req.ID).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProductNotFound
			}
			return err
		}

		targetStatus := p.Status
		if req.Status != nil {
			if *req.Status != 0 && *req.Status != 1 {
				return ErrInvalidStatus
			}
			targetStatus = *req.Status
		}

		sortVal := p.Sort
		if req.Sort != nil {
			sortVal = *req.Sort
		}
		virtualSales := p.VirtualSales
		if req.VirtualSales != nil {
			if *req.VirtualSales < 0 {
				return ErrInvalidVirtualSales
			}
			virtualSales = *req.VirtualSales
		}

		// Category check
		var cat model.ProductCategory
		if req.CategoryID > 0 {
			if err := tx.Where("id = ? AND deleted_at IS NULL", req.CategoryID).First(&cat).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrCategoryNotFound
				}
				return err
			}
		} else {
			req.CategoryID = 0
		}

		// Gallery IDs
		galleryIDs := deduplicatePositiveIDs(req.GalleryFileIDs)
		if len(galleryIDs) > 10 {
			return ErrTooManyGalleryImages
		}
		var oldGalleryIDs []int64
		if p.GalleryFileIDs != nil && *p.GalleryFileIDs != "" {
			_ = json.Unmarshal([]byte(*p.GalleryFileIDs), &oldGalleryIDs)
		}
		oldGallerySet := make(map[int64]bool, len(oldGalleryIDs))
		for _, id := range oldGalleryIDs {
			oldGallerySet[id] = true
		}
		for _, gid := range galleryIDs {
			if !oldGallerySet[gid] {
				if err := validateImage(tx, gid); err != nil {
					return err
				}
			}
		}

		// Main Image
		var mainImageID *int64
		if req.MainImageID != nil && *req.MainImageID > 0 {
			newID := *req.MainImageID
			if p.MainImageID == nil || *p.MainImageID != newID {
				if err := validateImage(tx, newID); err != nil {
					return err
				}
			}
			mainImageID = &newID
		}

		// Load all existing active SKUs for this product
		var existingSKUs []model.ProductSKU
		if err := tx.Where("product_id = ? AND deleted_at IS NULL", p.ID).Find(&existingSKUs).Error; err != nil {
			return err
		}
		existingByID := make(map[int64]model.ProductSKU, len(existingSKUs))
		for _, s := range existingSKUs {
			existingByID[s.ID] = s
		}

		// Process SKUs
		processed, err := s.processSKUsForUpdate(ctx, tx, p.ID, req.Specs, req.SKUs, existingByID)
		if err != nil {
			return err
		}

		// Online validation if targetStatus == 1
		if targetStatus == 1 {
			if req.CategoryID == 0 || cat.ID == 0 {
				return ErrCategoryNotFound
			}
			if cat.Status != 1 {
				return ErrCategoryDisabled
			}
			if mainImageID == nil {
				return ErrMainImageRequired
			}
			hasEnabled := false
			for _, sku := range processed {
				if sku.Status == 1 {
					hasEnabled = true
					cents, _ := util.ParseDecimalYuanToCent(sku.Price)
					if cents <= 0 {
						return ErrEnabledSKUZeroPrice
					}
				}
			}
			if !hasEnabled {
				return ErrEnabledSKURequired
			}
		}

		detailHTML := util.CleanHTML(req.Detail)

		specsJSON, err := json.Marshal(req.Specs)
		if err != nil {
			return err
		}
		specsStr := string(specsJSON)

		galleryJSON, err := json.Marshal(galleryIDs)
		if err != nil {
			return err
		}
		galleryStr := string(galleryJSON)

		// Save SKUs
		for _, sku := range processed {
			if sku.IsExisting {
				// Update existing SKU
				updates := map[string]any{
					"sku_code":     sku.SKUCode,
					"barcode":      sku.Barcode,
					"image_id":     sku.ImageID,
					"price":        sku.Price,
					"market_price": sku.MarketPrice,
					"cost_price":   sku.CostPrice,
					"unit":         sku.Unit,
					"weight_gram":  sku.WeightGram,
					"sort":         sku.Sort,
					"status":       sku.Status,
				}
				if err := tx.Model(&model.ProductSKU{}).Where("id = ?", sku.ID).Updates(updates).Error; err != nil {
					return fmt.Errorf("update sku %d: %w", sku.ID, err)
				}
			} else if sku.IsRestore {
				// Restore soft-deleted SKU: verify existing inventory exists
				var existingInv model.ProductSkuInventory
				if err := tx.Where("sku_id = ?", sku.ID).First(&existingInv).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return fmt.Errorf("sku %d: %w", sku.ID, ErrInventoryNotInitialized)
					}
					return fmt.Errorf("check sku %d inventory: %w", sku.ID, err)
				}
				updates := map[string]any{
					"spec_values":  sku.SpecValues,
					"sku_code":     sku.SKUCode,
					"barcode":      sku.Barcode,
					"image_id":     sku.ImageID,
					"price":        sku.Price,
					"market_price": sku.MarketPrice,
					"cost_price":   sku.CostPrice,
					"unit":         sku.Unit,
					"weight_gram":  sku.WeightGram,
					"sort":         sku.Sort,
					"status":       sku.Status,
					"deleted_at":   nil,
				}
				if err := tx.Unscoped().Model(&model.ProductSKU{}).Where("id = ?", sku.ID).Updates(updates).Error; err != nil {
					return fmt.Errorf("restore sku %d: %w", sku.ID, err)
				}
			} else {
				// Insert new SKU
				newSKU := model.ProductSKU{
					ProductID:   p.ID,
					SpecValues:  sku.SpecValues,
					SpecHash:    sku.SpecHash,
					SKUCode:     sku.SKUCode,
					Barcode:     sku.Barcode,
					ImageID:     sku.ImageID,
					Price:       sku.Price,
					MarketPrice: sku.MarketPrice,
					CostPrice:   sku.CostPrice,
					Unit:        sku.Unit,
					WeightGram:  sku.WeightGram,
					Sort:        sku.Sort,
					Status:      sku.Status,
				}
				if err := tx.Create(&newSKU).Error; err != nil {
					return fmt.Errorf("insert new sku: %w", err)
				}
				if err := productinventory.NewService(s.db).InitZeroInventoryTx(ctx, tx, p.ID, newSKU.ID); err != nil {
					return fmt.Errorf("init new sku inventory: %w", err)
				}
			}
		}

		var paramsVal any = nil
		if len(validParams) > 0 {
			paramsJSON, err := json.Marshal(validParams)
			if err != nil {
				return err
			}
			paramsVal = string(paramsJSON)
		}

		// Validate freight
		targetFreightType, normFreightFee, freightTmplID, err := validateProductFreight(ctx, tx, req.FreightType, req.FreightFee, req.FreightTemplateID, targetStatus, p.FreightTemplateID)
		if err != nil {
			return err
		}

		// Update product fields (NEVER touch sales_count)
		pUpdates := map[string]any{
			"category_id":         req.CategoryID,
			"name":                name,
			"subtitle":            subtitle,
			"main_image_id":       mainImageID,
			"gallery_file_ids":    galleryStr,
			"specs":               specsStr,
			"params":              paramsVal,
			"detail":              detailHTML,
			"status":              targetStatus,
			"sort":                sortVal,
			"virtual_sales":       virtualSales,
			"freight_type":        targetFreightType,
			"freight_fee":         normFreightFee,
			"freight_template_id": freightTmplID,
		}
		if err := tx.Model(&model.Product{}).Where("id = ?", p.ID).Updates(pUpdates).Error; err != nil {
			return fmt.Errorf("update product: %w", err)
		}

		updatedID = p.ID
		return nil
	})

	if err != nil {
		return ProductIDResponse{}, err
	}

	s.invalidateProductCache(updatedID, "update")

	return ProductIDResponse{ID: updatedID}, nil
}

func (s *Service) Status(ctx context.Context, req ProductStatusRequest) error {
	if s.db == nil {
		return ErrDatabaseUnavailable
	}
	if req.Status != 0 && req.Status != 1 {
		return ErrInvalidStatus
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p model.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", req.ID).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProductNotFound
			}
			return err
		}

		if req.Status == 1 {
			// Category check
			if p.CategoryID <= 0 {
				return ErrCategoryNotFound
			}
			var cat model.ProductCategory
			if err := tx.Where("id = ? AND deleted_at IS NULL", p.CategoryID).First(&cat).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrCategoryNotFound
				}
				return err
			}
			if cat.Status != 1 {
				return ErrCategoryDisabled
			}

			// Main Image check
			if p.MainImageID == nil || *p.MainImageID <= 0 {
				return ErrMainImageRequired
			}
			if err := validateImage(tx, *p.MainImageID); err != nil {
				return err
			}

			// Active SKUs check
			var skus []model.ProductSKU
			if err := tx.Where("product_id = ? AND deleted_at IS NULL", p.ID).Find(&skus).Error; err != nil {
				return err
			}
			hasEnabled := false
			for _, sku := range skus {
				if sku.Status == 1 {
					hasEnabled = true
					cents, _ := util.ParseDecimalYuanToCent(sku.Price)
					if cents <= 0 {
						return ErrEnabledSKUZeroPrice
					}
				}
			}
			if !hasEnabled {
				return ErrEnabledSKURequired
			}

			// Freight template check
			if p.FreightType == 3 {
				if p.FreightTemplateID == nil || *p.FreightTemplateID <= 0 {
					return errors.New("商品运费模板未配置")
				}
				var tmpl model.FreightTemplate
				if err := tx.Where("id = ? AND deleted_at IS NULL", *p.FreightTemplateID).First(&tmpl).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return errors.New("商品所选运费模板不存在")
					}
					return err
				}
				if tmpl.Status != 1 {
					return errors.New("商品所选运费模板已停用，无法上架")
				}
			}
		}

		return tx.Model(&model.Product{}).Where("id = ?", p.ID).Update("status", req.Status).Error
	})
	if err != nil {
		return err
	}

	s.invalidateProductCache(req.ID, "status")
	return nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if s.db == nil {
		return ErrDatabaseUnavailable
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Lock product
		var p model.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", id).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProductNotFound
			}
			return err
		}

		// 2. Lock all SKUs (including soft-deleted) in ID ascending order
		var allSKUs []model.ProductSKU
		if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("product_id = ?", id).Order("id ASC").Find(&allSKUs).Error; err != nil {
			return err
		}

		// 3. Lock all inventories for this product in sku_id ascending order
		var allInventories []model.ProductSkuInventory
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("product_id = ?", id).Order("sku_id ASC").Find(&allInventories).Error; err != nil {
			return err
		}

		// 4. Check unfinished orders (status 10, 20, 30)
		var unfinishedCount int64
		if err := tx.Table("order_item").
			Joins("JOIN order_main ON order_main.id = order_item.order_id").
			Where("order_item.product_id = ? AND order_main.deleted_at IS NULL AND order_main.status IN (?, ?, ?)",
				id, model.OrderStatusPendingPayment, model.OrderStatusPendingShipment, model.OrderStatusPendingReceipt).
			Count(&unfinishedCount).Error; err != nil {
			return err
		}
		if unfinishedCount > 0 {
			return ErrProductHasUnfinishedOrders
		}

		// 5. Check inventory integrity: no missing inventory, no mismatch, stock and locked_stock must be 0
		invMap := make(map[int64]model.ProductSkuInventory, len(allInventories))
		for _, inv := range allInventories {
			if inv.ProductID != id {
				return fmt.Errorf("库存商品归属异常: inventory_id=%d product_id=%d expected=%d", inv.ID, inv.ProductID, id)
			}
			invMap[inv.SKUID] = inv
		}

		for _, sku := range allSKUs {
			inv, ok := invMap[sku.ID]
			if !ok {
				return fmt.Errorf("sku_id=%d %w", sku.ID, ErrInventoryNotInitialized)
			}
			if inv.Stock != 0 || inv.LockedStock != 0 {
				return ErrProductHasStock
			}
		}

		// 6. Soft-delete product and active SKUs (keep inventory records and logs intact)
		if err := tx.Delete(&p).Error; err != nil {
			return err
		}
		return tx.Where("product_id = ?", id).Delete(&model.ProductSKU{}).Error
	})
	if err != nil {
		return err
	}

	s.invalidateProductCache(id, "delete")
	return nil
}

func (s *Service) Detail(ctx context.Context, id int64) (ProductDetail, error) {

	if s.db == nil {
		return ProductDetail{}, ErrDatabaseUnavailable
	}

	var p model.Product
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductDetail{}, ErrProductNotFound
		}
		return ProductDetail{}, err
	}

	categoryName := ""
	if p.CategoryID > 0 {
		var cat model.ProductCategory
		if err := s.db.WithContext(ctx).Select("name").Where("id = ?", p.CategoryID).First(&cat).Error; err == nil {
			categoryName = cat.Name
		}
	}

	specs := make([]ProductSpec, 0)
	if p.Specs != nil && *p.Specs != "" {
		_ = json.Unmarshal([]byte(*p.Specs), &specs)
	}

	params := make([]ProductParam, 0)
	if p.Params != nil && *p.Params != "" {
		_ = json.Unmarshal([]byte(*p.Params), &params)
	}

	galleryIDs := make([]int64, 0)
	if p.GalleryFileIDs != nil && *p.GalleryFileIDs != "" {
		_ = json.Unmarshal([]byte(*p.GalleryFileIDs), &galleryIDs)
	}

	var skuModels []model.ProductSKU
	if err := s.db.WithContext(ctx).Where("product_id = ? AND deleted_at IS NULL", p.ID).
		Order("sort ASC, id ASC").Find(&skuModels).Error; err != nil {
		return ProductDetail{}, err
	}

	// Resolve images: collect mainImageID, galleryIDs, SKU imageIDs
	imageIDs := make([]int64, 0, 1+len(galleryIDs)+len(skuModels))
	if p.MainImageID != nil && *p.MainImageID > 0 {
		imageIDs = append(imageIDs, *p.MainImageID)
	}
	imageIDs = append(imageIDs, galleryIDs...)
	for _, sm := range skuModels {
		if sm.ImageID != nil && *sm.ImageID > 0 {
			imageIDs = append(imageIDs, *sm.ImageID)
		}
	}

	urlMap := make(map[int64]string)
	if len(imageIDs) > 0 {
		resolved, err := systemfile.NewService(s.db).Resolve(ctx, systemfile.IDsRequest{IDs: imageIDs})
		if err == nil {
			for _, file := range resolved {
				urlMap[file.ID] = file.URL
			}
		}
	}

	var mainImageURL string
	if p.MainImageID != nil {
		mainImageURL = urlMap[*p.MainImageID]
	}

	galleryURLs := make([]string, 0, len(galleryIDs))
	for _, gid := range galleryIDs {
		galleryURLs = append(galleryURLs, urlMap[gid])
	}

	detailText := ""
	if p.Detail != nil {
		detailText = *p.Detail
	}

	skus := make([]ProductSKU, 0, len(skuModels))
	for _, sm := range skuModels {
		specValues := make([]ProductSKUValue, 0)
		if sm.SpecValues != "" {
			_ = json.Unmarshal([]byte(sm.SpecValues), &specValues)
		}
		var skuImageURL string
		if sm.ImageID != nil {
			skuImageURL = urlMap[*sm.ImageID]
		}
		price, _ := util.NormalizeDecimalYuan(sm.Price)
		marketPrice, _ := util.NormalizeDecimalYuan(sm.MarketPrice)
		costPrice, _ := util.NormalizeDecimalYuan(sm.CostPrice)

		skus = append(skus, ProductSKU{
			ID:          sm.ID,
			ProductID:   sm.ProductID,
			SpecValues:  specValues,
			SKUCode:     sm.SKUCode,
			Barcode:     sm.Barcode,
			ImageID:     sm.ImageID,
			ImageURL:    skuImageURL,
			Price:       price,
			MarketPrice: marketPrice,
			CostPrice:   costPrice,
			Unit:        sm.Unit,
			WeightGram:  sm.WeightGram,
			Sort:        sm.Sort,
			Status:      sm.Status,
			CreatedAt:   sm.CreatedAt,
			UpdatedAt:   sm.UpdatedAt,
		})
	}

	var tmplName *string
	if p.FreightTemplateID != nil && *p.FreightTemplateID > 0 {
		var tmpl model.FreightTemplate
		if err := s.db.WithContext(ctx).Select("name").Where("id = ?", *p.FreightTemplateID).First(&tmpl).Error; err == nil {
			tmplName = &tmpl.Name
		}
	}

	detail := ProductDetail{
		ID:                  p.ID,
		CategoryID:          p.CategoryID,
		CategoryName:        categoryName,
		Name:                p.Name,
		Subtitle:            p.Subtitle,
		MainImageID:         p.MainImageID,
		MainImageURL:        mainImageURL,
		GalleryFileIDs:      galleryIDs,
		GalleryImageURLs:    galleryURLs,
		Specs:               specs,
		Params:              params,
		Detail:              detailText,
		Status:              p.Status,
		Sort:                p.Sort,
		SalesCount:          p.SalesCount,
		VirtualSales:        p.VirtualSales,
		FreightType:         p.FreightType,
		FreightFee:          p.FreightFee,
		FreightTemplateID:   p.FreightTemplateID,
		FreightTemplateName: tmplName,
		SKUs:                skus,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}

	return detail, nil
}

func (s *Service) List(ctx context.Context, req ProductListRequest) ([]ProductRow, int64, error) {
	if s.db == nil {
		return nil, 0, ErrDatabaseUnavailable
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

	if s.db == nil {
		return nil, 0, ErrDatabaseUnavailable
	}

	q := s.db.WithContext(ctx).Model(&model.Product{}).Where("deleted_at IS NULL")

	// Category filter
	if req.CategoryID != nil {
		if *req.CategoryID == 0 {
			q = q.Where("category_id = 0")
		} else if *req.CategoryID > 0 {
			catIDs, err := s.getCategoryAndDescendantIDs(ctx, *req.CategoryID)
			if err != nil {
				return nil, 0, err
			}
			q = q.Where("category_id IN ?", catIDs)
		}
	}

	// Status filter
	if req.Status != nil {
		q = q.Where("status = ?", *req.Status)
	}

	// Keyword search
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		var skuProductIDs []int64
		_ = s.db.WithContext(ctx).Model(&model.ProductSKU{}).
			Where("deleted_at IS NULL AND (sku_code LIKE ? OR barcode LIKE ?)", "%"+keyword+"%", "%"+keyword+"%").
			Pluck("product_id", &skuProductIDs).Error

		if len(skuProductIDs) > 0 {
			q = q.Where("(name LIKE ? OR subtitle LIKE ? OR id IN ?)", "%"+keyword+"%", "%"+keyword+"%", skuProductIDs)
		} else {
			q = q.Where("(name LIKE ? OR subtitle LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []ProductRow{}, 0, nil
	}

	var products []model.Product
	if err := q.Order("sort ASC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&products).Error; err != nil {
		return nil, 0, err
	}

	productIDs := make([]int64, 0, len(products))
	categoryIDs := make([]int64, 0, len(products))
	mainImageIDs := make([]int64, 0, len(products))
	for _, p := range products {
		productIDs = append(productIDs, p.ID)
		if p.CategoryID > 0 {
			categoryIDs = append(categoryIDs, p.CategoryID)
		}
		if p.MainImageID != nil && *p.MainImageID > 0 {
			mainImageIDs = append(mainImageIDs, *p.MainImageID)
		}
	}

	// SKU Stats (group by product_id)
	type skuStat struct {
		ProductID int64   `gorm:"column:product_id"`
		SKUCount  int     `gorm:"column:sku_count"`
		MinPrice  *string `gorm:"column:min_price"`
		MaxPrice  *string `gorm:"column:max_price"`
	}
	var stats []skuStat
	_ = s.db.WithContext(ctx).Model(&model.ProductSKU{}).
		Select("product_id, COUNT(1) AS sku_count, MIN(CASE WHEN status = 1 THEN price END) AS min_price, MAX(CASE WHEN status = 1 THEN price END) AS max_price").
		Where("product_id IN ? AND deleted_at IS NULL", productIDs).
		Group("product_id").
		Scan(&stats).Error

	statMap := make(map[int64]skuStat, len(stats))
	for _, st := range stats {
		statMap[st.ProductID] = st
	}

	// Category names
	catMap := make(map[int64]string)
	if len(categoryIDs) > 0 {
		var cats []model.ProductCategory
		_ = s.db.WithContext(ctx).Select("id, name").Where("id IN ?", categoryIDs).Find(&cats).Error
		for _, c := range cats {
			catMap[c.ID] = c.Name
		}
	}

	// Main Image URLs
	imageURLMap := make(map[int64]string)
	if len(mainImageIDs) > 0 {
		resolved, err := systemfile.NewService(s.db).Resolve(ctx, systemfile.IDsRequest{IDs: mainImageIDs})
		if err == nil {
			for _, f := range resolved {
				imageURLMap[f.ID] = f.URL
			}
		}
	}

	rows := make([]ProductRow, 0, len(products))
	for _, p := range products {
		st := statMap[p.ID]
		var mainURL string
		if p.MainImageID != nil {
			mainURL = imageURLMap[*p.MainImageID]
		}

		var minP, maxP *string
		if st.MinPrice != nil {
			val, err := util.NormalizeDecimalYuan(*st.MinPrice)
			if err == nil {
				minP = &val
			}
		}
		if st.MaxPrice != nil {
			val, err := util.NormalizeDecimalYuan(*st.MaxPrice)
			if err == nil {
				maxP = &val
			}
		}

		rows = append(rows, ProductRow{
			ID:           p.ID,
			CategoryID:   p.CategoryID,
			CategoryName: catMap[p.CategoryID],
			Name:         p.Name,
			Subtitle:     p.Subtitle,
			MainImageID:  p.MainImageID,
			MainImageURL: mainURL,
			Status:       p.Status,
			Sort:         p.Sort,
			SalesCount:   p.SalesCount,
			VirtualSales: p.VirtualSales,
			SKUCount:          st.SKUCount,
			MinPrice:          minP,
			MaxPrice:          maxP,
			FreightType:       p.FreightType,
			FreightFee:        p.FreightFee,
			FreightTemplateID: p.FreightTemplateID,
			CreatedAt:         p.CreatedAt,
			UpdatedAt:         p.UpdatedAt,
		})
	}

	return rows, total, nil
}

func (s *Service) processSKUsForCreate(ctx context.Context, specs []ProductSpec, skus []ProductSKURequest) ([]processedSKU, error) {
	seenHashes := make(map[string]bool, len(skus))
	seenCodes := make(map[string]bool, len(skus))
	result := make([]processedSKU, 0, len(skus))

	for i, reqSKU := range skus {
		hash, err := CalculateSKUSpecHash(specs, reqSKU.SpecValues)
		if err != nil {
			return nil, fmt.Errorf("SKU #%d: %w", i+1, err)
		}
		if seenHashes[hash] {
			return nil, ErrDuplicateSKUCombo
		}
		seenHashes[hash] = true

		code, err := s.normalizeSKUCode(ctx, nil, reqSKU.SKUCode, seenCodes, 0)
		if err != nil {
			return nil, err
		}

		barcode := normalizeOptionalString(reqSKU.Barcode)
		unit := normalizeOptionalString(reqSKU.Unit)

		price, err := normalizeSKUPrice(reqSKU.Price)
		if err != nil {
			return nil, fmt.Errorf("SKU销售价格无效: %w", err)
		}
		marketPrice, err := normalizeSKUPrice(reqSKU.MarketPrice)
		if err != nil {
			return nil, fmt.Errorf("SKU划线价格无效: %w", err)
		}
		costPrice, err := normalizeSKUPrice(reqSKU.CostPrice)
		if err != nil {
			return nil, fmt.Errorf("SKU成本价格无效: %w", err)
		}

		if reqSKU.WeightGram != nil && *reqSKU.WeightGram < 0 {
			return nil, ErrInvalidWeight
		}

		var imageID *int64
		if reqSKU.ImageID != nil && *reqSKU.ImageID > 0 {
			if err := validateImage(s.db.WithContext(ctx), *reqSKU.ImageID); err != nil {
				return nil, err
			}
			id := *reqSKU.ImageID
			imageID = &id
		}

		status := int8(1)
		if reqSKU.Status != nil {
			if *reqSKU.Status != 0 && *reqSKU.Status != 1 {
				return nil, errors.New("SKU状态无效(仅支持0禁用/1启用)")
			}
			status = *reqSKU.Status
		}

		sortVal := 0
		if reqSKU.Sort != nil {
			sortVal = *reqSKU.Sort
		}

		specValuesJSON, _ := json.Marshal(reqSKU.SpecValues)

		result = append(result, processedSKU{
			SpecValues:  string(specValuesJSON),
			SpecHash:    hash,
			SKUCode:     code,
			Barcode:     barcode,
			ImageID:     imageID,
			Price:       price,
			MarketPrice: marketPrice,
			CostPrice:   costPrice,
			Unit:        unit,
			WeightGram:  reqSKU.WeightGram,
			Sort:        sortVal,
			Status:      status,
		})
	}
	return result, nil
}

func (s *Service) processSKUsForUpdate(ctx context.Context, tx *gorm.DB, productID int64, specs []ProductSpec, skus []ProductSKURequest, existingByID map[int64]model.ProductSKU) ([]processedSKU, error) {
	seenHashes := make(map[string]bool, len(skus))
	seenCodes := make(map[string]bool, len(skus))
	submittedExistingIDs := make(map[int64]bool, len(existingByID))
	result := make([]processedSKU, 0, len(skus))

	for i, reqSKU := range skus {
		hash, err := CalculateSKUSpecHash(specs, reqSKU.SpecValues)
		if err != nil {
			return nil, fmt.Errorf("SKU #%d: %w", i+1, err)
		}
		if seenHashes[hash] {
			return nil, ErrDuplicateSKUCombo
		}
		seenHashes[hash] = true

		isExisting := false
		isRestore := false
		targetSKUID := reqSKU.ID

		if reqSKU.ID > 0 {
			oldSKU, exists := existingByID[reqSKU.ID]
			if !exists {
				return nil, fmt.Errorf("%w: ID %d", ErrSKUNotBelongToProduct, reqSKU.ID)
			}
			if submittedExistingIDs[reqSKU.ID] {
				return nil, fmt.Errorf("%w: ID %d", ErrDuplicateSKUID, reqSKU.ID)
			}
			submittedExistingIDs[reqSKU.ID] = true
			if oldSKU.SpecHash != hash {
				return nil, fmt.Errorf("%w (SKU ID: %d)", ErrSKUIdentityChanged, reqSKU.ID)
			}
			isExisting = true
		} else {
			// New SKU - check if previously soft-deleted SKU exists with this hash
			var softDeleted model.ProductSKU
			err := tx.Unscoped().Where("product_id = ? AND spec_hash = ? AND deleted_at IS NOT NULL", productID, hash).First(&softDeleted).Error
			if err == nil {
				// Restore soft deleted SKU
				isRestore = true
				targetSKUID = softDeleted.ID
			}
		}

		code, err := s.normalizeSKUCode(ctx, tx, reqSKU.SKUCode, seenCodes, targetSKUID)
		if err != nil {
			return nil, err
		}

		barcode := normalizeOptionalString(reqSKU.Barcode)
		unit := normalizeOptionalString(reqSKU.Unit)

		price, err := normalizeSKUPrice(reqSKU.Price)
		if err != nil {
			return nil, fmt.Errorf("SKU销售价格无效: %w", err)
		}
		marketPrice, err := normalizeSKUPrice(reqSKU.MarketPrice)
		if err != nil {
			return nil, fmt.Errorf("SKU划线价格无效: %w", err)
		}
		costPrice, err := normalizeSKUPrice(reqSKU.CostPrice)
		if err != nil {
			return nil, fmt.Errorf("SKU成本价格无效: %w", err)
		}

		if reqSKU.WeightGram != nil && *reqSKU.WeightGram < 0 {
			return nil, ErrInvalidWeight
		}

		var imageID *int64
		if reqSKU.ImageID != nil && *reqSKU.ImageID > 0 {
			newID := *reqSKU.ImageID
			needCheck := true
			if isExisting {
				oldSKU := existingByID[reqSKU.ID]
				if oldSKU.ImageID != nil && *oldSKU.ImageID == newID {
					needCheck = false
				}
			}
			if needCheck {
				if err := validateImage(tx, newID); err != nil {
					return nil, err
				}
			}
			imageID = &newID
		}

		status := int8(1)
		if reqSKU.Status != nil {
			if *reqSKU.Status != 0 && *reqSKU.Status != 1 {
				return nil, errors.New("SKU状态无效(仅支持0禁用/1启用)")
			}
			status = *reqSKU.Status
		}

		sortVal := 0
		if reqSKU.Sort != nil {
			sortVal = *reqSKU.Sort
		}

		specValuesJSON, _ := json.Marshal(reqSKU.SpecValues)

		result = append(result, processedSKU{
			ID:          targetSKUID,
			SpecValues:  string(specValuesJSON),
			SpecHash:    hash,
			SKUCode:     code,
			Barcode:     barcode,
			ImageID:     imageID,
			Price:       price,
			MarketPrice: marketPrice,
			CostPrice:   costPrice,
			Unit:        unit,
			WeightGram:  reqSKU.WeightGram,
			Sort:        sortVal,
			Status:      status,
			IsExisting:  isExisting,
			IsRestore:   isRestore,
		})
	}

	// Verify all existing SKUs were submitted
	for existingID := range existingByID {
		if !submittedExistingIDs[existingID] {
			return nil, fmt.Errorf("%w: 缺少已有SKU ID %d", ErrMissingExistingSKU, existingID)
		}
	}

	return result, nil
}

func (s *Service) normalizeSKUCode(ctx context.Context, tx *gorm.DB, rawCode *string, seenCodes map[string]bool, currentSKUID int64) (*string, error) {
	if rawCode == nil {
		return nil, nil
	}
	code := strings.TrimSpace(*rawCode)
	if code == "" {
		return nil, nil
	}
	if seenCodes[code] {
		return nil, fmt.Errorf("提交的SKU列表中存在重复的SKU编码: %s", code)
	}
	seenCodes[code] = true

	// Check global uniqueness across product_sku (including soft-deleted)
	queryDB := s.db
	if tx != nil {
		queryDB = tx
	}
	q := queryDB.WithContext(ctx).Unscoped().Model(&model.ProductSKU{}).Where("sku_code = ?", code)
	if currentSKUID > 0 {
		q = q.Where("id != ?", currentSKUID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("SKU编码已存在: %s", code)
	}
	return &code, nil
}

func validateImage(db *gorm.DB, imageID int64) error {
	var count int64
	if err := db.Model(&model.SystemFile{}).
		Where("id = ? AND deleted_at IS NULL", imageID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%w: ID %d", ErrImageNotFound, imageID)
	}
	return nil
}

func (s *Service) getCategoryAndDescendantIDs(ctx context.Context, rootID int64) ([]int64, error) {
	var allCategories []model.ProductCategory
	if err := s.db.WithContext(ctx).Select("id, parent_id").Where("deleted_at IS NULL").Find(&allCategories).Error; err != nil {
		return nil, err
	}

	childrenMap := make(map[int64][]int64, len(allCategories))
	for _, c := range allCategories {
		childrenMap[c.ParentID] = append(childrenMap[c.ParentID], c.ID)
	}

	result := []int64{rootID}
	queue := []int64{rootID}
	visited := map[int64]bool{rootID: true}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, childID := range childrenMap[curr] {
			if visited[childID] {
				continue
			}
			visited[childID] = true
			result = append(result, childID)
			queue = append(queue, childID)
		}
	}
	return result, nil
}

func deduplicatePositiveIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

func normalizeOptionalString(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeSKUPrice(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "0.00", nil
	}
	return util.NormalizeDecimalYuan(trimmed)
}

func validateProductFreight(ctx context.Context, db *gorm.DB, freightType *int8, freightFee *string, freightTemplateID *int64, productStatus int8, currentTemplateID *int64) (int8, string, *int64, error) {
	targetType := int8(1)
	if freightType != nil {
		targetType = *freightType
	}

	switch targetType {
	case 1:
		return 1, "0.00", nil, nil
	case 2:
		if freightFee == nil {
			return 0, "", nil, errors.New("固定运费金额不能为空")
		}
		feeStr := strings.TrimSpace(*freightFee)
		if feeStr == "" {
			return 0, "", nil, errors.New("固定运费金额不能为空")
		}
		normFee, err := util.NormalizeDecimalYuan(feeStr)
		if err != nil {
			return 0, "", nil, fmt.Errorf("固定运费金额格式错误: %w", err)
		}
		return 2, normFee, nil, nil
	case 3:
		if freightTemplateID == nil || *freightTemplateID <= 0 {
			return 0, "", nil, errors.New("请选择运费模板")
		}
		var tmpl model.FreightTemplate
		if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *freightTemplateID).First(&tmpl).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, "", nil, errors.New("所选运费模板不存在")
			}
			return 0, "", nil, err
		}
		if tmpl.Status != 1 {
			// 新选禁用模板或商品为上架状态时均不允许
			if currentTemplateID == nil || *currentTemplateID != tmpl.ID || productStatus == 1 {
				return 0, "", nil, errors.New("所选运费模板已停用/不可用")
			}
		}
		return 3, "0.00", &tmpl.ID, nil
	default:
		return 0, "", nil, fmt.Errorf("非法配送方式: %d", targetType)
	}
}
