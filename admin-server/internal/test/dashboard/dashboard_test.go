package dashboard_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/dashboard"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/response"
	"mall/admin_core/routers"
)

const testAdminSecret = "test_admin_jwt_secret_1234567890"

type testEnv struct {
	db          *gorm.DB
	service     *dashboard.Service
	router      http.Handler
	adminUser   model.SystemUser
	adminToken  string
	authService *auth.Service
	enforcer    *casbin.SyncedEnforcer
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })

	if err := db.AutoMigrate(
		&model.SystemUser{},
		&model.SystemRole{},
		&model.Member{},
		&model.Product{},
		&model.ProductSKU{},
		&model.ProductSkuInventory{},
		&model.OrderMain{},
		&model.OrderItem{},
		&model.AfterSale{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Super Admin
	admin := model.SystemUser{
		Username: "superadmin",
		Nickname: "超级管理员",
		Status:   1,
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	superRole := model.SystemRole{
		Code:   auth.SuperAdminRoleCode,
		Name:   "超级管理员",
		Status: 1,
	}
	if err := db.Create(&superRole).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	m := casbinmodel.NewModel()
	m.AddDef("r", "r", "sub, perm")
	m.AddDef("p", "p", "sub, perm")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p_eft == allow))")
	m.AddDef("m", "m", "g(r_sub, p_sub) && r_perm == p_perm")
	enforcer, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	if _, err := enforcer.AddRoleForUser(auth.UserSubject(admin.ID), auth.SuperAdminRoleCode); err != nil {
		t.Fatalf("add super admin role: %v", err)
	}

	authService := auth.NewService(db, enforcer, testAdminSecret, 24*time.Hour)

	adminClaims := jwt.RegisteredClaims{
		Issuer:    "mall-admin-core",
		Subject:   strconv.FormatInt(admin.ID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims).SignedString([]byte(testAdminSecret))
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	bConfig := bootstrap.Config{
		JWTSecret:     "miniapp_jwt_secret_12345",
		JWTExpireHour: 24,
	}
	router := routers.NewRouter(bConfig, db, authService)

	return &testEnv{
		db:          db,
		service:     dashboard.NewService(db),
		router:      router,
		adminUser:   admin,
		adminToken:  token,
		authService: authService,
		enforcer:    enforcer,
	}
}

func (env *testEnv) generateUserToken(t *testing.T, userID int64) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		Issuer:    "mall-admin-core",
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testAdminSecret))
	if err != nil {
		t.Fatalf("generate user token: %v", err)
	}
	return token
}

// 1. 无订单时统计全部为 0
func TestDashboardOverviewEmpty(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.TodayOrderCount != 0 {
		t.Errorf("expected TodayOrderCount 0, got %d", res.TodayOrderCount)
	}
	if res.TodayPaidOrderCount != 0 {
		t.Errorf("expected TodayPaidOrderCount 0, got %d", res.TodayPaidOrderCount)
	}
	if res.TodaySalesAmount != "0.00" {
		t.Errorf("expected TodaySalesAmount '0.00', got %q", res.TodaySalesAmount)
	}
	if res.PendingPaymentCount != 0 {
		t.Errorf("expected PendingPaymentCount 0, got %d", res.PendingPaymentCount)
	}
	if res.PendingDeliveryCount != 0 {
		t.Errorf("expected PendingDeliveryCount 0, got %d", res.PendingDeliveryCount)
	}
	if res.PendingAftersaleCount != 0 {
		t.Errorf("expected PendingAftersaleCount 0, got %d", res.PendingAftersaleCount)
	}
	if res.ProductCount != 0 {
		t.Errorf("expected ProductCount 0, got %d", res.ProductCount)
	}
	if res.OnSaleProductCount != 0 {
		t.Errorf("expected OnSaleProductCount 0, got %d", res.OnSaleProductCount)
	}
	if res.LowStockSKUCount != 0 {
		t.Errorf("expected LowStockSKUCount 0, got %d", res.LowStockSKUCount)
	}
	if res.MemberCount != 0 {
		t.Errorf("expected MemberCount 0, got %d", res.MemberCount)
	}
	if res.TodayNewMemberCount != 0 {
		t.Errorf("expected TodayNewMemberCount 0, got %d", res.TodayNewMemberCount)
	}
}

// 2. 今日订单统计
func TestDashboardOverviewTodayOrders(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	// 2 orders today, 1 order yesterday
	yesterday := now.AddDate(0, 0, -1)
	env.db.Create(&model.OrderMain{OrderNo: "TODAY-1", MemberID: 1, RequestID: "r1", Status: model.OrderStatusPendingPayment, TotalAmount: "100.00", PayAmount: "100.00", CreatedAt: now})
	env.db.Create(&model.OrderMain{OrderNo: "TODAY-2", MemberID: 1, RequestID: "r2", Status: model.OrderStatusPendingShipment, TotalAmount: "50.00", PayAmount: "50.00", CreatedAt: now})
	env.db.Create(&model.OrderMain{OrderNo: "YEST-1", MemberID: 1, RequestID: "r3", Status: model.OrderStatusPendingShipment, TotalAmount: "80.00", PayAmount: "80.00", CreatedAt: yesterday})

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.TodayOrderCount != 2 {
		t.Errorf("expected TodayOrderCount 2, got %d", res.TodayOrderCount)
	}
}

// 3. 已支付订单销售额
func TestDashboardOverviewPaidSalesAmount(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	// 2 paid orders today
	env.db.Create(&model.OrderMain{OrderNo: "PAID-1", MemberID: 1, RequestID: "r1", Status: model.OrderStatusPendingShipment, TotalAmount: "199.90", PayAmount: "199.90", CreatedAt: now})
	env.db.Create(&model.OrderMain{OrderNo: "PAID-2", MemberID: 1, RequestID: "r2", Status: model.OrderStatusCompleted, TotalAmount: "300.10", PayAmount: "300.10", CreatedAt: now})

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.TodayPaidOrderCount != 2 {
		t.Errorf("expected TodayPaidOrderCount 2, got %d", res.TodayPaidOrderCount)
	}
	if res.TodaySalesAmount != "500.00" {
		t.Errorf("expected TodaySalesAmount '500.00', got %q", res.TodaySalesAmount)
	}
}

// 4. 未支付订单不计入销售额
func TestDashboardOverviewUnpaidOrderExcluded(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	// 1 paid, 1 unpaid
	env.db.Create(&model.OrderMain{OrderNo: "PAID-1", MemberID: 1, RequestID: "r1", Status: model.OrderStatusPendingShipment, TotalAmount: "120.00", PayAmount: "120.00", CreatedAt: now})
	env.db.Create(&model.OrderMain{OrderNo: "UNPAID-1", MemberID: 1, RequestID: "r2", Status: model.OrderStatusPendingPayment, TotalAmount: "888.00", PayAmount: "888.00", CreatedAt: now})

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.TodaySalesAmount != "120.00" {
		t.Errorf("expected TodaySalesAmount '120.00', got %q", res.TodaySalesAmount)
	}
	if res.PendingPaymentCount != 1 {
		t.Errorf("expected PendingPaymentCount 1, got %d", res.PendingPaymentCount)
	}
}

// 5. 已取消订单不计入销售额
func TestDashboardOverviewCancelledOrderExcluded(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	env.db.Create(&model.OrderMain{OrderNo: "CANCEL-1", MemberID: 1, RequestID: "r1", Status: model.OrderStatusCancelled, TotalAmount: "500.00", PayAmount: "500.00", CreatedAt: now})

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.TodaySalesAmount != "0.00" {
		t.Errorf("expected TodaySalesAmount '0.00', got %q", res.TodaySalesAmount)
	}
	if res.TodayPaidOrderCount != 0 {
		t.Errorf("expected TodayPaidOrderCount 0, got %d", res.TodayPaidOrderCount)
	}
}

// 6. 待发货数量
func TestDashboardOverviewPendingDelivery(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	env.db.Create(&model.OrderMain{OrderNo: "SHIP-1", MemberID: 1, RequestID: "r1", Status: model.OrderStatusPendingShipment, CreatedAt: now})
	env.db.Create(&model.OrderMain{OrderNo: "SHIP-2", MemberID: 1, RequestID: "r2", Status: model.OrderStatusPendingShipment, CreatedAt: now.AddDate(0, 0, -2)})
	env.db.Create(&model.OrderMain{OrderNo: "RECV-1", MemberID: 1, RequestID: "r3", Status: model.OrderStatusPendingReceipt, CreatedAt: now})

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.PendingDeliveryCount != 2 {
		t.Errorf("expected PendingDeliveryCount 2, got %d", res.PendingDeliveryCount)
	}
}

// 7. 待售后数量
func TestDashboardOverviewPendingAftersale(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	// Counted: 10 (待审核), 30 (待商家收货), 80 (退款失败)
	env.db.Create(&model.AfterSale{AfterSaleNo: "AS-1", RequestID: "r1", OrderID: 1, MemberID: 1, Type: 1, Status: model.AfterSaleStatusPendingReview, CreatedAt: now})
	env.db.Create(&model.AfterSale{AfterSaleNo: "AS-2", RequestID: "r2", OrderID: 2, MemberID: 1, Type: 2, Status: model.AfterSaleStatusWaitingMerchantReceive, CreatedAt: now})
	env.db.Create(&model.AfterSale{AfterSaleNo: "AS-3", RequestID: "r3", OrderID: 3, MemberID: 1, Type: 1, Status: model.AfterSaleStatusRefundFailed, CreatedAt: now})

	// Not counted: 20 (待买家退货), 50 (已完成), 60 (已拒绝)
	env.db.Create(&model.AfterSale{AfterSaleNo: "AS-4", RequestID: "r4", OrderID: 4, MemberID: 1, Type: 2, Status: model.AfterSaleStatusWaitingBuyerReturn, CreatedAt: now})
	env.db.Create(&model.AfterSale{AfterSaleNo: "AS-5", RequestID: "r5", OrderID: 5, MemberID: 1, Type: 1, Status: model.AfterSaleStatusCompleted, CreatedAt: now})
	env.db.Create(&model.AfterSale{AfterSaleNo: "AS-6", RequestID: "r6", OrderID: 6, MemberID: 1, Type: 1, Status: model.AfterSaleStatusRejected, CreatedAt: now})

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.PendingAftersaleCount != 3 {
		t.Errorf("expected PendingAftersaleCount 3, got %d", res.PendingAftersaleCount)
	}
}

// 8. 商品数量
func TestDashboardOverviewProductCounts(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	// 2 on sale, 1 off sale, 1 deleted
	env.db.Create(&model.Product{Name: "P1", Status: 1, CreatedAt: now})
	env.db.Create(&model.Product{Name: "P2", Status: 1, CreatedAt: now})
	env.db.Create(&model.Product{Name: "P3", Status: 0, CreatedAt: now})
	deletedProd := model.Product{Name: "P4", Status: 1, CreatedAt: now}
	env.db.Create(&deletedProd)
	env.db.Delete(&deletedProd)

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.ProductCount != 3 {
		t.Errorf("expected ProductCount 3, got %d", res.ProductCount)
	}
	if res.OnSaleProductCount != 2 {
		t.Errorf("expected OnSaleProductCount 2, got %d", res.OnSaleProductCount)
	}
}

// 9. 会员数量
func TestDashboardOverviewMemberCounts(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	env.db.Create(&model.Member{Nickname: "M1", Status: 1, CreatedAt: now})
	env.db.Create(&model.Member{Nickname: "M2", Status: 1, CreatedAt: now})
	deletedMem := model.Member{Nickname: "M3", Status: 1, CreatedAt: now}
	env.db.Create(&deletedMem)
	env.db.Delete(&deletedMem)

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.MemberCount != 2 {
		t.Errorf("expected MemberCount 2, got %d", res.MemberCount)
	}
}

// 10. 今日新增会员
func TestDashboardOverviewTodayNewMembers(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	env.db.Create(&model.Member{Nickname: "TODAY-M1", Status: 1, CreatedAt: now})
	env.db.Create(&model.Member{Nickname: "TODAY-M2", Status: 1, CreatedAt: now})
	env.db.Create(&model.Member{Nickname: "YEST-M1", Status: 1, CreatedAt: yesterday})

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.TodayNewMemberCount != 2 {
		t.Errorf("expected TodayNewMemberCount 2, got %d", res.TodayNewMemberCount)
	}
	if res.MemberCount != 3 {
		t.Errorf("expected MemberCount 3, got %d", res.MemberCount)
	}
}

// 11. 低库存 SKU
func TestDashboardOverviewLowStockSKU(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Create product
	prod := model.Product{Name: "Test Product", Status: 1}
	env.db.Create(&prod)

	// SKU 1: warning_stock = 0 (default 10 applies). Stock = 5, locked = 0 -> available = 5 <= 10 -> Low stock!
	sku1 := model.ProductSKU{ProductID: prod.ID, SpecValues: "{}", SpecHash: "h1", Price: "10.00", CostPrice: "5.00", MarketPrice: "20.00", Status: 1}
	env.db.Create(&sku1)
	env.db.Create(&model.ProductSkuInventory{ProductID: prod.ID, SKUID: sku1.ID, Stock: 5, LockedStock: 0, WarningStock: 0})

	// SKU 2: warning_stock = 20. Stock = 15, locked = 0 -> available = 15 <= 20 -> Low stock!
	sku2 := model.ProductSKU{ProductID: prod.ID, SpecValues: "{}", SpecHash: "h2", Price: "10.00", CostPrice: "5.00", MarketPrice: "20.00", Status: 1}
	env.db.Create(&sku2)
	env.db.Create(&model.ProductSkuInventory{ProductID: prod.ID, SKUID: sku2.ID, Stock: 15, LockedStock: 0, WarningStock: 20})

	// SKU 3: warning_stock = 5. Stock = 8, locked = 0 -> available = 8 > 5 -> Normal!
	sku3 := model.ProductSKU{ProductID: prod.ID, SpecValues: "{}", SpecHash: "h3", Price: "10.00", CostPrice: "5.00", MarketPrice: "20.00", Status: 1}
	env.db.Create(&sku3)
	env.db.Create(&model.ProductSkuInventory{ProductID: prod.ID, SKUID: sku3.ID, Stock: 8, LockedStock: 0, WarningStock: 5})

	// SKU 4: uninitialized inventory (no inventory row, stock=0 <= 10) -> Low stock!
	sku4 := model.ProductSKU{ProductID: prod.ID, SpecValues: "{}", SpecHash: "h4", Price: "10.00", CostPrice: "5.00", MarketPrice: "20.00", Status: 1}
	env.db.Create(&sku4)

	res, err := env.service.Overview(ctx)
	if err != nil {
		t.Fatalf("overview failed: %v", err)
	}

	if res.LowStockSKUCount != 3 {
		t.Errorf("expected LowStockSKUCount 3, got %d", res.LowStockSKUCount)
	}
}

// 12. 7 天趋势
func TestDashboardTrend7Days(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	// Create orders on today and 3 days ago
	threeDaysAgo := now.AddDate(0, 0, -3)
	env.db.Create(&model.OrderMain{OrderNo: "T-TODAY", MemberID: 1, RequestID: "r1", Status: model.OrderStatusPendingShipment, TotalAmount: "150.00", PayAmount: "150.00", CreatedAt: now})
	env.db.Create(&model.OrderMain{OrderNo: "T-3DAYS", MemberID: 1, RequestID: "r2", Status: model.OrderStatusCompleted, TotalAmount: "200.00", PayAmount: "200.00", CreatedAt: threeDaysAgo})

	trend, err := env.service.Trend(ctx, dashboard.DashboardTrendRequest{Days: 7})
	if err != nil {
		t.Fatalf("trend failed: %v", err)
	}

	if len(trend.Dates) != 7 {
		t.Fatalf("expected 7 dates, got %d", len(trend.Dates))
	}
	if len(trend.OrderCounts) != 7 {
		t.Fatalf("expected 7 order counts, got %d", len(trend.OrderCounts))
	}
	if len(trend.SalesAmounts) != 7 {
		t.Fatalf("expected 7 sales amounts, got %d", len(trend.SalesAmounts))
	}

	// Last item is today
	todayIdx := 6
	if trend.Dates[todayIdx] != now.Format("2006-01-02") {
		t.Errorf("expected last date %s, got %s", now.Format("2006-01-02"), trend.Dates[todayIdx])
	}
	if trend.OrderCounts[todayIdx] != 1 {
		t.Errorf("expected today order count 1, got %d", trend.OrderCounts[todayIdx])
	}
	if trend.SalesAmounts[todayIdx] != "150.00" {
		t.Errorf("expected today sales amount 150.00, got %s", trend.SalesAmounts[todayIdx])
	}

	// 3 days ago is index 3
	threeDaysIdx := 3
	if trend.OrderCounts[threeDaysIdx] != 1 {
		t.Errorf("expected 3 days ago order count 1, got %d", trend.OrderCounts[threeDaysIdx])
	}
	if trend.SalesAmounts[threeDaysIdx] != "200.00" {
		t.Errorf("expected 3 days ago sales amount 200.00, got %s", trend.SalesAmounts[threeDaysIdx])
	}
}

// 13. 30 天趋势
func TestDashboardTrend30Days(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	trend, err := env.service.Trend(ctx, dashboard.DashboardTrendRequest{Days: 30})
	if err != nil {
		t.Fatalf("trend failed: %v", err)
	}

	if len(trend.Dates) != 30 {
		t.Fatalf("expected 30 dates, got %d", len(trend.Dates))
	}
	if len(trend.OrderCounts) != 30 {
		t.Fatalf("expected 30 order counts, got %d", len(trend.OrderCounts))
	}
	if len(trend.SalesAmounts) != 30 {
		t.Fatalf("expected 30 sales amounts, got %d", len(trend.SalesAmounts))
	}
}

// 14. 无订单日期自动补 0
func TestDashboardTrendFillMissingDates(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// No orders inserted at all
	trend, err := env.service.Trend(ctx, dashboard.DashboardTrendRequest{Days: 7})
	if err != nil {
		t.Fatalf("trend failed: %v", err)
	}

	for i, d := range trend.Dates {
		if trend.OrderCounts[i] != 0 {
			t.Errorf("date %s: expected order count 0, got %d", d, trend.OrderCounts[i])
		}
		if trend.SalesAmounts[i] != "0.00" {
			t.Errorf("date %s: expected sales amount '0.00', got %q", d, trend.SalesAmounts[i])
		}
	}
}

// 15. Dashboard 权限校验
func TestDashboardPermissionCheck(t *testing.T) {
	env := setupTestEnv(t)

	// Scenario A: Super Admin with adminToken should succeed (200 OK)
	reqOverview, _ := http.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	reqOverview.Header.Set("Authorization", "Bearer "+env.adminToken)
	wOverview := httptest.NewRecorder()
	env.router.ServeHTTP(wOverview, reqOverview)

	if wOverview.Code != http.StatusOK {
		t.Fatalf("super admin overview expected 200, got %d: %s", wOverview.Code, wOverview.Body.String())
	}
	var respOverview response.Response[dashboard.DashboardOverviewResponse]
	if err := json.Unmarshal(wOverview.Body.Bytes(), &respOverview); err != nil {
		t.Fatalf("unmarshal overview response: %v", err)
	}
	if respOverview.Code != 1 {
		t.Fatalf("expected code 1, got %d: %s", respOverview.Code, respOverview.Message)
	}

	// Trend with days=7
	reqTrend, _ := http.NewRequest(http.MethodGet, "/api/dashboard/trend?days=7", nil)
	reqTrend.Header.Set("Authorization", "Bearer "+env.adminToken)
	wTrend := httptest.NewRecorder()
	env.router.ServeHTTP(wTrend, reqTrend)

	if wTrend.Code != http.StatusOK {
		t.Fatalf("super admin trend expected 200, got %d: %s", wTrend.Code, wTrend.Body.String())
	}

	// Scenario B: Regular user without permission should get 403 Forbidden
	userWithoutPerm := model.SystemUser{
		Username: "operator_noperm",
		Nickname: "无权限操作员",
		Status:   1,
	}
	env.db.Create(&userWithoutPerm)
	tokenWithoutPerm := env.generateUserToken(t, userWithoutPerm.ID)

	reqNoPerm, _ := http.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	reqNoPerm.Header.Set("Authorization", "Bearer "+tokenWithoutPerm)
	wNoPerm := httptest.NewRecorder()
	env.router.ServeHTTP(wNoPerm, reqNoPerm)

	var respNoPerm response.Response[any]
	if err := json.Unmarshal(wNoPerm.Body.Bytes(), &respNoPerm); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if respNoPerm.Code != 0 || respNoPerm.Message != "permission denied" {
		t.Fatalf("user without permission expected Code 0 permission denied, got Code %d, Message %q", respNoPerm.Code, respNoPerm.Message)
	}

	// Also verify trend API is rejected
	reqNoPermTrend, _ := http.NewRequest(http.MethodGet, "/api/dashboard/trend?days=7", nil)
	reqNoPermTrend.Header.Set("Authorization", "Bearer "+tokenWithoutPerm)
	wNoPermTrend := httptest.NewRecorder()
	env.router.ServeHTTP(wNoPermTrend, reqNoPermTrend)
	var respNoPermTrend response.Response[any]
	if err := json.Unmarshal(wNoPermTrend.Body.Bytes(), &respNoPermTrend); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if respNoPermTrend.Code != 0 || respNoPermTrend.Message != "permission denied" {
		t.Fatalf("user without permission expected trend Code 0 permission denied, got Code %d, Message %q", respNoPermTrend.Code, respNoPermTrend.Message)
	}

	// Scenario C: Regular user WITH permission should succeed
	userWithPerm := model.SystemUser{
		Username: "operator_withperm",
		Nickname: "有权限操作员",
		Status:   1,
	}
	env.db.Create(&userWithPerm)
	tokenWithPerm := env.generateUserToken(t, userWithPerm.ID)

	roleWithPerm := model.SystemRole{
		Code:   "dashboard_viewer",
		Name:   "看板查看者",
		Status: 1,
	}
	env.db.Create(&roleWithPerm)
	if _, err := env.enforcer.AddRoleForUser(auth.UserSubject(userWithPerm.ID), roleWithPerm.Code); err != nil {
		t.Fatalf("add role: %v", err)
	}
	if _, err := env.enforcer.AddPermissionForUser(roleWithPerm.Code, auth.PermissionDashboardView); err != nil {
		t.Fatalf("add perm: %v", err)
	}

	reqWithPerm, _ := http.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	reqWithPerm.Header.Set("Authorization", "Bearer "+tokenWithPerm)
	wWithPerm := httptest.NewRecorder()
	env.router.ServeHTTP(wWithPerm, reqWithPerm)

	// Also verify trend API succeeds for user with permission
	reqWithPermTrend, _ := http.NewRequest(http.MethodGet, "/api/dashboard/trend?days=7", nil)
	reqWithPermTrend.Header.Set("Authorization", "Bearer "+tokenWithPerm)
	wWithPermTrend := httptest.NewRecorder()
	env.router.ServeHTTP(wWithPermTrend, reqWithPermTrend)
	if wWithPermTrend.Code != http.StatusOK {
		t.Fatalf("user with permission trend expected 200, got %d: %s", wWithPermTrend.Code, wWithPermTrend.Body.String())
	}

	// Scenario D: Unauthenticated requests return code -1 (authentication required)
	reqUnauthOverview, _ := http.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	wUnauthOverview := httptest.NewRecorder()
	env.router.ServeHTTP(wUnauthOverview, reqUnauthOverview)
	var respUnauthOverview response.Response[any]
	if err := json.Unmarshal(wUnauthOverview.Body.Bytes(), &respUnauthOverview); err != nil {
		t.Fatalf("unmarshal unauthenticated overview: %v", err)
	}
	if respUnauthOverview.Code != -1 || respUnauthOverview.Message != "authentication required" {
		t.Fatalf("unauthenticated overview expected Code -1 authentication required, got Code %d, Message %q", respUnauthOverview.Code, respUnauthOverview.Message)
	}

	reqUnauthTrend, _ := http.NewRequest(http.MethodGet, "/api/dashboard/trend?days=7", nil)
	wUnauthTrend := httptest.NewRecorder()
	env.router.ServeHTTP(wUnauthTrend, reqUnauthTrend)
	var respUnauthTrend response.Response[any]
	if err := json.Unmarshal(wUnauthTrend.Body.Bytes(), &respUnauthTrend); err != nil {
		t.Fatalf("unmarshal unauthenticated trend: %v", err)
	}
	if respUnauthTrend.Code != -1 || respUnauthTrend.Message != "authentication required" {
		t.Fatalf("unauthenticated trend expected Code -1 authentication required, got Code %d, Message %q", respUnauthTrend.Code, respUnauthTrend.Message)
	}

	// Scenario E: Permission definitions check
	if auth.PermissionDashboardView != "dashboard:view" {
		t.Fatalf("expected PermissionDashboardView = %q, got %q", "dashboard:view", auth.PermissionDashboardView)
	}

	var foundDashboardView bool
	for _, perm := range auth.AllPermissions() {
		if perm.Code == "dashboard:overview:view" {
			t.Fatalf("stale permission code dashboard:overview:view must not exist in AllPermissions")
		}
		if perm.Code == auth.PermissionDashboardView {
			foundDashboardView = true
			if perm.Name != "查看数据看板" {
				t.Fatalf("expected permission name %q, got %q", "查看数据看板", perm.Name)
			}
			if perm.Module != "数据看板" {
				t.Fatalf("expected permission module %q, got %q", "数据看板", perm.Module)
			}
		}
	}
	if !foundDashboardView {
		t.Fatalf("dashboard:view not found in AllPermissions")
	}
	if auth.IsPermissionCode("dashboard:overview:view") {
		t.Fatalf("dashboard:overview:view must not be considered a valid permission code")
	}
	if !auth.IsPermissionCode(auth.PermissionDashboardView) {
		t.Fatalf("dashboard:view must be a valid permission code")
	}
}
