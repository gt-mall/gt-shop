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
	"gorm.io/gorm/logger"
	miniapporder "mall/admin_core/internal/miniapp/order"
	miniapppayment "mall/admin_core/internal/miniapp/payment"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/bizno"
)

// ORDER_TEST_MYSQL_DSN must point to a disposable database ending in _test.
// This test never uses the application's DATABASE_DSN.
func TestPaymentMySQLConcurrency(t *testing.T) {
	dsn := os.Getenv("ORDER_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("ORDER_TEST_MYSQL_DSN is not set; isolated MySQL test skipped")
	}
	cfg, err := drivermysql.ParseDSN(dsn)
	if err != nil || !strings.HasSuffix(cfg.DBName, "_test") {
		t.Fatal("ORDER_TEST_MYSQL_DSN must name a disposable database ending in _test")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot connect to isolated MySQL database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(16)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&model.MemberAddress{}, &model.Product{}, &model.ProductSKU{},
		&model.ProductSkuInventory{}, &model.ProductSkuInventoryLog{}, &model.Member{}, &model.MemberAuth{}, &model.SystemConfig{}, &model.OrderMain{},
		&model.OrderItem{}, &model.OrderAddress{}, &model.OrderLog{}, &model.OrderPayment{}); err != nil {
		t.Fatal(err)
	}
	gen, err := bizno.NewGenerator(1)
	if err != nil {
		t.Fatal(err)
	}
	orderSvc := miniapporder.NewService(db, gen)
	t.Run("different_orders_create_payments", func(t *testing.T) {
		var orders []model.OrderMain
		for i := 0; i < 12; i++ {
			orders = append(orders, model.OrderMain{
				OrderNo:  fmt.Sprintf("parallel-%d-%d", time.Now().UnixNano(), i),
				MemberID: 42, RequestID: fmt.Sprintf("parallel-%d-%d", time.Now().UnixNano(), i),
				Status: model.OrderStatusPendingPayment, PayAmount: "1.00",
			})
		}
		if err := db.Create(&orders).Error; err != nil {
			t.Fatal(err)
		}
		svc := miniapppayment.NewService(db, gen)
		var wg sync.WaitGroup
		start := make(chan struct{})
		for _, order := range orders {
			wg.Add(1)
			go func(orderID int64) {
				defer wg.Done()
				<-start
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if _, err := svc.CreatePayment(ctx, 42, miniapppayment.CreatePaymentRequest{OrderID: orderID}); err != nil {
					t.Errorf("create payment for different orders: %v", err)
				}
			}(order.ID)
		}
		close(start)
		wg.Wait()
	})
	t.Run("same_order_concurrent_wechat_callback", func(t *testing.T) {
		memberID := time.Now().UnixNano()
		_, skuID, addrID := createOrderTestData(t, db, memberID)
		order, err := orderSvc.Create(context.Background(), model.Member{ID: memberID}, miniapporder.OrderCreateRequest{
			RequestID: fmt.Sprintf("mysql-callback-%d", memberID), AddressID: addrID,
			Items: []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 2}},
		})
		if err != nil {
			t.Fatal(err)
		}
		fixture := newWechatPaymentFixture(t, db, memberID)
		p, err := fixture.service.CreatePayment(context.Background(), memberID, miniapppayment.CreatePaymentRequest{OrderID: order.ID})
		if err != nil {
			t.Fatal(err)
		}
		txn := fixture.transaction(t, p)
		const callers = 12
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := 0; i < callers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				if _, err := fixture.service.ConfirmWechat(context.Background(), txn); err != nil {
					t.Errorf("concurrent WeChat callback: %v", err)
				}
			}()
		}
		close(start)
		wg.Wait()
		assertPaymentStock(t, db, skuID, 48, 0)
		var logs int64
		if err := db.Model(&model.OrderLog{}).Where("order_id = ? AND action = ?", order.ID, "pay").Count(&logs).Error; err != nil || logs != 1 {
			t.Fatalf("pay log count=%d, err=%v", logs, err)
		}
	})
	t.Run("same_order_concurrent_create", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			memberID := time.Now().UnixNano()
			_, skuID, addrID := createOrderTestData(t, db, memberID)
			order, err := orderSvc.Create(context.Background(), model.Member{ID: memberID}, miniapporder.OrderCreateRequest{
				RequestID: fmt.Sprintf("mysql-%d", memberID), AddressID: addrID,
				Items: []miniapporder.CheckoutItemRequest{{SKUID: skuID, Quantity: 2}},
			})
			if err != nil {
				t.Fatal(err)
			}
			testConcurrentPaymentCreation(t, db, memberID, order.ID, skuID)
		}
	})
}
