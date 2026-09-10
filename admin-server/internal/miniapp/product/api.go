package product

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
// @Summary 获取商品列表
// @ID miniappProductList
// @Tags product
// @Produce json
// @Param page query int false "页码，默认 1" default(1)
// @Param page_size query int false "每页条数，默认 20，最大 100" default(20)
// @Param keyword query string false "关键字搜索(商品名称、副标题)"
// @Param category_id query int false "商品分类ID"
// @Param sort query string false "排序方式: default, sales, price_asc, price_desc" default(default)
// @Success 200 {object} response.Response[[]MiniappProductListItem]
// @Router /products [get]
func (a *API) List(c *gin.Context) {
	var req ProductListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, "参数错误")
		return
	}

	items, total, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, items, int(total))
}

// Detail godoc
// @Summary 获取商品详情
// @ID miniappProductDetail
// @Tags product
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response[MiniappProductDetailResponse]
// @Failure 200 {object} response.Response[any]
// @Router /products/{id} [get]
func (a *API) Detail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, "商品不存在或已下架")
		return
	}

	item, err := a.service.Detail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrProductNotFoundOrOffShelf) {
			response.Error(c, "商品不存在或已下架")
			return
		}
		response.Error(c, err.Error())
		return
	}

	response.Data(c, item, 0)
}
