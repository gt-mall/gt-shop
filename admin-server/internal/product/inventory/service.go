package inventory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/google/uuid"

	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) DB() *gorm.DB {
	return s.db
}

// InitZeroInventoryTx initializes a 0-balance inventory record inside the caller's transaction.
func (s *Service) InitZeroInventoryTx(ctx context.Context, tx *gorm.DB, productID, skuID int64) error {
	inv := model.ProductSkuInventory{
		ProductID:    productID,
		SKUID:        skuID,
		Stock:        0,
		LockedStock:  0,
		WarningStock: 0,
		Version:      0,
	}
	return tx.WithContext(ctx).Create(&inv).Error
}

// OrderDeductStockTx atomically deducts inventory for an order item within the caller's transaction.
func (s *Service) OrderDeductStockTx(
	ctx context.Context,
	tx *gorm.DB,
	productID, skuID int64,
	quantity int,
	orderNo string,
	operatorID int64,
	operatorName string,
) error {
	if quantity <= 0 {
		return errors.New("扣减库存数量必须大于0")
	}

	// 1. Lock product
	var prod model.Product
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", productID).
		First(&prod).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品不存在或已下架 (ID: %d)", productID)
		}
		return err
	}
	if prod.Status != 1 {
		return fmt.Errorf("商品未上架: %s", prod.Name)
	}

	// 2. Lock SKU
	var sku model.ProductSKU
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND product_id = ? AND deleted_at IS NULL", skuID, productID).
		First(&sku).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品规格不存在 (SKU ID: %d)", skuID)
		}
		return err
	}
	if sku.Status != 1 {
		return fmt.Errorf("商品规格已停售 (SKU ID: %d)", skuID)
	}

	// 3. Lock Inventory
	var inv model.ProductSkuInventory
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sku_id = ?", skuID).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品库存未初始化 (SKU ID: %d)", skuID)
		}
		return err
	}

	avail := inv.Stock - inv.LockedStock
	if avail < quantity {
		return fmt.Errorf("商品库存不足: %s (可用: %d, 需要: %d)", prod.Name, avail, quantity)
	}

	stockAfter := inv.Stock - quantity
	now := time.Now()

	res := tx.WithContext(ctx).Model(&model.ProductSkuInventory{}).
		Where("id = ? AND version = ?", inv.ID, inv.Version).
		Updates(map[string]any{
			"stock":      stockAfter,
			"version":    inv.Version + 1,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("库存并发更新冲突，请重试")
	}

	itemRequestID := uuid.New().String()
	remark := fmt.Sprintf("订单 %s 扣减库存", orderNo)
	if operatorName == "" {
		operatorName = "member"
	}

	invLog := model.ProductSkuInventoryLog{
		InventoryID:   inv.ID,
		ProductID:     prod.ID,
		SKUID:         sku.ID,
		RequestID:     itemRequestID,
		RequestHash:   "",
		ChangeType:    ChangeTypeOrderOut,
		Quantity:      quantity,
		StockBefore:   inv.Stock,
		StockDelta:    -quantity,
		StockAfter:    stockAfter,
		LockedBefore:  inv.LockedStock,
		LockedDelta:   0,
		LockedAfter:   inv.LockedStock,
		WarningBefore: inv.WarningStock,
		WarningAfter:  inv.WarningStock,
		VersionBefore: inv.Version,
		VersionAfter:  inv.Version + 1,
		SKUSnapshot:   buildSKUSnapshot(prod, sku),
		OperatorID:    operatorID,
		OperatorName:  operatorName,
		Remark:        remark,
		CreatedAt:     now,
	}

	return tx.WithContext(ctx).Create(&invLog).Error
}

// OrderRestoreStockTx restores stock when an order is cancelled within the caller's transaction.
func (s *Service) OrderRestoreStockTx(
	ctx context.Context,
	tx *gorm.DB,
	productID, skuID int64,
	quantity int,
	orderNo string,
	operatorID int64,
	operatorName string,
) error {
	if quantity <= 0 {
		return nil
	}

	// 1. Lock product
	var prod model.Product
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", productID).
		First(&prod).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品不存在 (ID: %d)", productID)
		}
		return err
	}

	// 2. Lock SKU
	var sku model.ProductSKU
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND product_id = ? AND deleted_at IS NULL", skuID, productID).
		First(&sku).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品规格不存在 (SKU ID: %d)", skuID)
		}
		return err
	}

	// 3. Lock Inventory
	var inv model.ProductSkuInventory
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sku_id = ?", skuID).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品库存未初始化 (SKU ID: %d)", skuID)
		}
		return err
	}

	stockAfter := inv.Stock + quantity
	now := time.Now()

	res := tx.WithContext(ctx).Model(&model.ProductSkuInventory{}).
		Where("id = ? AND version = ?", inv.ID, inv.Version).
		Updates(map[string]any{
			"stock":      stockAfter,
			"version":    inv.Version + 1,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("库存并发更新冲突，请重试")
	}

	itemRequestID := uuid.New().String()
	remark := fmt.Sprintf("订单 %s 取消恢复库存", orderNo)
	if operatorName == "" {
		operatorName = "member"
	}

	invLog := model.ProductSkuInventoryLog{
		InventoryID:   inv.ID,
		ProductID:     prod.ID,
		SKUID:         sku.ID,
		RequestID:     itemRequestID,
		RequestHash:   "",
		ChangeType:    ChangeTypeOrderCancel,
		Quantity:      quantity,
		StockBefore:   inv.Stock,
		StockDelta:    quantity,
		StockAfter:    stockAfter,
		LockedBefore:  inv.LockedStock,
		LockedDelta:   0,
		LockedAfter:   inv.LockedStock,
		WarningBefore: inv.WarningStock,
		WarningAfter:  inv.WarningStock,
		VersionBefore: inv.Version,
		VersionAfter:  inv.Version + 1,
		SKUSnapshot:   buildSKUSnapshot(prod, sku),
		OperatorID:    operatorID,
		OperatorName:  operatorName,
		Remark:        remark,
		CreatedAt:     now,
	}

	return tx.WithContext(ctx).Create(&invLog).Error
}

// OrderLockStockTx atomically locks inventory (locked_stock += quantity) when an order is created.
func (s *Service) OrderLockStockTx(
	ctx context.Context,
	tx *gorm.DB,
	productID, skuID int64,
	quantity int,
	orderNo string,
	operatorID int64,
	operatorName string,
) error {
	if quantity <= 0 {
		return errors.New("锁定库存数量必须大于0")
	}

	// 1. Lock product
	var prod model.Product
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", productID).
		First(&prod).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品不存在或已下架 (ID: %d)", productID)
		}
		return err
	}
	if prod.Status != 1 {
		return fmt.Errorf("商品未上架: %s", prod.Name)
	}

	// 2. Lock SKU
	var sku model.ProductSKU
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND product_id = ? AND deleted_at IS NULL", skuID, productID).
		First(&sku).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品规格不存在 (SKU ID: %d)", skuID)
		}
		return err
	}
	if sku.Status != 1 {
		return fmt.Errorf("商品规格已停售 (SKU ID: %d)", skuID)
	}

	// 3. Lock Inventory
	var inv model.ProductSkuInventory
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sku_id = ?", skuID).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品库存未初始化 (SKU ID: %d)", skuID)
		}
		return err
	}

	avail := inv.Stock - inv.LockedStock
	if avail < quantity {
		return fmt.Errorf("商品库存不足: %s (可用: %d, 需要: %d)", prod.Name, avail, quantity)
	}

	lockedAfter := inv.LockedStock + quantity
	now := time.Now()

	res := tx.WithContext(ctx).Model(&model.ProductSkuInventory{}).
		Where("id = ? AND version = ?", inv.ID, inv.Version).
		Updates(map[string]any{
			"locked_stock": lockedAfter,
			"version":      inv.Version + 1,
			"updated_at":   now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("库存并发更新冲突，请重试")
	}

	itemRequestID := uuid.New().String()
	remark := fmt.Sprintf("订单 %s 锁定库存", orderNo)
	if operatorName == "" {
		operatorName = "member"
	}

	invLog := model.ProductSkuInventoryLog{
		InventoryID:   inv.ID,
		ProductID:     prod.ID,
		SKUID:         sku.ID,
		RequestID:     itemRequestID,
		RequestHash:   "",
		ChangeType:    ChangeTypeOrderLock,
		Quantity:      quantity,
		StockBefore:   inv.Stock,
		StockDelta:    0,
		StockAfter:    inv.Stock,
		LockedBefore:  inv.LockedStock,
		LockedDelta:   quantity,
		LockedAfter:   lockedAfter,
		WarningBefore: inv.WarningStock,
		WarningAfter:  inv.WarningStock,
		VersionBefore: inv.Version,
		VersionAfter:  inv.Version + 1,
		SKUSnapshot:   buildSKUSnapshot(prod, sku),
		OperatorID:    operatorID,
		OperatorName:  operatorName,
		Remark:        remark,
		CreatedAt:     now,
	}

	return tx.WithContext(ctx).Create(&invLog).Error
}

// OrderDeductLockedStockTx deducts both stock and locked_stock upon successful payment.
func (s *Service) OrderDeductLockedStockTx(
	ctx context.Context,
	tx *gorm.DB,
	productID, skuID int64,
	quantity int,
	orderNo string,
	operatorID int64,
	operatorName string,
) error {
	if quantity <= 0 {
		return nil
	}

	// 1. Lock product
	var prod model.Product
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", productID).
		First(&prod).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品不存在 (ID: %d)", productID)
		}
		return err
	}

	// 2. Lock SKU
	var sku model.ProductSKU
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND product_id = ? AND deleted_at IS NULL", skuID, productID).
		First(&sku).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品规格不存在 (SKU ID: %d)", skuID)
		}
		return err
	}

	// 3. Lock Inventory
	var inv model.ProductSkuInventory
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sku_id = ?", skuID).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品库存未初始化 (SKU ID: %d)", skuID)
		}
		return err
	}

	stockAfter := inv.Stock - quantity
	if stockAfter < 0 {
		return fmt.Errorf("商品实际库存不足扣减: %s (当前: %d, 扣减: %d)", prod.Name, inv.Stock, quantity)
	}
	lockedAfter := inv.LockedStock - quantity
	if lockedAfter < 0 {
		lockedAfter = 0
	}
	now := time.Now()

	res := tx.WithContext(ctx).Model(&model.ProductSkuInventory{}).
		Where("id = ? AND version = ?", inv.ID, inv.Version).
		Updates(map[string]any{
			"stock":        stockAfter,
			"locked_stock": lockedAfter,
			"version":      inv.Version + 1,
			"updated_at":   now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("库存并发更新冲突，请重试")
	}

	itemRequestID := uuid.New().String()
	remark := fmt.Sprintf("订单 %s 支付扣减库存", orderNo)
	if operatorName == "" {
		operatorName = "member"
	}

	invLog := model.ProductSkuInventoryLog{
		InventoryID:   inv.ID,
		ProductID:     prod.ID,
		SKUID:         sku.ID,
		RequestID:     itemRequestID,
		RequestHash:   "",
		ChangeType:    ChangeTypeOrderOut,
		Quantity:      quantity,
		StockBefore:   inv.Stock,
		StockDelta:    -quantity,
		StockAfter:    stockAfter,
		LockedBefore:  inv.LockedStock,
		LockedDelta:   -quantity,
		LockedAfter:   lockedAfter,
		WarningBefore: inv.WarningStock,
		WarningAfter:  inv.WarningStock,
		VersionBefore: inv.Version,
		VersionAfter:  inv.Version + 1,
		SKUSnapshot:   buildSKUSnapshot(prod, sku),
		OperatorID:    operatorID,
		OperatorName:  operatorName,
		Remark:        remark,
		CreatedAt:     now,
	}

	return tx.WithContext(ctx).Create(&invLog).Error
}

// OrderUnlockStockTx releases locked_stock when an order is cancelled or times out.
func (s *Service) OrderUnlockStockTx(
	ctx context.Context,
	tx *gorm.DB,
	productID, skuID int64,
	quantity int,
	orderNo string,
	operatorID int64,
	operatorName string,
) error {
	if quantity <= 0 {
		return nil
	}

	// 1. Lock product
	var prod model.Product
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", productID).
		First(&prod).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品不存在 (ID: %d)", productID)
		}
		return err
	}

	// 2. Lock SKU
	var sku model.ProductSKU
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND product_id = ? AND deleted_at IS NULL", skuID, productID).
		First(&sku).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品规格不存在 (SKU ID: %d)", skuID)
		}
		return err
	}

	// 3. Lock Inventory
	var inv model.ProductSkuInventory
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sku_id = ?", skuID).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品库存未初始化 (SKU ID: %d)", skuID)
		}
		return err
	}

	lockedAfter := inv.LockedStock - quantity
	if lockedAfter < 0 {
		lockedAfter = 0
	}
	now := time.Now()

	res := tx.WithContext(ctx).Model(&model.ProductSkuInventory{}).
		Where("id = ? AND version = ?", inv.ID, inv.Version).
		Updates(map[string]any{
			"locked_stock": lockedAfter,
			"version":      inv.Version + 1,
			"updated_at":   now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errors.New("库存并发更新冲突，请重试")
	}

	itemRequestID := uuid.New().String()
	remark := fmt.Sprintf("订单 %s 释放锁定库存", orderNo)
	if operatorName == "" {
		operatorName = "member"
	}

	invLog := model.ProductSkuInventoryLog{
		InventoryID:   inv.ID,
		ProductID:     prod.ID,
		SKUID:         sku.ID,
		RequestID:     itemRequestID,
		RequestHash:   "",
		ChangeType:    ChangeTypeOrderUnlock,
		Quantity:      quantity,
		StockBefore:   inv.Stock,
		StockDelta:    0,
		StockAfter:    inv.Stock,
		LockedBefore:  inv.LockedStock,
		LockedDelta:   -quantity,
		LockedAfter:   lockedAfter,
		WarningBefore: inv.WarningStock,
		WarningAfter:  inv.WarningStock,
		VersionBefore: inv.Version,
		VersionAfter:  inv.Version + 1,
		SKUSnapshot:   buildSKUSnapshot(prod, sku),
		OperatorID:    operatorID,
		OperatorName:  operatorName,
		Remark:        remark,
		CreatedAt:     now,
	}

	return tx.WithContext(ctx).Create(&invLog).Error
}


type listRowQueryResult struct {
	SKUID          int64      `gorm:"column:sku_id"`
	ProductID      int64      `gorm:"column:product_id"`
	ProductName    string     `gorm:"column:product_name"`
	ProductMainImg *int64     `gorm:"column:product_main_image_id"`
	ProductSpecs   *string    `gorm:"column:product_specs"`
	ProductStatus  int8       `gorm:"column:product_status"`
	SKUCode        *string    `gorm:"column:sku_code"`
	Barcode        *string    `gorm:"column:barcode"`
	Unit           *string    `gorm:"column:unit"`
	SKUImageID     *int64     `gorm:"column:sku_image_id"`
	SKUSpecValues  string     `gorm:"column:sku_spec_values"`
	SKUStatus      int8       `gorm:"column:sku_status"`
	InventoryID    *int64     `gorm:"column:inventory_id"`
	Stock          *int       `gorm:"column:stock"`
	LockedStock    *int       `gorm:"column:locked_stock"`
	WarningStock   *int       `gorm:"column:warning_stock"`
	Version        *uint32    `gorm:"column:version"`
	InvUpdatedAt   *time.Time `gorm:"column:inv_updated_at"`
}

func (s *Service) List(ctx context.Context, req ProductInventoryListRequest) ([]ProductInventoryRow, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("数据库不可用")
	}

	query := s.db.WithContext(ctx).Table("product_sku AS s").
		Joins("INNER JOIN product AS p ON p.id = s.product_id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN product_sku_inventory AS i ON i.sku_id = s.id").
		Where("s.deleted_at IS NULL")

	if req.ProductID != nil {
		query = query.Where("s.product_id = ?", *req.ProductID)
	}
	if req.SKUID != nil {
		query = query.Where("s.id = ?", *req.SKUID)
	}
	if req.ProductStatus != nil {
		query = query.Where("p.status = ?", *req.ProductStatus)
	}
	if req.SKUStatus != nil {
		query = query.Where("s.status = ?", *req.SKUStatus)
	}

	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		kw := "%" + keyword + "%"
		query = query.Where("(p.name LIKE ? OR s.sku_code LIKE ? OR s.barcode LIKE ?)", kw, kw, kw)
	}

	switch strings.TrimSpace(req.StockFilter) {
	case "uninitialized":
		query = query.Where("i.id IS NULL")
	case "low":
		query = query.Where("i.id IS NOT NULL AND i.warning_stock > 0 AND (i.stock - i.locked_stock) <= i.warning_stock")
	case "empty":
		query = query.Where("i.id IS NOT NULL AND (i.stock - i.locked_stock) = 0")
	case "normal":
		query = query.Where("i.id IS NOT NULL AND (i.stock - i.locked_stock) > 0 AND NOT (i.warning_stock > 0 AND (i.stock - i.locked_stock) <= i.warning_stock)")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
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

	selectCols := `s.id AS sku_id,
		s.product_id AS product_id,
		p.name AS product_name,
		p.main_image_id AS product_main_image_id,
		p.specs AS product_specs,
		p.status AS product_status,
		s.sku_code AS sku_code,
		s.barcode AS barcode,
		s.unit AS unit,
		s.image_id AS sku_image_id,
		s.spec_values AS sku_spec_values,
		s.status AS sku_status,
		i.id AS inventory_id,
		i.stock AS stock,
		i.locked_stock AS locked_stock,
		i.warning_stock AS warning_stock,
		i.version AS version,
		i.updated_at AS inv_updated_at`

	var results []listRowQueryResult
	if err := query.Select(selectCols).Order("s.id DESC").Limit(pageSize).Offset(offset).Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	// Batch resolve images
	imageIDs := make([]int64, 0, len(results)*2)
	for _, r := range results {
		if r.SKUImageID != nil && *r.SKUImageID > 0 {
			imageIDs = append(imageIDs, *r.SKUImageID)
		}
		if r.ProductMainImg != nil && *r.ProductMainImg > 0 {
			imageIDs = append(imageIDs, *r.ProductMainImg)
		}
	}
	urlMap := make(map[int64]string)
	if len(imageIDs) > 0 {
		fileSvc := systemfile.NewService(s.db)
		resolved, err := fileSvc.Resolve(ctx, systemfile.IDsRequest{IDs: imageIDs})
		if err == nil {
			for _, f := range resolved {
				urlMap[f.ID] = f.URL
			}
		}
	}

	rows := make([]ProductInventoryRow, 0, len(results))
	for _, r := range results {
		// Image resolution: SKU image first, fallback to Product main image
		imageURL := ""
		if r.SKUImageID != nil && urlMap[*r.SKUImageID] != "" {
			imageURL = urlMap[*r.SKUImageID]
		} else if r.ProductMainImg != nil && urlMap[*r.ProductMainImg] != "" {
			imageURL = urlMap[*r.ProductMainImg]
		}

		// Parse specs
		specs := parseSKUSpecs(r.ProductSpecs, r.SKUSpecValues)

		row := ProductInventoryRow{
			ProductID:     r.ProductID,
			SKUID:         r.SKUID,
			ProductName:   r.ProductName,
			SKUCode:       r.SKUCode,
			Barcode:       r.Barcode,
			Unit:          r.Unit,
			Specs:         specs,
			ImageURL:      imageURL,
			ProductStatus: r.ProductStatus,
			SKUStatus:     r.SKUStatus,
			Initialized:   r.InventoryID != nil,
		}

		if r.InventoryID != nil {
			invID := *r.InventoryID
			stock := *r.Stock
			locked := *r.LockedStock
			warning := *r.WarningStock
			ver := *r.Version
			available := stock - locked
			isLow := warning > 0 && available <= warning
			isOut := available == 0

			row.InventoryID = &invID
			row.Stock = &stock
			row.LockedStock = &locked
			row.AvailableStock = &available
			row.WarningStock = &warning
			row.Version = &ver
			row.IsLowStock = &isLow
			row.IsOutOfStock = &isOut
			row.UpdatedAt = r.InvUpdatedAt
		}

		rows = append(rows, row)
	}

	return rows, total, nil
}

func (s *Service) Detail(ctx context.Context, skuID int64) (ProductInventoryRow, *BusinessError) {
	if s.db == nil {
		return ProductInventoryRow{}, NewBusinessError(ErrCodeRetryable, "数据库不可用")
	}

	var sku model.ProductSKU
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", skuID).First(&sku).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductInventoryRow{}, NewBusinessError(ErrCodeNotFound, "SKU不存在或已删除")
		}
		return ProductInventoryRow{}, NewBusinessError(ErrCodeRetryable, err.Error())
	}

	var prod model.Product
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", sku.ProductID).First(&prod).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductInventoryRow{}, NewBusinessError(ErrCodeNotFound, "商品不存在或已删除")
		}
		return ProductInventoryRow{}, NewBusinessError(ErrCodeRetryable, err.Error())
	}

	var inv model.ProductSkuInventory
	if err := s.db.WithContext(ctx).Where("sku_id = ?", skuID).First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductInventoryRow{}, NewBusinessError(ErrCodeNotInitialized, "库存未初始化")
		}
		return ProductInventoryRow{}, NewBusinessError(ErrCodeRetryable, err.Error())
	}

	// Resolve images
	imageIDs := make([]int64, 0, 2)
	if sku.ImageID != nil && *sku.ImageID > 0 {
		imageIDs = append(imageIDs, *sku.ImageID)
	}
	if prod.MainImageID != nil && *prod.MainImageID > 0 {
		imageIDs = append(imageIDs, *prod.MainImageID)
	}
	urlMap := make(map[int64]string)
	if len(imageIDs) > 0 {
		fileSvc := systemfile.NewService(s.db)
		resolved, err := fileSvc.Resolve(ctx, systemfile.IDsRequest{IDs: imageIDs})
		if err == nil {
			for _, f := range resolved {
				urlMap[f.ID] = f.URL
			}
		}
	}

	imageURL := ""
	if sku.ImageID != nil && urlMap[*sku.ImageID] != "" {
		imageURL = urlMap[*sku.ImageID]
	} else if prod.MainImageID != nil && urlMap[*prod.MainImageID] != "" {
		imageURL = urlMap[*prod.MainImageID]
	}

	specs := parseSKUSpecs(prod.Specs, sku.SpecValues)
	available := inv.Stock - inv.LockedStock
	isLow := inv.WarningStock > 0 && available <= inv.WarningStock
	isOut := available == 0

	return ProductInventoryRow{
		InventoryID:    &inv.ID,
		ProductID:      prod.ID,
		SKUID:          sku.ID,
		ProductName:    prod.Name,
		SKUCode:        sku.SKUCode,
		Barcode:        sku.Barcode,
		Unit:           sku.Unit,
		Specs:          specs,
		ImageURL:       imageURL,
		ProductStatus:  prod.Status,
		SKUStatus:      sku.Status,
		Initialized:    true,
		Stock:          &inv.Stock,
		LockedStock:    &inv.LockedStock,
		AvailableStock: &available,
		WarningStock:   &inv.WarningStock,
		Version:        &inv.Version,
		IsLowStock:     &isLow,
		IsOutOfStock:   &isOut,
		UpdatedAt:      &inv.UpdatedAt,
	}, nil
}

func (s *Service) Adjust(ctx context.Context, user model.SystemUser, req ProductInventoryAdjustRequest) (ProductInventoryMutationResult, *BusinessError) {
	if req.ChangeType != ChangeTypeManualIn && req.ChangeType != ChangeTypeManualOut && req.ChangeType != ChangeTypeStocktake {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "不支持的调整类型")
	}
	return s.executeMutation(ctx, user, req.SKUID, req.RequestID, req.ExpectedVersion, req.ChangeType, req.Quantity, req.Remark)
}

func (s *Service) Warning(ctx context.Context, user model.SystemUser, req ProductInventoryWarningRequest) (ProductInventoryMutationResult, *BusinessError) {
	return s.executeMutation(ctx, user, req.SKUID, req.RequestID, req.ExpectedVersion, ChangeTypeWarningSet, req.Quantity, req.Remark)
}

func (s *Service) executeMutation(
	ctx context.Context,
	user model.SystemUser,
	skuID int64,
	rawRequestID string,
	expectedVersion *uint32,
	changeType string,
	quantity *int,
	rawRemark string,
) (ProductInventoryMutationResult, *BusinessError) {
	if s.db == nil {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, "数据库不可用")
	}

	// 1. Validate UUID
	requestID := strings.ToLower(strings.TrimSpace(rawRequestID))
	if !uuidRegex.MatchString(requestID) {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "request_id必须为有效UUID")
	}

	// 2. Validate expected_version
	if expectedVersion == nil {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "expected_version必填")
	}

	// 3. Validate quantity
	if quantity == nil {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "quantity必填")
	}
	if *quantity < 0 || int64(*quantity) > math.MaxInt32 {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "操作数量超出有效范围")
	}

	// 4. Validate remark
	remark := strings.TrimSpace(rawRemark)
	runeCount := utf8.RuneCountInString(remark)
	if runeCount < 1 || runeCount > 500 {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "操作原因长度必须在1到500个字符之间")
	}

	// 5. Calculate normalized request_hash
	hashPayload := IdempotencyPayload{
		OperatorID:      user.ID,
		SKUID:           skuID,
		ChangeType:      changeType,
		Quantity:        *quantity,
		ExpectedVersion: *expectedVersion,
		Remark:          remark,
	}
	payloadBytes, err := json.Marshal(hashPayload)
	if err != nil {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "请求序列化失败")
	}
	sum := sha256.Sum256(payloadBytes)
	requestHash := hex.EncodeToString(sum[:])

	// 6. Pre-check idempotency outside transaction
	var preLog model.ProductSkuInventoryLog
	if err := s.db.WithContext(ctx).Where("request_id = ?", requestID).First(&preLog).Error; err == nil {
		if preLog.OperatorID != user.ID || preLog.RequestHash != requestHash {
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeIdempotencyConflict, "请求ID冲突或参数与原请求不一致")
		}
		logRow, parseErr := parseLogRow(preLog)
		if parseErr != nil {
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, parseErr.Error())
		}
		return ProductInventoryMutationResult{Log: &logRow, Replayed: true}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, err.Error())
	}

	// 7. Locate product_id from SKU
	var locSKU model.ProductSKU
	if err := s.db.WithContext(ctx).Select("id, product_id, deleted_at").Where("id = ?", skuID).First(&locSKU).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeNotFound, "SKU不存在")
		}
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, err.Error())
	}
	if locSKU.DeletedAt.Valid {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeNotFound, "SKU已被删除")
	}

	// 8. Execute in transaction with lock order: product -> product_sku -> product_sku_inventory
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		res, bErr, retry := s.executeTx(ctx, user, locSKU.ProductID, skuID, requestID, requestHash, *expectedVersion, changeType, *quantity, remark)
		if retry && attempt < maxRetries-1 {
			time.Sleep(time.Duration(20*(attempt+1)) * time.Millisecond)
			continue
		}
		return res, bErr
	}

	return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, "并发冲突频繁，请稍后重试")
}

func (s *Service) executeTx(
	ctx context.Context,
	user model.SystemUser,
	productID int64,
	skuID int64,
	requestID string,
	requestHash string,
	expectedVersion uint32,
	changeType string,
	quantity int,
	remark string,
) (ProductInventoryMutationResult, *BusinessError, bool) {
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, tx.Error.Error()), isDeadlockOrTimeout(tx.Error)
	}

	// Step 1: Lock product
	var prod model.Product
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", productID).
		First(&prod).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeNotFound, "商品不存在或已删除"), false
		}
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, err.Error()), isDeadlockOrTimeout(err)
	}

	// Step 2: Lock SKU
	var sku model.ProductSKU
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND product_id = ? AND deleted_at IS NULL", skuID, prod.ID).
		First(&sku).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeNotFound, "SKU不存在或不属于该商品"), false
		}
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, err.Error()), isDeadlockOrTimeout(err)
	}

	// Step 3: Lock inventory
	var inv model.ProductSkuInventory
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sku_id = ?", skuID).
		First(&inv).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeNotInitialized, "库存未初始化"), false
		}
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, err.Error()), isDeadlockOrTimeout(err)
	}
	if inv.ProductID != prod.ID {
		tx.Rollback()
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeNotFound, "库存商品归属不匹配"), false
	}

	// Step 4: Current read on request_id inside transaction (before version check!)
	var txLog model.ProductSkuInventoryLog
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("request_id = ?", requestID).
		First(&txLog).Error; err == nil {
		tx.Rollback()
		if txLog.OperatorID != user.ID || txLog.RequestHash != requestHash {
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeIdempotencyConflict, "请求ID冲突或参数与原请求不一致"), false
		}
		logRow, parseErr := parseLogRow(txLog)
		if parseErr != nil {
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, parseErr.Error()), false
		}
		return ProductInventoryMutationResult{Log: &logRow, Replayed: true}, nil, false
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, err.Error()), isDeadlockOrTimeout(err)
	}

	// Step 5: Version check
	if inv.Version != expectedVersion {
		tx.Rollback()
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeVersionConflict, "库存版本冲突，请刷新后重试"), false
	}
	if inv.Version >= math.MaxUint32 {
		tx.Rollback()
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "库存版本已达上限，拒绝变更"), false
	}

	// Step 6: Calculations & business checks
	var stockDelta, lockedDelta, stockAfter, lockedAfter, warningAfter int
	lockedAfter = inv.LockedStock
	lockedDelta = 0

	switch changeType {
	case ChangeTypeManualIn:
		if quantity <= 0 {
			tx.Rollback()
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "入库数量必须大于0"), false
		}
		if int64(inv.Stock)+int64(quantity) > math.MaxInt32 {
			tx.Rollback()
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "库存超出最大允许范围"), false
		}
		stockDelta = quantity
		stockAfter = inv.Stock + quantity
		warningAfter = inv.WarningStock

	case ChangeTypeManualOut:
		if quantity <= 0 {
			tx.Rollback()
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "出库数量必须大于0"), false
		}
		avail := inv.Stock - inv.LockedStock
		if quantity > avail {
			tx.Rollback()
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInsufficientStock, "可用库存不足"), false
		}
		stockDelta = -quantity
		stockAfter = inv.Stock - quantity
		warningAfter = inv.WarningStock

	case ChangeTypeStocktake:
		targetStock := quantity
		if targetStock < inv.LockedStock {
			tx.Rollback()
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "盘点目标总库存不能小于锁定库存"), false
		}
		if targetStock == inv.Stock {
			tx.Rollback()
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeNoChange, "库存数量未发生变化"), false
		}
		stockDelta = targetStock - inv.Stock
		stockAfter = targetStock
		warningAfter = inv.WarningStock

	case ChangeTypeWarningSet:
		targetWarning := quantity
		if targetWarning == inv.WarningStock {
			tx.Rollback()
			return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeNoChange, "预警阈值未发生变化"), false
		}
		stockDelta = 0
		stockAfter = inv.Stock
		warningAfter = targetWarning

	default:
		tx.Rollback()
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeInvalidArgument, "不支持的操作类型"), false
	}

	// Step 7: Build SKU Snapshot
	snapshotJSON := buildSKUSnapshot(prod, sku)

	// Step 8: Update inventory
	now := time.Now()
	res := tx.Model(&model.ProductSkuInventory{}).
		Where("id = ? AND version = ?", inv.ID, inv.Version).
		Updates(map[string]any{
			"stock":         stockAfter,
			"locked_stock":  lockedAfter,
			"warning_stock": warningAfter,
			"version":       inv.Version + 1,
			"updated_at":    now,
		})
	if res.Error != nil {
		tx.Rollback()
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, res.Error.Error()), isDeadlockOrTimeout(res.Error)
	}
	if res.RowsAffected != 1 {
		tx.Rollback()
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeVersionConflict, "库存版本冲突，请刷新后重试"), false
	}

	// Step 9: Insert inventory log
	operatorName := strings.TrimSpace(user.Nickname)
	if operatorName == "" {
		operatorName = user.Username
	}

	newLog := model.ProductSkuInventoryLog{
		InventoryID:   inv.ID,
		ProductID:     prod.ID,
		SKUID:         sku.ID,
		RequestID:     requestID,
		RequestHash:   requestHash,
		ChangeType:    changeType,
		Quantity:      quantity,
		StockBefore:   inv.Stock,
		StockDelta:    stockDelta,
		StockAfter:    stockAfter,
		LockedBefore:  inv.LockedStock,
		LockedDelta:   lockedDelta,
		LockedAfter:   lockedAfter,
		WarningBefore: inv.WarningStock,
		WarningAfter:  warningAfter,
		VersionBefore: inv.Version,
		VersionAfter:  inv.Version + 1,
		SKUSnapshot:   snapshotJSON,
		OperatorID:    user.ID,
		OperatorName:  operatorName,
		Remark:        remark,
		CreatedAt:     now,
	}
	if err := tx.Create(&newLog).Error; err != nil {
		tx.Rollback()
		if isUniqueKeyViolation(err) {
			// Outside query to check if request_id already committed
			var recheckLog model.ProductSkuInventoryLog
			if qErr := s.db.WithContext(ctx).Where("request_id = ?", requestID).First(&recheckLog).Error; qErr == nil {
				if recheckLog.OperatorID != user.ID || recheckLog.RequestHash != requestHash {
					return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeIdempotencyConflict, "请求ID冲突或参数与原请求不一致"), false
				}
				logRow, _ := parseLogRow(recheckLog)
				return ProductInventoryMutationResult{Log: &logRow, Replayed: true}, nil, false
			}
		}
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeRetryable, err.Error()), isDeadlockOrTimeout(err)
	}

	// Step 10: Commit transaction
	if err := tx.Commit().Error; err != nil {
		return ProductInventoryMutationResult{}, NewBusinessError(ErrCodeResultUnknown, "提交结果未知，请重试"), false
	}

	logRow, _ := parseLogRow(newLog)
	return ProductInventoryMutationResult{
		Log:      &logRow,
		Replayed: false,
	}, nil, false
}

func (s *Service) ListLogs(ctx context.Context, req ProductInventoryLogListRequest) ([]ProductInventoryLogRow, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("数据库不可用")
	}

	query := s.db.WithContext(ctx).Model(&model.ProductSkuInventoryLog{})

	if req.ProductID != nil {
		query = query.Where("product_id = ?", *req.ProductID)
	}
	if req.SKUID != nil {
		query = query.Where("sku_id = ?", *req.SKUID)
	}
	if req.RequestID != nil && strings.TrimSpace(*req.RequestID) != "" {
		query = query.Where("request_id = ?", strings.ToLower(strings.TrimSpace(*req.RequestID)))
	}
	if req.ChangeType != nil && strings.TrimSpace(*req.ChangeType) != "" {
		query = query.Where("change_type = ?", strings.TrimSpace(*req.ChangeType))
	}
	if req.OperatorID != nil {
		query = query.Where("operator_id = ?", *req.OperatorID)
	}

	var tFrom, tTo *time.Time
	if req.CreatedFrom != nil && strings.TrimSpace(*req.CreatedFrom) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.CreatedFrom))
		if err != nil {
			return nil, 0, fmt.Errorf("created_from 格式无效，必须为 RFC3339: %w", err)
		}
		tFrom = &t
		query = query.Where("created_at >= ?", t)
	}
	if req.CreatedTo != nil && strings.TrimSpace(*req.CreatedTo) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.CreatedTo))
		if err != nil {
			return nil, 0, fmt.Errorf("created_to 格式无效，必须为 RFC3339: %w", err)
		}
		tTo = &t
		query = query.Where("created_at < ?", t)
	}
	if tFrom != nil && tTo != nil && tFrom.After(*tTo) {
		return nil, 0, errors.New("开始时间不能晚于结束时间")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
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

	var logs []model.ProductSkuInventoryLog
	if err := query.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]ProductInventoryLogRow, 0, len(logs))
	for _, l := range logs {
		row, err := parseLogRow(l)
		if err != nil {
			continue
		}
		rows = append(rows, row)
	}

	return rows, total, nil
}

type rawSpecValue struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type rawSpec struct {
	ID     int64          `json:"id"`
	Name   string         `json:"name"`
	Values []rawSpecValue `json:"values"`
}

type rawSKUValue struct {
	SpecID  int64 `json:"spec_id"`
	ValueID int64 `json:"value_id"`
}

func parseSKUSpecs(specsJSON *string, specValuesJSON string) []ProductInventorySpecOption {
	options := make([]ProductInventorySpecOption, 0)
	if specsJSON == nil || *specsJSON == "" || strings.TrimSpace(specValuesJSON) == "" {
		return options
	}

	var specsDef []rawSpec
	if err := json.Unmarshal([]byte(*specsJSON), &specsDef); err != nil {
		return options
	}

	var skuValues []rawSKUValue
	if err := json.Unmarshal([]byte(specValuesJSON), &skuValues); err != nil {
		return options
	}

	specNameMap := make(map[int64]string)
	valNameMap := make(map[int64]map[int64]string)
	for _, s := range specsDef {
		specNameMap[s.ID] = s.Name
		valMap := make(map[int64]string)
		for _, v := range s.Values {
			valMap[v.ID] = v.Name
		}
		valNameMap[s.ID] = valMap
	}

	for _, sv := range skuValues {
		sName := specNameMap[sv.SpecID]
		vName := ""
		if vm, ok := valNameMap[sv.SpecID]; ok {
			vName = vm[sv.ValueID]
		}
		options = append(options, ProductInventorySpecOption{
			SpecID:    sv.SpecID,
			SpecName:  sName,
			ValueID:   sv.ValueID,
			ValueName: vName,
		})
	}
	return options
}

func buildSKUSnapshot(prod model.Product, sku model.ProductSKU) string {
	specs := parseSKUSpecs(prod.Specs, sku.SpecValues)
	snapshot := ProductInventorySKUSnapshot{
		ProductName: prod.Name,
		SKUCode:     sku.SKUCode,
		Unit:        sku.Unit,
		Specs:       specs,
	}
	bytes, err := json.Marshal(snapshot)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func parseLogRow(m model.ProductSkuInventoryLog) (ProductInventoryLogRow, error) {
	var snapshot ProductInventorySKUSnapshot
	if m.SKUSnapshot != "" {
		if err := json.Unmarshal([]byte(m.SKUSnapshot), &snapshot); err != nil {
			snapshot = ProductInventorySKUSnapshot{
				ProductName: "",
				Specs:       []ProductInventorySpecOption{},
			}
		}
	}
	if snapshot.Specs == nil {
		snapshot.Specs = []ProductInventorySpecOption{}
	}

	return ProductInventoryLogRow{
		ID:              m.ID,
		InventoryID:     m.InventoryID,
		ProductID:       m.ProductID,
		SKUID:           m.SKUID,
		RequestID:       m.RequestID,
		ChangeType:      m.ChangeType,
		Quantity:        m.Quantity,
		StockBefore:     m.StockBefore,
		StockDelta:      m.StockDelta,
		StockAfter:      m.StockAfter,
		LockedBefore:    m.LockedBefore,
		LockedDelta:     m.LockedDelta,
		LockedAfter:     m.LockedAfter,
		AvailableBefore: m.StockBefore - m.LockedBefore,
		AvailableAfter:  m.StockAfter - m.LockedAfter,
		WarningBefore:   m.WarningBefore,
		WarningAfter:    m.WarningAfter,
		VersionBefore:   m.VersionBefore,
		VersionAfter:    m.VersionAfter,
		SKUSnapshot:     snapshot,
		OperatorID:      m.OperatorID,
		OperatorName:    m.OperatorName,
		Remark:          m.Remark,
		CreatedAt:       m.CreatedAt,
	}, nil
}

func isUniqueKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "1062") ||
		strings.Contains(s, "Duplicate entry") ||
		strings.Contains(s, "UNIQUE constraint failed") ||
		strings.Contains(s, "unique constraint")
}

func isDeadlockOrTimeout(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "1213") ||
		strings.Contains(s, "Deadlock") ||
		strings.Contains(s, "1205") ||
		strings.Contains(s, "Lock wait timeout") ||
		strings.Contains(s, "database is locked") ||
		strings.Contains(s, "busy")
}
