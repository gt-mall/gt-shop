package member

import (
	miniappauth "mall/admin_core/internal/miniapp/auth"
	"mall/admin_core/pkg/response"

	"github.com/gin-gonic/gin"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// Me godoc
// @Summary 获取当前会员信息
// @ID miniappMemberMe
// @Tags member
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[MiniappMemberInfo]
// @Router /member/me [get]
func (a *API) Me(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	result, err := a.service.GetProfile(c.Request.Context(), member.ID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, result, 0)
}

// Profile godoc
// @Summary 更新当前会员资料
// @ID miniappMemberProfile
// @Tags member
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MiniappUpdateProfileRequest true "更新资料请求"
// @Success 200 {object} response.Response[MiniappMemberInfo]
// @Router /member/profile [post]
func (a *API) Profile(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	result, err := a.service.UpdateProfile(c.Request.Context(), member.ID, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, result, 0)
}

// Avatar godoc
// @Summary 上传并更新会员头像
// @ID miniappMemberAvatar
// @Tags member
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "头像图片文件"
// @Success 200 {object} response.Response[MiniappAvatarResponse]
// @Router /member/avatar [post]
func (a *API) Avatar(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, "请上传有效的图片文件")
		return
	}
	defer file.Close()

	result, err := a.service.UpdateAvatar(c.Request.Context(), member.ID, header.Filename, file)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, result, 0)
}

// Addresses godoc
// @Summary 获取会员收货地址列表
// @ID miniappMemberAddresses
// @Tags member
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[[]MiniappAddressResponse]
// @Router /member/addresses [get]
func (a *API) Addresses(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	result, err := a.service.ListAddresses(c.Request.Context(), member.ID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, result, 0)
}

// CreateAddress godoc
// @Summary 创建收货地址
// @ID miniappMemberAddressCreate
// @Tags member
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MiniappAddressCreateRequest true "创建收货地址请求"
// @Success 200 {object} response.Response[MiniappAddressResponse]
// @Router /member/addresses/create [post]
func (a *API) CreateAddress(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req AddressCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	result, err := a.service.CreateAddress(c.Request.Context(), member.ID, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, result, 0)
}

// UpdateAddress godoc
// @Summary 更新收货地址
// @ID miniappMemberAddressUpdate
// @Tags member
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MiniappAddressUpdateRequest true "更新收货地址请求"
// @Success 200 {object} response.Response[MiniappAddressResponse]
// @Router /member/addresses/update [post]
func (a *API) UpdateAddress(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req AddressUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	result, err := a.service.UpdateAddress(c.Request.Context(), member.ID, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Data(c, result, 0)
}

// DeleteAddress godoc
// @Summary 删除收货地址
// @ID miniappMemberAddressDelete
// @Tags member
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MiniappAddressIDRequest true "删除收货地址请求"
// @Success 200 {object} response.Response[any]
// @Router /member/addresses/delete [post]
func (a *API) DeleteAddress(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req AddressIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	if err := a.service.DeleteAddress(c.Request.Context(), member.ID, req.ID); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c)
}

// SetDefaultAddress godoc
// @Summary 设置默认收货地址
// @ID miniappMemberAddressDefault
// @Tags member
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MiniappAddressIDRequest true "设置默认收货地址请求"
// @Success 200 {object} response.Response[any]
// @Router /member/addresses/default [post]
func (a *API) SetDefaultAddress(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req AddressIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数无效")
		return
	}

	if err := a.service.SetDefaultAddress(c.Request.Context(), member.ID, req.ID); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c)
}
