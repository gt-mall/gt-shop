package returnaddress

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"mall/admin_core/pkg/response"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

func bindJSON(c *gin.Context, request any) bool {
	if err := c.ShouldBindJSON(request); err != nil {
		response.Error(c, "invalid request body")
		return false
	}
	return true
}

func writeError(c *gin.Context, err error) {
	if errors.Is(err, ErrUnavailable) {
		c.JSON(http.StatusServiceUnavailable, response.Response[any]{
			Code:    0,
			Message: "database unavailable",
		})
		return
	}
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, response.Response[any]{
			Code:    0,
			Message: err.Error(),
		})
		return
	}
	response.Error(c, err.Error())
}

// List godoc
// @Summary 获取退货地址列表
// @ID listOrderReturnAddresses
// @Tags order-return-address
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ListReturnAddressesRequest false "List query parameters"
// @Success 200 {object} response.Response[[]ReturnAddressItem]
// @Router /api/order/return-addresses/list [post]
func (a *API) List(c *gin.Context) {
	var req ListReturnAddressesRequest
	_ = c.ShouldBindJSON(&req)
	items, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, items, len(items))
}

// Detail godoc
// @Summary 获取退货地址详情
// @ID getOrderReturnAddressDetail
// @Tags order-return-address
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body IDRequest true "Return address ID request"
// @Success 200 {object} response.Response[ReturnAddressItem]
// @Router /api/order/return-addresses/detail [post]
func (a *API) Detail(c *gin.Context) {
	var req IDRequest
	if !bindJSON(c, &req) {
		return
	}
	item, err := a.service.Get(c.Request.Context(), req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, item, 0)
}

// GetDefault godoc
// @Summary 获取默认退货地址
// @ID getDefaultOrderReturnAddress
// @Tags order-return-address
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[ReturnAddressItem]
// @Router /api/order/return-addresses/default [get]
func (a *API) GetDefault(c *gin.Context) {
	item, err := a.service.GetDefault(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, item, 0)
}

// Create godoc
// @Summary 新增退货地址
// @ID createOrderReturnAddress
// @Tags order-return-address
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateReturnAddressRequest true "Create return address request"
// @Success 200 {object} response.Response[int64]
// @Router /api/order/return-addresses/create [post]
func (a *API) Create(c *gin.Context) {
	var req CreateReturnAddressRequest
	if !bindJSON(c, &req) {
		return
	}
	id, err := a.service.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, id, 0)
}

// Update godoc
// @Summary 更新退货地址
// @ID updateOrderReturnAddress
// @Tags order-return-address
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateReturnAddressRequest true "Update return address request"
// @Success 200 {object} response.Response[any]
// @Router /api/order/return-addresses/update [post]
func (a *API) Update(c *gin.Context) {
	var req UpdateReturnAddressRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := a.service.Update(c.Request.Context(), req); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Delete godoc
// @Summary 删除退货地址
// @ID deleteOrderReturnAddress
// @Tags order-return-address
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body IDRequest true "Delete return address request"
// @Success 200 {object} response.Response[any]
// @Router /api/order/return-addresses/delete [post]
func (a *API) Delete(c *gin.Context) {
	var req IDRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := a.service.Delete(c.Request.Context(), req.ID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// SetDefault godoc
// @Summary 设置默认退货地址
// @ID setDefaultOrderReturnAddress
// @Tags order-return-address
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body IDRequest true "Set default return address request"
// @Success 200 {object} response.Response[any]
// @Router /api/order/return-addresses/set-default [post]
func (a *API) SetDefault(c *gin.Context) {
	var req IDRequest
	if !bindJSON(c, &req) {
		return
	}
	if err := a.service.SetDefault(c.Request.Context(), req.ID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}
