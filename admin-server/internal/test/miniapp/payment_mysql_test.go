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
		&model.ProductSkuInventory{}, &model.ProductSkuInventoryLog{}, &model.OrderMain{},
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
	for _, scenario := range []string{"concurrent_create_and_pay", "cancel_and_pay", "timeout_and_pay"} {
		t.Run(scenario, func(t *testing.T) {
			// Multiple iterations exercise both lock acquisition orders.
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
				if scenario == "concurrent_create_and_pay" {
					testConcurrentPayment(t, db, memberID, order.ID, skuID)
				} else if scenario == "cancel_and_pay" {
					testCancelAndPay(t, db, memberID, order.ID, skuID)
				} else {
					testTimeoutAndPay(t, db, memberID, order.ID, skuID)
				}
			}
		})
	}
}
