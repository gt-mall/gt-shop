package review

import (
	"mall/admin_core/internal/auth"
	"mall/admin_core/pkg/response"

	"github.com/gin-gonic/gin"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// List godoc
// @Summary List product reviews (admin)
// @Description Query product reviews with status, rating, has_image, product_id, keyword and pagination filters
// @ID listProductReviews
// @Tags product-review
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ReviewListRequest true "Review list query"
// @Success 200 {object} response.Response[[]ReviewListItemResponse]
// @Router /api/product/reviews/list [post]
func (a *API) List(c *gin.Context) {
	var req ReviewListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败: "+err.Error())
		return
	}

	items, total, err := a.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, items, int(total))
}

// Detail godoc
// @Summary Get product review detail (admin)
// @Description Get full detail of a review including order, member, image URLs and audit history
// @ID getProductReviewDetail
// @Tags product-review
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ReviewDetailRequest true "Review detail query"
// @Success 200 {object} response.Response[ReviewDetailResponse]
// @Router /api/product/reviews/detail [post]
func (a *API) Detail(c *gin.Context) {
	var req ReviewDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败: "+err.Error())
		return
	}

	detail, err := a.service.Detail(c.Request.Context(), req.ID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Data(c, detail, 0)
}

// Audit godoc
// @Summary Audit or change product review status (admin)
// @Description Approve, reject, hide or restore a product review with optimistic lock protection
// @ID auditProductReview
// @Tags product-review
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ReviewAuditRequest true "Review audit request"
// @Success 200 {object} response.Response[string]
// @Router /api/product/reviews/audit [post]
func (a *API) Audit(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req ReviewAuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败: "+err.Error())
		return
	}

	adminName := user.Nickname
	if adminName == "" {
		adminName = user.Username
	}

	if err := a.service.Audit(c.Request.Context(), user.ID, adminName, req); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}

// Reply godoc
// @Summary Reply to product review (admin)
// @Description Official merchant reply to a product review
// @ID replyProductReview
// @Tags product-review
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ReviewReplyRequest true "Review reply request"
// @Success 200 {object} response.Response[string]
// @Router /api/product/reviews/reply [post]
func (a *API) Reply(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}

	var req ReviewReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数解析失败: "+err.Error())
		return
	}

	adminName := user.Nickname
	if adminName == "" {
		adminName = user.Username
	}

	if err := a.service.Reply(c.Request.Context(), user.ID, adminName, req); err != nil {
		response.Error(c, err.Error())
		return
	}

	response.Success(c)
}
