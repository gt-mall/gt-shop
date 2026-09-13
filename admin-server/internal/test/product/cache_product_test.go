package product_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"mall/admin_core/internal/model"
	"github.com/alicebob/miniredis/v2"
	productcategory "mall/admin_core/internal/product/category"
	productproduct "mall/admin_core/internal/product/product"
	"mall/admin_core/pkg/cache"
)

func getTestRedisCache(t *testing.T) (cache.Cache, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		client.Close()
	})
	c := cache.NewRedisCache(client)
	return c, client
}

// 1. Admin 商品管理直查 MySQL（不走缓存，强一致性）
func TestAdminProductDirectMySQL(t *testing.T) {
	ctx := context.Background()
	c, client := getTestRedisCache(t)
	defer client.Close()

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

// 2. Admin 创建 / 修改 / 上下架 / 删除 时版本号递增
func TestAdminProductMutationsBumpVersions(t *testing.T) {
	ctx := context.Background()
	c, client := getTestRedisCache(t)
	defer client.Close()

	db := productTestDB(t)

	catSvc := productcategory.NewService(db, c)
	catRes, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "数码配件"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	listVerKey := cache.MiniappProductListVersionKey()
	defer c.Delete(ctx, listVerKey)
	_ = c.Delete(ctx, listVerKey)

	listVer0, _ := c.GetVersion(ctx, listVerKey)

	// A. Create 商品 -> list version 递增
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

	listVer1, _ := c.GetVersion(ctx, listVerKey)
	if listVer1 <= listVer0 {
		t.Fatalf("expected list version to bump on Create, got %d <= %d", listVer1, listVer0)
	}

	detailVerKey := cache.MiniappProductDetailVersionKey(pRes.ID)
	defer c.Delete(ctx, detailVerKey)
	detailVer0, _ := c.GetVersion(ctx, detailVerKey)

	// B. Update 商品 -> detail version 和 list version 都递增
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

	detailVer1, _ := c.GetVersion(ctx, detailVerKey)
	if detailVer1 <= detailVer0 {
		t.Fatalf("expected detail version to bump on Update, got %d <= %d", detailVer1, detailVer0)
	}
	listVer2, _ := c.GetVersion(ctx, listVerKey)
	if listVer2 <= listVer1 {
		t.Fatalf("expected list version to bump on Update, got %d <= %d", listVer2, listVer1)
	}

	// C. Status 上下架 -> detail version 和 list version 都递增
	offStatus := int8(0)
	if err := prodSvc.Status(ctx, productproduct.ProductStatusRequest{ID: pRes.ID, Status: offStatus}); err != nil {
		t.Fatalf("status off: %v", err)
	}

	detailVer2, _ := c.GetVersion(ctx, detailVerKey)
	if detailVer2 <= detailVer1 {
		t.Fatalf("expected detail version to bump on Status, got %d <= %d", detailVer2, detailVer1)
	}
	listVer3, _ := c.GetVersion(ctx, listVerKey)
	if listVer3 <= listVer2 {
		t.Fatalf("expected list version to bump on Status, got %d <= %d", listVer3, listVer2)
	}

	// D. Delete 删除 -> detail version 和 list version 都递增
	if err := prodSvc.Delete(ctx, pRes.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	detailVer3, _ := c.GetVersion(ctx, detailVerKey)
	if detailVer3 <= detailVer2 {
		t.Fatalf("expected detail version to bump on Delete, got %d <= %d", detailVer3, detailVer2)
	}
	listVer4, _ := c.GetVersion(ctx, listVerKey)
	if listVer4 <= listVer3 {
		t.Fatalf("expected list version to bump on Delete, got %d <= %d", listVer4, listVer3)
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

	// BumpVersion 和 GetVersion 在 NoopCache 下正常工作不报错
	v, err := noop.BumpVersion(ctx, "any_key")
	if err != nil || v != 1 {
		t.Fatalf("noop BumpVersion error: %v, v=%d", err, v)
	}
	gv, err := noop.GetVersion(ctx, "any_key")
	if err != nil || gv != 0 {
		t.Fatalf("noop GetVersion error: %v, gv=%d", err, gv)
	}
}

// 4. Redis 不可用时优雅降级
func TestProductQueryWithRedisUnavailable(t *testing.T) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	_ = client.Close()
	brokenCache := cache.NewRedisCache(client)

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
