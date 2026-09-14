package product_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"

	"mall/admin_core/internal/model"
	productcategory "mall/admin_core/internal/product/category"
	productproduct "mall/admin_core/internal/product/product"
)

func createCustomTestFile(t *testing.T, db *gorm.DB, scene, fileType string, visibility int8, deleted bool) int64 {
	t.Helper()
	now := time.Now()
	var delAt gorm.DeletedAt
	if deleted {
		delAt = gorm.DeletedAt{Time: now, Valid: true}
	}
	f := model.SystemFile{
		GroupID:    0,
		Storage:    "local",
		Name:       fmt.Sprintf("file_%d.dat", now.UnixNano()),
		Path:       fmt.Sprintf("2026/09/%d.dat", now.UnixNano()),
		Ext:        "png",
		FileType:   fileType,
		Scene:      scene,
		Visibility: visibility,
		Size:       100,
		CreatedAt:  now,
		DeletedAt:  delAt,
	}
	if err := db.Create(&f).Error; err != nil {
		t.Fatal(err)
	}
	return f.ID
}

func TestImageValidation_ProductAndCategory(t *testing.T) {
	db := productTestDB(t)
	ctx := context.Background()

	catSvc := productcategory.NewService(db)
	prodSvc := productproduct.NewService(db)

	// Valid image: admin_material + image + public
	validImgID := createCustomTestFile(t, db, model.SceneAdminMaterial, model.FileTypeImage, model.VisibilityPublic, false)
	// Invalid 1: admin_material + video + public
	videoID := createCustomTestFile(t, db, model.SceneAdminMaterial, model.FileTypeVideo, model.VisibilityPublic, false)
	// Invalid 2: after_sale_proof + image + private
	proofImgID := createCustomTestFile(t, db, model.SceneAfterSaleProof, model.FileTypeImage, model.VisibilityPrivate, false)
	// Invalid 3: member_avatar + image + public
	avatarID := createCustomTestFile(t, db, model.SceneMemberAvatar, model.FileTypeImage, model.VisibilityPublic, false)
	// Invalid 4: admin_material + image + public + deleted
	deletedImgID := createCustomTestFile(t, db, model.SceneAdminMaterial, model.FileTypeImage, model.VisibilityPublic, true)
	// Invalid 5: admin_material + image + private
	privateMaterialID := createCustomTestFile(t, db, model.SceneAdminMaterial, model.FileTypeImage, model.VisibilityPrivate, false)

	// 1. Category ImageID validation
	t.Run("Category ImageID Validation", func(t *testing.T) {
		// Valid image succeeds
		cat, err := catSvc.Create(ctx, productcategory.CreateRequest{
			Name:    "有效分类",
			ImageID: &validImgID,
		})
		if err != nil || cat.ID <= 0 {
			t.Fatalf("expected valid category image to succeed, got %v", err)
		}

		// Video rejected
		_, err = catSvc.Create(ctx, productcategory.CreateRequest{
			Name:    "视频分类",
			ImageID: &videoID,
		})
		if !errors.Is(err, productcategory.ErrImageNotFound) {
			t.Fatalf("expected ErrImageNotFound for video category image, got %v", err)
		}

		// After-sale proof rejected
		_, err = catSvc.Create(ctx, productcategory.CreateRequest{
			Name:    "凭证分类",
			ImageID: &proofImgID,
		})
		if !errors.Is(err, productcategory.ErrImageNotFound) {
			t.Fatalf("expected ErrImageNotFound for proof category image, got %v", err)
		}

		// Member avatar rejected
		_, err = catSvc.Create(ctx, productcategory.CreateRequest{
			Name:    "头像分类",
			ImageID: &avatarID,
		})
		if !errors.Is(err, productcategory.ErrImageNotFound) {
			t.Fatalf("expected ErrImageNotFound for avatar category image, got %v", err)
		}

		// Deleted file rejected
		_, err = catSvc.Create(ctx, productcategory.CreateRequest{
			Name:    "删除分类",
			ImageID: &deletedImgID,
		})
		if !errors.Is(err, productcategory.ErrImageNotFound) {
			t.Fatalf("expected ErrImageNotFound for deleted category image, got %v", err)
		}

		// Private admin material rejected
		_, err = catSvc.Create(ctx, productcategory.CreateRequest{
			Name:    "私密素材分类",
			ImageID: &privateMaterialID,
		})
		if !errors.Is(err, productcategory.ErrImageNotFound) {
			t.Fatalf("expected ErrImageNotFound for private material category image, got %v", err)
		}
	})

	// 2. Product MainImageID, Gallery, and SKU ImageID validation
	t.Run("Product and SKU Image Validation", func(t *testing.T) {
		cat, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "商品分类"})
		if err != nil {
			t.Fatal(err)
		}

		// Valid product creation
		skuCode := "SKU_VALID_IMG"
		prod, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
			Name:           "有效商品",
			CategoryID:     cat.ID,
			MainImageID:    &validImgID,
			GalleryFileIDs: []int64{validImgID},
			SKUs: []productproduct.ProductSKURequest{
				{
					Price:       "10.00",
					MarketPrice: "20.00",
					CostPrice:   "5.00",
					SKUCode:     &skuCode,
					ImageID:     &validImgID,
				},
			},
		})
		if err != nil || prod.ID <= 0 {
			t.Fatalf("expected valid product image creation to succeed, got %v", err)
		}

		// MainImageID = Video rejected
		_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
			Name:        "视频主图商品",
			CategoryID:  cat.ID,
			MainImageID: &videoID,
			SKUs: []productproduct.ProductSKURequest{
				{Price: "10.00", MarketPrice: "20.00", CostPrice: "5.00"},
			},
		})
		if !errors.Is(err, productproduct.ErrImageNotFound) {
			t.Fatalf("expected ErrImageNotFound for video main image, got %v", err)
		}

		// MainImageID = Proof rejected
		_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
			Name:        "凭证主图商品",
			CategoryID:  cat.ID,
			MainImageID: &proofImgID,
			SKUs: []productproduct.ProductSKURequest{
				{Price: "10.00", MarketPrice: "20.00", CostPrice: "5.00"},
			},
		})
		if !errors.Is(err, productproduct.ErrImageNotFound) {
			t.Fatalf("expected ErrImageNotFound for proof main image, got %v", err)
		}

		// Gallery with Video rejected
		_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
			Name:           "视频图集商品",
			CategoryID:     cat.ID,
			MainImageID:    &validImgID,
			GalleryFileIDs: []int64{videoID},
			SKUs: []productproduct.ProductSKURequest{
				{Price: "10.00", MarketPrice: "20.00", CostPrice: "5.00"},
			},
		})
		if !errors.Is(err, productproduct.ErrImageNotFound) {
			t.Fatalf("expected ErrImageNotFound for video gallery, got %v", err)
		}

		// SKU ImageID = Avatar rejected
		skuCode2 := "SKU_AVATAR_IMG"
		_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
			Name:        "头像SKU商品",
			CategoryID:  cat.ID,
			MainImageID: &validImgID,
			SKUs: []productproduct.ProductSKURequest{
				{
					Price:       "10.00",
					MarketPrice: "20.00",
					CostPrice:   "5.00",
					SKUCode:     &skuCode2,
					ImageID:     &avatarID,
				},
			},
		})
		if !errors.Is(err, productproduct.ErrImageNotFound) {
			t.Fatalf("expected ErrImageNotFound for avatar sku image, got %v", err)
		}
	})
}
