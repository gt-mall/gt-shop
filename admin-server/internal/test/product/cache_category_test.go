package product_test

import (
	"context"
	"testing"

	"mall/admin_core/internal/model"
	productcategory "mall/admin_core/internal/product/category"
	"mall/admin_core/pkg/cache"
)

func TestCategoryCacheLifecycleWithRedis(t *testing.T) {
	ctx := context.Background()
	c, mr := getTestRueidisCache(t)

	db := productTestDB(t)

	catSvc := productcategory.NewService(db, c)
	treeKey := cache.CategoryTreeKey()
	optKey := cache.CategoryOptionsKey()
	defer c.Delete(ctx, treeKey, optKey)

	_ = c.Delete(ctx, treeKey, optKey)

	// 1. 创建分类
	c1, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "生鲜"})
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	// 2. 第一次查询 List: miss -> 写入缓存
	list1, err := catSvc.List(ctx)
	if err != nil {
		t.Fatalf("List query 1 failed: %v", err)
	}
	if len(list1) != 1 || list1[0].Name != "生鲜" {
		t.Fatalf("unexpected list1: %+v", list1)
	}

	// 验证 Redis 中存在 category tree 缓存
	if !mr.Exists(treeKey) {
		t.Fatalf("expected tree key %s to exist in redis", treeKey)
	}

	// 3. 在底层 DB 直接篡改
	if err := db.Model(&model.ProductCategory{}).Where("id = ?", c1.ID).Update("name", "直接改DB生鲜").Error; err != nil {
		t.Fatalf("direct db update: %v", err)
	}

	// 4. 第二次查询 List: 命中缓存，依然返回 "生鲜"
	list2, err := catSvc.List(ctx)
	if err != nil {
		t.Fatalf("List query 2 failed: %v", err)
	}
	if len(list2) != 1 || list2[0].Name != "生鲜" {
		t.Fatalf("expected cached '生鲜', got %q", list2[0].Name)
	}

	// 5. 更新分类: 应该删除缓存
	newName := "新鲜水果"
	_, err = catSvc.Update(ctx, []byte(`{"id":1,"name":"新鲜水果"}`), productcategory.UpdateRequest{
		ID:   c1.ID,
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update category failed: %v", err)
	}

	// 验证缓存已被删除
	if mr.Exists(treeKey) {
		t.Fatalf("expected tree key %s to be deleted after update", treeKey)
	}

	// 6. 第三次查询: 重新查 DB，返回 "新鲜水果"
	list3, err := catSvc.List(ctx)
	if err != nil {
		t.Fatalf("List query 3 failed: %v", err)
	}
	if len(list3) != 1 || list3[0].Name != "新鲜水果" {
		t.Fatalf("expected '新鲜水果', got %q", list3[0].Name)
	}
}
