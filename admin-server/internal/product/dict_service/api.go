package dict_service

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
// @Summary 分页获取服务保障列表
// @ID listProductDictServices
// @Tags product-dict-service
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "关键词"
// @Param status query int false "状态: 0禁用, 1启用"
// @Success 200 {object} response.Response[[]ProductDictServiceRow]
// @Router /api/product/dict-services [get]
func (a *API) List(c *gin.Context) {
	var req DictServiceListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req = DictServiceListRequest{Page: 1, PageSize: 20}
	}
	rows, total, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, rows, int(total))
}

// Create godoc
// @Summary 创建服务保障
// @ID createProductDictService
// @Tags product-dict-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body DictServiceCreateRequest true "创建服务保障参数"
// @Success 200 {object} response.Response[ProductDictServiceRow]
// @Router /api/product/dict-service [post]
func (a *API) Create(c *gin.Context) {
	var req DictServiceCreateRequest
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
// @Summary 更新服务保障
// @ID updateProductDictService
// @Tags product-dict-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "服务保障ID"
// @Param request body DictServiceUpdateRequest true "更新服务保障参数"
// @Success 200 {object} response.Response[ProductDictServiceRow]
// @Router /api/product/dict-service/{id} [put]
func (a *API) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, "无效的服务保障ID")
		return
	}

	var req DictServiceUpdateRequest
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
// @Summary 删除服务保障
// @ID deleteProductDictService
// @Tags product-dict-service
// @Produce json
// @Security BearerAuth
// @Param id path int true "服务保障ID"
// @Success 200 {object} response.Response[any]
// @Router /api/product/dict-service/{id} [delete]
func (a *API) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, "无效的服务保障ID")
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
