package analytics_test

import (
	"bytes"
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
	"mall/admin_core/internal/analytics/flow"
	"mall/admin_core/internal/analytics/member"
	"mall/admin_core/internal/analytics/overview"
	"mall/admin_core/internal/analytics/product"
	"mall/admin_core/internal/analytics/trade"
	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/response"
	"mall/admin_core/routers"
)

const testJWTSecret = "test_analytics_jwt_secret_1234567890"

type testEnv struct {
	db         *gorm.DB
	router     http.Handler
	adminToken string
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
		&model.OrderPayment{},
		&model.AfterSale{},
		&model.AfterSaleItem{},
		&model.OrderRefund{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

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
		t.Fatalf("add role: %v", err)
	}

	authService := auth.NewService(db, enforcer, testJWTSecret, time.Hour)
	r := routers.NewRouter(bootstrap.Config{
		DatabaseSet: true,
		JWTSecret:   testJWTSecret,
	}, db, authService)

	adminClaims := jwt.RegisteredClaims{
		Issuer:    "mall-admin-core",
		Subject:   strconv.FormatInt(admin.ID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return &testEnv{
		db:         db,
		router:     r,
		adminToken: token,
	}
}

func (e *testEnv) get(t *testing.T, path string, dest any) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+e.adminToken)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)

	if dest != nil && w.Code == http.StatusOK {
		var resp response.Response[json.RawMessage]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response (%s): %v", w.Body.String(), err)
		}
		if resp.Code != 1 {
			t.Fatalf("api returned error (%s): %s", path, resp.Message)
		}
		if err := json.Unmarshal(resp.Data, dest); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
	}
	return w.Code
}

func (e *testEnv) post(t *testing.T, path string, body any, dest any) int {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+e.adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)

	if dest != nil && w.Code == http.StatusOK {
		var resp response.Response[json.RawMessage]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response (%s): %v", w.Body.String(), err)
		}
		if resp.Code != 1 {
			t.Fatalf("api returned error (%s): %s", path, resp.Message)
		}
		if err := json.Unmarshal(resp.Data, dest); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
	}
	return w.Code
}

// 1. 测试空数据场景：所有接口正常返回0/空，不panic
func TestAnalyticsEmptyDatabase(t *testing.T) {
	env := setupTestEnv(t)

	var overviewResp overview.OverviewResponse
	code := env.get(t, "/api/analytics/overview?time_range=today", &overviewResp)
	if code != http.StatusOK {
		t.Fatalf("overview status: %d", code)
	}
	if overviewResp.Metrics.PayAmount != "0.00" || overviewResp.Metrics.NetSalesAmount != "0.00" {
		t.Fatalf("expected 0.00 amounts, got %+v", overviewResp.Metrics)
	}
	if len(overviewResp.TopProducts) != 0 || len(overviewResp.RecentFlows) != 0 {
		t.Fatalf("expected empty lists, got %+v", overviewResp)
	}

	var tradeResp trade.TradeResponse
	code = env.get(t, "/api/analytics/trade?time_range=today", &tradeResp)
	if code != http.StatusOK {
		t.Fatalf("trade status: %d", code)
	}
	if tradeResp.Metrics.OrderCount != 0 || tradeResp.Metrics.PayAmount != "0.00" {
		t.Fatalf("expected 0 metrics, got %+v", tradeResp.Metrics)
	}

	var rankingsResp product.ProductRankingsResponse
	code = env.get(t, "/api/analytics/product/rankings?time_range=today", &rankingsResp)
	if code != http.StatusOK {
		t.Fatalf("rankings status: %d", code)
	}
	if len(rankingsResp.SalesAmountRankings) != 0 {
		t.Fatalf("expected empty rankings, got %d", len(rankingsResp.SalesAmountRankings))
	}

	var prodList []product.ProductAnalysisRow
	code = env.post(t, "/api/analytics/product/list", product.ProductListRequest{
		Dimension: "product",
		TimeRange: "today",
		Page:      1,
		PageSize:  10,
	}, &prodList)
	if code != http.StatusOK {
		t.Fatalf("product list status: %d", code)
	}
	if len(prodList) != 0 {
		t.Fatalf("expected empty product list, got %d", len(prodList))
	}

	var memberResp member.MemberResponse
	code = env.get(t, "/api/analytics/member?time_range=today", &memberResp)
	if code != http.StatusOK {
		t.Fatalf("member status: %d", code)
	}
	if memberResp.Metrics.TotalMemberCount != 0 || memberResp.Metrics.PaidMemberCount != 0 {
		t.Fatalf("expected 0 member metrics, got %+v", memberResp.Metrics)
	}

	var flowResp flow.FlowListResponse
	code = env.post(t, "/api/analytics/flows/list", flow.FlowListRequest{
		Page:     1,
		PageSize: 10,
		FlowType: "all",
	}, &flowResp)
	if code != http.StatusOK {
		t.Fatalf("flow list status: %d", code)
	}
	if flowResp.Total != 0 || len(flowResp.Items) != 0 {
		t.Fatalf("expected empty flow list, got %+v", flowResp)
	}
}

// 2. 验证核心口径场景：
// - 昨天创建订单、今天支付 -> 销售收入统计到今天
// - 今天支付、明天退款 -> 今天产生收入，明天产生退款
// - 订单部分退款 -> 退款金额准确
// - 支付失败 -> 不计收入
// - 退款失败 -> 不计退款
func TestAnalyticsFinancialScenarios(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := todayStart.AddDate(0, 0, -1).Add(2 * time.Hour)
	todayPaid := todayStart.Add(3 * time.Hour)
	tomorrowRefund := todayStart.AddDate(0, 0, 1).Add(4 * time.Hour)

	// 会员
	m1 := model.Member{Nickname: "买家1", CreatedAt: yesterday}
	m2 := model.Member{Nickname: "买家2", CreatedAt: todayPaid}
	env.db.Create(&m1)
	env.db.Create(&m2)

	// 商品与SKU
	p1 := model.Product{Name: "高级卫衣", Status: 1}
	env.db.Create(&p1)
	sku1 := model.ProductSKU{ProductID: p1.ID, Price: "100.00"}
	env.db.Create(&sku1)

	// 订单1：昨天创建，今天支付 199.00
	order1 := model.OrderMain{
		OrderNo:     "ORD_001",
		MemberID:    m1.ID,
		Status:      model.OrderStatusPendingShipment,
		TotalAmount: "199.00",
		PayAmount:   "199.00",
		CreatedAt:   yesterday,
		PaidAt:      &todayPaid,
	}
	env.db.Create(&order1)

	item1 := model.OrderItem{
		OrderID:     order1.ID,
		ProductID:   int64(p1.ID),
		SKUID:       int64(sku1.ID),
		ProductName: "高级卫衣",
		Quantity:    2,
		UnitPrice:   "99.50",
		TotalAmount: "199.00",
		CreatedAt:   yesterday,
	}
	env.db.Create(&item1)

	pay1 := model.OrderPayment{
		PaymentNo: "PAY_001",
		OrderID:   order1.ID,
		Channel:   "wechat",
		Amount:    "199.00",
		Status:    model.PaymentStatusSuccess,
		PaidAt:    &todayPaid,
		CreatedAt: yesterday,
	}
	env.db.Create(&pay1)

	// 订单2：今天创建、今天支付 200.00，明天部分退款 50.00
	order2 := model.OrderMain{
		OrderNo:     "ORD_002",
		MemberID:    m2.ID,
		Status:      model.OrderStatusPendingReceipt,
		TotalAmount: "200.00",
		PayAmount:   "200.00",
		CreatedAt:   todayPaid,
		PaidAt:      &todayPaid,
	}
	env.db.Create(&order2)

	item2 := model.OrderItem{
		OrderID:     order2.ID,
		ProductID:   int64(p1.ID),
		SKUID:       int64(sku1.ID),
		ProductName: "高级卫衣",
		Quantity:    2,
		UnitPrice:   "100.00",
		TotalAmount: "200.00",
		CreatedAt:   todayPaid,
	}
	env.db.Create(&item2)

	pay2 := model.OrderPayment{
		PaymentNo: "PAY_002",
		OrderID:   order2.ID,
		Channel:   "mock",
		Amount:    "200.00",
		Status:    model.PaymentStatusSuccess,
		PaidAt:    &todayPaid,
		CreatedAt: todayPaid,
	}
	env.db.Create(&pay2)

	// 订单2的退款：明天退款成功 50.00
	as1 := model.AfterSale{
		AfterSaleNo:  "AS_001",
		OrderID:      order2.ID,
		MemberID:     m2.ID,
		RefundAmount: "50.00",
		Status:       model.AfterSaleStatusCompleted,
		CreatedAt:    tomorrowRefund,
	}
	env.db.Create(&as1)

	asItem1 := model.AfterSaleItem{
		AfterSaleID:  as1.ID,
		ProductID:    int64(p1.ID),
		SKUID:        int64(sku1.ID),
		Quantity:     1,
		RefundAmount: "50.00",
		CreatedAt:    tomorrowRefund,
	}
	env.db.Create(&asItem1)

	ref1 := model.OrderRefund{
		RefundNo:     "REF_001",
		AfterSaleID:  as1.ID,
		OrderID:      order2.ID,
		PaymentID:    pay2.ID,
		MemberID:     m2.ID,
		RefundAmount: "50.00",
		Status:       model.RefundStatusSuccess,
		RefundedAt:   &tomorrowRefund,
		CreatedAt:    tomorrowRefund,
	}
	env.db.Create(&ref1)

	// 订单3：支付失败（status=30）或待支付（status=10）不应计入销售收入
	order3 := model.OrderMain{
		OrderNo:     "ORD_003",
		MemberID:    m1.ID,
		Status:      model.OrderStatusPendingPayment,
		TotalAmount: "300.00",
		PayAmount:   "300.00",
		CreatedAt:   todayPaid,
	}
	env.db.Create(&order3)
	pay3Failed := model.OrderPayment{
		PaymentNo: "PAY_003_FAIL",
		OrderID:   order3.ID,
		Channel:   "wechat",
		Amount:    "300.00",
		Status:    model.PaymentStatusFailed, // 30 支付失败
		CreatedAt: todayPaid,
	}
	env.db.Create(&pay3Failed)

	// 订单4：退款失败（status=40）不应计入退款金额
	as2 := model.AfterSale{
		AfterSaleNo:  "AS_002",
		OrderID:      order2.ID,
		MemberID:     m2.ID,
		RefundAmount: "80.00",
		Status:       model.AfterSaleStatusRefundFailed,
		CreatedAt:    todayPaid,
	}
	env.db.Create(&as2)

	refFailed := model.OrderRefund{
		RefundNo:     "REF_002_FAIL",
		AfterSaleID:  as2.ID,
		OrderID:      order2.ID,
		PaymentID:    pay2.ID,
		MemberID:     m2.ID,
		RefundAmount: "80.00",
		Status:       model.RefundStatusFailed, // 40 退款失败
		CreatedAt:    todayPaid,
	}
	if err := env.db.Create(&refFailed).Error; err != nil {
		t.Fatalf("failed to create refFailed: %v", err)
	}

	// --- 验证1：查询今天（today）的数据 ---
	// 今天应有：
	// 支付金额: 199.00 (order1) + 200.00 (order2) = 399.00
	// 退款金额: 0.00 (因为退款在明天完成，且今天退款失败的不计入)
	// 净销售额: 399.00
	// 支付订单数: 2
	// 下单订单数: 今天创建了 order2, order3 -> 2单 (order1 是昨天创建的)
	var todayOverview overview.OverviewResponse
	env.get(t, "/api/analytics/overview?time_range=today", &todayOverview)

	if todayOverview.Metrics.PayAmount != "399.00" {
		t.Errorf("today pay_amount expected 399.00, got %s", todayOverview.Metrics.PayAmount)
	}
	if todayOverview.Metrics.RefundAmount != "0.00" {
		t.Errorf("today refund_amount expected 0.00, got %s", todayOverview.Metrics.RefundAmount)
	}
	if todayOverview.Metrics.NetSalesAmount != "399.00" {
		t.Errorf("today net_sales_amount expected 399.00, got %s", todayOverview.Metrics.NetSalesAmount)
	}
	if todayOverview.Metrics.PaidOrderCount != 2 {
		t.Errorf("today paid_order_count expected 2, got %d", todayOverview.Metrics.PaidOrderCount)
	}

	// 验证今天商品排行
	if len(todayOverview.TopProducts) == 0 {
		t.Fatalf("expected top products for today, got 0")
	}
	if todayOverview.TopProducts[0].SalesAmount != "399.00" || todayOverview.TopProducts[0].Quantity != 4 {
		t.Errorf("expected 4 quantity and 399.00 sales, got %+v", todayOverview.TopProducts[0])
	}

	// --- 验证2：查询昨天（yesterday）的数据 ---
	// 昨天创建了 order1 (199.00)，但昨天未支付（今天才支付）
	// 因此昨天的销售收入必须为 0.00！下单订单数为 1
	var yesterdayOverview overview.OverviewResponse
	env.get(t, "/api/analytics/overview?time_range=yesterday", &yesterdayOverview)

	if yesterdayOverview.Metrics.PayAmount != "0.00" {
		t.Errorf("yesterday pay_amount must be 0.00, got %s", yesterdayOverview.Metrics.PayAmount)
	}
	if yesterdayOverview.Metrics.PaidOrderCount != 0 {
		t.Errorf("yesterday paid_order_count must be 0, got %d", yesterdayOverview.Metrics.PaidOrderCount)
	}

	// 验证交易分析中的下单订单数统计
	var yesterdayTrade trade.TradeResponse
	env.get(t, "/api/analytics/trade?time_range=yesterday", &yesterdayTrade)
	if yesterdayTrade.Metrics.OrderCount != 1 {
		t.Errorf("yesterday order_count must be 1, got %d", yesterdayTrade.Metrics.OrderCount)
	}
	if yesterdayTrade.Metrics.PaidOrderCount != 0 {
		t.Errorf("yesterday paid_order_count must be 0, got %d", yesterdayTrade.Metrics.PaidOrderCount)
	}

	// --- 验证3：查询自定义时间跨度覆盖明天（包含退款）---
	startStr := todayStart.Format("2006-01-02 00:00:00")
	endStr := todayStart.AddDate(0, 0, 2).Format("2006-01-02 00:00:00")
	var customOverview overview.OverviewResponse
	env.get(t, "/api/analytics/overview?time_range=custom&start_time="+startStr+"&end_time="+endStr, &customOverview)

	// 在这个跨度中：
	// 支付金额: 399.00
	// 退款金额: 50.00
	// 净销售额: 399.00 - 50.00 = 349.00
	if customOverview.Metrics.PayAmount != "399.00" {
		t.Errorf("custom pay_amount expected 399.00, got %s", customOverview.Metrics.PayAmount)
	}
	if customOverview.Metrics.RefundAmount != "50.00" {
		t.Errorf("custom refund_amount expected 50.00, got %s", customOverview.Metrics.RefundAmount)
	}
	if customOverview.Metrics.NetSalesAmount != "349.00" {
		t.Errorf("custom net_sales_amount expected 349.00, got %s", customOverview.Metrics.NetSalesAmount)
	}

	// --- 验证4：资金流水统一接口 ---
	var flowResp flow.FlowListResponse
	env.post(t, "/api/analytics/flows/list", flow.FlowListRequest{
		Page:     1,
		PageSize: 10,
		FlowType: "all",
	}, &flowResp)

	// 总流水应有 5 条记录 (pay1, pay2, ref1, pay3Failed, refFailed)
	if flowResp.Total != 5 {
		t.Errorf("expected 5 total flows, got %d", flowResp.Total)
	}
	// 筛选仅成功流水的收入与支出汇总
	// 收入总额: 199.00 + 200.00 = 399.00
	// 支出总额: 50.00
	// 净流入: 349.00
	if flowResp.TotalIncomeAmount != "399.00" {
		t.Errorf("expected 399.00 total income, got %s", flowResp.TotalIncomeAmount)
	}
	if flowResp.TotalExpenseAmount != "50.00" {
		t.Errorf("expected 50.00 total expense, got %s", flowResp.TotalExpenseAmount)
	}
	if flowResp.NetFlowAmount != "349.00" {
		t.Errorf("expected 349.00 net flow, got %s", flowResp.NetFlowAmount)
	}

	// 单独查询 refund
	var refundOnlyFlow flow.FlowListResponse
	env.post(t, "/api/analytics/flows/list", flow.FlowListRequest{
		Page:     1,
		PageSize: 10,
		FlowType: "refund",
	}, &refundOnlyFlow)
	if refundOnlyFlow.Total != 2 {
		t.Errorf("expected 2 refund flows, got %d", refundOnlyFlow.Total)
	}

	// 过滤流水渠道: wechat (pay1 + pay3Failed) -> 2 条
	var wechatFlow flow.FlowListResponse
	env.post(t, "/api/analytics/flows/list", flow.FlowListRequest{
		Page:     1,
		PageSize: 10,
		Channel:  "wechat",
	}, &wechatFlow)
	if wechatFlow.Total != 2 {
		t.Errorf("expected 2 wechat flows, got %d", wechatFlow.Total)
	}

	// --- 验证5：商品分析排行榜与列表 ---
	var prodRankings product.ProductRankingsResponse
	env.get(t, "/api/analytics/product/rankings?time_range=today", &prodRankings)
	if len(prodRankings.SalesAmountRankings) == 0 {
		t.Fatalf("expected sales amount rankings not empty")
	}
	if prodRankings.SalesAmountRankings[0].SalesAmount != "399.00" {
		t.Errorf("expected 399.00 sales amount for top product, got %s", prodRankings.SalesAmountRankings[0].SalesAmount)
	}
	if prodRankings.SalesAmountRankings[0].SalesQuantity != 4 {
		t.Errorf("expected 4 sales quantity, got %d", prodRankings.SalesAmountRankings[0].SalesQuantity)
	}

	// 商品分析列表 (SKU 维度)
	var skuList []product.ProductAnalysisRow
	env.post(t, "/api/analytics/product/list", product.ProductListRequest{
		Dimension: "sku",
		TimeRange: "today",
		Page:      1,
		PageSize:  10,
	}, &skuList)
	if len(skuList) == 0 {
		t.Fatalf("expected sku list not empty")
	}
	if skuList[0].SalesAmount != "399.00" {
		t.Errorf("expected 399.00 for sku row, got %s", skuList[0].SalesAmount)
	}

	// --- 验证6：会员分析 ---
	var memberData member.MemberResponse
	env.get(t, "/api/analytics/member?time_range=today", &memberData)
	if memberData.Metrics.TotalMemberCount != 2 {
		t.Errorf("expected 2 total members, got %d", memberData.Metrics.TotalMemberCount)
	}
	if memberData.Metrics.PaidMemberCount != 2 {
		t.Errorf("expected 2 paid members today, got %d", memberData.Metrics.PaidMemberCount)
	}

	// --- 验证7：异常时间参数校验 ---
	req, _ := http.NewRequest(http.MethodGet, "/api/analytics/overview?time_range=custom&start_time=2026-09-20&end_time=2026-09-10", nil)
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	var errResp response.Response[json.RawMessage]
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Code != 0 {
		t.Errorf("expected code 0 for reversed date range, got %d", errResp.Code)
	}
}

func TestAnalyticsPermissions(t *testing.T) {
	env := setupTestEnv(t)

	// 1. 无 token 访问 -> Code: -1
	req, _ := http.NewRequest(http.MethodGet, "/api/analytics/overview", nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	var resp response.Response[json.RawMessage]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != -1 {
		t.Errorf("expected code -1 for unauthenticated request, got %d", resp.Code)
	}

	// 2. 普通无权限用户 -> Code: 0, Message: "permission denied"
	normalUser := model.SystemUser{
		Username: "operator",
		Nickname: "普通运营",
		Status:   1,
	}
	env.db.Create(&normalUser)
	userClaims := jwt.RegisteredClaims{
		Issuer:    "mall-admin-core",
		Subject:   strconv.FormatInt(normalUser.ID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	normalToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims).SignedString([]byte(testJWTSecret))

	req, _ = http.NewRequest(http.MethodGet, "/api/analytics/overview", nil)
	req.Header.Set("Authorization", "Bearer "+normalToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 || resp.Message != "permission denied" {
		t.Errorf("expected permission denied, got code %d, msg: %s", resp.Code, resp.Message)
	}
}

