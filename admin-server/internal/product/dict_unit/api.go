package dict_unit

import (
	"errors"
	"strconv"

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
// @Summary 分页获取商品单位列表
// @ID listProductDictUnits
// @Tags product-dict-unit
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "关键词"
// @Param status query int false "状态: 0禁用, 1启用"
// @Success 200 {object} response.Response[[]ProductDictUnitRow]
// @Router /api/product/dict-units [get]
func (a *API) List(c *gin.Context) {
	var req DictUnitListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = DictUnitListRequest{Page: 1, PageSize: 20}
	}
	rows, total, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, rows, int(total))
}

// Create godoc
// @Summary 创建商品单位
// @ID createProductDictUnit
// @Tags product-dict-unit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body DictUnitCreateRequest true "创建商品单位参数"
// @Success 200 {object} response.Response[ProductDictUnitRow]
// @Router /api/product/dict-unit [post]
func (a *API) Create(c *gin.Context) {
	var req DictUnitCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数错误: "+err.Error())
		return
	}
	row, err := a.service.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, row, 0)
}

// Update godoc
// @Summary 更新商品单位
// @ID updateProductDictUnit
// @Tags product-dict-unit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "商品单位ID"
// @Param request body DictUnitUpdateRequest true "更新商品单位参数"
// @Success 200 {object} response.Response[ProductDictUnitRow]
// @Router /api/product/dict-unit/{id} [put]
func (a *API) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, "无效的商品单位ID")
		return
	}

	var req DictUnitUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数错误: "+err.Error())
		return
	}

	row, err := a.service.Update(c.Request.Context(), id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, row, 0)
}

// Delete godoc
// @Summary 删除商品单位
// @ID deleteProductDictUnit
// @Tags product-dict-unit
// @Produce json
// @Security BearerAuth
// @Param id path int true "商品单位ID"
// @Success 200 {object} response.Response[any]
// @Router /api/product/dict-unit/{id} [delete]
func (a *API) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, "无效的商品单位ID")
		return
	}

	if err := a.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrDatabaseUnavailable),
		errors.Is(err, ErrNotFound),
		errors.Is(err, ErrNameRequired),
		errors.Is(err, ErrNameTooLong),
		errors.Is(err, ErrNameExists),
		errors.Is(err, ErrDescriptionTooLong),
		errors.Is(err, ErrInvalidStatus):
		response.Error(c, err.Error())
	default:
		response.Error(c, err.Error())
	}
}
