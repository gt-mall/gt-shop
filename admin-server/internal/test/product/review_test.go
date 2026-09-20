package product_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/internal/model"
	productreview "mall/admin_core/internal/product/review"
	systemfile "mall/admin_core/internal/system/file"
)

func setupReviewTestDB(t *testing.T) (*gorm.DB, *systemfile.Service, *productreview.Service) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(
		&model.SystemConfig{},
		&model.SystemFile{},
		&model.SystemFileGroup{},
		&model.SystemUser{},
		&model.ProductCategory{},
		&model.Product{},
		&model.ProductSKU{},
		&model.ProductReview{},
		&model.ProductReviewImage{},
		&model.ProductReviewLog{},
		&model.Member{},
		&model.OrderMain{},
		&model.OrderItem{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	fileSvc := systemfile.NewService(db)
	cfg := systemfile.DefaultConfig()
	cfg.Local.Path = t.TempDir()
	if err := fileSvc.SaveConfig(context.Background(), cfg); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	reviewSvc := productreview.NewService(db, fileSvc, nil)
	return db, fileSvc, reviewSvc
}

func TestAdminReview_ListAndDetail(t *testing.T) {
	db, _, reviewSvc := setupReviewTestDB(t)
	ctx := context.Background()
	now := time.Now()

	// Seed member
	mobile := "13800000001"
	member := model.Member{
		ID:       101,
		Nickname: "测试用户1",
		Mobile:   &mobile,
	}
	_ = db.Create(&member)

	// Seed order and items
	order := model.OrderMain{
		ID:        201,
		OrderNo:   "ORD202609200001",
		MemberID:  101,
		RequestID: "req-ord-1",
		Status:    model.OrderStatusCompleted,
	}
	_ = db.Create(&order)

	item := model.OrderItem{
		ID:          301,
		OrderID:     201,
		ProductID:   1,
		SKUID:       1,
		ProductName: "高端智能手机",
		Quantity:    1,
		UnitPrice:   "4999.00",
		TotalAmount: "4999.00",
	}
	_ = db.Create(&item)

	// Seed file
	file := model.SystemFile{
		ID:         501,
		Storage:    "local",
		Name:       "review1.png",
		Path:       "2026/09/review1.png",
		Ext:        "png",
		FileType:   model.FileTypeImage,
		Scene:      model.SceneReview,
		OwnerType:  model.OwnerTypeMember,
		OwnerID:    101,
		Visibility: model.VisibilityPrivate,
	}
	_ = db.Create(&file)

	// Seed review 1 (PendingReview, 5 stars, with image)
	r1 := model.ProductReview{
		ID:          1,
		OrderID:     201,
		OrderItemID: 301,
		MemberID:    101,
		RequestID:   "req-rev-1",
		ProductID:   1,
		SKUID:       1,
		ProductName: "高端智能手机",
		Rating:      5,
		Content:     "做工精良，屏幕显示通透，拍照极其清晰！",
		Status:      model.ReviewStatusPendingReview,
		HasImage:    1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = db.Create(&r1)
	_ = db.Create(&model.ProductReviewImage{ReviewID: 1, FileID: 501, Sort: 0})
	_ = db.Create(&model.ProductReviewLog{
		ReviewID:     1,
		OperatorType: model.ReviewOperatorTypeMember,
		OperatorID:   101,
		OperatorName: "测试用户1",
		Action:       model.ReviewActionCreate,
		Remark:       "用户提交评价",
		CreatedAt:    now,
	})

	// Seed review 2 (Approved, 4 stars, no image)
	r2 := model.ProductReview{
		ID:          2,
		OrderID:     201,
		OrderItemID: 302,
		MemberID:    101,
		RequestID:   "req-rev-2",
		ProductID:   1,
		SKUID:       1,
		ProductName: "高端智能手机",
		Rating:      4,
		Content:     "总体还不错，电池续航尚可。",
		Status:      model.ReviewStatusApproved,
		HasImage:    0,
		CreatedAt:   now.Add(time.Hour),
		UpdatedAt:   now.Add(time.Hour),
	}
	_ = db.Create(&r2)

	// 1. List all
	list, total, err := reviewSvc.List(ctx, productreview.ReviewListRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}

	// 2. Filter by status = 10 (PendingReview)
	st10 := model.ReviewStatusPendingReview
	list10, total10, err := reviewSvc.List(ctx, productreview.ReviewListRequest{Status: &st10})
	if err != nil || total10 != 1 {
		t.Fatalf("expected 1 pending review, got %d", total10)
	}
	if list10[0].ID != 1 || list10[0].HasImage != 1 || len(list10[0].Images) != 1 {
		t.Fatalf("expected review 1 with 1 image, got %+v", list10[0])
	}
	if list10[0].MemberMobile != "13800000001" {
		t.Fatalf("expected member mobile, got %s", list10[0].MemberMobile)
	}

	// 3. Filter by has_image = 0
	hasImg0 := int8(0)
	listNoImg, totalNoImg, err := reviewSvc.List(ctx, productreview.ReviewListRequest{HasImage: &hasImg0})
	if err != nil || totalNoImg != 1 {
		t.Fatalf("expected 1 review without image, got %d", totalNoImg)
	}
	if listNoImg[0].ID != 2 {
		t.Fatalf("expected review 2")
	}

	// 4. Keyword search
	listKw, totalKw, err := reviewSvc.List(ctx, productreview.ReviewListRequest{Keyword: "屏幕显示"})
	if err != nil || totalKw != 1 || listKw[0].ID != 1 {
		t.Fatalf("expected keyword match review 1")
	}

	// 5. Detail review 1
	detail, err := reviewSvc.Detail(ctx, 1)
	if err != nil {
		t.Fatalf("detail error: %v", err)
	}
	if detail.ID != 1 || detail.OrderNo != "ORD202609200001" || len(detail.Images) != 1 || len(detail.Logs) != 1 {
		t.Fatalf("unexpected detail: %+v", detail)
	}
}

func TestAdminReview_AuditAndReply(t *testing.T) {
	db, _, reviewSvc := setupReviewTestDB(t)
	ctx := context.Background()
	now := time.Now()

	// Seed file
	file := model.SystemFile{
		ID:         601,
		Storage:    "local",
		Name:       "p.png",
		Path:       "2026/09/p.png",
		Ext:        "png",
		FileType:   model.FileTypeImage,
		Scene:      model.SceneReview,
		OwnerType:  model.OwnerTypeMember,
		OwnerID:    101,
		Visibility: model.VisibilityPrivate,
	}
	_ = db.Create(&file)

	// Seed review
	r := model.ProductReview{
		ID:          10,
		OrderID:     201,
		OrderItemID: 301,
		MemberID:    101,
		RequestID:   "audit-rev-1",
		ProductID:   88,
		SKUID:       881,
		ProductName: "测试商品",
		Rating:      5,
		Content:     "很好非常好！",
		Status:      model.ReviewStatusPendingReview,
		HasImage:    1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = db.Create(&r)
	_ = db.Create(&model.ProductReviewImage{ReviewID: 10, FileID: 601, Sort: 0})

	// 1. Audit: optimistic lock mismatch
	err := reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
		ID:             10,
		ExpectedStatus: model.ReviewStatusApproved, // Mismatch (current is PendingReview)
		Action:         "approve",
	})
	if err == nil {
		t.Fatalf("expected optimistic lock error on status mismatch")
	}

	// 2. Audit approve: status -> 20, file visibility -> public
	err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
		ID:             10,
		ExpectedStatus: model.ReviewStatusPendingReview,
		Action:         "approve",
		Reason:         "符合社区规范",
	})
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}

	var updated model.ProductReview
	_ = db.First(&updated, 10)
	if updated.Status != model.ReviewStatusApproved {
		t.Fatalf("expected status 20, got %d", updated.Status)
	}
	if updated.AuditReason != "符合社区规范" || updated.AuditUserID == nil || *updated.AuditUserID != 1 {
		t.Fatalf("audit info not recorded")
	}

	var updatedFile model.SystemFile
	_ = db.First(&updatedFile, 601)
	if updatedFile.Visibility != model.VisibilityPublic {
		t.Fatalf("expected file visibility to become public (1), got %d", updatedFile.Visibility)
	}

	// Verify audit log
	var logs []model.ProductReviewLog
	_ = db.Where("review_id = ?", 10).Find(&logs)
	if len(logs) != 1 || logs[0].Action != model.ReviewActionAuditApprove {
		t.Fatalf("expected approve log, got %+v", logs)
	}

	// 3. Audit hide: status -> 40, file visibility -> private
	err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
		ID:             10,
		ExpectedStatus: model.ReviewStatusApproved,
		Action:         "hide",
		Reason:         "暂时隐藏",
	})
	if err != nil {
		t.Fatalf("hide failed: %v", err)
	}
	_ = db.First(&updated, 10)
	if updated.Status != model.ReviewStatusHidden {
		t.Fatalf("expected status 40, got %d", updated.Status)
	}
	_ = db.First(&updatedFile, 601)
	if updatedFile.Visibility != model.VisibilityPrivate {
		t.Fatalf("expected file visibility to become private (2), got %d", updatedFile.Visibility)
	}

	// 4. Audit restore: status -> 20
	err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
		ID:             10,
		ExpectedStatus: model.ReviewStatusHidden,
		Action:         "restore",
		Reason:         "恢复展示",
	})
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	_ = db.First(&updated, 10)
	if updated.Status != model.ReviewStatusApproved {
		t.Fatalf("expected status 20, got %d", updated.Status)
	}

	// 5. Reply: reject invalid empty / whitespace
	err = reviewSvc.Reply(ctx, 1, "admin", productreview.ReviewReplyRequest{
		ID:      10,
		Content: "   ",
	})
	if err == nil {
		t.Fatalf("expected empty reply to be rejected")
	}

	// 6. Reply: valid response
	replyText := "感谢您的支持与厚爱，我们将继续努力为您提供优质的产品与服务！"
	err = reviewSvc.Reply(ctx, 1, "admin", productreview.ReviewReplyRequest{
		ID:      10,
		Content: replyText,
	})
	if err != nil {
		t.Fatalf("reply failed: %v", err)
	}

	_ = db.First(&updated, 10)
	if updated.ReplyContent != replyText || updated.ReplyUserID == nil || *updated.ReplyUserID != 1 || updated.ReplyAt == nil {
		t.Fatalf("reply fields not updated: %+v", updated)
	}
}

func TestAdminReview_AuditStateTransitions(t *testing.T) {
	db, _, reviewSvc := setupReviewTestDB(t)
	ctx := context.Background()
	now := time.Now()

	// Seed review in PendingReview (10)
	r := model.ProductReview{
		ID:          20,
		OrderID:     201,
		OrderItemID: 399,
		MemberID:    101,
		RequestID:   "audit-trans-1",
		ProductID:   88,
		SKUID:       881,
		ProductName: "测试商品",
		Rating:      5,
		Content:     "测试状态机",
		Status:      model.ReviewStatusPendingReview,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = db.Create(&r)

	// 1. Status 10: restore and hide must be rejected
	err := reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
		ID:             20,
		ExpectedStatus: model.ReviewStatusPendingReview,
		Action:         "restore",
	})
	if err == nil {
		t.Fatalf("expected restore on pending review to be rejected")
	}

	err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
		ID:             20,
		ExpectedStatus: model.ReviewStatusPendingReview,
		Action:         "hide",
	})
	if err == nil {
		t.Fatalf("expected hide on pending review to be rejected")
	}

	// 2. Status 10 -> approve -> Status 20
	err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
		ID:             20,
		ExpectedStatus: model.ReviewStatusPendingReview,
		Action:         "approve",
		Reason:         "通过",
	})
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}

	// 3. Status 20: approve, reject, restore must be rejected
	for _, invalidAction := range []string{"approve", "reject", "restore"} {
		err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
			ID:             20,
			ExpectedStatus: model.ReviewStatusApproved,
			Action:         invalidAction,
			Reason:         "测试",
		})
		if err == nil {
			t.Fatalf("expected %s on approved review to be rejected", invalidAction)
		}
	}

	// 4. Status 20 -> hide -> Status 40
	err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
		ID:             20,
		ExpectedStatus: model.ReviewStatusApproved,
		Action:         "hide",
		Reason:         "隐藏",
	})
	if err != nil {
		t.Fatalf("hide failed: %v", err)
	}

	// 5. Status 40: approve, reject, hide must be rejected
	for _, invalidAction := range []string{"approve", "reject", "hide"} {
		err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
			ID:             20,
			ExpectedStatus: model.ReviewStatusHidden,
			Action:         invalidAction,
			Reason:         "测试",
		})
		if err == nil {
			t.Fatalf("expected %s on hidden review to be rejected", invalidAction)
		}
	}

	// 6. Status 40 -> restore -> Status 20
	err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
		ID:             20,
		ExpectedStatus: model.ReviewStatusHidden,
		Action:         "restore",
		Reason:         "恢复",
	})
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// 7. Seed a rejected review (Status 30) -> cannot do any actions
	r30 := model.ProductReview{
		ID:          30,
		OrderID:     201,
		OrderItemID: 400,
		MemberID:    101,
		RequestID:   "audit-trans-30",
		ProductID:   88,
		SKUID:       881,
		ProductName: "测试商品",
		Rating:      1,
		Content:     "驳回评价",
		Status:      model.ReviewStatusRejected,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_ = db.Create(&r30)

	for _, action := range []string{"approve", "reject", "hide", "restore"} {
		err = reviewSvc.Audit(ctx, 1, "admin", productreview.ReviewAuditRequest{
			ID:             30,
			ExpectedStatus: model.ReviewStatusRejected,
			Action:         action,
			Reason:         "测试",
		})
		if err == nil {
			t.Fatalf("expected %s on rejected review to be rejected", action)
		}
	}
}
