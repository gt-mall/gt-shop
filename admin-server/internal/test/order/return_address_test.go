package order_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/order/returnaddress"
)

func setupReturnAddressTest(t *testing.T) (*returnaddress.Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&model.OrderReturnAddress{}); err != nil {
		t.Fatalf("migrate return address: %v", err)
	}

	return returnaddress.NewService(db), db
}

func TestReturnAddressCRUDAndDefault(t *testing.T) {
	svc, _ := setupReturnAddressTest(t)
	ctx := context.Background()

	// 1. Create first address as default
	req1 := returnaddress.CreateReturnAddressRequest{
		Name:         "默认华东总仓",
		Receiver:     "张三",
		Phone:        "13800000001",
		ProvinceCode: "310000",
		CityCode:     "310100",
		DistrictCode: "310101",
		Address:      "上海市黄浦区人民大道100号",
		IsDefault:    1,
		Status:       1,
		Sort:         10,
	}
	id1, err := svc.Create(ctx, req1)
	if err != nil {
		t.Fatalf("create address 1 failed: %v", err)
	}
	if id1 <= 0 {
		t.Fatalf("expected positive ID, got %d", id1)
	}

	// Verify it is default
	def1, err := svc.GetDefault(ctx)
	if err != nil || def1.ID != id1 || def1.IsDefault != 1 {
		t.Fatalf("expected address 1 as default, got %+v, err: %v", def1, err)
	}

	// 2. Create second address, also marked default -> should unset first
	req2 := returnaddress.CreateReturnAddressRequest{
		Name:         "华南第二仓",
		Receiver:     "李四",
		Phone:        "13800000002",
		ProvinceCode: "440000",
		CityCode:     "440100",
		DistrictCode: "440106",
		Address:      "广州市天河区天河路1号",
		IsDefault:    1,
		Status:       1,
		Sort:         20,
	}
	id2, err := svc.Create(ctx, req2)
	if err != nil {
		t.Fatalf("create address 2 failed: %v", err)
	}

	// Verify second is now default, first is no longer default
	def2, err := svc.GetDefault(ctx)
	if err != nil || def2.ID != id2 || def2.IsDefault != 1 {
		t.Fatalf("expected address 2 as default, got %+v, err: %v", def2, err)
	}
	d1, err := svc.Get(ctx, id1)
	if err != nil || d1.IsDefault != 0 {
		t.Fatalf("expected address 1 is_default 0, got %+v, err: %v", d1, err)
	}

	// 3. List
	list, err := svc.List(ctx, returnaddress.ListReturnAddressesRequest{})
	if err != nil || len(list) != 2 {
		t.Fatalf("expected 2 items, got len=%d, err=%v", len(list), err)
	}

	// 4. Update address 1 to become default again
	err = svc.Update(ctx, returnaddress.UpdateReturnAddressRequest{
		ID:           id1,
		Name:         "默认华东总仓-更新",
		Receiver:     "张三",
		Phone:        "13800000001",
		ProvinceCode: "310000",
		CityCode:     "310100",
		DistrictCode: "310101",
		Address:      "上海市黄浦区人民大道100号A座",
		IsDefault:    1,
		Status:       1,
		Sort:         5,
	})
	if err != nil {
		t.Fatalf("update address 1 failed: %v", err)
	}
	defAfterUpdate, err := svc.GetDefault(ctx)
	if err != nil || defAfterUpdate.ID != id1 {
		t.Fatalf("expected address 1 to be default after update, got %+v", defAfterUpdate)
	}

	// 5. Explicit SetDefault for address 2
	if err := svc.SetDefault(ctx, id2); err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}
	defAfterSet, err := svc.GetDefault(ctx)
	if err != nil || defAfterSet.ID != id2 {
		t.Fatalf("expected address 2 default after SetDefault, got %+v", defAfterSet)
	}

	// 6. Delete address 1
	if err := svc.Delete(ctx, id1); err != nil {
		t.Fatalf("delete address 1 failed: %v", err)
	}
	_, err = svc.Get(ctx, id1)
	if err == nil {
		t.Fatal("expected error getting deleted address 1")
	}

	listAfterDel, err := svc.List(ctx, returnaddress.ListReturnAddressesRequest{})
	if err != nil || len(listAfterDel) != 1 {
		t.Fatalf("expected 1 item after delete, got len=%d, err=%v", len(listAfterDel), err)
	}
}
