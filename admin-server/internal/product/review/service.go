package review

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"

	"gorm.io/gorm"
)

type Service struct {
	db      *gorm.DB
	fileSvc *systemfile.Service
	cache   cache.Cache
}

func NewService(db *gorm.DB, fileSvc *systemfile.Service, c cache.Cache) *Service {
	return &Service{
		db:      db,
		fileSvc: fileSvc,
		cache:   c,
	}
}

// List queries reviews for admin management with filters and pagination.
func (s *Service) List(ctx context.Context, req ReviewListRequest) ([]ReviewListItemResponse, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	q := s.db.WithContext(ctx).Model(&model.ProductReview{})

	if req.ProductID != nil && *req.ProductID > 0 {
		q = q.Where("product_id = ?", *req.ProductID)
	}
	if req.Status != nil {
		q = q.Where("status = ?", *req.Status)
	}
	if req.Rating != nil && *req.Rating > 0 {
		q = q.Where("rating = ?", *req.Rating)
	}
	if req.HasImage != nil {
		q = q.Where("has_image = ?", *req.HasImage)
	}
	if req.MemberID != nil && *req.MemberID > 0 {
		q = q.Where("member_id = ?", *req.MemberID)
	}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("content LIKE ? OR product_name LIKE ?", like, like)
	}
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			q = q.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count reviews: %w", err)
	}

	var reviews []model.ProductReview
	offset := (req.Page - 1) * req.PageSize
	if err := q.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&reviews).Error; err != nil {
		return nil, 0, fmt.Errorf("find reviews: %w", err)
	}

	if len(reviews) == 0 {
		return []ReviewListItemResponse{}, total, nil
	}

	// Batch fetch member info, audit user names, reply user names, and images
	memberIDs := make([]int64, 0, len(reviews))
	adminUserIDs := make([]int64, 0, len(reviews))
	reviewIDs := make([]int64, 0, len(reviews))

	for _, r := range reviews {
		memberIDs = append(memberIDs, r.MemberID)
		reviewIDs = append(reviewIDs, r.ID)
		if r.AuditUserID != nil && *r.AuditUserID > 0 {
			adminUserIDs = append(adminUserIDs, *r.AuditUserID)
		}
		if r.ReplyUserID != nil && *r.ReplyUserID > 0 {
			adminUserIDs = append(adminUserIDs, *r.ReplyUserID)
		}
	}

	memberMap := make(map[int64]model.Member)
	if len(memberIDs) > 0 {
		var members []model.Member
		_ = s.db.WithContext(ctx).Where("id IN ?", memberIDs).Find(&members).Error
		for _, m := range members {
			memberMap[m.ID] = m
		}
	}

	adminUserMap := make(map[int64]string)
	if len(adminUserIDs) > 0 {
		var users []model.SystemUser
		_ = s.db.WithContext(ctx).Where("id IN ?", adminUserIDs).Find(&users).Error
		for _, u := range users {
			name := u.Nickname
			if name == "" {
				name = u.Username
			}
			adminUserMap[u.ID] = name
		}
	}

	// Fetch review images
	var reviewImages []model.ProductReviewImage
	_ = s.db.WithContext(ctx).Where("review_id IN ?", reviewIDs).Order("sort ASC, id ASC").Find(&reviewImages).Error

	fileIDsByReview := make(map[int64][]int64)
	allFileIDs := make([]int64, 0, len(reviewImages))
	for _, ri := range reviewImages {
		fileIDsByReview[ri.ReviewID] = append(fileIDsByReview[ri.ReviewID], ri.FileID)
		allFileIDs = append(allFileIDs, ri.FileID)
	}

	filesMap := make(map[int64]systemfile.FileResponse)
	if len(allFileIDs) > 0 && s.fileSvc != nil {
		// Admin uses ResolvePrivate to get authorized signed URLs for both public and private images
		if resolved, err := s.fileSvc.ResolvePrivate(ctx, allFileIDs); err == nil {
			for _, f := range resolved {
				filesMap[f.ID] = f
			}
		}
	}

	items := make([]ReviewListItemResponse, 0, len(reviews))
	for _, r := range reviews {
		m := memberMap[r.MemberID]
		var memberMobile string
		if m.Mobile != nil {
			memberMobile = *m.Mobile
		}
		var auditUserName, replyUserName string
		if r.AuditUserID != nil {
			auditUserName = adminUserMap[*r.AuditUserID]
		}
		if r.ReplyUserID != nil {
			replyUserName = adminUserMap[*r.ReplyUserID]
		}

		fIDs := fileIDsByReview[r.ID]
		imgs := make([]systemfile.FileResponse, 0, len(fIDs))
		for _, fID := range fIDs {
			if f, ok := filesMap[fID]; ok {
				imgs = append(imgs, f)
			}
		}

		items = append(items, ReviewListItemResponse{
			ID:             r.ID,
			OrderID:        r.OrderID,
			OrderItemID:    r.OrderItemID,
			MemberID:       r.MemberID,
			MemberNickname: m.Nickname,
			MemberMobile:   memberMobile,
			ProductID:      r.ProductID,
			ProductName:    r.ProductName,
			SKUID:          r.SKUID,
			SKUSpecValues:  r.SKUSpecValues,
			Rating:         r.Rating,
			Content:        r.Content,
			IsAnonymous:    r.IsAnonymous,
			Status:         r.Status,
			StatusText:     model.ReviewStatusText(r.Status),
			HasImage:       r.HasImage,
			Images:         imgs,
			AuditReason:    r.AuditReason,
			AuditUserName:  auditUserName,
			AuditedAt:      r.AuditedAt,
			ReplyContent:   r.ReplyContent,
			ReplyUserName:  replyUserName,
			ReplyAt:        r.ReplyAt,
			CreatedAt:      r.CreatedAt,
		})
	}

	return items, total, nil
}

// Detail returns full details of a review in admin.
func (s *Service) Detail(ctx context.Context, id int64) (ReviewDetailResponse, error) {
	var r model.ProductReview
	if err := s.db.WithContext(ctx).First(&r, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ReviewDetailResponse{}, errors.New("评价不存在")
		}
		return ReviewDetailResponse{}, err
	}

	// Fetch member
	var m model.Member
	_ = s.db.WithContext(ctx).First(&m, r.MemberID).Error

	// Fetch order
	var o model.OrderMain
	_ = s.db.WithContext(ctx).First(&o, r.OrderID).Error

	// Fetch admin names
	var auditUserName, replyUserName string
	if r.AuditUserID != nil && *r.AuditUserID > 0 {
		var u model.SystemUser
		if err := s.db.WithContext(ctx).First(&u, *r.AuditUserID).Error; err == nil {
			auditUserName = u.Nickname
			if auditUserName == "" {
				auditUserName = u.Username
			}
		}
	}
	if r.ReplyUserID != nil && *r.ReplyUserID > 0 {
		var u model.SystemUser
		if err := s.db.WithContext(ctx).First(&u, *r.ReplyUserID).Error; err == nil {
			replyUserName = u.Nickname
			if replyUserName == "" {
				replyUserName = u.Username
			}
		}
	}

	// Fetch images
	var reviewImages []model.ProductReviewImage
	_ = s.db.WithContext(ctx).Where("review_id = ?", r.ID).Order("sort ASC, id ASC").Find(&reviewImages).Error
	fileIDs := make([]int64, 0, len(reviewImages))
	for _, ri := range reviewImages {
		fileIDs = append(fileIDs, ri.FileID)
	}

	var imgs []systemfile.FileResponse
	if len(fileIDs) > 0 && s.fileSvc != nil {
		if resolved, err := s.fileSvc.ResolvePrivate(ctx, fileIDs); err == nil {
			imgs = resolved
		}
	}

	// Fetch logs
	var logs []model.ProductReviewLog
	_ = s.db.WithContext(ctx).Where("review_id = ?", r.ID).Order("id ASC").Find(&logs).Error
	logResponses := make([]ReviewLogResponse, 0, len(logs))
	for _, l := range logs {
		var opTypeName string
		switch l.OperatorType {
		case model.ReviewOperatorTypeMember:
			opTypeName = "会员"
		case model.ReviewOperatorTypeAdmin:
			opTypeName = "管理员"
		case model.ReviewOperatorTypeSystem:
			opTypeName = "系统"
		}
		logResponses = append(logResponses, ReviewLogResponse{
			ID:               l.ID,
			OperatorType:     l.OperatorType,
			OperatorTypeName: opTypeName,
			OperatorID:       l.OperatorID,
			OperatorName:     l.OperatorName,
			FromStatus:       l.FromStatus,
			FromStatusText:   model.ReviewStatusText(l.FromStatus),
			ToStatus:         l.ToStatus,
			ToStatusText:     model.ReviewStatusText(l.ToStatus),
			Action:           l.Action,
			Remark:           l.Remark,
			CreatedAt:        l.CreatedAt,
		})
	}

	var memberMobile string
	if m.Mobile != nil {
		memberMobile = *m.Mobile
	}
	var memberAvatarURL string
	if m.AvatarID != nil && *m.AvatarID > 0 && s.fileSvc != nil {
		if res, err := s.fileSvc.Resolve(ctx, systemfile.IDsRequest{IDs: []int64{*m.AvatarID}}); err == nil && len(res) > 0 {
			memberAvatarURL = res[0].URL
		}
	}

	return ReviewDetailResponse{
		ID:              r.ID,
		OrderID:         r.OrderID,
		OrderNo:         o.OrderNo,
		OrderItemID:     r.OrderItemID,
		MemberID:        r.MemberID,
		MemberNickname:  m.Nickname,
		MemberMobile:    memberMobile,
		MemberAvatarURL: memberAvatarURL,
		ProductID:       r.ProductID,
		ProductName:     r.ProductName,
		SKUID:           r.SKUID,
		SKUSpecValues:   r.SKUSpecValues,
		Rating:          r.Rating,
		Content:         r.Content,
		IsAnonymous:     r.IsAnonymous,
		Status:          r.Status,
		StatusText:      model.ReviewStatusText(r.Status),
		HasImage:        r.HasImage,
		Images:          imgs,
		AuditReason:     r.AuditReason,
		AuditUserName:   auditUserName,
		AuditedAt:       r.AuditedAt,
		ReplyContent:    r.ReplyContent,
		ReplyUserName:   replyUserName,
		ReplyAt:         r.ReplyAt,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		Logs:            logResponses,
	}, nil
}

// Audit audits or changes the status of a review with optimistic lock protection.
func (s *Service) Audit(ctx context.Context, adminID int64, adminName string, req ReviewAuditRequest) error {
	if req.Action != "approve" && req.Action != "reject" && req.Action != "hide" && req.Action != "restore" {
		return errors.New("不支持的审核动作")
	}
	if req.Action == "reject" && strings.TrimSpace(req.Reason) == "" {
		return errors.New("驳回时必须填写原因")
	}

	now := time.Now()
	var productID int64

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.ProductReview
		if err := tx.Where("id = ?", req.ID).First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("评价不存在")
			}
			return err
		}
		productID = current.ProductID

		// Optimistic locking check against ExpectedStatus
		if current.Status != req.ExpectedStatus {
			return fmt.Errorf("当前状态为【%s】，与预期状态【%s】不一致，请刷新重试",
				model.ReviewStatusText(current.Status), model.ReviewStatusText(req.ExpectedStatus))
		}

		// State transition matrix:
		// 10 (PendingReview) -> approve (20), reject (30)
		// 20 (Approved)      -> hide (40)
		// 40 (Hidden)        -> restore (20)
		// 30 (Rejected)      -> no audit actions allowed
		var targetStatus int8
		var logAction string
		switch req.Action {
		case "approve":
			if current.Status != model.ReviewStatusPendingReview {
				return fmt.Errorf("仅待审核状态的评价可以通过审核，当前状态为【%s】", model.ReviewStatusText(current.Status))
			}
			targetStatus = model.ReviewStatusApproved
			logAction = model.ReviewActionAuditApprove
		case "reject":
			if current.Status != model.ReviewStatusPendingReview {
				return fmt.Errorf("仅待审核状态的评价可以驳回，当前状态为【%s】", model.ReviewStatusText(current.Status))
			}
			targetStatus = model.ReviewStatusRejected
			logAction = model.ReviewActionAuditReject
		case "hide":
			if current.Status != model.ReviewStatusApproved {
				return fmt.Errorf("仅已公开状态的评价可以隐藏，当前状态为【%s】", model.ReviewStatusText(current.Status))
			}
			targetStatus = model.ReviewStatusHidden
			logAction = model.ReviewActionHide
		case "restore":
			if current.Status != model.ReviewStatusHidden {
				return fmt.Errorf("仅已隐藏状态的评价可以恢复展示，当前状态为【%s】", model.ReviewStatusText(current.Status))
			}
			targetStatus = model.ReviewStatusApproved
			logAction = model.ReviewActionRestore
		default:
			return errors.New("不支持的审核动作")
		}

		res := tx.Model(&model.ProductReview{}).
			Where("id = ? AND status = ?", req.ID, req.ExpectedStatus).
			Updates(map[string]interface{}{
				"status":        targetStatus,
				"audit_reason":  strings.TrimSpace(req.Reason),
				"audit_user_id": adminID,
				"audited_at":    now,
				"updated_at":    now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("评价状态已被其他操作更新，请刷新重试")
		}

		// Adjust visibility of review images
		var reviewImages []model.ProductReviewImage
		if err := tx.Where("review_id = ?", req.ID).Find(&reviewImages).Error; err != nil {
			return err
		}
		if len(reviewImages) > 0 {
			fileIDs := make([]int64, 0, len(reviewImages))
			for _, ri := range reviewImages {
				fileIDs = append(fileIDs, ri.FileID)
			}
			var fileVisibility int8
			if targetStatus == model.ReviewStatusApproved {
				fileVisibility = model.VisibilityPublic
			} else {
				fileVisibility = model.VisibilityPrivate
			}
			if err := tx.Model(&model.SystemFile{}).Where("id IN ?", fileIDs).Update("visibility", fileVisibility).Error; err != nil {
				return fmt.Errorf("update file visibility: %w", err)
			}
		}

		// Record audit log
		reviewLog := model.ProductReviewLog{
			ReviewID:     req.ID,
			OperatorType: model.ReviewOperatorTypeAdmin,
			OperatorID:   adminID,
			OperatorName: adminName,
			FromStatus:   req.ExpectedStatus,
			ToStatus:     targetStatus,
			Action:       logAction,
			Remark:       strings.TrimSpace(req.Reason),
			CreatedAt:    now,
		}
		if err := tx.Create(&reviewLog).Error; err != nil {
			return fmt.Errorf("record review log: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Invalidate product detail cache if configured
	if s.cache != nil && productID > 0 {
		_, _ = cache.BumpProductDetailGeneration(ctx, s.cache, productID)
	}

	return nil
}

// Reply handles official merchant reply to a review.
func (s *Service) Reply(ctx context.Context, adminID int64, adminName string, req ReviewReplyRequest) error {
	content := strings.TrimSpace(req.Content)
	count := utf8.RuneCountInString(content)
	if count < 1 || count > 500 {
		return errors.New("回复内容必须在 1-500 个字符之间")
	}

	now := time.Now()
	var productID int64

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.ProductReview
		if err := tx.Where("id = ?", req.ID).First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("评价不存在")
			}
			return err
		}
		productID = current.ProductID

		res := tx.Model(&model.ProductReview{}).
			Where("id = ?", req.ID).
			Updates(map[string]interface{}{
				"reply_content": content,
				"reply_user_id": adminID,
				"reply_at":      now,
				"updated_at":    now,
			})
		if res.Error != nil {
			return res.Error
		}

		// Record reply log
		reviewLog := model.ProductReviewLog{
			ReviewID:     req.ID,
			OperatorType: model.ReviewOperatorTypeAdmin,
			OperatorID:   adminID,
			OperatorName: adminName,
			FromStatus:   current.Status,
			ToStatus:     current.Status,
			Action:       model.ReviewActionReply,
			Remark:       content,
			CreatedAt:    now,
		}
		if err := tx.Create(&reviewLog).Error; err != nil {
			return fmt.Errorf("record review log: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Invalidate product detail cache if configured
	if s.cache != nil && productID > 0 {
		_, _ = cache.BumpProductDetailGeneration(ctx, s.cache, productID)
	}

	return nil
}
