package cart

// AddRequest represents parameters for adding an item to the shopping cart.
type AddRequest struct {
	SKUID    int64 `json:"sku_id" binding:"required,gt=0"`
	Quantity int   `json:"quantity" binding:"required,gt=0"`
} // @name MiniappCartAddRequest

// UpdateQuantityRequest represents parameters for updating item quantity in the cart.
type UpdateQuantityRequest struct {
	ID       int64 `json:"id" binding:"required,gt=0"`
	Quantity int   `json:"quantity" binding:"required,gt=0"`
} // @name MiniappCartUpdateQuantityRequest

// DeleteRequest represents parameters for deleting one or more items from the cart.
type DeleteRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
} // @name MiniappCartDeleteRequest

// CartSpecValue represents an enriched SKU specification option.
type CartSpecValue struct {
	SpecID    int64  `json:"spec_id"`
	SpecName  string `json:"spec_name"`
	ValueID   int64  `json:"value_id"`
	ValueName string `json:"value_name"`
} // @name MiniappCartSpecValue

// CartItemResponse represents a single item returned in the cart list.
type CartItemResponse struct {
	ID                int64           `json:"id"`
	ProductID         int64           `json:"product_id"`
	SKUID             int64           `json:"sku_id"`
	ProductName       string          `json:"product_name"`
	SpecValues        []CartSpecValue `json:"spec_values"`
	ImageURL          string          `json:"image_url"`
	Price             string          `json:"price"`
	MarketPrice       string          `json:"market_price"`
	Quantity          int             `json:"quantity"`
	Stock             int             `json:"stock"`
	Available         bool            `json:"available"`
	UnavailableReason string          `json:"unavailable_reason"`
} // @name MiniappCartItemResponse
