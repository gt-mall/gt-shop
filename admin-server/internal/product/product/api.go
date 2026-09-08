package product

import (
	"errors"

	"github.com/gin-gonic/gin"
	"mall/admin_core/pkg/response"
	"mall/admin_core/pkg/util"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// List godoc
// @Summary List products with pagination and filters
// @ID listProducts
// @Tags product
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProductListRequest true "List products request"
// @Success 200 {object} response.Response[[]ProductRow]
// @Router /api/product/products/list [post]
func (a *API) List(c *gin.Context) {
	var req ProductListRequest
	if !bindJSON(c, &req) {
		return
	}
	result, total, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, int(total))
}

// Detail godoc
// @Summary Get product detail with all SKUs
// @ID getProductDetail
// @Tags product
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProductIDRequest true "Get product request"
// @Success 200 {object} response.Response[ProductDetail]
// @Router /api/product/products/detail [post]
func (a *API) Detail(c *gin.Context) {
	var req ProductIDRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.ID <= 0 {
		response.Error(c, "invalid product id")
		return
	}
	result, err := a.service.Detail(c.Request.Context(), req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// Create godoc
// @Summary Create a product with skus
// @ID createProduct
// @Tags product
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProductCreateRequest true "Create product request"
// @Success 200 {object} response.Response[ProductIDResponse]
// @Router /api/product/products/create [post]
func (a *API) Create(c *gin.Context) {
	var req ProductCreateRequest
	if !bindJSON(c, &req) {
		return
	}
	result, err := a.service.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// Update godoc
// @Summary Update a product and all its skus
// @ID updateProduct
// @Tags product
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProductUpdateRequest true "Update product request"
// @Success 200 {object} response.Response[ProductIDResponse]
// @Router /api/product/products/update [post]
func (a *API) Update(c *gin.Context) {
	var req ProductUpdateRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.ID <= 0 {
		response.Error(c, "invalid product id")
		return
	}
	result, err := a.service.Update(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// Status godoc
// @Summary Modify product online/offline status
// @ID updateProductStatus
// @Tags product
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProductStatusRequest true "Product status request"
// @Success 200 {object} response.Response[any]
// @Router /api/product/products/status [post]
func (a *API) Status(c *gin.Context) {
	var req ProductStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.ID <= 0 {
		response.Error(c, "invalid product id")
		return
	}
	if err := a.service.Status(c.Request.Context(), req); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Delete godoc
// @Summary Soft delete a product and its skus
// @ID deleteProduct
// @Tags product
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProductIDRequest true "Delete product request"
// @Success 200 {object} response.Response[any]
// @Router /api/product/products/delete [post]
func (a *API) Delete(c *gin.Context) {
	var req ProductIDRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.ID <= 0 {
		response.Error(c, "invalid product id")
		return
	}
	if err := a.service.Delete(c.Request.Context(), req.ID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

func bindJSON(c *gin.Context, request any) bool {
	if err := c.ShouldBindJSON(request); err != nil {
		response.Error(c, "invalid request")
		return false
	}
	return true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrDatabaseUnavailable),
		errors.Is(err, ErrProductNotFound),
		errors.Is(err, ErrInvalidProductName),
		errors.Is(err, ErrInvalidSubtitle),
		errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrInvalidVirtualSales),
		errors.Is(err, ErrInvalidWeight),
		errors.Is(err, ErrTooManyGalleryImages),
		errors.Is(err, ErrImageNotFound),
		errors.Is(err, ErrCategoryNotFound),
		errors.Is(err, ErrCategoryDisabled),
		errors.Is(err, ErrMainImageRequired),
		errors.Is(err, ErrEnabledSKURequired),
		errors.Is(err, ErrEnabledSKUZeroPrice),
		errors.Is(err, ErrSKUNotBelongToProduct),
		errors.Is(err, ErrDuplicateSKUID),
		errors.Is(err, ErrSKUIdentityChanged),
		errors.Is(err, ErrMissingExistingSKU),
		errors.Is(err, ErrTooManySpecs),
		errors.Is(err, ErrTooManySpecValues),
		errors.Is(err, ErrTooManySKUs),
		errors.Is(err, ErrEmptySKUs),
		errors.Is(err, ErrInvalidSpecID),
		errors.Is(err, ErrInvalidValueID),
		errors.Is(err, ErrEmptySpecName),
		errors.Is(err, ErrEmptyValueName),
		errors.Is(err, ErrDuplicateSpecID),
		errors.Is(err, ErrDuplicateSpecName),
		errors.Is(err, ErrDuplicateValueID),
		errors.Is(err, ErrDuplicateValueName),
		errors.Is(err, ErrEmptySpecValues),
		errors.Is(err, ErrDefaultSKURequired),
		errors.Is(err, ErrDefaultSKUNoValues),
		errors.Is(err, ErrSKUMismatchSpecs),
		errors.Is(err, ErrDuplicateSKUCombo),
		errors.Is(err, util.ErrInvalidAmount),
		errors.Is(err, util.ErrNegativeAmount),
		errors.Is(err, util.ErrAmountTooLarge):
		response.Error(c, err.Error())
	default:
		response.Error(c, err.Error())
	}
}
