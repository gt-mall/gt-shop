package product_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	productdictservice "mall/admin_core/internal/product/dict_service"
	productdictunit "mall/admin_core/internal/product/dict_unit"
)

func dictTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open sqlite memory db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(
		&model.ProductDictService{},
		&model.ProductDictUnit{},
	); err != nil {
		t.Fatalf("failed to automigrate dict models: %v", err)
	}

	return db
}

func TestProductDictService_CRUD(t *testing.T) {
	db := dictTestDB(t)
	svc := productdictservice.NewService(db)
	ctx := context.Background()

	// 1. Create with default status and sort
	created1, err := svc.Create(ctx, productdictservice.DictServiceCreateRequest{
		Name:        "7天无理由退货",
		Description: "消费者在收到商品后7天内可无理由退货",
	})
	if err != nil {
		t.Fatalf("create dict service 1 failed: %v", err)
	}
	if created1.ID <= 0 {
		t.Fatalf("expected positive ID, got %d", created1.ID)
	}
	if created1.Name != "7天无理由退货" {
		t.Errorf("expected name '7天无理由退货', got %s", created1.Name)
	}
	if created1.Status != 1 {
		t.Errorf("expected default status 1, got %d", created1.Status)
	}
	if created1.Sort != 0 {
		t.Errorf("expected default sort 0, got %d", created1.Sort)
	}

	// 2. Create another with explicit sort and status
	statusZero := int8(0)
	created2, err := svc.Create(ctx, productdictservice.DictServiceCreateRequest{
		Name:        "极速退款",
		Description: "退货申请通过后快速打款",
		Sort:        10,
		Status:      &statusZero,
	})
	if err != nil {
		t.Fatalf("create dict service 2 failed: %v", err)
	}
	if created2.Status != 0 {
		t.Errorf("expected status 0, got %d", created2.Status)
	}

	// 3. List all
	list, total, err := svc.List(ctx, productdictservice.DictServiceListRequest{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("list dict service failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
	// Sort ascending: sort 0 comes before sort 10
	if list[0].ID != created1.ID || list[1].ID != created2.ID {
		t.Errorf("expected order: id %d then %d", created1.ID, created2.ID)
	}

	// 4. Keyword search
	listSearch, totalSearch, err := svc.List(ctx, productdictservice.DictServiceListRequest{
		Page:     1,
		PageSize: 20,
		Keyword:  "极速",
	})
	if err != nil {
		t.Fatalf("list with keyword failed: %v", err)
	}
	if totalSearch != 1 || len(listSearch) != 1 || listSearch[0].ID != created2.ID {
		t.Errorf("expected 1 result matching '极速', got total=%d", totalSearch)
	}

	// 5. Status filter
	statusOne := int8(1)
	listStatus, totalStatus, err := svc.List(ctx, productdictservice.DictServiceListRequest{
		Page:     1,
		PageSize: 20,
		Status:   &statusOne,
	})
	if err != nil {
		t.Fatalf("list with status filter failed: %v", err)
	}
	if totalStatus != 1 || listStatus[0].ID != created1.ID {
		t.Errorf("expected 1 active record, got total=%d", totalStatus)
	}

	// 6. Update
	newSort := 5
	newStatus := int8(1)
	updated, err := svc.Update(ctx, created2.ID, productdictservice.DictServiceUpdateRequest{
		Name:        "极速退款-升级版",
		Description: "升级版说明",
		Sort:        &newSort,
		Status:      &newStatus,
	})
	if err != nil {
		t.Fatalf("update dict service failed: %v", err)
	}
	if updated.Name != "极速退款-升级版" || updated.Sort != 5 || updated.Status != 1 {
		t.Errorf("unexpected updated data: %+v", updated)
	}

	// 7. Delete
	if err := svc.Delete(ctx, created1.ID); err != nil {
		t.Fatalf("delete dict service failed: %v", err)
	}
	listAfterDelete, totalAfterDelete, _ := svc.List(ctx, productdictservice.DictServiceListRequest{Page: 1, PageSize: 20})
	if totalAfterDelete != 1 || len(listAfterDelete) != 1 {
		t.Errorf("expected 1 item after delete, got %d", totalAfterDelete)
	}
}

func TestProductDictService_Validation(t *testing.T) {
	db := dictTestDB(t)
	svc := productdictservice.NewService(db)
	ctx := context.Background()

	// 1. Empty name
	_, err := svc.Create(ctx, productdictservice.DictServiceCreateRequest{Name: "   "})
	if !errors.Is(err, productdictservice.ErrNameRequired) {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}

	// 2. Name too long (>64 runes)
	longName := strings.Repeat("服务", 33)
	_, err = svc.Create(ctx, productdictservice.DictServiceCreateRequest{Name: longName})
	if !errors.Is(err, productdictservice.ErrNameTooLong) {
		t.Errorf("expected ErrNameTooLong, got %v", err)
	}

	// 3. Description too long (>255 runes)
	longDesc := strings.Repeat("说明", 129)
	_, err = svc.Create(ctx, productdictservice.DictServiceCreateRequest{
		Name:        "正常服务",
		Description: longDesc,
	})
	if !errors.Is(err, productdictservice.ErrDescriptionTooLong) {
		t.Errorf("expected ErrDescriptionTooLong, got %v", err)
	}

	// 4. Invalid status on create
	badStatus := int8(2)
	_, err = svc.Create(ctx, productdictservice.DictServiceCreateRequest{
		Name:   "服务A",
		Status: &badStatus,
	})
	if !errors.Is(err, productdictservice.ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}

	// 5. Duplicate name on create
	_, err = svc.Create(ctx, productdictservice.DictServiceCreateRequest{Name: "正品保障"})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err = svc.Create(ctx, productdictservice.DictServiceCreateRequest{Name: "正品保障"})
	if !errors.Is(err, productdictservice.ErrNameExists) {
		t.Errorf("expected ErrNameExists, got %v", err)
	}

	// 6. Non-existent ID on update and delete
	_, err = svc.Update(ctx, 99999, productdictservice.DictServiceUpdateRequest{Name: "不存在"})
	if !errors.Is(err, productdictservice.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	err = svc.Delete(ctx, 99999)
	if !errors.Is(err, productdictservice.ErrNotFound) {
		t.Errorf("expected ErrNotFound on delete, got %v", err)
	}
}

func TestProductDictUnit_CRUD(t *testing.T) {
	db := dictTestDB(t)
	svc := productdictunit.NewService(db)
	ctx := context.Background()

	// 1. Create units
	u1, err := svc.Create(ctx, productdictunit.DictUnitCreateRequest{
		Name:        "件",
		Description: "单件商品",
		Sort:        1,
	})
	if err != nil {
		t.Fatalf("create unit 1 failed: %v", err)
	}
	if u1.Name != "件" || u1.Status != 1 || u1.Sort != 1 {
		t.Errorf("unexpected unit 1 data: %+v", u1)
	}

	u2, err := svc.Create(ctx, productdictunit.DictUnitCreateRequest{
		Name:        "台",
		Description: "电子设备单位",
		Sort:        2,
	})
	if err != nil {
		t.Fatalf("create unit 2 failed: %v", err)
	}

	// 2. List
	list, total, err := svc.List(ctx, productdictunit.DictUnitListRequest{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list units failed: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Errorf("expected 2 units, got total=%d, len=%d", total, len(list))
	}

	// 3. Update
	newSort := 10
	statusZero := int8(0)
	updated, err := svc.Update(ctx, u2.ID, productdictunit.DictUnitUpdateRequest{
		Name:        "台(台式)",
		Description: "更新说明",
		Sort:        &newSort,
		Status:      &statusZero,
	})
	if err != nil {
		t.Fatalf("update unit failed: %v", err)
	}
	if updated.Name != "台(台式)" || updated.Sort != 10 || updated.Status != 0 {
		t.Errorf("unexpected updated unit: %+v", updated)
	}

	// 4. Delete
	if err := svc.Delete(ctx, u1.ID); err != nil {
		t.Fatalf("delete unit failed: %v", err)
	}
	listAfterDelete, totalAfterDelete, _ := svc.List(ctx, productdictunit.DictUnitListRequest{Page: 1, PageSize: 10})
	if totalAfterDelete != 1 || listAfterDelete[0].ID != u2.ID {
		t.Errorf("expected 1 unit remaining, got total=%d", totalAfterDelete)
	}
}

func TestProductDictUnit_Validation(t *testing.T) {
	db := dictTestDB(t)
	svc := productdictunit.NewService(db)
	ctx := context.Background()

	// 1. Empty name
	_, err := svc.Create(ctx, productdictunit.DictUnitCreateRequest{Name: "   "})
	if !errors.Is(err, productdictunit.ErrNameRequired) {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}

	// 2. Name too long (>32 runes)
	longName := strings.Repeat("单位", 17)
	_, err = svc.Create(ctx, productdictunit.DictUnitCreateRequest{Name: longName})
	if !errors.Is(err, productdictunit.ErrNameTooLong) {
		t.Errorf("expected ErrNameTooLong, got %v", err)
	}

	// 3. Description too long (>255 runes)
	longDesc := strings.Repeat("说明", 129)
	_, err = svc.Create(ctx, productdictunit.DictUnitCreateRequest{
		Name:        "箱",
		Description: longDesc,
	})
	if !errors.Is(err, productdictunit.ErrDescriptionTooLong) {
		t.Errorf("expected ErrDescriptionTooLong, got %v", err)
	}

	// 4. Duplicate name
	_, err = svc.Create(ctx, productdictunit.DictUnitCreateRequest{Name: "个"})
	if err != nil {
		t.Fatalf("create unit failed: %v", err)
	}
	_, err = svc.Create(ctx, productdictunit.DictUnitCreateRequest{Name: "个"})
	if !errors.Is(err, productdictunit.ErrNameExists) {
		t.Errorf("expected ErrNameExists, got %v", err)
	}

	// 5. Non-existent ID on update and delete
	_, err = svc.Update(ctx, 99999, productdictunit.DictUnitUpdateRequest{Name: "不存在"})
	if !errors.Is(err, productdictunit.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	err = svc.Delete(ctx, 99999)
	if !errors.Is(err, productdictunit.ErrNotFound) {
		t.Errorf("expected ErrNotFound on delete, got %v", err)
	}
}

func TestProductDict_Permissions(t *testing.T) {
	expectedCodes := []string{
		auth.PermissionProductDictServiceList,
		auth.PermissionProductDictServiceCreate,
		auth.PermissionProductDictServiceUpdate,
		auth.PermissionProductDictServiceDelete,
		auth.PermissionProductDictUnitList,
		auth.PermissionProductDictUnitCreate,
		auth.PermissionProductDictUnitUpdate,
		auth.PermissionProductDictUnitDelete,
	}

	for _, code := range expectedCodes {
		if !auth.IsPermissionCode(code) {
			t.Errorf("expected permission code %s to be valid", code)
		}
	}
}
