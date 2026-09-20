package review

import (
	"net/http"
	"strconv"

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

// Upload godoc
// @Summary 上传评价图片 (最大 5MB)
// @Description 会员上传评价图片，仅支持 jpg/jpeg/png/webp 格式图片，自动绑定本人与 review 场景
// @ID uploadMiniappReviewImage
// @Tags miniapp-review
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "评价图片文件"
// @Success 200 {object} response.Response[MiniappReviewUploadResponse]
// @Router /reviews/upload [post]
func (a *API) Upload(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20+(1<<20))
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

	resp, err := a.service.Upload(c.Request.Context(), member.ID, file.Filename, stream, file.Size)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, resp, 0)
}

// Create godoc
// @Summary 提交商品评价 (miniapp)
// @Description 已完成订单商品明细可提交评价，支持星级、正文、图片、匿名及弱网幂等
// @ID createMiniappReview
// @Tags miniapp-review
// @Accept json
// @Produce json
// @Param request body MiniappReviewCreateRequest true "评价请求参数"
// @Success 200 {object} response.Response[MiniappReviewCreateResponse]
// @Router /reviews/create [post]
func (a *API) Create(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req MiniappReviewCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败: "+err.Error())
		return
	}

	memberName := member.Nickname
	if memberName == "" {
		memberName = "会员"
	}

	res, err := a.service.Create(c.Request.Context(), member.ID, memberName, req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, res, 0)
}

// Mine godoc
// @Summary 获取我的评价列表 (miniapp)
// @Description 分页查询当前登录会员发表的所有评价（含待审核/已公开/已驳回/已隐藏状态）
// @ID getMiniappMineReviews
// @Tags miniapp-review
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response[[]MiniappMineReviewItemResponse]
// @Router /reviews/mine [get]
func (a *API) Mine(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	items, total, err := a.service.Mine(c.Request.Context(), member.ID, page, pageSize)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, items, int(total))
}

// Detail godoc
// @Summary 获取评价详情 (miniapp)
// @Description 获取单条评价详情；未公开评价仅作者本人可查看
// @ID getMiniappReviewDetail
// @Tags miniapp-review
// @Produce json
// @Param id path int true "评价ID"
// @Success 200 {object} response.Response[MiniappMineReviewItemResponse]
// @Router /reviews/{id} [get]
func (a *API) Detail(c *gin.Context) {
	member, ok := miniappauth.CurrentMember(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, "无效的评价ID")
		return
	}

	detail, err := a.service.Detail(c.Request.Context(), member.ID, id)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, detail, 0)
}

// ProductReviews godoc
// @Summary 获取商品的公开评价列表及统计汇总 (miniapp)
// @Description 公开接口，返回商品下已审核通过的评价，支持评分/有图过滤与统计概要
// @ID getMiniappProductReviews
// @Tags miniapp-review
// @Produce json
// @Param id path int true "商品ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param filter query string false "过滤类型: all, has_image, good, medium, bad"
// @Param rating query int false "星级筛选 (1-5)"
// @Success 200 {object} response.Response[MiniappProductReviewsResponse]
// @Router /products/{id}/reviews [get]
func (a *API) ProductReviews(c *gin.Context) {
	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || productID <= 0 {
		response.Error(c, "无效的商品ID")
		return
	}

	var query MiniappProductReviewsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, "参数解析失败: "+err.Error())
		return
	}

	res, err := a.service.ProductReviews(c.Request.Context(), productID, query)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, res, 0)
}
