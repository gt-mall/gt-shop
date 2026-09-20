package review

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db      *gorm.DB
	fileSvc *systemfile.Service
}

func NewService(db *gorm.DB, fileSvc *systemfile.Service) *Service {
	return &Service{
		db:      db,
		fileSvc: fileSvc,
	}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if mysqlErr.Number == 1062 {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique")
}

func maskNickname(name string) string {
	runes := []rune(strings.TrimSpace(name))
	if len(runes) == 0 {
		return "匿***名"
	}
	if len(runes) == 1 {
		return string(runes[0]) + "***"
	}
	if len(runes) == 2 {
		return string(runes[0]) + "*"
	}
	return string(runes[0]) + "***" + string(runes[len(runes)-1])
}

// UploadProof uploads an image for product review (max 5MB, jpg/jpeg/png/webp, private).
func (s *Service) Upload(ctx context.Context, memberID int64, filename string, reader io.Reader, declaredSize int64) (MiniappReviewUploadResponse, error) {
	if s.fileSvc == nil {
		return MiniappReviewUploadResponse{}, errors.New("文件服务不可用")
	}

	res, err := s.fileSvc.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     filename,
		GroupID:      0,
		Scene:        model.SceneReview,
		OwnerType:    model.OwnerTypeMember,
		OwnerID:      memberID,
		Visibility:   model.VisibilityPrivate,
		Reader:       reader,
		DeclaredSize: declaredSize,
	})
	if err != nil {
		return MiniappReviewUploadResponse{}, err
	}

	return MiniappReviewUploadResponse{
		FileID: res.ID,
		URL:    res.URL,
	}, nil
}

func (s *Service) matchesReviewPayload(ctx context.Context, db *gorm.DB, review *model.ProductReview, req MiniappReviewCreateRequest, cleanContent string) (bool, error) {
	if review.OrderItemID != req.OrderItemID ||
		review.Rating != req.Rating ||
		review.IsAnonymous != req.IsAnonymous {
		return false, nil
	}

	if review.Content != cleanContent {
		if strings.Contains(review.Content, "&") && html.UnescapeString(review.Content) == cleanContent {
			// matches legacy HTML entity escaped record
		} else {
			return false, nil
		}
	}

	var existingImgs []model.ProductReviewImage
	if err := db.WithContext(ctx).Where("review_id = ?", review.ID).Order("sort ASC, id ASC").Find(&existingImgs).Error; err != nil {
		return false, err
	}

	if len(existingImgs) != len(req.ImageIDs) {
		return false, nil
	}

	for i, img := range existingImgs {
		if img.FileID != req.ImageIDs[i] {
			return false, nil
		}
	}

	return true, nil
}

// Create submits a product review with idempotency, purchase qualification, and after-sale checks.
func (s *Service) Create(ctx context.Context, memberID int64, memberName string, req MiniappReviewCreateRequest) (MiniappReviewCreateResponse, error) {
	// 1. Parameter validations
	if req.Rating < 1 || req.Rating > 5 {
		return MiniappReviewCreateResponse{}, errors.New("评分必须在 1 至 5 星之间")
	}

	content := strings.TrimSpace(req.Content)
	count := utf8.RuneCountInString(content)
	if count < 1 || count > 500 {
		return MiniappReviewCreateResponse{}, errors.New("评价内容长度必须在 1 至 500 个字符之间")
	}

	if len(req.ImageIDs) > 6 {
		return MiniappReviewCreateResponse{}, errors.New("评价图片最多上传 6 张")
	}

	// Deduplicate check on image IDs
	imgSet := make(map[int64]bool, len(req.ImageIDs))
	for _, fID := range req.ImageIDs {
		if fID <= 0 {
			return MiniappReviewCreateResponse{}, errors.New("无效的图片ID")
		}
		if imgSet[fID] {
			return MiniappReviewCreateResponse{}, errors.New("评价图片包含重复图片")
		}
		imgSet[fID] = true
	}

	reqID := strings.TrimSpace(req.RequestID)
	if reqID == "" {
		return MiniappReviewCreateResponse{}, errors.New("缺少幂等请求ID")
	}

	// 2. Pre-check idempotency by request_id
	var existing model.ProductReview
	err := s.db.WithContext(ctx).Where("member_id = ? AND request_id = ?", memberID, reqID).First(&existing).Error
	if err == nil {
		matched, mErr := s.matchesReviewPayload(ctx, s.db, &existing, req, content)
		if mErr != nil {
			return MiniappReviewCreateResponse{}, mErr
		}
		if matched {
			return MiniappReviewCreateResponse{
				ID:         existing.ID,
				Status:     existing.Status,
				StatusText: model.ReviewStatusText(existing.Status),
			}, nil
		}
		return MiniappReviewCreateResponse{}, errors.New("请求ID已被使用但参数不一致")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return MiniappReviewCreateResponse{}, err
	}

	// 3. Execute in transaction with row locking
	now := time.Now()
	var createdReview model.ProductReview

	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check order item first
		var orderItem model.OrderItem
		if err := tx.Where("id = ?", req.OrderItemID).First(&orderItem).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("订单商品明细不存在")
			}
			return err
		}

		// Lock order main FIRST with clause.Locking (ordered lock on order_main, matching aftersale lock order)
		var orderMain model.OrderMain
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", orderItem.OrderID).First(&orderMain).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("关联订单不存在")
			}
			return err
		}

		// Verify ownership
		if orderMain.MemberID != memberID {
			return errors.New("无权评价该订单商品")
		}

		// Verify order status (only completed orders can be reviewed)
		if orderMain.Status == model.OrderStatusRefunded {
			return errors.New("订单已退款，不可评价")
		}
		if orderMain.Status != model.OrderStatusCompleted {
			return errors.New("仅已完成的订单可以进行评价")
		}

		// Idempotency re-check after acquiring order lock
		var existingInTx model.ProductReview
		if err := tx.Where("member_id = ? AND request_id = ?", memberID, reqID).First(&existingInTx).Error; err == nil {
			matched, mErr := s.matchesReviewPayload(ctx, tx, &existingInTx, req, content)
			if mErr != nil {
				return mErr
			}
			if matched {
				createdReview = existingInTx
				return nil
			}
			return errors.New("请求ID已被使用但参数不一致")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// Verify order item has not already been reviewed
		var itemReview model.ProductReview
		if err := tx.Where("order_item_id = ?", req.OrderItemID).First(&itemReview).Error; err == nil {
			if itemReview.MemberID == memberID && itemReview.RequestID == reqID {
				matched, mErr := s.matchesReviewPayload(ctx, tx, &itemReview, req, content)
				if mErr != nil {
					return mErr
				}
				if matched {
					createdReview = itemReview
					return nil
				}
				return errors.New("请求ID已被使用但参数不一致")
			}
			return errors.New("该订单商品已评价，不可重复评价")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// Verify after-sale status:
		// Active after-sale or completed after-sale prevents new review
		type afterSaleItemCheck struct {
			Status int8
		}
		var afterSales []afterSaleItemCheck
		if err := tx.Table("after_sale_item").
			Select("after_sale.status").
			Joins("JOIN after_sale ON after_sale.id = after_sale_item.after_sale_id").
			Where("after_sale_item.order_item_id = ? AND after_sale.member_id = ?", req.OrderItemID, memberID).
			Scan(&afterSales).Error; err != nil {
			return err
		}
		for _, af := range afterSales {
			if model.IsActiveAfterSaleStatus(af.Status) {
				return errors.New("该商品正在售后处理中，暂不可评价")
			}
			if af.Status == model.AfterSaleStatusCompleted {
				return errors.New("该商品已申请退款完成，不可评价")
			}
		}

		// Validate images
		if len(req.ImageIDs) > 0 {
			var files []model.SystemFile
			if err := tx.Where("id IN ?", req.ImageIDs).Find(&files).Error; err != nil {
				return err
			}
			if len(files) != len(req.ImageIDs) {
				return errors.New("部分图片不存在")
			}
			fileMap := make(map[int64]model.SystemFile, len(files))
			for _, f := range files {
				fileMap[f.ID] = f
			}
			for _, fID := range req.ImageIDs {
				f, ok := fileMap[fID]
				if !ok {
					return errors.New("图片不存在")
				}
				if f.FileType != model.FileTypeImage {
					return errors.New("上传文件类型必须为图片")
				}
				if f.Scene != model.SceneReview {
					return errors.New("图片场景不匹配")
				}
				if f.OwnerType != model.OwnerTypeMember || f.OwnerID != memberID {
					return errors.New("无权使用非本人上传的图片")
				}
			}

			// Exclusive image check: images cannot be bound to multiple reviews
			var boundCount int64
			if err := tx.Model(&model.ProductReviewImage{}).Where("file_id IN ?", req.ImageIDs).Count(&boundCount).Error; err != nil {
				return err
			}
			if boundCount > 0 {
				return errors.New("所选图片已被其他评价使用")
			}
		}

		hasImage := int8(0)
		if len(req.ImageIDs) > 0 {
			hasImage = 1
		}

		createdReview = model.ProductReview{
			OrderID:       orderItem.OrderID,
			OrderItemID:   orderItem.ID,
			MemberID:      memberID,
			RequestID:     reqID,
			ProductID:     orderItem.ProductID,
			SKUID:         orderItem.SKUID,
			ProductName:   orderItem.ProductName,
			SKUSpecValues: orderItem.SKUSpecValues,
			Rating:        req.Rating,
			Content:       content,
			IsAnonymous:   req.IsAnonymous,
			Status:        model.ReviewStatusPendingReview,
			HasImage:      hasImage,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := tx.Create(&createdReview).Error; err != nil {
			return err
		}

		// Insert review images
		if len(req.ImageIDs) > 0 {
			reviewImgs := make([]model.ProductReviewImage, 0, len(req.ImageIDs))
			for sortIdx, fID := range req.ImageIDs {
				reviewImgs = append(reviewImgs, model.ProductReviewImage{
					ReviewID:  createdReview.ID,
					FileID:    fID,
					Sort:      sortIdx,
					CreatedAt: now,
				})
			}
			if err := tx.Create(&reviewImgs).Error; err != nil {
				return err
			}
		}

		// Insert review log
		opName := memberName
		if opName == "" {
			opName = fmt.Sprintf("会员#%d", memberID)
		}
		reviewLog := model.ProductReviewLog{
			ReviewID:     createdReview.ID,
			OperatorType: model.ReviewOperatorTypeMember,
			OperatorID:   memberID,
			OperatorName: opName,
			FromStatus:   0,
			ToStatus:     model.ReviewStatusPendingReview,
			Action:       model.ReviewActionCreate,
			Remark:       "用户提交评价",
			CreatedAt:    now,
		}
		if err := tx.Create(&reviewLog).Error; err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		if isDuplicateKeyError(txErr) {
			// 1. Check idempotency recovery by member_id + request_id
			var dup model.ProductReview
			if err := s.db.WithContext(ctx).Where("member_id = ? AND request_id = ?", memberID, reqID).First(&dup).Error; err == nil {
				matched, mErr := s.matchesReviewPayload(ctx, s.db, &dup, req, content)
				if mErr == nil && matched {
					return MiniappReviewCreateResponse{
						ID:         dup.ID,
						Status:     dup.Status,
						StatusText: model.ReviewStatusText(dup.Status),
					}, nil
				}
				return MiniappReviewCreateResponse{}, errors.New("请求ID已被使用但参数不一致")
			}
			// 2. Check if image file_id conflict
			if len(req.ImageIDs) > 0 {
				var imgCount int64
				if err := s.db.WithContext(ctx).Model(&model.ProductReviewImage{}).Where("file_id IN ?", req.ImageIDs).Count(&imgCount).Error; err == nil && imgCount > 0 {
					return MiniappReviewCreateResponse{}, errors.New("所选图片已被其他评价使用")
				}
			}
			// 3. Otherwise was duplicate order_item_id
			return MiniappReviewCreateResponse{}, errors.New("该订单商品已评价，不可重复评价")
		}
		return MiniappReviewCreateResponse{}, txErr
	}

	return MiniappReviewCreateResponse{
		ID:         createdReview.ID,
		Status:     createdReview.Status,
		StatusText: model.ReviewStatusText(createdReview.Status),
	}, nil
}

// Mine returns reviews submitted by the current member.
func (s *Service) Mine(ctx context.Context, memberID int64, page, pageSize int) ([]MiniappMineReviewItemResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	q := s.db.WithContext(ctx).Model(&model.ProductReview{}).Where("member_id = ?", memberID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var reviews []model.ProductReview
	offset := (page - 1) * pageSize
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&reviews).Error; err != nil {
		return nil, 0, err
	}

	if len(reviews) == 0 {
		return []MiniappMineReviewItemResponse{}, total, nil
	}

	reviewIDs := make([]int64, 0, len(reviews))
	for _, r := range reviews {
		reviewIDs = append(reviewIDs, r.ID)
	}

	var reviewImages []model.ProductReviewImage
	_ = s.db.WithContext(ctx).Where("review_id IN ?", reviewIDs).Order("sort ASC, id ASC").Find(&reviewImages).Error

	fileIDsByReview := make(map[int64][]int64)
	allFileIDs := make([]int64, 0, len(reviewImages))
	for _, ri := range reviewImages {
		fileIDsByReview[ri.ReviewID] = append(fileIDsByReview[ri.ReviewID], ri.FileID)
		allFileIDs = append(allFileIDs, ri.FileID)
	}

	urlMap := make(map[int64]string)
	if len(allFileIDs) > 0 && s.fileSvc != nil {
		// Author can resolve private images with signature
		if resolved, err := s.fileSvc.ResolvePrivate(ctx, allFileIDs); err == nil {
			for _, f := range resolved {
				urlMap[f.ID] = f.URL
			}
		}
	}

	items := make([]MiniappMineReviewItemResponse, 0, len(reviews))
	for _, r := range reviews {
		fIDs := fileIDsByReview[r.ID]
		imgURLs := make([]string, 0, len(fIDs))
		for _, fID := range fIDs {
			if u, ok := urlMap[fID]; ok && u != "" {
				imgURLs = append(imgURLs, u)
			}
		}

		items = append(items, MiniappMineReviewItemResponse{
			ID:            r.ID,
			OrderID:       r.OrderID,
			OrderItemID:   r.OrderItemID,
			ProductID:     r.ProductID,
			ProductName:   r.ProductName,
			SKUID:         r.SKUID,
			SKUSpecValues: r.SKUSpecValues,
			Rating:        r.Rating,
			Content:       r.Content,
			IsAnonymous:   r.IsAnonymous,
			Status:        r.Status,
			StatusText:    model.ReviewStatusText(r.Status),
			AuditReason:   r.AuditReason,
			HasImage:      r.HasImage,
			Images:        imgURLs,
			ReplyContent:  r.ReplyContent,
			ReplyAt:       r.ReplyAt,
			CreatedAt:     r.CreatedAt,
		})
	}

	return items, total, nil
}

// Detail returns detail of a single review for miniapp.
func (s *Service) Detail(ctx context.Context, memberID int64, reviewID int64) (MiniappReviewDetailResponse, error) {
	var r model.ProductReview
	if err := s.db.WithContext(ctx).First(&r, reviewID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MiniappReviewDetailResponse{}, errors.New("评价不存在")
		}
		return MiniappReviewDetailResponse{}, err
	}

	isAuthor := (r.MemberID == memberID)

	// Non-author cannot view pending/rejected/hidden reviews
	if !isAuthor && r.Status != model.ReviewStatusApproved {
		return MiniappReviewDetailResponse{}, errors.New("无权查看该评价")
	}

	var reviewImages []model.ProductReviewImage
	_ = s.db.WithContext(ctx).Where("review_id = ?", r.ID).Order("sort ASC, id ASC").Find(&reviewImages).Error
	fileIDs := make([]int64, 0, len(reviewImages))
	for _, ri := range reviewImages {
		fileIDs = append(fileIDs, ri.FileID)
	}

	imgURLs := make([]string, 0, len(fileIDs))
	if len(fileIDs) > 0 && s.fileSvc != nil {
		if isAuthor {
			if resolved, err := s.fileSvc.ResolvePrivate(ctx, fileIDs); err == nil {
				for _, f := range resolved {
					if f.URL != "" {
						imgURLs = append(imgURLs, f.URL)
					}
				}
			}
		} else {
			if resolved, err := s.fileSvc.Resolve(ctx, systemfile.IDsRequest{IDs: fileIDs}); err == nil {
				for _, f := range resolved {
					if f.URL != "" {
						imgURLs = append(imgURLs, f.URL)
					}
				}
			}
		}
	}

	var nickname, avatar string
	if isAuthor {
		var author model.Member
		_ = s.db.WithContext(ctx).First(&author, r.MemberID).Error
		nickname = author.Nickname
		if author.AvatarID != nil && *author.AvatarID > 0 && s.fileSvc != nil {
			if res, err := s.fileSvc.Resolve(ctx, systemfile.IDsRequest{IDs: []int64{*author.AvatarID}}); err == nil && len(res) > 0 {
				avatar = res[0].URL
			}
		}
	} else {
		if r.IsAnonymous == 1 {
			nickname = "匿***名"
			avatar = ""
		} else {
			var author model.Member
			_ = s.db.WithContext(ctx).First(&author, r.MemberID).Error
			nickname = maskNickname(author.Nickname)
			if author.AvatarID != nil && *author.AvatarID > 0 && s.fileSvc != nil {
				if res, err := s.fileSvc.Resolve(ctx, systemfile.IDsRequest{IDs: []int64{*author.AvatarID}}); err == nil && len(res) > 0 {
					avatar = res[0].URL
				}
			}
		}
	}

	resp := MiniappReviewDetailResponse{
		ID:            r.ID,
		ProductID:     r.ProductID,
		ProductName:   r.ProductName,
		SKUID:         r.SKUID,
		SKUSpecValues: r.SKUSpecValues,
		Rating:        r.Rating,
		Content:       r.Content,
		IsAnonymous:   r.IsAnonymous,
		Status:        r.Status,
		StatusText:    model.ReviewStatusText(r.Status),
		HasImage:      r.HasImage,
		Images:        imgURLs,
		ReplyContent:  r.ReplyContent,
		ReplyAt:       r.ReplyAt,
		CreatedAt:     r.CreatedAt,
		IsAuthor:      isAuthor,
		UserNickname:  nickname,
		UserAvatar:    avatar,
	}

	if isAuthor {
		resp.OrderID = &r.OrderID
		resp.OrderItemID = &r.OrderItemID
		resp.AuditReason = &r.AuditReason
	}

	return resp, nil
}

// ProductReviews returns public approved reviews for a product, along with summary.
func (s *Service) ProductReviews(ctx context.Context, productID int64, query MiniappProductReviewsQuery) (MiniappProductReviewsResponse, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 10
	}
	if query.PageSize > 50 {
		query.PageSize = 50
	}

	baseQ := s.db.WithContext(ctx).Model(&model.ProductReview{}).
		Where("product_id = ? AND status = ?", productID, model.ReviewStatusApproved)

	// Summary aggregation query: total, has_image, good, medium, bad, sum(rating)
	type summaryAgg struct {
		TotalCount    int64
		HasImageCount int64
		GoodCount     int64
		MediumCount   int64
		BadCount      int64
		SumRating     int64
	}
	var agg summaryAgg
	err := baseQ.Select(
		"COUNT(*) AS total_count, " +
			"COALESCE(SUM(CASE WHEN has_image = 1 THEN 1 ELSE 0 END), 0) AS has_image_count, " +
			"COALESCE(SUM(CASE WHEN rating >= 4 THEN 1 ELSE 0 END), 0) AS good_count, " +
			"COALESCE(SUM(CASE WHEN rating = 3 THEN 1 ELSE 0 END), 0) AS medium_count, " +
			"COALESCE(SUM(CASE WHEN rating <= 2 THEN 1 ELSE 0 END), 0) AS bad_count, " +
			"COALESCE(SUM(rating), 0) AS sum_rating",
	).Scan(&agg).Error
	if err != nil {
		return MiniappProductReviewsResponse{}, err
	}

	avgRatingStr := "5.0"
	goodRateStr := "100%"
	if agg.TotalCount > 0 {
		// Calculate decimal average rating without float inaccuracy
		// e.g. sum_rating * 10 / total_count -> 48 -> "4.8"
		scaled := (agg.SumRating*100 + agg.TotalCount/2) / agg.TotalCount // round to 2 decimals * 100
		avgInt := scaled / 100
		avgDec := (scaled % 100) / 10
		avgRatingStr = fmt.Sprintf("%d.%d", avgInt, avgDec)

		// Calculate good rate
		rateScaled := (agg.GoodCount * 1000) / agg.TotalCount
		rateInt := rateScaled / 10
		rateDec := rateScaled % 10
		if rateDec == 0 {
			goodRateStr = fmt.Sprintf("%d%%", rateInt)
		} else {
			goodRateStr = fmt.Sprintf("%d.%d%%", rateInt, rateDec)
		}
	}

	summary := MiniappProductReviewSummary{
		TotalCount:    agg.TotalCount,
		HasImageCount: agg.HasImageCount,
		GoodCount:     agg.GoodCount,
		MediumCount:   agg.MediumCount,
		BadCount:      agg.BadCount,
		AverageRating: avgRatingStr,
		GoodRate:      goodRateStr,
	}

	// Apply filters for item list
	itemQ := s.db.WithContext(ctx).Model(&model.ProductReview{}).
		Where("product_id = ? AND status = ?", productID, model.ReviewStatusApproved)

	switch query.Filter {
	case "has_image":
		itemQ = itemQ.Where("has_image = 1")
	case "good":
		itemQ = itemQ.Where("rating >= 4")
	case "medium":
		itemQ = itemQ.Where("rating = 3")
	case "bad":
		itemQ = itemQ.Where("rating <= 2")
	}

	if query.Rating >= 1 && query.Rating <= 5 {
		itemQ = itemQ.Where("rating = ?", query.Rating)
	}

	var reviews []model.ProductReview
	offset := (query.Page - 1) * query.PageSize
	if err := itemQ.Order("id DESC").Offset(offset).Limit(query.PageSize).Find(&reviews).Error; err != nil {
		return MiniappProductReviewsResponse{}, err
	}

	if len(reviews) == 0 {
		return MiniappProductReviewsResponse{
			Summary: summary,
			Items:   []MiniappProductReviewItemResponse{},
		}, nil
	}

	// Fetch members for non-anonymous reviews
	memberIDs := make([]int64, 0, len(reviews))
	reviewIDs := make([]int64, 0, len(reviews))
	for _, r := range reviews {
		reviewIDs = append(reviewIDs, r.ID)
		if r.IsAnonymous == 0 {
			memberIDs = append(memberIDs, r.MemberID)
		}
	}

	memberMap := make(map[int64]model.Member)
	avatarURLMap := make(map[int64]string)
	if len(memberIDs) > 0 {
		var members []model.Member
		_ = s.db.WithContext(ctx).Where("id IN ?", memberIDs).Find(&members).Error
		avatarIDs := make([]int64, 0, len(members))
		for _, m := range members {
			memberMap[m.ID] = m
			if m.AvatarID != nil && *m.AvatarID > 0 {
				avatarIDs = append(avatarIDs, *m.AvatarID)
			}
		}
		if len(avatarIDs) > 0 && s.fileSvc != nil {
			if res, err := s.fileSvc.Resolve(ctx, systemfile.IDsRequest{IDs: avatarIDs}); err == nil {
				for _, f := range res {
					avatarURLMap[f.ID] = f.URL
				}
			}
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

	urlMap := make(map[int64]string)
	if len(allFileIDs) > 0 && s.fileSvc != nil {
		// Public resolve: since review is approved, file visibility is public
		if resolved, err := s.fileSvc.Resolve(ctx, systemfile.IDsRequest{IDs: allFileIDs}); err == nil {
			for _, f := range resolved {
				if f.URL != "" {
					urlMap[f.ID] = f.URL
				}
			}
		}
	}

	items := make([]MiniappProductReviewItemResponse, 0, len(reviews))
	for _, r := range reviews {
		var nickname, avatar string
		if r.IsAnonymous == 1 {
			nickname = "匿***名"
			avatar = ""
		} else {
			m := memberMap[r.MemberID]
			nickname = maskNickname(m.Nickname)
			if m.AvatarID != nil {
				avatar = avatarURLMap[*m.AvatarID]
			}
		}

		fIDs := fileIDsByReview[r.ID]
		imgs := make([]string, 0, len(fIDs))
		for _, fID := range fIDs {
			if u, ok := urlMap[fID]; ok && u != "" {
				imgs = append(imgs, u)
			}
		}

		items = append(items, MiniappProductReviewItemResponse{
			ID:            r.ID,
			Rating:        r.Rating,
			Content:       r.Content,
			SKUSpecValues: r.SKUSpecValues,
			IsAnonymous:   r.IsAnonymous,
			UserNickname:  nickname,
			UserAvatar:    avatar,
			HasImage:      r.HasImage,
			Images:        imgs,
			ReplyContent:  r.ReplyContent,
			ReplyAt:       r.ReplyAt,
			CreatedAt:     r.CreatedAt,
		})
	}

	return MiniappProductReviewsResponse{
		Summary: summary,
		Items:   items,
	}, nil
}
