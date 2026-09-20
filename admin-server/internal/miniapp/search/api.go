package search

import (
	"github.com/gin-gonic/gin"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/response"
)

type API struct {
	settingService *systemsetting.Service
}

func NewAPI(settingService *systemsetting.Service) *API {
	return &API{settingService: settingService}
}

// Config godoc
// @Summary 获取小程序搜索公开配置（热门搜索关键词）
// @ID getMiniappSearchConfig
// @Tags miniapp-search
// @Accept json
// @Produce json
// @Success 200 {object} response.Response[systemsetting.MiniappSearchConfigResponse]
// @Router /api/miniapp/search/config [get]
func (a *API) Config(c *gin.Context) {
	result, err := a.settingService.GetPublicSearchConfig(c.Request.Context())
	if err != nil {
		// 容错降级返回空列表，不阻断前端搜索
		response.Data(c, systemsetting.MiniappSearchConfigResponse{
			HotKeywords: make([]string, 0),
		}, 0)
		return
	}
	response.Data(c, result, 0)
}
