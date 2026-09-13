package product_test

import (
	"context"
	"testing"

	"mall/admin_core/internal/model"
	productcategory "mall/admin_core/internal/product/category"
	"mall/admin_core/pkg/cache"
)

func TestCategoryDirectMySQLAndProductListGenBump(t *testing.T) {
	ctx := context.Background()
	c, _ := getTestRueidisCache(t)

	db := productTestDB(t)
	catSvc := productcategory.NewService(db, c)

	gen0, _ := cache.GetProductListGeneration(ctx, c)

	// 1. 创建分类: 触发 product list generation 增加
	c1, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "生鲜"})
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	gen1, _ := cache.GetProductListGeneration(ctx, c)
	if gen1 <= gen0 {
		t.Fatalf("expected product list generation bumped from %d, got %d", gen0, gen1)
	}

	// 2. 第一次查询 List: 直接查 MySQL
	list1, err := catSvc.List(ctx)
	if err != nil {
		t.Fatalf("List query 1 failed: %v", err)
	}
	if len(list1) != 1 || list1[0].Name != "生鲜" {
		t.Fatalf("unexpected list1: %+v", list1)
	}

	// 3. 在底层 DB 直接篡改，证明无缓存，立即反映
	if err := db.Model(&model.ProductCategory{}).Where("id = ?", c1.ID).Update("name", "直接改DB生鲜").Error; err != nil {
		t.Fatalf("direct db update: %v", err)
	}

	// 4. 第二次查询 List: 直接返回 DB 最新值 "直接改DB生鲜"
	list2, err := catSvc.List(ctx)
	if err != nil {
		t.Fatalf("List query 2 failed: %v", err)
	}
	if len(list2) != 1 || list2[0].Name != "直接改DB生鲜" {
		t.Fatalf("expected immediate reflection of DB '直接改DB生鲜', got %q", list2[0].Name)
	}

	// 5. 更新分类: 再次 bump product list generation
	newName := "新鲜水果"
	_, err = catSvc.Update(ctx, []byte(`{"id":1,"name":"新鲜水果"}`), productcategory.UpdateRequest{
		ID:   c1.ID,
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update category failed: %v", err)
	}

	gen2, _ := cache.GetProductListGeneration(ctx, c)
	if gen2 <= gen1 {
		t.Fatalf("expected product list generation bumped from %d, got %d", gen1, gen2)
	}

	// 6. 第三次查询: 返回 "新鲜水果"
	list3, err := catSvc.List(ctx)
	if err != nil {
		t.Fatalf("List query 3 failed: %v", err)
	}
	if len(list3) != 1 || list3[0].Name != "新鲜水果" {
		t.Fatalf("expected '新鲜水果', got %q", list3[0].Name)
	}
}
