package product

import "time"

type ProductSpecValue struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
} // @name ProductSpecValue

type ProductSpec struct {
	ID     int64              `json:"id"`
	Name   string             `json:"name"`
	Values []ProductSpecValue `json:"values"`
} // @name ProductSpec

type ProductSKUValue struct {
	SpecID  int64 `json:"spec_id"`
	ValueID int64 `json:"value_id"`
} // @name ProductSKUValue

type ProductSKURequest struct {
	ID          int64             `json:"id"` // 0 for new SKU, >0 for existing SKU
	SpecValues  []ProductSKUValue `json:"spec_values"`
	SKUCode     *string           `json:"sku_code"`
	Barcode     *string           `json:"barcode"`
	ImageID     *int64            `json:"image_id"`
	Price       string            `json:"price"`        // e.g. "199.90"
	MarketPrice string            `json:"market_price"` // e.g. "199.90"
	CostPrice   string            `json:"cost_price"`   // e.g. "199.90"
	Unit        *string           `json:"unit"`
	WeightGram  *int              `json:"weight_gram"`
	Sort        *int              `json:"sort"`
	Status      *int8             `json:"status"` // 0:禁用, 1:启用
} // @name ProductSKURequest

type ProductCreateRequest struct {
	Name           string              `json:"name" binding:"required,max=255"`
	Subtitle       string              `json:"subtitle" binding:"max=255"`
	CategoryID     int64               `json:"category_id"`
	MainImageID    *int64              `json:"main_image_id"`
	GalleryFileIDs []int64             `json:"gallery_file_ids"`
	Specs          []ProductSpec       `json:"specs"`
	Detail         string              `json:"detail"`
	Sort           *int                `json:"sort"`
	Status         *int8               `json:"status"` // 0:下架, 1:上架
	VirtualSales   *int                `json:"virtual_sales"`
	SKUs           []ProductSKURequest `json:"skus" binding:"required"`
} // @name ProductCreateRequest

type ProductUpdateRequest struct {
	ID             int64               `json:"id" binding:"required"`
	Name           string              `json:"name" binding:"required,max=255"`
	Subtitle       string              `json:"subtitle" binding:"max=255"`
	CategoryID     int64               `json:"category_id"`
	MainImageID    *int64              `json:"main_image_id"`
	GalleryFileIDs []int64             `json:"gallery_file_ids"`
	Specs          []ProductSpec       `json:"specs"`
	Detail         string              `json:"detail"`
	Sort           *int                `json:"sort"`
	Status         *int8               `json:"status"`
	VirtualSales   *int                `json:"virtual_sales"`
	SKUs           []ProductSKURequest `json:"skus" binding:"required"`
} // @name ProductUpdateRequest

type ProductIDRequest struct {
	ID int64 `json:"id" binding:"required"`
} // @name ProductIDRequest

type ProductIDResponse struct {
	ID int64 `json:"id"`
} // @name ProductIDResponse

type ProductStatusRequest struct {
	ID     int64 `json:"id" binding:"required"`
	Status int8  `json:"status"` // 0:下架, 1:上架
} // @name ProductStatusRequest

type ProductListRequest struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	Keyword    string `json:"keyword"`
	CategoryID *int64 `json:"category_id"`
	Status     *int8  `json:"status"`
} // @name ProductListRequest

type ProductRow struct {
	ID           int64     `json:"id"`
	CategoryID   int64     `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Name         string    `json:"name"`
	Subtitle     string    `json:"subtitle"`
	MainImageID  *int64    `json:"main_image_id"`
	MainImageURL string    `json:"main_image_url"`
	Status       int8      `json:"status"`
	Sort         int       `json:"sort"`
	SalesCount   int64     `json:"sales_count"`
	VirtualSales int       `json:"virtual_sales"`
	SKUCount     int       `json:"sku_count"`
	MinPrice     *string   `json:"min_price"`
	MaxPrice     *string   `json:"max_price"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
} // @name ProductRow

type ProductSKU struct {
	ID          int64             `json:"id"`
	ProductID   int64             `json:"product_id"`
	SpecValues  []ProductSKUValue `json:"spec_values"`
	SKUCode     *string           `json:"sku_code"`
	Barcode     *string           `json:"barcode"`
	ImageID     *int64            `json:"image_id"`
	ImageURL    string            `json:"image_url"`
	Price       string            `json:"price"`
	MarketPrice string            `json:"market_price"`
	CostPrice   string            `json:"cost_price"`
	Unit        *string           `json:"unit"`
	WeightGram  *int              `json:"weight_gram"`
	Sort        int               `json:"sort"`
	Status      int8              `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
} // @name ProductSKU

type ProductDetail struct {
	ID               int64             `json:"id"`
	CategoryID       int64             `json:"category_id"`
	CategoryName     string            `json:"category_name"`
	Name             string            `json:"name"`
	Subtitle         string            `json:"subtitle"`
	MainImageID      *int64            `json:"main_image_id"`
	MainImageURL     string            `json:"main_image_url"`
	GalleryFileIDs   []int64           `json:"gallery_file_ids"`
	GalleryImageURLs []string          `json:"gallery_image_urls"`
	Specs            []ProductSpec     `json:"specs"`
	Detail           string            `json:"detail"`
	Status           int8              `json:"status"`
	Sort             int               `json:"sort"`
	SalesCount       int64             `json:"sales_count"`
	VirtualSales     int               `json:"virtual_sales"`
	SKUs             []ProductSKU      `json:"skus"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
} // @name ProductDetail
