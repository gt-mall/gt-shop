package aftersale

import (
	"net/http"

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

// Apply godoc
// @Summary 申请售后
// @ID miniappAfterSaleApply
// @Tags aftersale
// @Accept json
// @Produce json
// @Param request body MiniappApplyAfterSaleRequest true "申请售后参数"
// @Success 200 {object} response.Response[MiniappAfterSaleDetailResponse]
// @Router /after-sales/apply [post]
func (a *API) Apply(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req MiniappApplyAfterSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	as, err := a.service.Apply(c.Request.Context(), member.ID, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	detail, err := a.service.Detail(c.Request.Context(), member.ID, as.ID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, detail, 0)
}

// List godoc
// @Summary 获取我的售后单列表
// @ID miniappAfterSaleList
// @Tags aftersale
// @Accept json
// @Produce json
// @Param request body MiniappAfterSaleListRequest true "查询参数"
// @Success 200 {object} response.Response[[]MiniappAfterSaleListItemResponse]
// @Router /after-sales/list [post]
func (a *API) List(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req MiniappAfterSaleListRequest
	_ = c.ShouldBindJSON(&req)

	rows, total, err := a.service.List(c.Request.Context(), member.ID, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, rows, int(total))
}

// Detail godoc
// @Summary 获取售后单详情
// @ID miniappAfterSaleDetail
// @Tags aftersale
// @Accept json
// @Produce json
// @Param request body MiniappAfterSaleDetailRequest true "售后单ID参数"
// @Success 200 {object} response.Response[MiniappAfterSaleDetailResponse]
// @Router /after-sales/detail [post]
func (a *API) Detail(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req MiniappAfterSaleDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	detail, err := a.service.Detail(c.Request.Context(), member.ID, req.AfterSaleID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, detail, 0)
}

// Cancel godoc
// @Summary 取消售后申请
// @ID miniappAfterSaleCancel
// @Tags aftersale
// @Accept json
// @Produce json
// @Param request body MiniappCancelAfterSaleRequest true "取消参数"
// @Success 200 {object} response.Response[any]
// @Router /after-sales/cancel [post]
func (a *API) Cancel(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req MiniappCancelAfterSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	if err := a.service.Cancel(c.Request.Context(), member.ID, req.AfterSaleID); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}

// ReturnLogistics godoc
// @Summary 提交退货物流信息
// @ID miniappAfterSaleReturnLogistics
// @Tags aftersale
// @Accept json
// @Produce json
// @Param request body MiniappReturnLogisticsRequest true "物流参数"
// @Success 200 {object} response.Response[any]
// @Router /after-sales/return-logistics [post]
func (a *API) ReturnLogistics(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req MiniappReturnLogisticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	if err := a.service.SubmitReturnLogistics(c.Request.Context(), member.ID, req.AfterSaleID, req.Company, req.TrackingNo); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}

// UploadProof godoc
// @Summary 上传售后凭证附件 (图片最大 10MB，视频最大 100MB)
// @ID uploadMiniappAfterSaleProof
// @Tags aftersale
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "凭证文件"
// @Success 200 {object} response.Response[MiniappAfterSaleProofUploadResponse]
// @Router /after-sales/upload [post]
func (a *API) UploadProof(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 100<<20+(1<<20))
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, "请选择要上传的文件")
		return
	}

	stream, err := file.Open()
	if err != nil {
		response.Error(c, "读取文件失败")
		return
	}
	defer stream.Close()

	resp, err := a.service.UploadProof(c.Request.Context(), member.ID, file.Filename, stream, file.Size)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, resp, 0)
}

