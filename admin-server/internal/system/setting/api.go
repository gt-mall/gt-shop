package setting

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
	response.Error(c, err.Error())
}

// GetShop godoc
// @Summary 获取商城基础设置
// @ID getSystemShopSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[ShopBasicConfig]
// @Router /api/system/settings/shop [get]
func (a *API) GetShop(c *gin.Context) {
	result, err := a.service.GetShop(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveShop godoc
// @Summary 保存商城基础设置
// @ID saveSystemShopSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ShopBasicConfig true "Shop basic settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/shop/save [post]
func (a *API) SaveShop(c *gin.Context) {
	var request ShopBasicConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveShop(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// GetSite godoc
// @Summary 获取站点信息设置
// @ID getSystemSiteSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[SiteBasicConfig]
// @Router /api/system/settings/site [get]
func (a *API) GetSite(c *gin.Context) {
	result, err := a.service.GetSite(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveSite godoc
// @Summary 保存站点信息设置
// @ID saveSystemSiteSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SiteBasicConfig true "Site basic settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/site/save [post]
func (a *API) SaveSite(c *gin.Context) {
	var request SiteBasicConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveSite(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// GetMp godoc
// @Summary 获取小程序设置
// @ID getSystemMpSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[MpConfig]
// @Router /api/system/settings/mp [get]
func (a *API) GetMp(c *gin.Context) {
	result, err := a.service.GetMp(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveMp godoc
// @Summary 保存小程序设置
// @ID saveSystemMpSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MpConfig true "Mini program settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/mp/save [post]
func (a *API) SaveMp(c *gin.Context) {
	var request MpConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveMp(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}
