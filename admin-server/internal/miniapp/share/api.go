package share

import (
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

// ProductShareCode godoc
// @Summary 获取商品分享小程序码
// @ID miniappProductShareCode
// @Tags share
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} response.Response[ProductShareCodeResponse]
// @Failure 200 {object} response.Response[any]
// @Router /products/{id}/share-code [post]
func (a *API) ProductShareCode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, "无效的商品ID")
		return
	}

	resp, err := a.service.GetOrCreateProductShareCode(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, resp, 0)
}
