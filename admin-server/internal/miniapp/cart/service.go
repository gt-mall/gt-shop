package cart

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/util"
)

var (
	ErrDatabaseUnavailable = errors.New("数据库不可用")
	ErrCartItemNotFound    = errors.New("购物车商品不存在")
	ErrQuantityInvalid     = errors.New("数量必须大于0")
	ErrQuantityLimit       = errors.New("加购数量超出上限(最大999)")
	ErrSKUNotFound         = errors.New("商品规格不存在")
	ErrSKUOffSale          = errors.New("商品规格已停售")
	ErrProductNotFound     = errors.New("商品不存在")
	ErrProductOffShelf     = errors.New("商品已下架")
	ErrInventoryNotFound   = errors.New("商品库存不存在")
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
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Add adds an item to member's cart or increments quantity if already exists.
func (s *Service) Add(ctx context.Context, memberID int64, req AddRequest) error {
	if s.db == nil {
		return ErrDatabaseUnavailable
	}
	if req.SKUID <= 0 {
		return ErrSKUNotFound
	}
	if req.Quantity <= 0 {
		return ErrQuantityInvalid
	}
	if req.Quantity > 999 {
		return ErrQuantityLimit
	}

	// 1. Verify SKU existence and status
	var sku model.ProductSKU
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", req.SKUID).
		First(&sku).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSKUNotFound
		}
		return err
	}
	if sku.Status != 1 {
		return ErrSKUOffSale
	}

	// 2. Verify Product existence and status
	var prod model.Product
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", sku.ProductID).
		First(&prod).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProductNotFound
		}
		return err
	}
	if prod.Status != 1 {
		return ErrProductOffShelf
	}

	// 3. Verify Inventory exists
	var inv model.ProductSkuInventory
	if err := s.db.WithContext(ctx).
		Where("sku_id = ?", sku.ID).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInventoryNotFound
		}
		return err
	}

	// 4. Upsert cart item without locking inventory
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item model.CartItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("member_id = ? AND sku_id = ?", memberID, req.SKUID).
			First(&item).Error

		if err == nil {
			newQty := item.Quantity + req.Quantity
			if newQty > 999 {
				newQty = 999
			}
			return tx.Model(&item).Updates(map[string]interface{}{
				"quantity":   newQty,
				"updated_at": time.Now(),
			}).Error
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			qty := req.Quantity
			if qty > 999 {
				qty = 999
			}
			newItem := model.CartItem{
				MemberID:  memberID,
				ProductID: prod.ID,
				SKUID:     sku.ID,
				Quantity:  qty,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if createErr := tx.Create(&newItem).Error; createErr != nil {
				if isDuplicateKeyError(createErr) {
					var existing model.CartItem
					if fErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
						Where("member_id = ? AND sku_id = ?", memberID, req.SKUID).
						First(&existing).Error; fErr != nil {
						return fErr
					}
					newQty := existing.Quantity + req.Quantity
					if newQty > 999 {
						newQty = 999
					}
					return tx.Model(&existing).Updates(map[string]interface{}{
						"quantity":   newQty,
						"updated_at": time.Now(),
					}).Error
				}
				return createErr
			}
			return nil
		}

		return err
	})
}

// UpdateQuantity updates the quantity of a specific cart item belonging to member.
func (s *Service) UpdateQuantity(ctx context.Context, memberID int64, req UpdateQuantityRequest) error {
	if s.db == nil {
		return ErrDatabaseUnavailable
	}
	if req.ID <= 0 {
		return ErrCartItemNotFound
	}
	if req.Quantity <= 0 {
		return ErrQuantityInvalid
	}
	if req.Quantity > 999 {
		return ErrQuantityLimit
	}

	result := s.db.WithContext(ctx).
		Model(&model.CartItem{}).
		Where("id = ? AND member_id = ?", req.ID, memberID).
		Updates(map[string]interface{}{
			"quantity":   req.Quantity,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCartItemNotFound
	}
	return nil
}

// Delete deletes one or more cart items belonging to member.
func (s *Service) Delete(ctx context.Context, memberID int64, req DeleteRequest) error {
	if s.db == nil {
		return ErrDatabaseUnavailable
	}
	if len(req.IDs) == 0 {
		return errors.New("请选择要删除的商品")
	}

	return s.db.WithContext(ctx).
		Where("member_id = ? AND id IN ?", memberID, req.IDs).
		Delete(&model.CartItem{}).Error
}

// List returns all cart items for member with real-time resolved product, sku, inventory, and specs.
func (s *Service) List(ctx context.Context, memberID int64) ([]CartItemResponse, error) {
	if s.db == nil {
		return nil, ErrDatabaseUnavailable
	}

	var cartItems []model.CartItem
	if err := s.db.WithContext(ctx).
		Where("member_id = ?", memberID).
		Order("updated_at DESC, id DESC").
		Find(&cartItems).Error; err != nil {
		return nil, err
	}
	if len(cartItems) == 0 {
		return []CartItemResponse{}, nil
	}

	skuIDs := make([]int64, 0, len(cartItems))
	prodIDMap := make(map[int64]struct{}, len(cartItems))
	for _, it := range cartItems {
		skuIDs = append(skuIDs, it.SKUID)
		prodIDMap[it.ProductID] = struct{}{}
	}

	// 1. Batch fetch SKUs
	var skus []model.ProductSKU
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", skuIDs).
		Find(&skus).Error; err != nil {
		return nil, err
	}
	skuMap := make(map[int64]model.ProductSKU, len(skus))
	for _, sku := range skus {
		skuMap[sku.ID] = sku
		prodIDMap[sku.ProductID] = struct{}{}
	}

	// 2. Batch fetch Products
	prodIDs := make([]int64, 0, len(prodIDMap))
	for pid := range prodIDMap {
		prodIDs = append(prodIDs, pid)
	}
	var prods []model.Product
	if len(prodIDs) > 0 {
		if err := s.db.WithContext(ctx).
			Where("id IN ? AND deleted_at IS NULL", prodIDs).
			Find(&prods).Error; err != nil {
			return nil, err
		}
	}
	prodMap := make(map[int64]model.Product, len(prods))
	for _, prod := range prods {
		prodMap[prod.ID] = prod
	}

	// 3. Batch fetch Inventories
	var invs []model.ProductSkuInventory
	if err := s.db.WithContext(ctx).
		Where("sku_id IN ?", skuIDs).
		Find(&invs).Error; err != nil {
		return nil, err
	}
	invMap := make(map[int64]model.ProductSkuInventory, len(invs))
	for _, inv := range invs {
		invMap[inv.SKUID] = inv
	}

	// 4. Batch resolve images
	imageIDs := make([]int64, 0, len(skus)+len(prods))
	for _, sku := range skus {
		if sku.ImageID != nil && *sku.ImageID > 0 {
			imageIDs = append(imageIDs, *sku.ImageID)
		}
	}
	for _, prod := range prods {
		if prod.MainImageID != nil && *prod.MainImageID > 0 {
			imageIDs = append(imageIDs, *prod.MainImageID)
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

	// 5. Build responses
	res := make([]CartItemResponse, 0, len(cartItems))
	for _, item := range cartItems {
		sku, skuOk := skuMap[item.SKUID]
		prod, prodOk := prodMap[item.ProductID]
		if !prodOk && skuOk {
			prod, prodOk = prodMap[sku.ProductID]
		}
		inv, invOk := invMap[item.SKUID]

		respItem := CartItemResponse{
			ID:                item.ID,
			ProductID:         item.ProductID,
			SKUID:             item.SKUID,
			Quantity:          item.Quantity,
			Available:         true,
			UnavailableReason: "",
			SpecValues:        []CartSpecValue{},
		}

		if prodOk {
			respItem.ProductName = prod.Name
			if prod.MainImageID != nil {
				respItem.ImageURL = urlMap[*prod.MainImageID]
			}
		} else {
			respItem.ProductName = "商品已失效"
		}

		if skuOk {
			if sku.ImageID != nil && urlMap[*sku.ImageID] != "" {
				respItem.ImageURL = urlMap[*sku.ImageID]
			}
			priceStr := sku.Price
			if norm, err := util.NormalizeDecimalYuan(sku.Price); err == nil {
				priceStr = norm
			}
			respItem.Price = priceStr

			marketPriceStr := sku.MarketPrice
			if norm, err := util.NormalizeDecimalYuan(sku.MarketPrice); err == nil {
				marketPriceStr = norm
			}
			respItem.MarketPrice = marketPriceStr

			var prodSpecs *string
			if prodOk {
				prodSpecs = prod.Specs
			}
			enrichedJSON := util.EnrichSKUSpecValues(sku.SpecValues, prodSpecs)
			if enrichedJSON != "" && enrichedJSON != "[]" {
				var specs []CartSpecValue
				if err := json.Unmarshal([]byte(enrichedJSON), &specs); err == nil {
					respItem.SpecValues = specs
				}
			}
		} else {
			respItem.Price = "0.00"
			respItem.MarketPrice = "0.00"
		}

		if invOk {
			availStock := inv.Stock - inv.LockedStock
			if availStock < 0 {
				availStock = 0
			}
			respItem.Stock = availStock
		} else {
			respItem.Stock = 0
		}

		// Check availability and reason
		if !skuOk {
			respItem.Available = false
			respItem.UnavailableReason = "规格不存在"
		} else if !prodOk {
			respItem.Available = false
			respItem.UnavailableReason = "商品不存在"
		} else if prod.Status != 1 {
			respItem.Available = false
			respItem.UnavailableReason = "商品已下架"
		} else if sku.Status != 1 {
			respItem.Available = false
			respItem.UnavailableReason = "规格已停售"
		} else if !invOk {
			respItem.Available = false
			respItem.UnavailableReason = "库存异常"
		} else if respItem.Stock <= 0 {
			respItem.Available = false
			respItem.UnavailableReason = "暂时缺货"
		}

		res = append(res, respItem)
	}

	return res, nil
}
