package setting

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"mall/admin_core/internal/order/wechatpay"
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
	result.AppSecret = ""
	result.UploadPrivateKey = ""
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

// GetSearch godoc
// @Summary 获取搜索与热门词设置
// @ID getSystemSearchSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[SearchBasicConfig]
// @Router /api/system/settings/search [get]
func (a *API) GetSearch(c *gin.Context) {
	result, err := a.service.GetSearch(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveSearch godoc
// @Summary 保存搜索与热门词设置
// @ID saveSystemSearchSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SearchBasicConfig true "Search basic settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/search/save [post]
func (a *API) SaveSearch(c *gin.Context) {
	var request SearchBasicConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveSearch(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// GetWechatPay godoc
// @Summary 获取微信支付配置状态（不返回密钥）
// @ID getSystemWechatPaySetting
// @Tags system-setting
// @Security BearerAuth
// @Success 200 {object} response.Response[wechatpay.PublicConfig]
// @Router /api/system/settings/wechat-pay [get]
func (a *API) GetWechatPay(c *gin.Context) {
	cfg, err := (wechatpay.Store{DB: a.service.db}).Load(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, cfg.Public(), 0)
}

// SaveWechatPay godoc
// @Summary 保存微信支付商户配置
// @ID saveSystemWechatPaySetting
// @Tags system-setting
// @Accept json
// @Security BearerAuth
// @Param request body wechatpay.Config true "商户配置；密钥留空表示保持原值"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/wechat-pay/save [post]
func (a *API) SaveWechatPay(c *gin.Context) {
	var req wechatpay.Config
	if !bindJSON(c, &req) {
		return
	}
	if err := (wechatpay.Store{DB: a.service.db}).Save(c.Request.Context(), req); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// GetTrade godoc
// @Summary 获取交易基础设置
// @ID getSystemTradeSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[TradeBasicConfig]
// @Router /api/system/settings/trade [get]
func (a *API) GetTrade(c *gin.Context) {
	result, err := a.service.GetTrade(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveTrade godoc
// @Summary 保存交易基础设置
// @ID saveSystemTradeSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body TradeBasicConfig true "Trade basic settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/trade/save [post]
func (a *API) SaveTrade(c *gin.Context) {
	var request TradeBasicConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveTrade(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// GetPayment godoc
// @Summary 获取支付方式配置
// @ID getSystemPaymentSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[PaymentBasicConfig]
// @Router /api/system/settings/payment [get]
func (a *API) GetPayment(c *gin.Context) {
	result, err := a.service.GetPayment(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SavePayment godoc
// @Summary 保存支付方式配置
// @ID saveSystemPaymentSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PaymentBasicConfig true "Payment basic settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/payment/save [post]
func (a *API) SavePayment(c *gin.Context) {
	var request PaymentBasicConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SavePayment(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// GetLogistics godoc
// @Summary 获取配送基础设置
// @ID getSystemLogisticsSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[LogisticsBasicConfig]
// @Router /api/system/settings/logistics [get]
func (a *API) GetLogistics(c *gin.Context) {
	result, err := a.service.GetLogistics(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveLogistics godoc
// @Summary 保存配送基础设置
// @ID saveSystemLogisticsSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body LogisticsBasicConfig true "Logistics basic settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/logistics/save [post]
func (a *API) SaveLogistics(c *gin.Context) {
	var request LogisticsBasicConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveLogistics(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// GetMember godoc
// @Summary 获取用户注册与登录设置
// @ID getSystemMemberSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[MemberBasicConfig]
// @Router /api/system/settings/member [get]
func (a *API) GetMember(c *gin.Context) {
	result, err := a.service.GetMember(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveMember godoc
// @Summary 保存用户注册与登录设置
// @ID saveSystemMemberSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MemberBasicConfig true "Member basic settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/member/save [post]
func (a *API) SaveMember(c *gin.Context) {
	var request MemberBasicConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveMember(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// GetReview godoc
// @Summary 获取商品评价设置
// @ID getSystemReviewSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[ReviewBasicConfig]
// @Router /api/system/settings/review [get]
func (a *API) GetReview(c *gin.Context) {
	result, err := a.service.GetReview(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveReview godoc
// @Summary 保存商品评价设置
// @ID saveSystemReviewSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ReviewBasicConfig true "Review basic settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/review/save [post]
func (a *API) SaveReview(c *gin.Context) {
	var request ReviewBasicConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveReview(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}

// GetAgreement godoc
// @Summary 获取政策协议设置
// @ID getSystemAgreementSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[AgreementBasicConfig]
// @Router /api/system/settings/agreement [get]
func (a *API) GetAgreement(c *gin.Context) {
	result, err := a.service.GetAgreement(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.Data(c, result, 0)
}

// SaveAgreement godoc
// @Summary 保存政策协议设置
// @ID saveSystemAgreementSetting
// @Tags system-setting
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AgreementBasicConfig true "Agreement basic settings request"
// @Success 200 {object} response.Response[any]
// @Router /api/system/settings/agreement/save [post]
func (a *API) SaveAgreement(c *gin.Context) {
	var request AgreementBasicConfig
	if !bindJSON(c, &request) {
		return
	}
	if err := a.service.SaveAgreement(c.Request.Context(), request); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c)
}
