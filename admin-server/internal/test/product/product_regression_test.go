package product_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/internal/model"
	category "mall/admin_core/internal/product/category"
	product "mall/admin_core/internal/product/product"
	"mall/admin_core/pkg/response"
	"mall/admin_core/pkg/util"
)

func TestProductHTMLPreservesNestedContent(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{`<section><p>first</p><p>second</p></section>`, `<p>first</p><p>second</p>`},
		{`<p><custom>normal<strong>bold</strong>tail</custom></p>`, `<p>normal<strong>bold</strong>tail</p>`},
		{`<section><custom>a<b>b</b></custom><custom>c<i>d</i></custom></section>`, `a<b>b</b>c<i>d</i>`},
		{`<p><mark>normal<strong>bold</strong></mark><sub>2</sub><sup>3</sup></p>`, `<p><mark>normal<strong>bold</strong></mark><sub>2</sub><sup>3</sup></p>`},
		{`<section><script>bad()</script><p onclick="bad()">safe</p><a href="javascript:bad()">link</a></section>`, `<p>safe</p><a>link</a>`},
	} {
		t.Run(tc.input, func(t *testing.T) {
			if got := util.CleanHTML(tc.input); got != tc.want {
				t.Fatalf("CleanHTML = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProductImageValidationUsesTransactionConnection(t *testing.T) {
	for _, imageField := range []string{"main", "gallery", "sku"} {
		t.Run(imageField, func(t *testing.T) {
			db := productTestDB(t) // One connection: an accidental pool query would deadlock.
			svc := product.NewService(db)
			imageID := createTestImage(t, db)
			created, err := svc.Create(context.Background(), product.ProductCreateRequest{
				Name: "image update", SKUs: []product.ProductSKURequest{{Price: "199.90"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			var sku model.ProductSKU
			if err := db.Where("product_id = ?", created.ID).First(&sku).Error; err != nil {
				t.Fatal(err)
			}
			req := product.ProductUpdateRequest{
				ID: created.ID, Name: "image update",
				SKUs: []product.ProductSKURequest{{ID: sku.ID, Price: "199.90"}},
			}
			switch imageField {
			case "main":
				req.MainImageID = &imageID
			case "gallery":
				req.GalleryFileIDs = []int64{imageID}
			case "sku":
				req.SKUs[0].ImageID = &imageID
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if _, err := svc.Update(ctx, req); err != nil {
				t.Fatal(err)
			}
			detail, err := svc.Detail(ctx, created.ID)
			if err != nil {
				t.Fatal(err)
			}
			switch imageField {
			case "main":
				if detail.MainImageID == nil || *detail.MainImageID != imageID {
					t.Fatal("main image was not persisted")
				}
			case "gallery":
				if len(detail.GalleryFileIDs) != 1 || detail.GalleryFileIDs[0] != imageID {
					t.Fatal("gallery image was not persisted")
				}
			case "sku":
				if detail.SKUs[0].ImageID == nil || *detail.SKUs[0].ImageID != imageID {
					t.Fatal("SKU image was not persisted")
				}
			}
			sqlDB, _ := db.DB()
			if sqlDB.Stats().WaitCount != 0 {
				t.Fatal("image validation borrowed another connection during the transaction")
			}
		})
	}
}

func TestProductPublishWithSingleConnection(t *testing.T) {
	db := productTestDB(t)
	svc := product.NewService(db)
	cat, err := category.NewService(db).Create(context.Background(), category.CreateRequest{Name: "category"})
	if err != nil {
		t.Fatal(err)
	}
	imageID := createTestImage(t, db)
	created, err := svc.Create(context.Background(), product.ProductCreateRequest{
		Name: "publish", CategoryID: cat.ID, MainImageID: &imageID,
		SKUs: []product.ProductSKURequest{{Price: "19.90"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := svc.Status(ctx, product.ProductStatusRequest{ID: created.ID, Status: 1}); err != nil {
		t.Fatal(err)
	}
	var saved model.Product
	if err := db.First(&saved, created.ID).Error; err != nil || saved.Status != 1 {
		t.Fatalf("publish was not persisted: status=%d, err=%v", saved.Status, err)
	}
}

func TestCategoryImageOmittedNullAndReplacement(t *testing.T) {
	db := productTestDB(t)
	svc := category.NewService(db)
	imageID := createTestImage(t, db)
	otherImage := model.SystemFile{Storage: "local", Name: "replacement.png", Path: "2026/09/replacement.png", Ext: "png"}
	if err := db.Create(&otherImage).Error; err != nil {
		t.Fatal(err)
	}
	otherImageID := otherImage.ID
	cat, err := svc.Create(context.Background(), category.CreateRequest{Name: "image", ImageID: &imageID})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.POST("/update", category.NewAPI(svc).Update)
	for _, tc := range []struct {
		name, fields string
		want         *int64
	}{
		{"omitted preserves", `"name":"renamed"`, &imageID},
		{"replacement", fmt.Sprintf(`"image_id":%d`, otherImageID), &otherImageID},
		{"null clears", `"image_id":null`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			req := httptest.NewRequest("POST", "/update", strings.NewReader(fmt.Sprintf(`{"id":%d,%s}`, cat.ID, tc.fields))).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			var result response.Response[category.CategoryResponse]
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Code != 1 {
				t.Fatalf("update failed: %s, err=%v", w.Body, err)
			}
			var saved model.ProductCategory
			if err := db.First(&saved, cat.ID).Error; err != nil {
				t.Fatal(err)
			}
			for _, got := range []*int64{result.Data.ImageID, saved.ImageID} {
				if (got == nil) != (tc.want == nil) || got != nil && *got != *tc.want {
					t.Fatalf("response or stored image mismatch: got=%v, want=%v", got, tc.want)
				}
			}
		})
	}
}

func TestProductInvalidRequestsReturnSingleJSON(t *testing.T) {
	prodAPI := product.NewAPI(product.NewService(nil))
	catAPI := category.NewAPI(category.NewService(nil))
	for name, handler := range map[string]gin.HandlerFunc{
		"detail": prodAPI.Detail, "update": prodAPI.Update, "status": prodAPI.Status,
		"delete": prodAPI.Delete, "category_delete": catAPI.Delete,
	} {
		t.Run(name, func(t *testing.T) {
			router := gin.New()
			router.POST("/", handler)
			for _, body := range []string{`{`, `{}`, `{"id":0}`, `{"id":-1,"name":"test","skus":[{}]}`, `{"id":"invalid"}`} {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)
				var result response.Response[any]
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Code != 0 || result.Message == "" {
					t.Fatalf("body=%s: expected a single error envelope, got=%s, err=%v", body, w.Body, err)
				}
			}
		})
	}
}

func TestCategoryConcurrentUpdatesRejectCycle(t *testing.T) {
	testCategoryConcurrentUpdates(t, productTestDB(t))
}

// Use a disposable MySQL database, never the application's DATABASE_DSN.
func TestCategoryConcurrentUpdatesMySQL(t *testing.T) {
	dsn := os.Getenv("PRODUCT_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("PRODUCT_TEST_MYSQL_DSN is not set; requires an isolated MySQL database")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("could not connect to the isolated MySQL database")
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	sqlDB.SetMaxOpenConns(4)
	var databaseName string
	if err := db.Raw("SELECT DATABASE()").Scan(&databaseName).Error; err != nil || !strings.HasPrefix(databaseName, "mall_product_regression_") {
		t.Fatal("test database name must start with mall_product_regression_")
	}
	if err := db.AutoMigrate(&model.ProductCategory{}); err != nil {
		t.Fatal(err)
	}
	testCategoryConcurrentUpdates(t, db)
}

func testCategoryConcurrentUpdates(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := category.NewService(db)
	a, err := svc.Create(context.Background(), category.CreateRequest{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), category.CreateRequest{Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Unscoped().Where("id IN ?", []int64{a.ID, b.ID}).Delete(&model.ProductCategory{}) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	firstReady := make(chan struct{})
	releaseFirst := make(chan struct{})
	var readyOnce, releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(releaseFirst) }) }
	defer unblock()
	type firstWriterKey struct{}
	// Pause A after validation but before writing; B must wait for A's commit
	// and then validate against the changed tree instead of also succeeding.
	if err := db.Callback().Update().Before("gorm:begin_transaction").Register("product_regression:pause", func(tx *gorm.DB) {
		if tx.Statement.Context.Value(firstWriterKey{}) != true {
			return
		}
		readyOnce.Do(func() { close(firstReady) })
		select {
		case <-releaseFirst:
		case <-ctx.Done():
			tx.AddError(ctx.Err())
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Callback().Update().Remove("product_regression:pause") })
	update := func(ctx context.Context, id, parentID int64) error {
		body := []byte(fmt.Sprintf(`{"id":%d,"parent_id":%d}`, id, parentID))
		_, err := svc.Update(ctx, body, category.UpdateRequest{ID: id, ParentID: &parentID})
		return err
	}
	firstDone := make(chan error, 1)
	go func() { firstDone <- update(context.WithValue(ctx, firstWriterKey{}, true), a.ID, b.ID) }()
	select {
	case <-firstReady:
	case err := <-firstDone:
		t.Fatalf("first update did not reach the write: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	secondDone := make(chan error, 1)
	go func() { secondDone <- update(ctx, b.ID, a.ID) }()
	select {
	case err := <-secondDone:
		t.Errorf("second update finished before the first committed: %v", err)
		unblock()
		<-firstDone
		return
	case <-time.After(100 * time.Millisecond):
		unblock()
	}
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; !errors.Is(err, category.ErrParentCategoryCycle) {
		t.Fatalf("expected cycle rejection after first commit, got %v", err)
	}
	var savedA, savedB model.ProductCategory
	if err := db.First(&savedA, a.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&savedB, b.ID).Error; err != nil {
		t.Fatal(err)
	}
	if savedA.ParentID != b.ID || savedB.ParentID != 0 {
		t.Fatalf("unexpected tree: A parent=%d, B parent=%d", savedA.ParentID, savedB.ParentID)
	}
}

func TestProductCategoryFilterHandlesExistingCycle(t *testing.T) {
	db := productTestDB(t)
	a := model.ProductCategory{Name: "A"}
	b := model.ProductCategory{Name: "B"}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	b.ParentID = a.ID
	if err := db.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&a).Update("parent_id", b.ID).Error; err != nil {
		t.Fatal(err)
	}
	svc := product.NewService(db)
	for _, id := range []int64{a.ID, b.ID} {
		if _, err := svc.Create(context.Background(), product.ProductCreateRequest{
			Name: "filter", CategoryID: id, SKUs: []product.ProductSKURequest{{Price: "9.90"}},
		}); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rows, total, err := svc.List(ctx, product.ProductListRequest{CategoryID: &a.ID})
	if err != nil || total != 2 || len(rows) != 2 {
		t.Fatalf("cyclic category filter: rows=%d total=%d err=%v", len(rows), total, err)
	}
}
