package product

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"
	"mall/admin_core/pkg/util"
)

var (
	ErrDatabaseUnavailable       = errors.New("数据库不可用")
	ErrProductNotFoundOrOffShelf = errors.New("商品不存在或已下架")
)

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

type productRow struct {
	ID           int64  `gorm:"column:id"`
	CategoryID   int64  `gorm:"column:category_id"`
	Name         string `gorm:"column:name"`
	Subtitle     string `gorm:"column:subtitle"`
	MainImageID  *int64 `gorm:"column:main_image_id"`
	SalesCount   int64  `gorm:"column:sales_count"`
	VirtualSales int    `gorm:"column:virtual_sales"`
	Sort         int    `gorm:"column:sort"`
}

type miniappListCache struct {
	Items []ProductListItem `json:"items"`
	Total int64             `json:"total"`
}

// List queries products with pagination, filtering, and sorting without N+1 queries.
// Static list metadata is cached via rueidisaside with generation, while live inventory is queried dynamically.
func (s *Service) List(ctx context.Context, req ProductListRequest) ([]ProductListItem, int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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

	var catID int64 = -1
	if req.CategoryID != nil {
		catID = *req.CategoryID
	}
	paramStr := fmt.Sprintf("p=%d:s=%d:c=%d:sort=%s:kw=%s", page, pageSize, catID, req.Sort, strings.TrimSpace(req.Keyword))

	gen := cache.GetProductListGeneration(ctx, s.cache)
	cacheKey := cache.ProductListKey(gen, paramStr)

	cached, err := cache.Get(ctx, s.cache, cacheKey, cache.TTLProductList, func(innerCtx context.Context) (*miniappListCache, error) {
		items, total, err := s.loadStaticListFromDB(innerCtx, req, page, pageSize)
		if err != nil {
			return nil, err
		}
		if total == 0 {
			cache.SetTTL(innerCtx, cache.TTLEmptyProductList)
		}
		return &miniappListCache{Items: items, Total: total}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	if cached == nil {
		return []ProductListItem{}, 0, nil
	}

	items, err := s.fillListStock(ctx, cached.Items)
	if err != nil {
		return nil, 0, err
	}
	return items, cached.Total, nil
}

func (s *Service) loadStaticListFromDB(ctx context.Context, req ProductListRequest, page, pageSize int) ([]ProductListItem, int64, error) {
	if s.db == nil {
		return nil, 0, ErrDatabaseUnavailable
	}

	keyword := strings.TrimSpace(req.Keyword)

	// Count query: only status=1, not deleted products with at least one valid SKU
	countQuery := s.db.WithContext(ctx).Table("product AS p").
		Joins("INNER JOIN product_sku AS s ON s.product_id = p.id AND s.status = 1 AND s.deleted_at IS NULL").
		Where("p.status = 1 AND p.deleted_at IS NULL")

	if req.CategoryID != nil && *req.CategoryID > 0 {
		countQuery = countQuery.Where("p.category_id = ?", *req.CategoryID)
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		countQuery = countQuery.Where("(p.name LIKE ? OR p.subtitle LIKE ?)", kw, kw)
	}

	var total int64
	if err := countQuery.Select("COUNT(DISTINCT p.id)").Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []ProductListItem{}, 0, nil
	}

	offset := (page - 1) * pageSize
	if int64(offset) >= total {
		return []ProductListItem{}, total, nil
	}

	listQuery := s.db.WithContext(ctx).Table("product AS p").
		Joins("INNER JOIN product_sku AS s ON s.product_id = p.id AND s.status = 1 AND s.deleted_at IS NULL").
		Where("p.status = 1 AND p.deleted_at IS NULL")

	if req.CategoryID != nil && *req.CategoryID > 0 {
		listQuery = listQuery.Where("p.category_id = ?", *req.CategoryID)
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		listQuery = listQuery.Where("(p.name LIKE ? OR p.subtitle LIKE ?)", kw, kw)
	}

	listQuery = listQuery.Group("p.id, p.category_id, p.name, p.subtitle, p.main_image_id, p.sales_count, p.virtual_sales, p.sort")

	switch req.Sort {
	case "sales":
		listQuery = listQuery.Order("(p.sales_count + p.virtual_sales) DESC, p.sort ASC, p.id DESC")
	case "price_asc":
		listQuery = listQuery.Order("MIN(CAST(s.price AS DECIMAL(12,2))) ASC, p.sort ASC, p.id DESC")
	case "price_desc":
		listQuery = listQuery.Order("MIN(CAST(s.price AS DECIMAL(12,2))) DESC, p.sort ASC, p.id DESC")
	default:
		listQuery = listQuery.Order("p.sort ASC, p.id DESC")
	}

	var rows []productRow
	if err := listQuery.Select("p.id, p.category_id, p.name, p.subtitle, p.main_image_id, p.sales_count, p.virtual_sales, p.sort").
		Offset(offset).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	if len(rows) == 0 {
		return []ProductListItem{}, total, nil
	}

	productIDs := make([]int64, 0, len(rows))
	imageIDs := make([]int64, 0, len(rows))
	for _, r := range rows {
		productIDs = append(productIDs, r.ID)
		if r.MainImageID != nil && *r.MainImageID > 0 {
			imageIDs = append(imageIDs, *r.MainImageID)
		}
	}

	// Batch fetch valid SKUs for the products
	var skus []model.ProductSKU
	if err := s.db.WithContext(ctx).
		Where("product_id IN ? AND status = 1 AND deleted_at IS NULL", productIDs).
		Order("sort ASC, id ASC").
		Find(&skus).Error; err != nil {
		return nil, 0, err
	}

	skusByProductID := make(map[int64][]model.ProductSKU, len(productIDs))
	for _, sku := range skus {
		skusByProductID[sku.ProductID] = append(skusByProductID[sku.ProductID], sku)
	}

	// Batch resolve main images
	urlMap := make(map[int64]string)
	if len(imageIDs) > 0 {
		resolved, err := systemfile.NewService(s.db).Resolve(ctx, systemfile.IDsRequest{IDs: imageIDs})
		if err == nil {
			for _, f := range resolved {
				urlMap[f.ID] = f.URL
			}
		}
	}

	items := make([]ProductListItem, 0, len(rows))
	for _, r := range rows {
		pSKUs := skusByProductID[r.ID]
		if len(pSKUs) == 0 {
			continue
		}

		var lowestSKU *model.ProductSKU
		var minCents int64 = -1

		for i := range pSKUs {
			sku := &pSKUs[i]
			cents, err := util.ParseDecimalYuanToCent(sku.Price)
			if err != nil {
				continue
			}
			if minCents < 0 || cents < minCents {
				minCents = cents
				lowestSKU = sku
			}
		}

		if lowestSKU == nil {
			continue
		}

		priceStr := util.FormatCentToYuan(minCents)
		marketPriceStr := lowestSKU.MarketPrice
		if norm, err := util.NormalizeDecimalYuan(lowestSKU.MarketPrice); err == nil {
			marketPriceStr = norm
		}

		var mainURL string
		if r.MainImageID != nil {
			mainURL = urlMap[*r.MainImageID]
		}

		sales := r.SalesCount + int64(r.VirtualSales)

		items = append(items, ProductListItem{
			ID:           r.ID,
			CategoryID:   r.CategoryID,
			Name:         r.Name,
			Subtitle:     r.Subtitle,
			MainImageURL: mainURL,
			Price:        priceStr,
			MarketPrice:  marketPriceStr,
			Sales:        sales,
			Stock:        0, // dynamic stock filled in fillListStock
		})
	}

	return items, total, nil
}

func (s *Service) fillListStock(ctx context.Context, staticItems []ProductListItem) ([]ProductListItem, error) {
	if len(staticItems) == 0 {
		return []ProductListItem{}, nil
	}
	if s.db == nil {
		return nil, ErrDatabaseUnavailable
	}

	items := make([]ProductListItem, len(staticItems))
	copy(items, staticItems)

	productIDs := make([]int64, 0, len(items))
	for _, it := range items {
		productIDs = append(productIDs, it.ID)
	}

	type stockResult struct {
		ProductID  int64 `gorm:"column:product_id"`
		TotalStock int   `gorm:"column:total_stock"`
	}
	var results []stockResult
	err := s.db.WithContext(ctx).Table("product_sku AS s").
		Select("s.product_id, COALESCE(SUM(CASE WHEN i.stock > i.locked_stock THEN i.stock - i.locked_stock ELSE 0 END), 0) AS total_stock").
		Joins("JOIN product_sku_inventory AS i ON i.sku_id = s.id").
		Where("s.product_id IN ? AND s.status = 1 AND s.deleted_at IS NULL", productIDs).
		Group("s.product_id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	stockMap := make(map[int64]int, len(results))
	for _, r := range results {
		stockMap[r.ProductID] = r.TotalStock
	}

	for i := range items {
		items[i].Stock = stockMap[items[i].ID]
	}

	return items, nil
}

// Detail retrieves product detail by id. Only published and non-deleted products with valid SKUs are returned.
// Static detail is cached via rueidisaside; live inventory is fetched on response.
func (s *Service) Detail(ctx context.Context, id int64) (*ProductDetailResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if id <= 0 {
		return nil, ErrProductNotFoundOrOffShelf
	}
	if s.db == nil {
		return nil, ErrDatabaseUnavailable
	}

	cacheKey := cache.ProductDetailKey(id)
	staticDetail, err := cache.Get(ctx, s.cache, cacheKey, cache.TTLProductDetail, func(innerCtx context.Context) (*ProductDetailResponse, error) {
		return s.loadStaticDetailFromDB(innerCtx, id)
	})
	if err != nil {
		return nil, err
	}
	if staticDetail == nil {
		return nil, ErrProductNotFoundOrOffShelf
	}

	return s.fillDetailStock(ctx, staticDetail)
}

func (s *Service) loadStaticDetailFromDB(ctx context.Context, id int64) (*ProductDetailResponse, error) {
	if s.db == nil {
		return nil, ErrDatabaseUnavailable
	}

	var p model.Product
	if err := s.db.WithContext(ctx).
		Where("id = ? AND status = 1 AND deleted_at IS NULL", id).
		First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var skuModels []model.ProductSKU
	if err := s.db.WithContext(ctx).
		Where("product_id = ? AND status = 1 AND deleted_at IS NULL", p.ID).
		Order("sort ASC, id ASC").
		Find(&skuModels).Error; err != nil {
		return nil, err
	}
	if len(skuModels) == 0 {
		return nil, nil
	}

	var galleryIDs []int64
	if p.GalleryFileIDs != nil && *p.GalleryFileIDs != "" {
		_ = json.Unmarshal([]byte(*p.GalleryFileIDs), &galleryIDs)
	}

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
		if u, ok := urlMap[gid]; ok && u != "" {
			galleryURLs = append(galleryURLs, u)
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

	detail := ""
	if p.Detail != nil {
		detail = *p.Detail
	}

	skus := make([]ProductSKUItem, 0, len(skuModels))
	var lowestSKU *model.ProductSKU
	var minCents int64 = -1

	for i := range skuModels {
		sm := &skuModels[i]

		cents, err := util.ParseDecimalYuanToCent(sm.Price)
		if err == nil {
			if minCents < 0 || cents < minCents {
				minCents = cents
				lowestSKU = sm
			}
		}

		priceStr := sm.Price
		if norm, err := util.NormalizeDecimalYuan(sm.Price); err == nil {
			priceStr = norm
		}
		marketPriceStr := sm.MarketPrice
		if norm, err := util.NormalizeDecimalYuan(sm.MarketPrice); err == nil {
			marketPriceStr = norm
		}

		var skuImageURL string
		if sm.ImageID != nil {
			skuImageURL = urlMap[*sm.ImageID]
		}

		specValues := make([]ProductSKUValue, 0)
		if sm.SpecValues != "" {
			_ = json.Unmarshal([]byte(sm.SpecValues), &specValues)
		}

		skus = append(skus, ProductSKUItem{
			ID:          sm.ID,
			SpecValues:  specValues,
			Price:       priceStr,
			MarketPrice: marketPriceStr,
			ImageURL:    skuImageURL,
			Unit:        sm.Unit,
			WeightGram:  sm.WeightGram,
			Stock:       0, // dynamic stock filled in fillDetailStock
		})
	}

	productPrice := ""
	productMarketPrice := ""
	if lowestSKU != nil {
		productPrice = util.FormatCentToYuan(minCents)
		if norm, err := util.NormalizeDecimalYuan(lowestSKU.MarketPrice); err == nil {
			productMarketPrice = norm
		} else {
			productMarketPrice = lowestSKU.MarketPrice
		}
	}

	sales := p.SalesCount + int64(p.VirtualSales)

	return &ProductDetailResponse{
		ID:           p.ID,
		CategoryID:   p.CategoryID,
		Name:         p.Name,
		Subtitle:     p.Subtitle,
		MainImageURL: mainImageURL,
		GalleryURLs:  galleryURLs,
		Price:        productPrice,
		MarketPrice:  productMarketPrice,
		Sales:        sales,
		Specs:        specs,
		Params:       params,
		SKUs:         skus,
		Detail:       detail,
	}, nil
}

func (s *Service) fillDetailStock(ctx context.Context, staticDetail *ProductDetailResponse) (*ProductDetailResponse, error) {
	if staticDetail == nil {
		return nil, nil
	}
	if s.db == nil {
		return nil, ErrDatabaseUnavailable
	}

	resp := *staticDetail
	resp.SKUs = make([]ProductSKUItem, len(staticDetail.SKUs))
	copy(resp.SKUs, staticDetail.SKUs)

	if len(resp.SKUs) == 0 {
		return &resp, nil
	}

	skuIDs := make([]int64, 0, len(resp.SKUs))
	for _, sku := range resp.SKUs {
		skuIDs = append(skuIDs, sku.ID)
	}

	var inventories []model.ProductSkuInventory
	if err := s.db.WithContext(ctx).
		Where("sku_id IN ?", skuIDs).
		Find(&inventories).Error; err != nil {
		return nil, err
	}

	invMap := make(map[int64]model.ProductSkuInventory, len(inventories))
	for _, inv := range inventories {
		invMap[inv.SKUID] = inv
	}

	for i := range resp.SKUs {
		sku := &resp.SKUs[i]
		skuStock := 0
		if inv, ok := invMap[sku.ID]; ok {
			avail := inv.Stock - inv.LockedStock
			if avail > 0 {
				skuStock = avail
			}
		}
		sku.Stock = skuStock
	}

	return &resp, nil
}
