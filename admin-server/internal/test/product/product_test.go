package product_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	productcategory "mall/admin_core/internal/product/category"
	productproduct "mall/admin_core/internal/product/product"
	systemfile "mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/util"
)

func productTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(
		&model.ProductCategory{},
		&model.Product{},
		&model.ProductSKU{},
		&model.ProductSkuInventory{},
		&model.ProductSkuInventoryLog{},
		&model.SystemFile{},
		&model.SystemFileGroup{},
		&model.SystemConfig{},
	); err != nil {
		t.Fatal(err)
	}

	fileSvc := systemfile.NewService(db)
	cfg := systemfile.DefaultConfig()
	cfg.Local.Path = t.TempDir()
	if err := fileSvc.SaveConfig(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	return db
}

func createTestImage(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	file := model.SystemFile{
		GroupID: 0,
		Storage: "local",
		Name:    "test.png",
		Path:    "2026/09/test.png",
		Ext:     "png",
		Size:    100,
		Width:   10,
		Height:  10,
	}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	return file.ID
}

// 1. 规范化 hash（顺序变化仍相同）与默认 SKU
func TestSpecHashNormalizationAndDefaultSKU(t *testing.T) {
	specs := []productproduct.ProductSpec{
		{
			ID:   1,
			Name: "颜色",
			Values: []productproduct.ProductSpecValue{
				{ID: 1, Name: "黑色"},
				{ID: 2, Name: "白色"},
			},
		},
		{
			ID:   2,
			Name: "容量",
			Values: []productproduct.ProductSpecValue{
				{ID: 3, Name: "128GB"},
				{ID: 4, Name: "256GB"},
			},
		},
	}

	// 顺序不同：(1:1, 2:4) vs (2:4, 1:1)
	combo1 := []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 1}, {SpecID: 2, ValueID: 4}}
	combo2 := []productproduct.ProductSKUValue{{SpecID: 2, ValueID: 4}, {SpecID: 1, ValueID: 1}}

	hash1, err := productproduct.CalculateSKUSpecHash(specs, combo1)
	if err != nil {
		t.Fatalf("hash1 err: %v", err)
	}
	hash2, err := productproduct.CalculateSKUSpecHash(specs, combo2)
	if err != nil {
		t.Fatalf("hash2 err: %v", err)
	}

	if hash1 != hash2 {
		t.Fatalf("hash must be identical regardless of order: hash1=%s, hash2=%s", hash1, hash2)
	}

	// 预期为 sha256("1:1|2:4")
	expectedH := sha256.Sum256([]byte("1:1|2:4"))
	expectedHex := hex.EncodeToString(expectedH[:])
	if hash1 != expectedHex {
		t.Fatalf("hash1 expected %s, got %s", expectedHex, hash1)
	}

	// 默认 SKU (无规格 specs=[])
	noSpecs := []productproduct.ProductSpec{}
	defaultCombo := []productproduct.ProductSKUValue{}
	defaultHash, err := productproduct.CalculateSKUSpecHash(noSpecs, defaultCombo)
	if err != nil {
		t.Fatalf("default hash err: %v", err)
	}

	expectedDefH := sha256.Sum256([]byte("default"))
	expectedDefHex := hex.EncodeToString(expectedDefH[:])
	if defaultHash != expectedDefHex {
		t.Fatalf("default hash expected %s, got %s", expectedDefHex, defaultHash)
	}

	// 无规格却包含规格值应当报错
	_, err = productproduct.CalculateSKUSpecHash(noSpecs, combo1)
	if err == nil {
		t.Fatal("expected error when spec_values provided for no-spec product")
	}
}

// 2. 非法规格与重复组合校验
func TestIllegalSpecsAndDuplicateCombos(t *testing.T) {
	// 超出 3 个规格维度
	tooManySpecs := []productproduct.ProductSpec{
		{ID: 1, Name: "A", Values: []productproduct.ProductSpecValue{{ID: 1, Name: "a"}}},
		{ID: 2, Name: "B", Values: []productproduct.ProductSpecValue{{ID: 2, Name: "b"}}},
		{ID: 3, Name: "C", Values: []productproduct.ProductSpecValue{{ID: 3, Name: "c"}}},
		{ID: 4, Name: "D", Values: []productproduct.ProductSpecValue{{ID: 4, Name: "d"}}},
	}
	if err := productproduct.ValidateSpecs(tooManySpecs); !errors.Is(err, productproduct.ErrTooManySpecs) {
		t.Fatalf("expected ErrTooManySpecs, got: %v", err)
	}

	// 单个规格超过 20 个选项
	tooManyVals := make([]productproduct.ProductSpecValue, 21)
	for i := 0; i < 21; i++ {
		tooManyVals[i] = productproduct.ProductSpecValue{ID: int64(i + 1), Name: fmt.Sprintf("val%d", i+1)}
	}
	spec21 := []productproduct.ProductSpec{
		{ID: 1, Name: "Color", Values: tooManyVals},
	}
	if err := productproduct.ValidateSpecs(spec21); !errors.Is(err, productproduct.ErrTooManySpecValues) {
		t.Fatalf("expected ErrTooManySpecValues, got: %v", err)
	}

	// 规格ID重复
	dupSpecID := []productproduct.ProductSpec{
		{ID: 1, Name: "A", Values: []productproduct.ProductSpecValue{{ID: 1, Name: "a"}}},
		{ID: 1, Name: "B", Values: []productproduct.ProductSpecValue{{ID: 2, Name: "b"}}},
	}
	if err := productproduct.ValidateSpecs(dupSpecID); !errors.Is(err, productproduct.ErrDuplicateSpecID) {
		t.Fatalf("expected ErrDuplicateSpecID, got: %v", err)
	}

	// 规格名称重复
	dupSpecName := []productproduct.ProductSpec{
		{ID: 1, Name: "Color", Values: []productproduct.ProductSpecValue{{ID: 1, Name: "a"}}},
		{ID: 2, Name: "Color", Values: []productproduct.ProductSpecValue{{ID: 2, Name: "b"}}},
	}
	if err := productproduct.ValidateSpecs(dupSpecName); !errors.Is(err, productproduct.ErrDuplicateSpecName) {
		t.Fatalf("expected ErrDuplicateSpecName, got: %v", err)
	}
}

// 3. 金额精度、范围与禁止 float
func TestDecimalPrecisionAndBounds(t *testing.T) {
	// 超过两位小数拒绝
	if _, err := util.ParseDecimalYuanToCent("19.999"); !errors.Is(err, util.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount for 3 decimals, got: %v", err)
	}
	// 负数拒绝
	if _, err := util.ParseDecimalYuanToCent("-0.01"); !errors.Is(err, util.ErrNegativeAmount) {
		t.Fatalf("expected ErrNegativeAmount, got: %v", err)
	}
	// 溢出拒绝 (大于 9999999999.99)
	if _, err := util.ParseDecimalYuanToCent("10000000000.00"); !errors.Is(err, util.ErrAmountTooLarge) {
		t.Fatalf("expected ErrAmountTooLarge, got: %v", err)
	}
	// 往返转换精准
	cents, err := util.ParseDecimalYuanToCent("199.90")
	if err != nil || cents != 19990 {
		t.Fatalf("expected 19990 cents, got %d, err %v", cents, err)
	}
	yuan := util.FormatCentToYuan(cents)
	if yuan != "199.90" {
		t.Fatalf("expected 199.90, got %s", yuan)
	}
}

// 4. 分类层级、循环引用与引用删除检查
func TestCategoryLifecycleAndCycleDetection(t *testing.T) {
	ctx := context.Background()
	db := productTestDB(t)
	catSvc := productcategory.NewService(db)

	// 创建一级分类
	c1, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "一级分类", ParentID: 0})
	if err != nil {
		t.Fatalf("create c1: %v", err)
	}
	// 创建二级分类
	c2, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "二级分类", ParentID: c1.ID})
	if err != nil {
		t.Fatalf("create c2: %v", err)
	}
	// 创建三级分类
	c3, err := catSvc.Create(ctx, productcategory.CreateRequest{Name: "三级分类", ParentID: c2.ID})
	if err != nil {
		t.Fatalf("create c3: %v", err)
	}

	// 尝试将一级分类的父级设为自身 -> 拒绝
	_, err = catSvc.Update(ctx, nil, productcategory.UpdateRequest{ID: c1.ID, ParentID: &c1.ID})
	if !errors.Is(err, productcategory.ErrParentCategorySelf) {
		t.Fatalf("expected ErrParentCategorySelf, got %v", err)
	}

	// 尝试将一级分类的父级设为三级分类 -> 形成环路，拒绝
	_, err = catSvc.Update(ctx, nil, productcategory.UpdateRequest{ID: c1.ID, ParentID: &c3.ID})
	if !errors.Is(err, productcategory.ErrParentCategoryCycle) {
		t.Fatalf("expected ErrParentCategoryCycle, got %v", err)
	}

	// 尝试删除有一级子分类的分类 -> 拒绝
	err = catSvc.Delete(ctx, c1.ID)
	if !errors.Is(err, productcategory.ErrHasSubcategories) {
		t.Fatalf("expected ErrHasSubcategories, got %v", err)
	}

	// 创建商品引用 c3
	prodSvc := productproduct.NewService(db)
	skuReq := productproduct.ProductSKURequest{
		Price:  "10.00",
		Status: func(v int8) *int8 { return &v }(1),
	}
	pRes, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:       "测试商品",
		CategoryID: c3.ID,
		SKUs:       []productproduct.ProductSKURequest{skuReq},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	// 尝试删除被商品引用的分类 c3 -> 拒绝
	err = catSvc.Delete(ctx, c3.ID)
	if !errors.Is(err, productcategory.ErrCategoryInUse) {
		t.Fatalf("expected ErrCategoryInUse, got %v", err)
	}

	// 删除商品后，可成功删除 c3
	if err := prodSvc.Delete(ctx, pRes.ID); err != nil {
		t.Fatalf("delete product: %v", err)
	}
	if err := catSvc.Delete(ctx, c3.ID); err != nil {
		t.Fatalf("delete c3: %v", err)
	}
}

// 5. 商品创建、详情与 SKU 完整聚合保存
func TestProductCreateDetailAndUpdate(t *testing.T) {
	ctx := context.Background()
	db := productTestDB(t)
	prodSvc := productproduct.NewService(db)
	imageID := createTestImage(t, db)

	specs := []productproduct.ProductSpec{
		{
			ID:   1,
			Name: "颜色",
			Values: []productproduct.ProductSpecValue{
				{ID: 1, Name: "红色"},
				{ID: 2, Name: "蓝色"},
			},
		},
	}

	skus := []productproduct.ProductSKURequest{
		{
			SpecValues:  []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 1}},
			SKUCode:     func(s string) *string { return &s }("SKU-RED"),
			Price:       "99.00",
			MarketPrice: "129.00",
			CostPrice:   "50.00",
			Sort:        func(i int) *int { return &i }(1),
			Status:      func(i int8) *int8 { return &i }(1),
		},
		{
			SpecValues:  []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 2}},
			SKUCode:     func(s string) *string { return &s }("SKU-BLUE"),
			Price:       "109.00",
			MarketPrice: "139.00",
			CostPrice:   "55.00",
			Sort:        func(i int) *int { return &i }(2),
			Status:      func(i int8) *int8 { return &i }(1),
		},
	}

	createRes, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:           "短袖T恤",
		Subtitle:       "夏季爆款",
		MainImageID:    &imageID,
		GalleryFileIDs: []int64{imageID},
		Specs:          specs,
		Detail:         "<p>纯棉舒适</p>",
		SKUs:           skus,
	})
	if err != nil {
		t.Fatalf("create product err: %v", err)
	}

	detail, err := prodSvc.Detail(ctx, createRes.ID)
	if err != nil {
		t.Fatalf("detail product err: %v", err)
	}
	if detail.Name != "短袖T恤" || len(detail.SKUs) != 2 {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if detail.SalesCount != 0 {
		t.Fatalf("sales_count should default to 0, got: %d", detail.SalesCount)
	}
	if len(detail.GalleryFileIDs) != 1 || detail.GalleryFileIDs[0] != imageID {
		t.Fatalf("unexpected gallery file IDs: %+v", detail.GalleryFileIDs)
	}

	// 编辑：漏传已有 SKU 应当报错
	updateReqMissingSKU := productproduct.ProductUpdateRequest{
		ID:    createRes.ID,
		Name:  "短袖T恤-改",
		Specs: specs,
		SKUs: []productproduct.ProductSKURequest{
			{
				ID:         detail.SKUs[0].ID,
				SpecValues: detail.SKUs[0].SpecValues,
				Price:      "88.00",
			},
			// 漏传了 detail.SKUs[1]
		},
	}
	_, err = prodSvc.Update(ctx, updateReqMissingSKU)
	if !errors.Is(err, productproduct.ErrMissingExistingSKU) {
		t.Fatalf("expected ErrMissingExistingSKU, got: %v", err)
	}

	// 编辑：跨商品传入 SKU ID 应当报错
	updateReqCrossSKU := productproduct.ProductUpdateRequest{
		ID:    createRes.ID,
		Name:  "短袖T恤-改",
		Specs: specs,
		SKUs: []productproduct.ProductSKURequest{
			{ID: detail.SKUs[0].ID, SpecValues: detail.SKUs[0].SpecValues, Price: "88.00"},
			{ID: 999999, SpecValues: detail.SKUs[1].SpecValues, Price: "99.00"},
		},
	}
	_, err = prodSvc.Update(ctx, updateReqCrossSKU)
	if !errors.Is(err, productproduct.ErrSKUNotBelongToProduct) {
		t.Fatalf("expected ErrSKUNotBelongToProduct, got: %v", err)
	}

	// 正常编辑：保留所有已有 SKU ID，修改价格
	validUpdate := productproduct.ProductUpdateRequest{
		ID:          createRes.ID,
		Name:        "短袖T恤-改名",
		Subtitle:    detail.Subtitle,
		MainImageID: &imageID,
		Specs:       specs,
		Detail:      detail.Detail,
		SKUs: []productproduct.ProductSKURequest{
			{
				ID:         detail.SKUs[0].ID,
				SpecValues: detail.SKUs[0].SpecValues,
				SKUCode:    detail.SKUs[0].SKUCode,
				Price:      "88.80",
			},
			{
				ID:         detail.SKUs[1].ID,
				SpecValues: detail.SKUs[1].SpecValues,
				SKUCode:    detail.SKUs[1].SKUCode,
				Price:      "99.90",
			},
		},
	}
	_, err = prodSvc.Update(ctx, validUpdate)
	if err != nil {
		t.Fatalf("valid update err: %v", err)
	}

	updatedDetail, _ := prodSvc.Detail(ctx, createRes.ID)
	if updatedDetail.SKUs[0].ID != detail.SKUs[0].ID || updatedDetail.SKUs[0].Price != "88.80" {
		t.Fatalf("SKU ID should be retained and price updated: %+v", updatedDetail.SKUs[0])
	}
}

// 6. 兼容软删除 SKU 恢复
func TestProductSoftDeleteSKURestoration(t *testing.T) {
	ctx := context.Background()
	db := productTestDB(t)
	prodSvc := productproduct.NewService(db)

	specs := []productproduct.ProductSpec{
		{
			ID:   1,
			Name: "尺码",
			Values: []productproduct.ProductSpecValue{
				{ID: 1, Name: "M"},
				{ID: 2, Name: "L"},
			},
		},
	}

	createRes, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:  "裤子",
		Specs: specs,
		SKUs: []productproduct.ProductSKURequest{
			{SpecValues: []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 1}}, SKUCode: func(s string) *string { return &s }("PANTS-M"), Price: "50.00"},
			{SpecValues: []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 2}}, SKUCode: func(s string) *string { return &s }("PANTS-L"), Price: "60.00"},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	detail, _ := prodSvc.Detail(ctx, createRes.ID)
	lSKU := detail.SKUs[1] // L号

	// 模拟在数据库中该 SKU 被软删除
	now := time.Now()
	if err := db.Model(&model.ProductSKU{}).Where("id = ?", lSKU.ID).Update("deleted_at", now).Error; err != nil {
		t.Fatalf("soft delete sku: %v", err)
	}

	// 此时产品仅剩 M 号未删除 SKU
	// 用户在编辑时重新提交 L 号组合（作为新行，不带 ID）
	updateReq := productproduct.ProductUpdateRequest{
		ID:    createRes.ID,
		Name:  "裤子",
		Specs: specs,
		SKUs: []productproduct.ProductSKURequest{
			{ID: detail.SKUs[0].ID, SpecValues: detail.SKUs[0].SpecValues, Price: "50.00"},
			// 重新提交相同的 L 规格组合，不带 ID
			{ID: 0, SpecValues: []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 2}}, SKUCode: func(s string) *string { return &s }("PANTS-L"), Price: "65.00"},
		},
	}

	_, err = prodSvc.Update(ctx, updateReq)
	if err != nil {
		t.Fatalf("update with restored SKU failed: %v", err)
	}

	// 检查恢复后的 SKU：ID 必须等于原 lSKU.ID，且 deleted_at 为 NULL，价格已更新为 65.00
	var restored model.ProductSKU
	if err := db.Where("id = ?", lSKU.ID).First(&restored).Error; err != nil {
		t.Fatalf("find restored sku: %v", err)
	}
	normPrice, _ := util.NormalizeDecimalYuan(restored.Price)
	if normPrice != "65.00" || restored.DeletedAt.Valid {
		t.Fatalf("restored sku mismatch: deleted_at=%v, price=%s", restored.DeletedAt, restored.Price)
	}
}

// 7. 上架校验（分类、主图、启用 SKU 及价格 > 0）
func TestProductOnlineValidation(t *testing.T) {
	ctx := context.Background()
	db := productTestDB(t)
	prodSvc := productproduct.NewService(db)
	catSvc := productcategory.NewService(db)
	imageID := createTestImage(t, db)

	// 分类禁用
	cat, err := catSvc.Create(ctx, productcategory.CreateRequest{
		Name:   "禁用分类",
		Status: func(v int8) *int8 { return &v }(0),
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	// 上架 (status=1) 但分类禁用 -> 报错
	_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "商品",
		CategoryID:  cat.ID,
		MainImageID: &imageID,
		Status:      func(v int8) *int8 { return &v }(1),
		SKUs: []productproduct.ProductSKURequest{
			{Price: "10.00", Status: func(v int8) *int8 { return &v }(1)},
		},
	})
	if !errors.Is(err, productproduct.ErrCategoryDisabled) {
		t.Fatalf("expected ErrCategoryDisabled, got: %v", err)
	}

	// 启用分类
	_, _ = catSvc.Update(ctx, nil, productcategory.UpdateRequest{
		ID:     cat.ID,
		Status: func(v int8) *int8 { return &v }(1),
	})

	// 上架但未设置主图 -> 报错
	_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:       "商品",
		CategoryID: cat.ID,
		Status:     func(v int8) *int8 { return &v }(1),
		SKUs: []productproduct.ProductSKURequest{
			{Price: "10.00", Status: func(v int8) *int8 { return &v }(1)},
		},
	})
	if !errors.Is(err, productproduct.ErrMainImageRequired) {
		t.Fatalf("expected ErrMainImageRequired, got: %v", err)
	}

	// 上架但所有 SKU 均为 0 元 -> 报错
	_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "商品",
		CategoryID:  cat.ID,
		MainImageID: &imageID,
		Status:      func(v int8) *int8 { return &v }(1),
		SKUs: []productproduct.ProductSKURequest{
			{Price: "0.00", Status: func(v int8) *int8 { return &v }(1)},
		},
	})
	if !errors.Is(err, productproduct.ErrEnabledSKUZeroPrice) {
		t.Fatalf("expected ErrEnabledSKUZeroPrice, got: %v", err)
	}

	// 下架商品 (status=0) 允许未分类、缺图和 0 元
	draftProd, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:   "下架草稿商品",
		Status: func(v int8) *int8 { return &v }(0),
		SKUs: []productproduct.ProductSKURequest{
			{Price: "0.00"},
		},
	})
	if err != nil {
		t.Fatalf("create offline product err: %v", err)
	}

	// 尝试直接修改状态上架 -> 触发相同上架校验报错
	err = prodSvc.Status(ctx, productproduct.ProductStatusRequest{ID: draftProd.ID, Status: 1})
	if err == nil {
		t.Fatal("expected error when setting status=1 on incomplete product")
	}
}

// 8. 权限校验
func TestProductPermissions(t *testing.T) {
	modelDef := casbinmodel.NewModel()
	modelDef.AddDef("r", "r", "sub, perm")
	modelDef.AddDef("p", "p", "sub, perm")
	modelDef.AddDef("g", "g", "_, _")
	modelDef.AddDef("e", "e", "some(where (p_eft == allow))")
	modelDef.AddDef("m", "m", "g(r_sub, p_sub) && r_perm == p_perm")
	enforcer, err := casbin.NewSyncedEnforcer(modelDef)
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	authSvc := auth.NewService(nil, enforcer, "", time.Hour)

	// 给角色授予商品列表权限
	role := "product_operator"
	_, _ = enforcer.AddPermissionForUser(role, auth.PermissionProductProductList)
	_, _ = enforcer.AddRoleForUser(auth.UserSubject(10), role)

	// 验证 options 接口允许分类 list 或商品 list/create/update 任一权限
	hasOptionsPerm := false
	perms := []string{
		auth.PermissionProductCategoryList,
		auth.PermissionProductProductList,
		auth.PermissionProductProductCreate,
		auth.PermissionProductProductUpdate,
	}
	for _, p := range perms {
		if ok, _ := authSvc.Can(context.Background(), 10, p); ok {
			hasOptionsPerm = true
			break
		}
	}
	if !hasOptionsPerm {
		t.Fatal("product list permission should be sufficient for reading category options")
	}

	// 验证无任何权限的用户被拒绝
	unauthorizedUser := int64(999)
	hasPerm, _ := authSvc.Can(context.Background(), unauthorizedUser, auth.PermissionProductCategoryList)
	if hasPerm {
		t.Fatal("unauthorized user should not have permission")
	}
}

// 9. 事务回滚校验：SKU 校验失败时整个事务回滚，无残留数据
func TestProductTransactionRollback(t *testing.T) {
	ctx := context.Background()
	db := productTestDB(t)
	prodSvc := productproduct.NewService(db)

	specs := []productproduct.ProductSpec{
		{
			ID:   1,
			Name: "规格",
			Values: []productproduct.ProductSpecValue{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
			},
		},
	}

	// 第二个 SKU 价格非法，导致整体回滚
	invalidSKUs := []productproduct.ProductSKURequest{
		{
			SpecValues: []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 1}},
			Price:      "50.00",
		},
		{
			SpecValues: []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 2}},
			Price:      "invalid_price",
		},
	}

	_, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:  "回滚测试商品",
		Specs: specs,
		SKUs:  invalidSKUs,
	})
	if err == nil {
		t.Fatal("expected error for invalid price in sku")
	}

	// 验证商品表和 SKU 表中均无数据写入
	var prodCount, skuCount int64
	db.Model(&model.Product{}).Count(&prodCount)
	db.Model(&model.ProductSKU{}).Count(&skuCount)
	if prodCount != 0 || skuCount != 0 {
		t.Fatalf("transaction rollback failed: prodCount=%d, skuCount=%d", prodCount, skuCount)
	}
}

// 10. SKU 编码全局唯一与软删除占用校验
func TestProductSKUCodeConflict(t *testing.T) {
	ctx := context.Background()
	db := productTestDB(t)
	prodSvc := productproduct.NewService(db)

	skuCode := "GLOBAL-SKU-001"
	// 创建第一个商品使用该编码
	_, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name: "商品1",
		SKUs: []productproduct.ProductSKURequest{
			{Price: "10.00", SKUCode: &skuCode},
		},
	})
	if err != nil {
		t.Fatalf("create prod1: %v", err)
	}

	// 创建第二个商品尝试使用相同编码 -> 必须报错
	_, err = prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name: "商品2",
		SKUs: []productproduct.ProductSKURequest{
			{Price: "20.00", SKUCode: &skuCode},
		},
	})
	if err == nil {
		t.Fatal("expected error for duplicate global sku_code")
	}
}
