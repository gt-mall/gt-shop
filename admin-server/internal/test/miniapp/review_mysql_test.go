package miniapp_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	miniappreview "mall/admin_core/internal/miniapp/review"
	"mall/admin_core/internal/model"
)

// This regression requires InnoDB REPEATABLE READ. It creates only uniquely
// named rows in the disposable *_test database and removes those rows afterward.
func TestReviewMySQLSeesAfterSaleCommittedWhileWaitingForOrderLock(t *testing.T) {
	dsn := os.Getenv("ORDER_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("ORDER_TEST_MYSQL_DSN is not set; review snapshot regression skipped")
	}
	cfg, err := drivermysql.ParseDSN(dsn)
	if err != nil || !strings.HasSuffix(cfg.DBName, "_test") {
		t.Fatal("ORDER_TEST_MYSQL_DSN must name a disposable database ending in _test")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot connect to isolated MySQL database")
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.OrderMain{}, &model.OrderItem{}, &model.ProductReview{}, &model.ProductReviewImage{}, &model.AfterSale{}, &model.AfterSaleItem{}); err != nil {
		t.Fatal(err)
	}

	stamp := time.Now().UnixNano()
	memberID := stamp
	now := time.Now()
	order := model.OrderMain{OrderNo: fmt.Sprintf("review-race-%d", stamp), MemberID: memberID, RequestID: fmt.Sprintf("review-race-order-%d", stamp), Status: model.OrderStatusCompleted, TotalAmount: "1.00", PayAmount: "1.00", CompletedAt: &now}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	item := model.OrderItem{OrderID: order.ID, ProductID: stamp, SKUID: stamp, ProductName: "并发评价商品", UnitPrice: "1.00", Quantity: 1, TotalAmount: "1.00"}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Where("order_item_id = ?", item.ID).Delete(&model.ProductReview{})
		db.Where("order_item_id = ?", item.ID).Delete(&model.AfterSaleItem{})
		db.Where("order_id = ?", order.ID).Delete(&model.AfterSale{})
		db.Delete(&item)
		db.Unscoped().Delete(&order)
	})

	locker := db.Begin()
	if locker.Error != nil {
		t.Fatal(locker.Error)
	}
	var locked model.OrderMain
	if err := locker.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, order.ID).Error; err != nil {
		locker.Rollback()
		t.Fatal(err)
	}

	lockAttempted := make(chan struct{})
	callbackName := fmt.Sprintf("test:review-order-lock:%d", stamp)
	var once sync.Once
	if err := db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "order_main" {
			once.Do(func() { close(lockAttempted) })
		}
	}); err != nil {
		locker.Rollback()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callbackName) })

	result := make(chan error, 1)
	go func() {
		_, createErr := miniappreview.NewService(db, nil).Create(context.Background(), memberID, "并发用户", miniappreview.MiniappReviewCreateRequest{
			OrderItemID: item.ID,
			Rating:      5,
			Content:     "不应越过售后创建",
			RequestID:   fmt.Sprintf("review-race-request-%d", stamp),
		})
		result <- createErr
	}()

	select {
	case <-lockAttempted:
	case <-time.After(5 * time.Second):
		locker.Rollback()
		t.Fatal("review did not reach the order lock")
	}
	afterSale := model.AfterSale{AfterSaleNo: fmt.Sprintf("AS-RACE-%d", stamp), RequestID: fmt.Sprintf("as-race-request-%d", stamp), OrderID: order.ID, MemberID: memberID, Type: model.AfterSaleTypeRefundOnly, Status: model.AfterSaleStatusPendingReview, OrderStatusAtApply: model.OrderStatusCompleted, Reason: "并发测试", RefundAmount: "1.00", AppliedAt: now}
	if err := locker.Create(&afterSale).Error; err != nil {
		locker.Rollback()
		t.Fatal(err)
	}
	if err := locker.Create(&model.AfterSaleItem{AfterSaleID: afterSale.ID, OrderItemID: item.ID, ProductID: item.ProductID, SKUID: item.SKUID, Quantity: 1, RefundAmount: "1.00", ProductName: item.ProductName}).Error; err != nil {
		locker.Rollback()
		t.Fatal(err)
	}
	if err := locker.Commit().Error; err != nil {
		t.Fatal(err)
	}

	select {
	case createErr := <-result:
		if createErr == nil || !strings.Contains(createErr.Error(), "售后") {
			t.Fatalf("review missed the newly committed after-sale: %v", createErr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("review remained blocked after after-sale commit")
	}
}
