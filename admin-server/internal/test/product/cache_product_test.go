package product_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"mall/admin_core/internal/model"
	productcategory "mall/admin_core/internal/product/category"
	productproduct "mall/admin_core/internal/product/product"
	"mall/admin_core/pkg/cache"
)

func getTestRueidisCache(t *testing.T) (cache.Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	c, err := cache.NewRueidisCache(cache.Config{
		Addr:         mr.Addr(),
		DisableCache: true,
		UseLuaLock:   true,
	})
	if err != nil {
		t.Fatalf("failed to create rueidis cache: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Close()
	})
	return c, mr
}

// 1. Admin 商品管理直查 MySQL（不走缓存，强一致性）
func TestAdminProductDirectMySQL(t *testing.T) {
	ctx := context.Background()
	c, _ := getTestRueidisCache(t)

	db := productTestDB(t)

	catSvc := productcategory.NewService(db, c)
	catRes, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "数码"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	prodSvc := productproduct.NewService(db, c)
	skuReq := productproduct.ProductSKURequest{
		Price:  "1999.00",
		Status: func(v int8) *int8 { return &v }(1),
	}
	pRes, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:       "初始商品",
		CategoryID: catRes.ID,
		SKUs:       []productproduct.ProductSKURequest{skuReq},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	// 第一次查询详情
	d1, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("Detail query 1 failed: %v", err)
	}
	if d1.Name != "初始商品" {
		t.Fatalf("expected '初始商品', got %q", d1.Name)
	}

	// 直接修改数据库
	if err := db.Model(&model.Product{}).Where("id = ?", pRes.ID).Update("name", "直接篡改DB名称").Error; err != nil {
		t.Fatalf("direct db update: %v", err)
	}

	// 后台再次查询详情：必须立即反映数据库真实值（直查MySQL，无缓存滞后）
	d2, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("Detail query 2 failed: %v", err)
	}
	if d2.Name != "直接篡改DB名称" {
		t.Fatalf("expected admin detail to reflect DB directly, got %q", d2.Name)
	}

	// 后台查询列表：必须直查 MySQL
	rows, total, err := prodSvc.List(ctx, productproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List query failed: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Name != "直接篡改DB名称" {
		t.Fatalf("expected admin list to reflect DB directly, got %q", rows[0].Name)
	}
}

// 2. Admin 创建 / 修改 / 上下架 / 删除 时列表 Generation 递增与详情缓存删除
func TestAdminProductMutationsBumpGenerationAndInvalidateDetail(t *testing.T) {
	ctx := context.Background()
	c, _ := getTestRueidisCache(t)

	db := productTestDB(t)

	catSvc := productcategory.NewService(db, c)
	catRes, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "数码配件"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	listGen0, err := cache.GetProductListGeneration(ctx, c)
	if err != nil {
		t.Fatalf("get list gen: %v", err)
	}

	// A. Create 商品 -> list generation 递增, detail generation 递增
	prodSvc := productproduct.NewService(db, c)
	skuReq := productproduct.ProductSKURequest{
		Price:  "99.00",
		Status: func(v int8) *int8 { return &v }(1),
	}
	pRes, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:       "版本测试商品",
		CategoryID: catRes.ID,
		SKUs:       []productproduct.ProductSKURequest{skuReq},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	listGen1, err := cache.GetProductListGeneration(ctx, c)
	if err != nil || listGen1 <= listGen0 {
		t.Fatalf("expected list generation to bump on Create, got %d <= %d (err=%v)", listGen1, listGen0, err)
	}
	detailGen0, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil || detailGen0 < 1 {
		t.Fatalf("expected detail generation >= 1 on Create, got %d (err=%v)", detailGen0, err)
	}

	// B. Update 商品 -> detail generation 递增, list generation 递增
	d, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	stVal := int8(1)
	_, err = prodSvc.Update(ctx, productproduct.ProductUpdateRequest{
		ID:         pRes.ID,
		Name:       "版本测试商品-改名",
		CategoryID: catRes.ID,
		SKUs: []productproduct.ProductSKURequest{
			{
				ID:     d.SKUs[0].ID,
				Price:  "109.00",
				Status: &stVal,
			},
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	detailGen1, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil || detailGen1 <= detailGen0 {
		t.Fatalf("expected detail generation to bump on Update, got %d <= %d (err=%v)", detailGen1, detailGen0, err)
	}
	listGen2, err := cache.GetProductListGeneration(ctx, c)
	if err != nil || listGen2 <= listGen1 {
		t.Fatalf("expected list generation to bump on Update, got %d <= %d (err=%v)", listGen2, listGen1, err)
	}

	// C. Status 上下架 -> detail generation 递增, list generation 递增
	offStatus := int8(0)
	if err := prodSvc.Status(ctx, productproduct.ProductStatusRequest{ID: pRes.ID, Status: offStatus}); err != nil {
		t.Fatalf("status off: %v", err)
	}

	detailGen2, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil || detailGen2 <= detailGen1 {
		t.Fatalf("expected detail generation to bump on Status change, got %d <= %d (err=%v)", detailGen2, detailGen1, err)
	}
	listGen3, err := cache.GetProductListGeneration(ctx, c)
	if err != nil || listGen3 <= listGen2 {
		t.Fatalf("expected list generation to bump on Status change, got %d <= %d (err=%v)", listGen3, listGen2, err)
	}

	// D. Delete 删除 -> detail generation 递增, list generation 递增
	if err := prodSvc.Delete(ctx, pRes.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	detailGen3, err := cache.GetProductDetailGeneration(ctx, c, pRes.ID)
	if err != nil || detailGen3 <= detailGen2 {
		t.Fatalf("expected detail generation to bump on Delete, got %d <= %d (err=%v)", detailGen3, detailGen2, err)
	}
	listGen4, err := cache.GetProductListGeneration(ctx, c)
	if err != nil || listGen4 <= listGen3 {
		t.Fatalf("expected list generation to bump on Delete, got %d <= %d (err=%v)", listGen4, listGen3, err)
	}
}

// 3. NoopCache 行为验证
func TestProductQueryWithRedisDisabled(t *testing.T) {
	ctx := context.Background()
	noop := cache.NewNoop()
	db := productTestDB(t)

	catSvc := productcategory.NewService(db, noop)
	catRes, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "服装"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	prodSvc := productproduct.NewService(db, noop)
	skuReq := productproduct.ProductSKURequest{
		Price:  "99.00",
		Status: func(v int8) *int8 { return &v }(1),
	}
	pRes, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:       "T恤",
		CategoryID: catRes.ID,
		SKUs:       []productproduct.ProductSKURequest{skuReq},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	d1, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("Detail failed: %v", err)
	}
	if d1.Name != "T恤" {
		t.Fatalf("expected 'T恤', got %q", d1.Name)
	}

	// BumpProductListGeneration 和 GetProductListGeneration 在 NoopCache 下正常工作不报错
	v, err := cache.BumpProductListGeneration(ctx, noop)
	if err != nil || v != 0 {
		t.Fatalf("noop BumpProductListGeneration error: %v, v=%d", err, v)
	}
	gv, err := cache.GetProductListGeneration(ctx, noop)
	if err != nil || gv != 0 {
		t.Fatalf("noop GetProductListGeneration gv=%d (err=%v), expected 0", gv, err)
	}
}

// 4. Redis 不可用时优雅降级
func TestProductQueryWithRedisUnavailable(t *testing.T) {
	mr := miniredis.RunT(t)
	brokenCache, err := cache.NewRueidisCache(cache.Config{
		Addr:         mr.Addr(),
		DisableCache: true,
		UseLuaLock:   true,
	})
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}
	defer brokenCache.Close()

	// 模拟 Redis 运行时宕机
	mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db := productTestDB(t)

	catSvc := productcategory.NewService(db, brokenCache)
	catRes, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "居家"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	prodSvc := productproduct.NewService(db, brokenCache)
	skuReq := productproduct.ProductSKURequest{
		Price:  "59.00",
		Status: func(v int8) *int8 { return &v }(1),
	}
	pRes, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:       "四件套",
		CategoryID: catRes.ID,
		SKUs:       []productproduct.ProductSKURequest{skuReq},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	d1, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("expected Detail to succeed despite Redis failure, got %v", err)
	}
	if d1.Name != "四件套" {
		t.Fatalf("expected '四件套', got %q", d1.Name)
	}

	upReq := productproduct.ProductUpdateRequest{
		ID:         pRes.ID,
		Name:       "纯棉四件套",
		CategoryID: catRes.ID,
		SKUs: []productproduct.ProductSKURequest{
			{
				ID:     d1.SKUs[0].ID,
				Price:  "69.00",
				Status: func(v int8) *int8 { return &v }(1),
			},
		},
	}
	if _, err := prodSvc.Update(ctx, upReq); err != nil {
		t.Fatalf("expected Update to succeed despite Redis failure, got %v", err)
	}

	d2, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("expected Detail to succeed despite Redis failure, got %v", err)
	}
	if d2.Name != "纯棉四件套" {
		t.Fatalf("expected '纯棉四件套', got %q", d2.Name)
	}
}
