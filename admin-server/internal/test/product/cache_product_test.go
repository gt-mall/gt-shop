package product_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"mall/admin_core/internal/model"
	productcategory "mall/admin_core/internal/product/category"
	productproduct "mall/admin_core/internal/product/product"
	"mall/admin_core/pkg/cache"
)

func getTestRedisCache(t *testing.T) (cache.Cache, *redis.Client) {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Skipf("skipping Redis cache integration tests: local redis not reachable: %v", err)
		return nil, nil
	}
	c := cache.NewRedisCache(client)
	return c, client
}

// 2. Redis 启用情况下：
//    - 第一次商品查询 MISS → MySQL → 写缓存
//    - 第二次查询命中缓存
// 5. 商品修改：
//    - MySQL 修改成功后删除对应详情缓存
//    - 再查询得到最新数据
func TestProductCacheLifecycleWithRedis(t *testing.T) {
	ctx := context.Background()
	c, client := getTestRedisCache(t)
	defer client.Close()

	db := productTestDB(t)

	// 1. 创建分类与商品
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

	detailKey := cache.ProductDetailKey(pRes.ID)
	miniappDetailKey := cache.MiniappProductDetailKey(pRes.ID)
	defer c.Delete(ctx, detailKey, miniappDetailKey)

	// 确保测试前没有旧缓存
	_ = c.Delete(ctx, detailKey, miniappDetailKey)

	// 第一次查询：MISS → 查 MySQL → 写入缓存
	d1, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("Detail query 1 failed: %v", err)
	}
	if d1.Name != "初始商品" {
		t.Fatalf("expected '初始商品', got %q", d1.Name)
	}

	// 检查 Redis 中确实已存在缓存
	exists, err := client.Exists(ctx, detailKey).Result()
	if err != nil || exists != 1 {
		t.Fatalf("expected key %s to exist in redis after first query, exists=%d, err=%v", detailKey, exists, err)
	}

	// 验证缓存命中：在底层 MySQL 中直接篡改商品名称
	if err := db.Model(&model.Product{}).Where("id = ?", pRes.ID).Update("name", "直接篡改DB名称").Error; err != nil {
		t.Fatalf("direct db update: %v", err)
	}

	// 第二次查询：命中缓存，因此依然返回缓存中的“初始商品”，而不是数据库里的“直接篡改DB名称”
	d2, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("Detail query 2 failed: %v", err)
	}
	if d2.Name != "初始商品" {
		t.Fatalf("expected cache hit with '初始商品', but got %q", d2.Name)
	}

	// 通过服务更新商品：应成功更新数据库并主动删除缓存
	newName := "正式更新后的商品"
	upReq := productproduct.ProductUpdateRequest{
		ID:         pRes.ID,
		Name:       newName,
		CategoryID: catRes.ID,
		SKUs: []productproduct.ProductSKURequest{
			{
				ID:     d1.SKUs[0].ID,
				Price:  "2099.00",
				Status: func(v int8) *int8 { return &v }(1),
			},
		},
	}
	if _, err := prodSvc.Update(ctx, upReq); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 确认缓存 key 已被主动删除
	existsAfterUpdate, err := client.Exists(ctx, detailKey).Result()
	if err != nil || existsAfterUpdate != 0 {
		t.Fatalf("expected key %s to be deleted after update, exists=%d", detailKey, existsAfterUpdate)
	}

	// 第三次查询：缓存已删除，重新查 MySQL 并写入最新缓存
	d3, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("Detail query 3 failed: %v", err)
	}
	if d3.Name != newName {
		t.Fatalf("expected fresh name %q after update, got %q", newName, d3.Name)
	}
}

// 3. Redis 关闭（NoopCache）：
//    - 商品查询正常直接走 MySQL
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

	// 查询详情正常返回
	d1, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("Detail failed: %v", err)
	}
	if d1.Name != "T恤" {
		t.Fatalf("expected 'T恤', got %q", d1.Name)
	}

	// 直接修改 DB
	if err := db.Model(&model.Product{}).Where("id = ?", pRes.ID).Update("name", "衬衫").Error; err != nil {
		t.Fatalf("update db: %v", err)
	}

	// 无缓存情况下，再次查询立即看到最新 DB 数据
	d2, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("Detail failed: %v", err)
	}
	if d2.Name != "衬衫" {
		t.Fatalf("expected '衬衫', got %q", d2.Name)
	}
}

// 4. Redis 不可用：
//    - 商品查询仍可通过 MySQL 成功返回
//    - 修改数据仍可成功保存至 MySQL
func TestProductQueryWithRedisUnavailable(t *testing.T) {
	ctx := context.Background()
	// 使用已关闭的客户端模拟 Redis 故障/网络不可达
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

	// Redis GET 异常时，应当优雅降级查 MySQL，查询仍成功返回
	d1, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("expected Detail to succeed despite Redis failure, got %v", err)
	}
	if d1.Name != "四件套" {
		t.Fatalf("expected '四件套', got %q", d1.Name)
	}

	// Redis DELETE 异常时，应当不阻断数据库更新
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

	// 再次查询仍然能成功获取最新数据
	d2, err := prodSvc.Detail(ctx, pRes.ID)
	if err != nil {
		t.Fatalf("expected Detail to succeed despite Redis failure, got %v", err)
	}
	if d2.Name != "纯棉四件套" {
		t.Fatalf("expected '纯棉四件套', got %q", d2.Name)
	}
}

func TestProductListCacheWithRedis(t *testing.T) {
	ctx := context.Background()
	c, client := getTestRedisCache(t)
	defer client.Close()

	db := productTestDB(t)

	catSvc := productcategory.NewService(db, c)
	catRes, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "数码配件"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	prodSvc := productproduct.NewService(db, c)
	skuReq := productproduct.ProductSKURequest{
		Price:  "29.00",
		Status: func(v int8) *int8 { return &v }(1),
	}
	_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:       "手机壳",
		CategoryID: catRes.ID,
		SKUs:       []productproduct.ProductSKURequest{skuReq},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	// 第一次查列表：查 MySQL 并写入缓存
	rows1, total1, err := prodSvc.List(ctx, productproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List query 1 failed: %v", err)
	}
	if total1 != 1 || len(rows1) != 1 || rows1[0].Name != "手机壳" {
		t.Fatalf("unexpected list result: total=%d, len=%d", total1, len(rows1))
	}

	// 在底层 MySQL 中直接修改
	if err := db.Model(&model.Product{}).Where("name = ?", "手机壳").Update("name", "篡改手机壳").Error; err != nil {
		t.Fatalf("update db: %v", err)
	}

	// 第二次查列表：命中列表缓存，依然返回 "手机壳"
	rows2, total2, err := prodSvc.List(ctx, productproduct.ProductListRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List query 2 failed: %v", err)
	}
	if total2 != 1 || len(rows2) != 1 || rows2[0].Name != "手机壳" {
		t.Fatalf("expected cached list with '手机壳', got %q", rows2[0].Name)
	}
}
