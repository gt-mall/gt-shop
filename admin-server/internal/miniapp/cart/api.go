package cart

import (
	"github.com/gin-gonic/gin"

	miniappauth "mall/admin_core/internal/miniapp/auth"
	"mall/admin_core/pkg/response"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// List godoc
// @Summary 获取购物车列表
// @ID miniappCartList
// @Tags cart
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[[]MiniappCartItemResponse]
// @Failure 200 {object} response.Response[any]
// @Router /cart [get]
func (a *API) List(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	items, err := a.service.List(c.Request.Context(), member.ID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, items, 0)
}

// Add godoc
// @Summary 加入购物车
// @ID miniappCartAdd
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MiniappCartAddRequest true "加入购物车参数"
// @Success 200 {object} response.Response[any]
// @Failure 200 {object} response.Response[any]
// @Router /cart/add [post]
func (a *API) Add(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req AddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	if err := a.service.Add(c.Request.Context(), member.ID, req); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c)
}

// UpdateQuantity godoc
// @Summary 修改购物车商品数量
// @ID miniappCartUpdateQuantity
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MiniappCartUpdateQuantityRequest true "修改数量参数"
// @Success 200 {object} response.Response[any]
// @Failure 200 {object} response.Response[any]
// @Router /cart/quantity [post]
func (a *API) UpdateQuantity(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req UpdateQuantityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	if err := a.service.UpdateQuantity(c.Request.Context(), member.ID, req); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c)
}

// Delete godoc
// @Summary 删除购物车商品 (支持批量)
// @ID miniappCartDelete
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MiniappCartDeleteRequest true "删除购物车参数"
// @Success 200 {object} response.Response[any]
// @Failure 200 {object} response.Response[any]
// @Router /cart/delete [post]
func (a *API) Delete(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	if err := a.service.Delete(c.Request.Context(), member.ID, req); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c)
}
