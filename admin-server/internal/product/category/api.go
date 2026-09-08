package category

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"github.com/gin-gonic/gin"
	"mall/admin_core/pkg/response"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// List godoc
// @Summary List all product categories (flat array)
// @ID listProductCategories
// @Tags product-category
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[[]ProductCategoryRow]
// @Router /api/product/categories/list [post]
func (a *API) List(c *gin.Context) {
	result, err := a.service.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, len(result))
}

// Options godoc
// @Summary Get lightweight product category options
// @ID getProductCategoryOptions
// @Tags product-category
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[[]ProductCategoryOption]
// @Router /api/product/categories/options [post]
func (a *API) Options(c *gin.Context) {
	result, err := a.service.Options(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, len(result))
}

// Create godoc
// @Summary Create a product category
// @ID createProductCategory
// @Tags product-category
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProductCategoryCreateRequest true "Create category request"
// @Success 200 {object} response.Response[ProductCategoryRow]
// @Router /api/product/categories/create [post]
func (a *API) Create(c *gin.Context) {
	var req CreateRequest
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
// @Summary Update a product category
// @ID updateProductCategory
// @Tags product-category
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProductCategoryUpdateRequest true "Update category request"
// @Success 200 {object} response.Response[ProductCategoryRow]
// @Router /api/product/categories/update [post]
func (a *API) Update(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, "invalid request body")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var req UpdateRequest
	if err := json.Unmarshal(body, &req); err != nil || req.ID <= 0 {
		response.Error(c, "invalid request")
		return
	}

	result, err := a.service.Update(c.Request.Context(), body, req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// Delete godoc
// @Summary Delete a product category
// @ID deleteProductCategory
// @Tags product-category
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProductCategoryDeleteRequest true "Delete category request"
// @Success 200 {object} response.Response[any]
// @Router /api/product/categories/delete [post]
func (a *API) Delete(c *gin.Context) {
	var req DeleteRequest
	if !bindJSON(c, &req) {
		return
	}
	if req.ID <= 0 {
		response.Error(c, "invalid category id")
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
		errors.Is(err, ErrCategoryNotFound),
		errors.Is(err, ErrParentCategoryNotFound),
		errors.Is(err, ErrParentCategorySelf),
		errors.Is(err, ErrParentCategoryCycle),
		errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrHasSubcategories),
		errors.Is(err, ErrCategoryInUse),
		errors.Is(err, ErrImageNotFound):
		response.Error(c, err.Error())
	default:
		response.Error(c, err.Error())
	}
}
