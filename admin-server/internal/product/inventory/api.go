package inventory

import (
	"github.com/gin-gonic/gin"

	"mall/admin_core/internal/auth"
	"mall/admin_core/pkg/response"
)

func ptr[T any](v T) *T {
	return &v
}

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// List godoc
// @Summary 按SKU分页查询商品库存
// @Description 支持商品、SKU、状态及库存状态过滤
// @Tags 商品库存
// @Accept json
// @Produce json
// @Param request body ProductInventoryListRequest true "查询参数"
// @Success 200 {object} response.Response[[]ProductInventoryRow]
// @Failure 200 {object} response.Response[any]
// @Router /api/product/inventories/list [post]
// @ID listProductInventories
func (a *API) List(c *gin.Context) {
	var req ProductInventoryListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败: "+err.Error())
		return
	}

	rows, total, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, rows, int(total))
}

// Detail godoc
// @Summary 获取单个SKU最新库存
// @Description 打开库存调整框前获取SKU最新库存信息
// @Tags 商品库存
// @Accept json
// @Produce json
// @Param request body ProductInventoryDetailRequest true "SKU ID"
// @Success 200 {object} response.Response[ProductInventoryRow]
// @Failure 200 {object} response.Response[any]
// @Router /api/product/inventories/detail [post]
// @ID getProductInventory
func (a *API) Detail(c *gin.Context) {
	var req ProductInventoryDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败: "+err.Error())
		return
	}

	row, bErr := a.service.Detail(c.Request.Context(), req.SKUID)
	if bErr != nil {
		response.Error(c, bErr.Message)
		return
	}

	response.Data(c, row, 1)
}

// Adjust godoc
// @Summary 调整商品库存
// @Description 支持人工入库、人工出库、盘点校正
// @Tags 商品库存
// @Accept json
// @Produce json
// @Param request body ProductInventoryAdjustRequest true "库存调整参数"
// @Success 200 {object} response.Response[ProductInventoryMutationResult]
// @Failure 200 {object} response.Response[ProductInventoryMutationResult]
// @Router /api/product/inventories/adjust [post]
// @ID adjustProductInventory
func (a *API) Adjust(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req ProductInventoryAdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSON(c, response.Response[ProductInventoryMutationResult]{
			Code:    0,
			Message: "参数解析失败: " + err.Error(),
			Data: ProductInventoryMutationResult{
				ErrorCode: ptr(ErrCodeInvalidArgument),
			},
		})
		return
	}

	res, bErr := a.service.Adjust(c.Request.Context(), user, req)
	if bErr != nil {
		response.JSON(c, response.Response[ProductInventoryMutationResult]{
			Code:    0,
			Message: bErr.Message,
			Data: ProductInventoryMutationResult{
				ErrorCode: &bErr.Code,
			},
		})
		return
	}

	response.JSON(c, response.Response[ProductInventoryMutationResult]{
		Code:    1,
		Message: "ok",
		Data:    res,
	})
}

// Warning godoc
// @Summary 设置库存预警阈值
// @Description 设置SKU可用库存预警阈值，0表示关闭预警
// @Tags 商品库存
// @Accept json
// @Produce json
// @Param request body ProductInventoryWarningRequest true "预警设置参数"
// @Success 200 {object} response.Response[ProductInventoryMutationResult]
// @Failure 200 {object} response.Response[ProductInventoryMutationResult]
// @Router /api/product/inventories/warning [post]
// @ID setProductInventoryWarning
func (a *API) Warning(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req ProductInventoryWarningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSON(c, response.Response[ProductInventoryMutationResult]{
			Code:    0,
			Message: "参数解析失败: " + err.Error(),
			Data: ProductInventoryMutationResult{
				ErrorCode: ptr(ErrCodeInvalidArgument),
			},
		})
		return
	}

	res, bErr := a.service.Warning(c.Request.Context(), user, req)
	if bErr != nil {
		response.JSON(c, response.Response[ProductInventoryMutationResult]{
			Code:    0,
			Message: bErr.Message,
			Data: ProductInventoryMutationResult{
				ErrorCode: &bErr.Code,
			},
		})
		return
	}

	response.JSON(c, response.Response[ProductInventoryMutationResult]{
		Code:    1,
		Message: "ok",
		Data:    res,
	})
}

// LogsList godoc
// @Summary 查询库存变更流水
// @Description 只读查询SKU库存与预警历史变更流水
// @Tags 商品库存
// @Accept json
// @Produce json
// @Param request body ProductInventoryLogListRequest true "流水查询参数"
// @Success 200 {object} response.Response[[]ProductInventoryLogRow]
// @Failure 200 {object} response.Response[any]
// @Router /api/product/inventories/logs/list [post]
// @ID listProductInventoryLogs
func (a *API) LogsList(c *gin.Context) {
	var req ProductInventoryLogListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败: "+err.Error())
		return
	}

	rows, total, err := a.service.ListLogs(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, rows, int(total))
}
