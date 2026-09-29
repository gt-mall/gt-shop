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

type ProductParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
} // @name ProductParam

type ProductCreateRequest struct {
	Name              string              `json:"name" binding:"required,max=255"`
	Subtitle          string              `json:"subtitle" binding:"max=255"`
	CategoryID        int64               `json:"category_id"`
	MainImageID       *int64              `json:"main_image_id"`
	GalleryFileIDs    []int64             `json:"gallery_file_ids"`
	Specs             []ProductSpec       `json:"specs"`
	Params            []ProductParam      `json:"params"`
	Detail            string              `json:"detail"`
	Sort              *int                `json:"sort"`
	Status            *int8               `json:"status"` // 0:下架, 1:上架
	VirtualSales      *int                `json:"virtual_sales"`
	FreightType       *int8               `json:"freight_type"`        // 1:包邮, 2:固定运费, 3:运费模板
	FreightFee        *string             `json:"freight_fee"`         // 固定运费金额
	FreightTemplateID *int64              `json:"freight_template_id"` // 运费模板ID
	SKUs              []ProductSKURequest `json:"skus" binding:"required"`
} // @name ProductCreateRequest

type ProductUpdateRequest struct {
	ID                int64               `json:"id" binding:"required"`
	Name              string              `json:"name" binding:"required,max=255"`
	Subtitle          string              `json:"subtitle" binding:"max=255"`
	CategoryID        int64               `json:"category_id"`
	MainImageID       *int64              `json:"main_image_id"`
	GalleryFileIDs    []int64             `json:"gallery_file_ids"`
	Specs             []ProductSpec       `json:"specs"`
	Params            []ProductParam      `json:"params"`
	Detail            string              `json:"detail"`
	Sort              *int                `json:"sort"`
	Status            *int8               `json:"status"`
	VirtualSales      *int                `json:"virtual_sales"`
	FreightType       *int8               `json:"freight_type"`
	FreightFee        *string             `json:"freight_fee"`
	FreightTemplateID *int64              `json:"freight_template_id"`
	SKUs              []ProductSKURequest `json:"skus" binding:"required"`
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

type ProductTabCountsResponse struct {
	All       int64 `json:"all"`
	Selling   int64 `json:"selling"`
	Warning   int64 `json:"warning"`
	SoldOut   int64 `json:"sold_out"`
	Warehouse int64 `json:"warehouse"`
} // @name ProductTabCountsResponse

type ProductListRequest struct {
	Page           int     `json:"page"`
	PageSize       int     `json:"page_size"`
	Tab            string  `json:"tab"` // all, selling, warning, sold_out, warehouse
	Keyword        string  `json:"keyword"`
	CategoryID     *int64  `json:"category_id"`
	Status         *int8   `json:"status"`
	ProductCode    string  `json:"product_code"`     // 商品编码/ID
	SKUCode        string  `json:"sku_code"`         // SKU编码/条形码
	MinPrice       *string `json:"min_price"`        // 最低价格（元）
	MaxPrice       *string `json:"max_price"`        // 最高价格（元）
	MinStock       *int    `json:"min_stock"`        // 最低库存
	MaxStock       *int    `json:"max_stock"`        // 最高库存
	MinSales       *int64  `json:"min_sales"`        // 最低销量
	MaxSales       *int64  `json:"max_sales"`        // 最高销量
	FreightType    *int8   `json:"freight_type"`     // 运费类型：1包邮，2固定运费，3运费模板
	CreatedAtStart *string `json:"created_at_start"` // 创建时间开始
	CreatedAtEnd   *string `json:"created_at_end"`   // 创建时间结束
	UpdatedAtStart *string `json:"updated_at_start"` // 更新时间开始
	UpdatedAtEnd   *string `json:"updated_at_end"`   // 更新时间结束
} // @name ProductListRequest

type ProductRow struct {
	ID                int64     `json:"id"`
	CategoryID        int64     `json:"category_id"`
	CategoryName      string    `json:"category_name"`
	Name              string    `json:"name"`
	Subtitle          string    `json:"subtitle"`
	MainImageID       *int64    `json:"main_image_id"`
	MainImageURL      string    `json:"main_image_url"`
	Status            int8      `json:"status"`
	Sort              int       `json:"sort"`
	SalesCount        int64     `json:"sales_count"`
	VirtualSales      int       `json:"virtual_sales"`
	SKUCount          int       `json:"sku_count"`
	MinPrice          *string   `json:"min_price"`
	MaxPrice          *string   `json:"max_price"`
	FreightType       int8      `json:"freight_type"`
	FreightFee        string    `json:"freight_fee"`
	FreightTemplateID *int64    `json:"freight_template_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
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
	ID                  int64          `json:"id"`
	CategoryID          int64          `json:"category_id"`
	CategoryName        string         `json:"category_name"`
	Name                string         `json:"name"`
	Subtitle            string         `json:"subtitle"`
	MainImageID         *int64         `json:"main_image_id"`
	MainImageURL        string         `json:"main_image_url"`
	GalleryFileIDs      []int64        `json:"gallery_file_ids"`
	GalleryImageURLs    []string       `json:"gallery_image_urls"`
	Specs               []ProductSpec  `json:"specs"`
	Params              []ProductParam `json:"params"`
	Detail              string         `json:"detail"`
	Status              int8           `json:"status"`
	Sort                int            `json:"sort"`
	SalesCount          int64          `json:"sales_count"`
	VirtualSales        int            `json:"virtual_sales"`
	FreightType         int8           `json:"freight_type"`
	FreightFee          string         `json:"freight_fee"`
	FreightTemplateID   *int64         `json:"freight_template_id"`
	FreightTemplateName *string        `json:"freight_template_name"`
	SKUs                []ProductSKU   `json:"skus"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
} // @name ProductDetail
