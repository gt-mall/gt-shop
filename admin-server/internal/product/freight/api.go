package freight

import (
	"errors"

	"github.com/gin-gonic/gin"

	"mall/admin_core/internal/region"
	"mall/admin_core/pkg/response"
)

type RegionNode = region.RegionNode

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// List godoc
// @Summary 分页获取运费模板列表
// @ID listFreightTemplates
// @Tags product-freight
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body FreightTemplateListRequest false "查询参数"
// @Success 200 {object} response.Response[[]FreightTemplateRow]
// @Router /api/product/freight-templates/list [post]
func (a *API) List(c *gin.Context) {
	var req FreightTemplateListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = FreightTemplateListRequest{Page: 1, PageSize: 20}
	}
	list, total, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, list, int(total))
}

// Options godoc
// @Summary 获取启用的运费模板选项
// @ID getFreightTemplateOptions
// @Tags product-freight
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[[]FreightTemplateOption]
// @Router /api/product/freight-templates/options [post]
func (a *API) Options(c *gin.Context) {
	opts, err := a.service.Options(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, opts, len(opts))
}

// Detail godoc
// @Summary 获取运费模板详情
// @ID getFreightTemplateDetail
// @Tags product-freight
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body FreightTemplateIDRequest true "模板ID参数"
// @Success 200 {object} response.Response[FreightTemplateDetail]
// @Router /api/product/freight-templates/detail [post]
func (a *API) Detail(c *gin.Context) {
	var req FreightTemplateIDRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID <= 0 {
		response.Error(c, "模板ID无效")
		return
	}
	detail, err := a.service.Detail(c.Request.Context(), req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, detail, 0)
}

// Create godoc
// @Summary 创建运费模板
// @ID createFreightTemplate
// @Tags product-freight
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body FreightTemplateCreateRequest true "模板内容"
// @Success 200 {object} response.Response[FreightTemplateIDRequest]
// @Router /api/product/freight-templates/create [post]
func (a *API) Create(c *gin.Context) {
	var req FreightTemplateCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效: "+err.Error())
		return
	}
	id, err := a.service.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, FreightTemplateIDRequest{ID: id}, 0)
}

// Update godoc
// @Summary 更新运费模板
// @ID updateFreightTemplate
// @Tags product-freight
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body FreightTemplateUpdateRequest true "模板内容"
// @Success 200 {object} response.Response[any]
// @Router /api/product/freight-templates/update [post]
func (a *API) Update(c *gin.Context) {
	var req FreightTemplateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效: "+err.Error())
		return
	}
	if err := a.service.Update(c.Request.Context(), req); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Status godoc
// @Summary 切换运费模板启用状态
// @ID updateFreightTemplateStatus
// @Tags product-freight
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body FreightTemplateStatusRequest true "状态参数"
// @Success 200 {object} response.Response[any]
// @Router /api/product/freight-templates/status [post]
func (a *API) Status(c *gin.Context) {
	var req FreightTemplateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID <= 0 {
		response.Error(c, "请求参数无效")
		return
	}
	if err := a.service.Status(c.Request.Context(), req.ID, req.Status); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Delete godoc
// @Summary 删除运费模板
// @ID deleteFreightTemplate
// @Tags product-freight
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body FreightTemplateIDRequest true "模板ID参数"
// @Success 200 {object} response.Response[any]
// @Router /api/product/freight-templates/delete [post]
func (a *API) Delete(c *gin.Context) {
	var req FreightTemplateIDRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID <= 0 {
		response.Error(c, "模板ID无效")
		return
	}
	if err := a.service.Delete(c.Request.Context(), req.ID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// Regions godoc
// @Summary 获取全国省市区三级行政区划树（用于运费模板地区配置）
// @ID getFreightRegions
// @Tags product-freight
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[[]region.RegionNode]
// @Router /api/product/freight-templates/regions [post]
func (a *API) Regions(c *gin.Context) {
	regions := a.service.GetRegions()
	response.Data(c, regions, len(regions))
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrDatabaseUnavailable),
		errors.Is(err, ErrTemplateNotFound),
		errors.Is(err, ErrTemplateDisabled),
		errors.Is(err, ErrTemplateInUse),
		errors.Is(err, ErrInvalidTemplateName),
		errors.Is(err, ErrInvalidBaseFee),
		errors.Is(err, ErrDuplicateRule),
		errors.Is(err, ErrConflictRule),
		errors.Is(err, ErrInvalidRuleFee),
		errors.Is(err, ErrInvalidProvinceCode):
		response.Error(c, err.Error())
	default:
		response.Error(c, err.Error())
	}
}
