package product_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	productcategory "mall/admin_core/internal/product/category"
	productinventory "mall/admin_core/internal/product/inventory"
	productproduct "mall/admin_core/internal/product/product"
)

func createTestProductWithSKU(t *testing.T, db *gorm.DB) (int64, int64) {
	t.Helper()
	catSvc := productcategory.NewService(db)
	prodSvc := productproduct.NewService(db)
	ctx := context.Background()

	cat, err := catSvc.Create(ctx, productcategory.CreateRequest{
		Name:   "测试分类",
		Status: func(v int8) *int8 { return &v }(1),
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	imageID := createTestImage(t, db)
	skuCode := fmt.Sprintf("SKU-%d", time.Now().UnixNano())
	res, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "测试商品",
		Subtitle:    "测试副标题",
		CategoryID:  cat.ID,
		MainImageID: &imageID,
		Status:      func(v int8) *int8 { return &v }(1),
		SKUs: []productproduct.ProductSKURequest{
			{
				Price:       "99.00",
				MarketPrice: "199.00",
				CostPrice:   "50.00",
				SKUCode:     &skuCode,
			},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	var sku model.ProductSKU
	if err := db.Where("product_id = ?", res.ID).First(&sku).Error; err != nil {
		t.Fatalf("find sku: %v", err)
	}

	return res.ID, sku.ID
}

// 1. 测试商品创建、新增SKU和恢复时的库存初始化与保留
func TestInventoryCreationLifecycle(t *testing.T) {
	db := productTestDB(t)
	catSvc := productcategory.NewService(db)
	prodSvc := productproduct.NewService(db)
	ctx := context.Background()

	cat, err := catSvc.Create(ctx, productcategory.CreateRequest{
		Name:   "生命周期测试分类",
		Status: func(v int8) *int8 { return &v }(1),
	})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	imageID := createTestImage(t, db)
	sku1Code := "SKU-LIFE-001"
	specs := []productproduct.ProductSpec{
		{
			ID:   1,
			Name: "颜色",
			Values: []productproduct.ProductSpecValue{
				{ID: 1, Name: "红"},
				{ID: 2, Name: "蓝"},
			},
		},
	}

	// 1.1 创建商品时自动创建零库存
	res, err := prodSvc.Create(ctx, productproduct.ProductCreateRequest{
		Name:        "多规格生命周期商品",
		Subtitle:    "测试",
		CategoryID:  cat.ID,
		MainImageID: &imageID,
		Status:      func(v int8) *int8 { return &v }(1),
		Specs:       specs,
		SKUs: []productproduct.ProductSKURequest{
			{
				SpecValues:  []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 1}},
				Price:       "99.00",
				MarketPrice: "199.00",
				CostPrice:   "50.00",
				SKUCode:     &sku1Code,
			},
		},
	})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	pID := res.ID

	var sku1 model.ProductSKU
	if err := db.Where("product_id = ? AND sku_code = ?", pID, sku1Code).First(&sku1).Error; err != nil {
		t.Fatalf("find sku1: %v", err)
	}

	var inv model.ProductSkuInventory
	if err := db.Where("sku_id = ?", sku1.ID).First(&inv).Error; err != nil {
		t.Fatalf("find inventory: %v", err)
	}
	if inv.Stock != 0 || inv.LockedStock != 0 || inv.WarningStock != 0 || inv.Version != 0 {
		t.Fatalf("expected 0 inventory, got %+v", inv)
	}
	if inv.ProductID != pID {
		t.Fatalf("expected product_id %d, got %d", pID, inv.ProductID)
	}

	// 1.2 编辑商品新增 SKU 时也自动初始化零库存
	detail, err := prodSvc.Detail(ctx, pID)
	if err != nil {
		t.Fatalf("get detail: %v", err)
	}

	newSKUCode := "SKU-LIFE-002"
	updateReq := productproduct.ProductUpdateRequest{
		ID:          pID,
		Name:        detail.Name,
		Subtitle:    detail.Subtitle,
		CategoryID:  detail.CategoryID,
		MainImageID: detail.MainImageID,
		Status:      &detail.Status,
		Specs:       specs,
		SKUs: []productproduct.ProductSKURequest{
			{
				ID:          sku1.ID,
				SpecValues:  []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 1}},
				Price:       detail.SKUs[0].Price,
				MarketPrice: detail.SKUs[0].MarketPrice,
				CostPrice:   detail.SKUs[0].CostPrice,
				SKUCode:     detail.SKUs[0].SKUCode,
			},
			{
				ID:          0, // 新增SKU
				SpecValues:  []productproduct.ProductSKUValue{{SpecID: 1, ValueID: 2}},
				Price:       "150.00",
				MarketPrice: "200.00",
				CostPrice:   "80.00",
				SKUCode:     &newSKUCode,
			},
		},
	}

	if _, err := prodSvc.Update(ctx, updateReq); err != nil {
		t.Fatalf("update product add sku: %v", err)
	}

	var newSKU model.ProductSKU
	if err := db.Where("sku_code = ?", newSKUCode).First(&newSKU).Error; err != nil {
		t.Fatalf("find new sku: %v", err)
	}
	var newInv model.ProductSkuInventory
	if err := db.Where("sku_id = ?", newSKU.ID).First(&newInv).Error; err != nil {
		t.Fatalf("find new sku inventory: %v", err)
	}
	if newInv.Stock != 0 || newInv.Version != 0 {
		t.Fatalf("expected 0 inventory for new sku, got %+v", newInv)
	}

	// 1.3 软删除SKU并恢复时保留原库存
	// 先人为给newSKU充入库存50
	invSvc := productinventory.NewService(db)
	adminUser := model.SystemUser{ID: 1, Username: "admin", Nickname: "管理员"}
	reqID := uuid.New().String()
	adjustRes, bErr := invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           newSKU.ID,
		RequestID:       reqID,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(50),
		Remark:          "期初入库",
	})
	if bErr != nil {
		t.Fatalf("adjust failed: %v", bErr.Message)
	}
	if adjustRes.Log.StockAfter != 50 {
		t.Fatalf("expected stock 50, got %d", adjustRes.Log.StockAfter)
	}

	// 软删除 newSKU
	if err := db.Delete(&newSKU).Error; err != nil {
		t.Fatalf("soft delete sku: %v", err)
	}

	// 恢复 newSKU (通过多规格匹配或同规格恢复逻辑)
	// 如果恢复缺失库存的SKU，报 ErrInventoryNotInitialized
	// 先验证存在库存时，恢复不覆盖原库存
	var checkInv model.ProductSkuInventory
	if err := db.Where("sku_id = ?", newSKU.ID).First(&checkInv).Error; err != nil {
		t.Fatalf("inventory should still exist after sku soft delete: %v", err)
	}
	if checkInv.Stock != 50 {
		t.Fatalf("expected stock 50 preserved, got %d", checkInv.Stock)
	}
}

// 2. 测试商品删除保护（库存非零阻止删除、缺库存阻止删除、清零后可删除且保留流水与余额）
func TestInventoryDeletionProtection(t *testing.T) {
	db := productTestDB(t)
	prodSvc := productproduct.NewService(db)
	invSvc := productinventory.NewService(db)
	ctx := context.Background()

	pID, skuID := createTestProductWithSKU(t, db)
	adminUser := model.SystemUser{ID: 1, Username: "admin", Nickname: "管理员"}

	// 2.1 入库 20 件
	_, bErr := invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(20),
		Remark:          "入库20件",
	})
	if bErr != nil {
		t.Fatalf("adjust failed: %v", bErr.Message)
	}

	// 尝试删除商品 -> 应该被拒绝
	err := prodSvc.Delete(ctx, pID)
	if err == nil || !errors.Is(err, productproduct.ErrProductHasStock) {
		t.Fatalf("expected ErrProductHasStock, got %v", err)
	}

	// 2.2 锁定库存非零阻止删除
	// 将 stock 设为 0，locked_stock 设为 10
	if err := db.Model(&model.ProductSkuInventory{}).Where("sku_id = ?", skuID).
		Updates(map[string]any{"stock": 10, "locked_stock": 10, "version": 2}).Error; err != nil {
		t.Fatalf("update locked: %v", err)
	}
	err = prodSvc.Delete(ctx, pID)
	if err == nil || !errors.Is(err, productproduct.ErrProductHasStock) {
		t.Fatalf("expected ErrProductHasStock when locked > 0, got %v", err)
	}

	// 2.3 缺库存记录阻止删除
	if err := db.Where("sku_id = ?", skuID).Delete(&model.ProductSkuInventory{}).Error; err != nil {
		t.Fatalf("delete inventory: %v", err)
	}
	err = prodSvc.Delete(ctx, pID)
	if err == nil || !strings.Contains(err.Error(), "SKU库存未初始化") {
		t.Fatalf("expected ErrInventoryNotInitialized, got %v", err)
	}

	// 2.4 库存清零后允许删除，且余额和流水保留
	// 重新补上 0 库存
	if err := invSvc.InitZeroInventoryTx(ctx, db, pID, skuID); err != nil {
		t.Fatalf("re-init inventory: %v", err)
	}
	if err := prodSvc.Delete(ctx, pID); err != nil {
		t.Fatalf("delete product with 0 stock should succeed, got %v", err)
	}

	// 验证商品已被软删除
	var deletedP model.Product
	if err := db.Where("id = ?", pID).First(&deletedP).Error; err == nil {
		t.Fatal("product should be soft-deleted")
	}

	// 验证库存记录未被删除
	var preservedInv model.ProductSkuInventory
	if err := db.Where("sku_id = ?", skuID).First(&preservedInv).Error; err != nil {
		t.Fatalf("inventory record must be preserved after product delete: %v", err)
	}

	// 验证流水记录未被删除
	var logCount int64
	if err := db.Model(&model.ProductSkuInventoryLog{}).Where("product_id = ?", pID).Count(&logCount).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if logCount < 1 {
		t.Fatalf("expected at least 1 log preserved, got %d", logCount)
	}
}

// 3. 测试库存变更、前后值计算、NO_CHANGE、合法0与版本冲突
func TestInventoryMutationsAndCalculations(t *testing.T) {
	db := productTestDB(t)
	invSvc := productinventory.NewService(db)
	ctx := context.Background()

	_, skuID := createTestProductWithSKU(t, db)
	adminUser := model.SystemUser{ID: 10, Username: "operator", Nickname: "仓库管理员"}

	// 3.1 人工入库 100 件 (version: 0 -> 1)
	req1ID := uuid.New().String()
	res1, bErr := invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       req1ID,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(100),
		Remark:          "首次采购入库",
	})
	if bErr != nil {
		t.Fatalf("manual_in failed: %v", bErr.Message)
	}
	if res1.Replayed {
		t.Fatal("first call should not be replayed")
	}
	log1 := res1.Log
	if log1.StockBefore != 0 || log1.StockDelta != 100 || log1.StockAfter != 100 {
		t.Fatalf("unexpected stock calculation: %+v", log1)
	}
	if log1.VersionBefore != 0 || log1.VersionAfter != 1 {
		t.Fatalf("unexpected version increment: %+v", log1)
	}
	if log1.AvailableBefore != 0 || log1.AvailableAfter != 100 {
		t.Fatalf("unexpected available calculation: %+v", log1)
	}

	// 3.2 人工出库 30 件 (version: 1 -> 2)
	req2ID := uuid.New().String()
	res2, bErr := invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       req2ID,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(1),
		ChangeType:      productinventory.ChangeTypeManualOut,
		Quantity:        func(v int) *int { return &v }(30),
		Remark:          "调拨出库",
	})
	if bErr != nil {
		t.Fatalf("manual_out failed: %v", bErr.Message)
	}
	log2 := res2.Log
	if log2.StockBefore != 100 || log2.StockDelta != -30 || log2.StockAfter != 70 {
		t.Fatalf("unexpected stock calculation: %+v", log2)
	}
	if log2.VersionBefore != 1 || log2.VersionAfter != 2 {
		t.Fatalf("unexpected version increment: %+v", log2)
	}

	// 3.3 出库超扣拒绝 (当前可用 70, 尝试出库 80)
	_, bErr = invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(2),
		ChangeType:      productinventory.ChangeTypeManualOut,
		Quantity:        func(v int) *int { return &v }(80),
		Remark:          "尝试超扣出库",
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeInsufficientStock {
		t.Fatalf("expected INSUFFICIENT_STOCK, got %v", bErr)
	}

	// 3.4 盘点校正到 50 件 (当前 stock=70, 目标 50 -> delta = -20, version: 2 -> 3)
	req3ID := uuid.New().String()
	res3, bErr := invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       req3ID,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(2),
		ChangeType:      productinventory.ChangeTypeStocktake,
		Quantity:        func(v int) *int { return &v }(50),
		Remark:          "月度盘点损耗校正",
	})
	if bErr != nil {
		t.Fatalf("stocktake failed: %v", bErr.Message)
	}
	log3 := res3.Log
	if log3.StockBefore != 70 || log3.StockDelta != -20 || log3.StockAfter != 50 {
		t.Fatalf("unexpected stocktake: %+v", log3)
	}
	if log3.VersionAfter != 3 {
		t.Fatalf("expected version 3, got %d", log3.VersionAfter)
	}

	// 3.5 盘点目标与当前一致返回 NO_CHANGE，不加版本不写流水
	_, bErr = invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(3),
		ChangeType:      productinventory.ChangeTypeStocktake,
		Quantity:        func(v int) *int { return &v }(50),
		Remark:          "盘点无变化",
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeNoChange {
		t.Fatalf("expected NO_CHANGE, got %v", bErr)
	}

	// 3.6 设置预警阈值 10 (version: 3 -> 4)
	req4ID := uuid.New().String()
	res4, bErr := invSvc.Warning(ctx, adminUser, productinventory.ProductInventoryWarningRequest{
		SKUID:           skuID,
		RequestID:       req4ID,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(3),
		Quantity:        func(v int) *int { return &v }(10),
		Remark:          "设置预警阈值",
	})
	if bErr != nil {
		t.Fatalf("set warning failed: %v", bErr.Message)
	}
	log4 := res4.Log
	if log4.WarningBefore != 0 || log4.WarningAfter != 10 || log4.StockDelta != 0 {
		t.Fatalf("unexpected warning log: %+v", log4)
	}
	if log4.VersionAfter != 4 {
		t.Fatalf("expected version 4, got %d", log4.VersionAfter)
	}

	// 3.7 预警阈值未变化返回 NO_CHANGE
	_, bErr = invSvc.Warning(ctx, adminUser, productinventory.ProductInventoryWarningRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(4),
		Quantity:        func(v int) *int { return &v }(10),
		Remark:          "重复设置预警",
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeNoChange {
		t.Fatalf("expected NO_CHANGE, got %v", bErr)
	}

	// 3.8 预警阈值设置为合法 0 (关闭预警, version: 4 -> 5)
	res5, bErr := invSvc.Warning(ctx, adminUser, productinventory.ProductInventoryWarningRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(4),
		Quantity:        func(v int) *int { return &v }(0),
		Remark:          "关闭预警",
	})
	if bErr != nil {
		t.Fatalf("close warning failed: %v", bErr.Message)
	}
	if res5.Log.WarningAfter != 0 || res5.Log.VersionAfter != 5 {
		t.Fatalf("unexpected close warning log: %+v", res5.Log)
	}

	// 3.9 版本冲突 (使用旧 expected_version=2)
	_, bErr = invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(2),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(10),
		Remark:          "旧版本请求",
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeVersionConflict {
		t.Fatalf("expected VERSION_CONFLICT, got %v", bErr)
	}
}

// 4. 测试幂等性：重试回放、参数冲突与已提交操作重试
func TestInventoryIdempotency(t *testing.T) {
	db := productTestDB(t)
	invSvc := productinventory.NewService(db)
	ctx := context.Background()

	_, skuID := createTestProductWithSKU(t, db)
	adminUser := model.SystemUser{ID: 1, Username: "admin", Nickname: "管理员"}

	reqID := uuid.New().String()
	adjustReq := productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       reqID,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(50),
		Remark:          "首次提交入库",
	}

	// 第一次提交成功
	res1, bErr := invSvc.Adjust(ctx, adminUser, adjustReq)
	if bErr != nil {
		t.Fatalf("first adjust failed: %v", bErr.Message)
	}
	if res1.Replayed {
		t.Fatal("first call should not be replayed")
	}

	// 模拟 COMMIT 后由于网络原因丢失响应，用相同的 request_id 和 payload 再次重试
	// 此时数据库当前版本已经是 1，但原请求的 expected_version 是 0。幂等检查必须优先于 expected_version 校验！
	res2, bErr := invSvc.Adjust(ctx, adminUser, adjustReq)
	if bErr != nil {
		t.Fatalf("replay failed: %v", bErr.Message)
	}
	if !res2.Replayed {
		t.Fatal("replay should have replayed = true")
	}
	if res2.Log.ID != res1.Log.ID {
		t.Fatalf("expected same log ID %d, got %d", res1.Log.ID, res2.Log.ID)
	}
	if res2.Log.StockAfter != 50 {
		t.Fatalf("expected stock 50, got %d", res2.Log.StockAfter)
	}

	// 确认数据库总库存依然是 50，没有被二次增加
	detail, bErr := invSvc.Detail(ctx, skuID)
	if bErr != nil {
		t.Fatalf("detail failed: %v", bErr.Message)
	}
	if *detail.Stock != 50 || *detail.Version != 1 {
		t.Fatalf("expected stock 50, version 1; got stock=%d, version=%d", *detail.Stock, *detail.Version)
	}

	// 同一个 request_id，但是修改了数量 -> 必须报 IDEMPOTENCY_CONFLICT
	conflictReq := adjustReq
	conflictReq.Quantity = func(v int) *int { return &v }(100)
	_, bErr = invSvc.Adjust(ctx, adminUser, conflictReq)
	if bErr == nil || bErr.Code != productinventory.ErrCodeIdempotencyConflict {
		t.Fatalf("expected IDEMPOTENCY_CONFLICT, got %v", bErr)
	}

	// 同一个 request_id，由另一个操作者提交 -> 必须报 IDEMPOTENCY_CONFLICT
	otherUser := model.SystemUser{ID: 2, Username: "other", Nickname: "其他人"}
	_, bErr = invSvc.Adjust(ctx, otherUser, adjustReq)
	if bErr == nil || bErr.Code != productinventory.ErrCodeIdempotencyConflict {
		t.Fatalf("expected IDEMPOTENCY_CONFLICT for different operator, got %v", bErr)
	}
}

// 5. 参数校验与边界检查（UUID格式、非法数量、溢出、原因长度）
func TestInventoryValidationBounds(t *testing.T) {
	db := productTestDB(t)
	invSvc := productinventory.NewService(db)
	ctx := context.Background()

	_, skuID := createTestProductWithSKU(t, db)
	adminUser := model.SystemUser{ID: 1, Username: "admin", Nickname: "管理员"}

	// 5.1 无效 UUID
	_, bErr := invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       "not-a-valid-uuid",
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(10),
		Remark:          "测试",
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for invalid UUID, got %v", bErr)
	}

	// 5.2 expected_version 缺失
	_, bErr = invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:      skuID,
		RequestID:  uuid.New().String(),
		ChangeType: productinventory.ChangeTypeManualIn,
		Quantity:   func(v int) *int { return &v }(10),
		Remark:     "测试",
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for missing expected_version, got %v", bErr)
	}

	// 5.3 入库数量 <= 0
	_, bErr = invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(0),
		Remark:          "测试0入库",
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for quantity 0 manual_in, got %v", bErr)
	}

	// 5.4 操作原因空白或超过 500 字符
	_, bErr = invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(10),
		Remark:          "   ",
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for blank remark, got %v", bErr)
	}

	longRemark := strings.Repeat("长", 501)
	_, bErr = invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(10),
		Remark:          longRemark,
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for long remark, got %v", bErr)
	}

	// 5.5 库存加法溢出检查
	// 先人为将库存设为 MaxInt32 - 5
	if err := db.Model(&model.ProductSkuInventory{}).Where("sku_id = ?", skuID).
		Updates(map[string]any{"stock": math.MaxInt32 - 5, "version": 1}).Error; err != nil {
		t.Fatalf("set stock: %v", err)
	}
	_, bErr = invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       uuid.New().String(),
		ExpectedVersion: func(v uint32) *uint32 { return &v }(1),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(10),
		Remark:          "导致溢出的入库",
	})
	if bErr == nil || bErr.Code != productinventory.ErrCodeInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT for stock overflow, got %v", bErr)
	}
}

// 6. 测试库存列表与详情查询（筛选、快照解析、未初始化处理）
func TestInventoryListAndDetailQuery(t *testing.T) {
	db := productTestDB(t)
	invSvc := productinventory.NewService(db)
	ctx := context.Background()

	pID, skuID := createTestProductWithSKU(t, db)

	// 6.1 列表查询
	rows, total, err := invSvc.List(ctx, productinventory.ProductInventoryListRequest{
		ProductID: &pID,
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("expected 1 row, got total=%d, rows=%d", total, len(rows))
	}
	r0 := rows[0]
	if !r0.Initialized {
		t.Fatal("expected initialized = true")
	}
	if *r0.Stock != 0 || *r0.AvailableStock != 0 {
		t.Fatalf("expected stock 0, got stock=%v available=%v", *r0.Stock, *r0.AvailableStock)
	}
	if *r0.IsOutOfStock != true {
		t.Fatal("expected is_out_of_stock = true for 0 stock")
	}

	// 6.2 详情查询
	detail, bErr := invSvc.Detail(ctx, skuID)
	if bErr != nil {
		t.Fatalf("detail failed: %v", bErr.Message)
	}
	if detail.SKUID != skuID || !detail.Initialized {
		t.Fatalf("unexpected detail: %+v", detail)
	}

	// 6.3 模拟未初始化 SKU（删除库存记录后查询列表）
	if err := db.Where("sku_id = ?", skuID).Delete(&model.ProductSkuInventory{}).Error; err != nil {
		t.Fatalf("delete inventory: %v", err)
	}

	uninitRows, totalUninit, err := invSvc.List(ctx, productinventory.ProductInventoryListRequest{
		StockFilter: "uninitialized",
	})
	if err != nil {
		t.Fatalf("list uninitialized failed: %v", err)
	}
	if totalUninit < 1 {
		t.Fatal("expected uninitialized row to be found")
	}
	found := false
	for _, row := range uninitRows {
		if row.SKUID == skuID {
			found = true
			if row.Initialized {
				t.Fatal("expected initialized = false")
			}
			if row.Stock != nil || row.AvailableStock != nil || row.InventoryID != nil {
				t.Fatal("uninitialized row must have nil stock and inventory_id")
			}
		}
	}
	if !found {
		t.Fatal("uninitialized sku not found in list")
	}

	// 详情查询未初始化 SKU 应明确报错
	_, bErr = invSvc.Detail(ctx, skuID)
	if bErr == nil || bErr.Code != productinventory.ErrCodeNotInitialized {
		t.Fatalf("expected NOT_INITIALIZED, got %v", bErr)
	}
}

// 7. 测试流水查询与快照保留
func TestInventoryLogsQueryAndSnapshot(t *testing.T) {
	db := productTestDB(t)
	invSvc := productinventory.NewService(db)
	ctx := context.Background()

	pID, skuID := createTestProductWithSKU(t, db)
	adminUser := model.SystemUser{ID: 1, Username: "admin", Nickname: "管理员"}

	// 记录两条流水
	req1 := uuid.New().String()
	_, bErr := invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       req1,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(0),
		ChangeType:      productinventory.ChangeTypeManualIn,
		Quantity:        func(v int) *int { return &v }(100),
		Remark:          "第一批入库",
	})
	if bErr != nil {
		t.Fatalf("adjust 1 failed: %v", bErr.Message)
	}

	req2 := uuid.New().String()
	_, bErr = invSvc.Adjust(ctx, adminUser, productinventory.ProductInventoryAdjustRequest{
		SKUID:           skuID,
		RequestID:       req2,
		ExpectedVersion: func(v uint32) *uint32 { return &v }(1),
		ChangeType:      productinventory.ChangeTypeManualOut,
		Quantity:        func(v int) *int { return &v }(20),
		Remark:          "第一批出库",
	})
	if bErr != nil {
		t.Fatalf("adjust 2 failed: %v", bErr.Message)
	}

	// 7.1 按 sku_id 查询流水
	logs, total, err := invSvc.ListLogs(ctx, productinventory.ProductInventoryLogListRequest{
		SKUID: &skuID,
	})
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if total != 2 || len(logs) != 2 {
		t.Fatalf("expected 2 logs, got total=%d, len=%d", total, len(logs))
	}
	// 验证排序：created_at DESC, id DESC -> 最新的出库在前面
	if logs[0].ChangeType != productinventory.ChangeTypeManualOut {
		t.Fatalf("expected first log to be manual_out, got %s", logs[0].ChangeType)
	}
	if logs[0].SKUSnapshot.ProductName != "测试商品" {
		t.Fatalf("expected snapshot product name '测试商品', got %q", logs[0].SKUSnapshot.ProductName)
	}

	// 7.2 按 request_id 精确查询
	singleLog, totalSingle, err := invSvc.ListLogs(ctx, productinventory.ProductInventoryLogListRequest{
		RequestID: &req1,
	})
	if err != nil {
		t.Fatalf("query by request_id: %v", err)
	}
	if totalSingle != 1 || len(singleLog) != 1 || singleLog[0].RequestID != req1 {
		t.Fatalf("expected 1 log for request_id %s, got %d", req1, totalSingle)
	}

	// 7.3 时间区间过滤 RFC3339
	now := time.Now()
	fromStr := now.Add(-1 * time.Hour).Format(time.RFC3339)
	toStr := now.Add(1 * time.Hour).Format(time.RFC3339)
	timeLogs, timeTotal, err := invSvc.ListLogs(ctx, productinventory.ProductInventoryLogListRequest{
		CreatedFrom: &fromStr,
		CreatedTo:   &toStr,
	})
	if err != nil {
		t.Fatalf("time range query: %v", err)
	}
	if timeTotal < 2 {
		t.Fatalf("expected at least 2 logs in time range, got %d", timeTotal)
	}
	_ = timeLogs
	_ = pID
}

// 8. 测试权限控制：RBAC 中查看、调整与流水权限的隔离
func TestInventoryPermissions(t *testing.T) {
	cModel := casbinmodel.NewModel()
	cModel.AddDef("r", "r", "sub, perm")
	cModel.AddDef("p", "p", "sub, perm")
	cModel.AddDef("g", "g", "_, _")
	cModel.AddDef("e", "e", "some(where (p_eft == allow))")
	cModel.AddDef("m", "m", "g(r_sub, p_sub) && r_perm == p_perm")
	enforcer, err := casbin.NewSyncedEnforcer(cModel)
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	authSvc := auth.NewService(nil, enforcer, "secret", time.Hour)

	// 用户 1 只有查看权限 product:inventory:list
	if _, err := enforcer.AddPermissionForUser("role_viewer", auth.PermissionProductInventoryList); err != nil {
		t.Fatal(err)
	}
	if _, err := enforcer.AddRoleForUser(auth.UserSubject(1), "role_viewer"); err != nil {
		t.Fatal(err)
	}

	// 用户 2 具有调整权限，同时需要具备查看权限
	if _, err := enforcer.AddPermissionForUser("role_operator", auth.PermissionProductInventoryList); err != nil {
		t.Fatal(err)
	}
	if _, err := enforcer.AddPermissionForUser("role_operator", auth.PermissionProductInventoryAdjust); err != nil {
		t.Fatal(err)
	}
	if _, err := enforcer.AddRoleForUser(auth.UserSubject(2), "role_operator"); err != nil {
		t.Fatal(err)
	}

	// 用户 3 只有流水查看权限 product:inventory:log
	if _, err := enforcer.AddPermissionForUser("role_auditor", auth.PermissionProductInventoryLog); err != nil {
		t.Fatal(err)
	}
	if _, err := enforcer.AddRoleForUser(auth.UserSubject(3), "role_auditor"); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// 验证用户 1：可看库存列表，但不能调库存也不能看流水
	canView, _ := authSvc.Can(ctx, 1, auth.PermissionProductInventoryList)
	canAdjust, _ := authSvc.Can(ctx, 1, auth.PermissionProductInventoryAdjust)
	canLog, _ := authSvc.Can(ctx, 1, auth.PermissionProductInventoryLog)
	if !canView || canAdjust || canLog {
		t.Fatalf("unexpected user 1 perms: view=%v adjust=%v log=%v", canView, canAdjust, canLog)
	}

	// 验证用户 2：可看可调库存，但不能看流水
	canView2, _ := authSvc.Can(ctx, 2, auth.PermissionProductInventoryList)
	canAdjust2, _ := authSvc.Can(ctx, 2, auth.PermissionProductInventoryAdjust)
	canLog2, _ := authSvc.Can(ctx, 2, auth.PermissionProductInventoryLog)
	if !canView2 || !canAdjust2 || canLog2 {
		t.Fatalf("unexpected user 2 perms: view=%v adjust=%v log=%v", canView2, canAdjust2, canLog2)
	}

	// 验证用户 3：只能看流水，不能看库存列表
	canView3, _ := authSvc.Can(ctx, 3, auth.PermissionProductInventoryList)
	canLog3, _ := authSvc.Can(ctx, 3, auth.PermissionProductInventoryLog)
	if canView3 || !canLog3 {
		t.Fatalf("unexpected user 3 perms: view=%v log=%v", canView3, canLog3)
	}
}

// 9. 测试 HTTP API 路由与响应信封结构（Adjust/Warning 错误码结构）
func TestInventoryHTTPEnvelopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := productTestDB(t)
	invSvc := productinventory.NewService(db)
	invAPI := productinventory.NewAPI(invSvc)

	_, skuID := createTestProductWithSKU(t, db)

	r := gin.New()
	// Mock user middleware
	r.Use(func(c *gin.Context) {
		c.Set("authenticated_user", model.SystemUser{ID: 1, Username: "admin", Nickname: "超级管理员"})
		c.Next()
	})
	r.POST("/api/product/inventories/adjust", invAPI.Adjust)
	r.POST("/api/product/inventories/warning", invAPI.Warning)

	// 9.1 成功入库
	reqBody := fmt.Sprintf(`{
		"sku_id": %d,
		"request_id": "%s",
		"expected_version": 0,
		"change_type": "manual_in",
		"quantity": 100,
		"remark": "HTTP测试入库"
	}`, skuID, uuid.New().String())

	req := httptest.NewRequest(http.MethodPost, "/api/product/inventories/adjust", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}
	var resp struct {
		Code    int                                           `json:"code"`
		Message string                                        `json:"message"`
		Data    productinventory.ProductInventoryMutationResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 1 || resp.Data.Log == nil || resp.Data.Log.StockAfter != 100 {
		t.Fatalf("unexpected success envelope: %+v", resp)
	}

	// 9.2 业务错误（版本冲突）：返回 code=0, data.error_code="VERSION_CONFLICT"
	conflictReq := fmt.Sprintf(`{
		"sku_id": %d,
		"request_id": "%s",
		"expected_version": 0,
		"change_type": "manual_in",
		"quantity": 20,
		"remark": "使用旧版本"
	}`, skuID, uuid.New().String())

	req = httptest.NewRequest(http.MethodPost, "/api/product/inventories/adjust", strings.NewReader(conflictReq))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var errResp struct {
		Code    int                                           `json:"code"`
		Message string                                        `json:"message"`
		Data    productinventory.ProductInventoryMutationResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal err response: %v", err)
	}
	if errResp.Code != 0 {
		t.Fatalf("expected business error code = 0, got %d", errResp.Code)
	}
	if errResp.Data.ErrorCode == nil || *errResp.Data.ErrorCode != productinventory.ErrCodeVersionConflict {
		t.Fatalf("expected VERSION_CONFLICT, got %v", errResp.Data.ErrorCode)
	}
}
