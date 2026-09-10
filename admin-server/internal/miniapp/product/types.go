package product

// ProductListRequest represents the query parameters for product listing.
type ProductListRequest struct {
	Page       int    `form:"page" json:"page"`
	PageSize   int    `form:"page_size" json:"page_size"`
	Keyword    string `form:"keyword" json:"keyword"`
	CategoryID *int64 `form:"category_id" json:"category_id"`
	Sort       string `form:"sort" json:"sort"` // default, sales, price_asc, price_desc
} // @name MiniappProductListRequest

// ProductListItem represents a product item in the miniapp product list.
type ProductListItem struct {
	ID           int64  `json:"id"`
	CategoryID   int64  `json:"category_id"`
	Name         string `json:"name"`
	Subtitle     string `json:"subtitle"`
	MainImageURL string `json:"main_image_url"`
	Price        string `json:"price"`
	MarketPrice  string `json:"market_price"`
	Sales        int64  `json:"sales"`
	Stock        int    `json:"stock"`
} // @name MiniappProductListItem

// ProductSpecValue represents an option value within a specification dimension.
type ProductSpecValue struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
} // @name MiniappProductSpecValue

// ProductSpec represents a product specification dimension.
type ProductSpec struct {
	ID     int64              `json:"id"`
	Name   string             `json:"name"`
	Values []ProductSpecValue `json:"values"`
} // @name MiniappProductSpec

// ProductSKUValue maps a spec ID to a value ID.
type ProductSKUValue struct {
	SpecID  int64 `json:"spec_id"`
	ValueID int64 `json:"value_id"`
} // @name MiniappProductSKUValue

// ProductParam represents a key-value parameter of the product.
type ProductParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
} // @name MiniappProductParam

// ProductSKUItem represents a SKU item in the miniapp product detail.
type ProductSKUItem struct {
	ID          int64             `json:"id"`
	SpecValues  []ProductSKUValue `json:"spec_values"`
	Price       string            `json:"price"`
	MarketPrice string            `json:"market_price"`
	ImageURL    string            `json:"image_url"`
	Unit        *string           `json:"unit" extensions:"x-nullable"`
	WeightGram  *int              `json:"weight_gram" extensions:"x-nullable"`
	Stock       int               `json:"stock"`
} // @name MiniappProductSKUItem

// ProductDetailResponse represents the product detail in miniapp.
type ProductDetailResponse struct {
	ID           int64            `json:"id"`
	CategoryID   int64            `json:"category_id"`
	Name         string           `json:"name"`
	Subtitle     string           `json:"subtitle"`
	MainImageURL string           `json:"main_image_url"`
	GalleryURLs  []string         `json:"gallery_urls"`
	Price        string           `json:"price"`
	MarketPrice  string           `json:"market_price"`
	Sales        int64            `json:"sales"`
	Specs        []ProductSpec    `json:"specs"`
	Params       []ProductParam   `json:"params"`
	SKUs         []ProductSKUItem `json:"skus"`
	Detail       string           `json:"detail"`
} // @name MiniappProductDetailResponse
