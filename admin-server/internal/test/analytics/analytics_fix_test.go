package analytics_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mall/admin_core/internal/analytics"
	"mall/admin_core/internal/analytics/flow"
	"mall/admin_core/internal/analytics/overview"
	"mall/admin_core/internal/analytics/product"
	"mall/admin_core/internal/analytics/trade"
	"mall/admin_core/internal/dashboard"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/response"
)

// 1. 资金流水按全部流水查询时，忽略 status 参数，能同时查出支付成功(20)和退款处理中(20)两条不同含义记录，不冲突。
func TestFlowAllIgnoresStatus(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now()

	// Insert 1 payment with status 20 (success)
	env.db.Create(&model.OrderPayment{
		PaymentNo: "PAY-1",
		OrderID:   1,
		Amount:    "100.00",
		Status:    model.PaymentStatusSuccess, // 20
		PaidAt:    &now,
		CreatedAt: now,
	})

	// Insert 1 refund with status 20 (processing)
	env.db.Create(&model.OrderRefund{
		RefundNo:     "REF-1",
		OrderID:      1,
		RefundAmount: "30.00",
		Status:       model.RefundStatusProcessing, // 20
		CreatedAt:    now,
	})

	// Query "all" flows with status=20
	statusVal := int8(20)
	body, _ := json.Marshal(flow.FlowListRequest{
		FlowType: "all",
		Status:   &statusVal,
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/analytics/flows/list", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var resp response.Response[flow.FlowListResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.Total != 2 {
		t.Fatalf("expected 2 flows in 'all' tab when status=20 is passed, got %d", resp.Data.Total)
	}
}

// 2. 资金流水按单类型（收入流水）查询时，status=20 仅查出支付成功的收款记录。
func TestFlowIncomeAppliesStatus(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now()

	env.db.Create(&model.OrderPayment{
		PaymentNo: "PAY-SUCC",
		OrderID:   1,
		Amount:    "100.00",
		Status:    model.PaymentStatusSuccess, // 20
		PaidAt:    &now,
		CreatedAt: now,
	})
	env.db.Create(&model.OrderPayment{
		PaymentNo: "PAY-PEND",
		OrderID:   2,
		Amount:    "50.00",
		Status:    model.PaymentStatusPending, // 10
		CreatedAt: now,
	})

	statusVal := int8(20)
	body, _ := json.Marshal(flow.FlowListRequest{
		FlowType: "payment",
		Status:   &statusVal,
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/analytics/flows/list", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var resp response.Response[flow.FlowListResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.Total != 1 {
		t.Fatalf("expected 1 flow, got %d", resp.Data.Total)
	}
	if resp.Data.Items[0].BizNo != "PAY-SUCC" {
		t.Errorf("expected PAY-SUCC, got %s", resp.Data.Items[0].BizNo)
	}
}

// 3. 资金流水按单类型（退款流水）查询时，status=30 仅查出退款成功的退款记录。
func TestFlowRefundAppliesStatus(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now()

	env.db.Create(&model.OrderRefund{
		RefundNo:     "REF-SUCC",
		OrderID:      1,
		RefundAmount: "40.00",
		Status:       model.RefundStatusSuccess, // 30
		RefundedAt:   &now,
		CreatedAt:    now,
	})
	env.db.Create(&model.OrderRefund{
		RefundNo:     "REF-PROC",
		OrderID:      2,
		RefundAmount: "20.00",
		Status:       model.RefundStatusProcessing, // 20
		CreatedAt:    now,
	})

	statusVal := int8(30)
	body, _ := json.Marshal(flow.FlowListRequest{
		FlowType: "refund",
		Status:   &statusVal,
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/analytics/flows/list", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var resp response.Response[flow.FlowListResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.Total != 1 {
		t.Fatalf("expected 1 refund flow, got %d", resp.Data.Total)
	}
	if resp.Data.Items[0].BizNo != "REF-SUCC" {
		t.Errorf("expected REF-SUCC, got %s", resp.Data.Items[0].BizNo)
	}
}

// 4. 跨日支付订单（昨天创建、今天支付）：
// Dashboard 今日销售额 = 该订单金额, 今日已支付订单数 = 1, 今日下单量 = 0
// Analytics 今日销售额 = 该订单金额
// 两者今日数据完全一致。
func TestCrossDayPaymentAlignment(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	todayPaid := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())

	// Created yesterday, paid today
	order := model.OrderMain{
		OrderNo:     "ORD-YEST-PAY-TODAY",
		MemberID:    1,
		Status:      model.OrderStatusPendingShipment,
		TotalAmount: "128.00",
		PayAmount:   "128.00",
		CreatedAt:   yesterday,
	}
	env.db.Create(&order)

	payment := model.OrderPayment{
		PaymentNo: "PAY-TODAY-1",
		OrderID:   order.ID,
		Amount:    "128.00",
		Status:    model.PaymentStatusSuccess,
		PaidAt:    &todayPaid,
		CreatedAt: todayPaid,
	}
	env.db.Create(&payment)

	// Check Dashboard Overview
	dashService := dashboard.NewService(env.db)
	dashOverview, err := dashService.Overview(context.Background())
	if err != nil {
		t.Fatalf("dashboard overview failed: %v", err)
	}

	if dashOverview.TodayOrderCount != 0 {
		t.Errorf("expected Dashboard TodayOrderCount 0, got %d", dashOverview.TodayOrderCount)
	}
	if dashOverview.TodayPaidOrderCount != 1 {
		t.Errorf("expected Dashboard TodayPaidOrderCount 1, got %d", dashOverview.TodayPaidOrderCount)
	}
	if dashOverview.TodaySalesAmount != "128.00" {
		t.Errorf("expected Dashboard TodaySalesAmount 128.00, got %s", dashOverview.TodaySalesAmount)
	}

	// Check Analytics Overview
	req, _ := http.NewRequest(http.MethodGet, "/api/analytics/overview?time_range=today", nil)
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var anResp response.Response[overview.OverviewResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &anResp)

	if anResp.Data.Metrics.PayAmount != "128.00" {
		t.Errorf("expected Analytics PayAmount 128.00, got %s", anResp.Data.Metrics.PayAmount)
	}
	if anResp.Data.Metrics.PaidOrderCount != 1 {
		t.Errorf("expected Analytics PaidOrderCount 1, got %d", anResp.Data.Metrics.PaidOrderCount)
	}

	// Ensure Dashboard and Analytics match exactly
	if dashOverview.TodaySalesAmount != anResp.Data.Metrics.PayAmount {
		t.Errorf("Dashboard (%s) and Analytics (%s) TodaySalesAmount mismatch", dashOverview.TodaySalesAmount, anResp.Data.Metrics.PayAmount)
	}
	if dashOverview.TodayPaidOrderCount != anResp.Data.Metrics.PaidOrderCount {
		t.Errorf("Dashboard (%d) and Analytics (%d) TodayPaidOrderCount mismatch", dashOverview.TodayPaidOrderCount, anResp.Data.Metrics.PaidOrderCount)
	}
}

// 5. 取消订单时间归属：
// 9 月 1 日创建订单、9 月 5 日取消：
// 查询 9 月 1 日交易概览：CancelledOrderCount = 0
// 查询 9 月 5 日交易概览：CancelledOrderCount = 1
func TestCancelledOrderAttribution(t *testing.T) {
	env := setupTestEnv(t)
	createdTime := time.Date(2026, 9, 1, 10, 0, 0, 0, time.Local)
	cancelledTime := time.Date(2026, 9, 5, 15, 0, 0, 0, time.Local)

	env.db.Create(&model.OrderMain{
		OrderNo:     "ORD-CANCEL",
		MemberID:    1,
		Status:      model.OrderStatusCancelled,
		TotalAmount: "50.00",
		PayAmount:   "50.00",
		CancelledAt: &cancelledTime,
		CreatedAt:   createdTime,
	})

	// Query 2026-09-01 (should have 0 cancelled)
	reqD1, _ := http.NewRequest(http.MethodGet, "/api/analytics/trade?time_range=custom&start_time=2026-09-01 00:00:00&end_time=2026-09-01 23:59:59", nil)
	reqD1.Header.Set("Authorization", "Bearer "+env.adminToken)
	wD1 := httptest.NewRecorder()
	env.router.ServeHTTP(wD1, reqD1)

	var respD1 response.Response[trade.TradeResponse]
	_ = json.Unmarshal(wD1.Body.Bytes(), &respD1)
	if respD1.Data.Metrics.CancelledOrderCount != 0 {
		t.Errorf("expected 0 cancelled orders on 2026-09-01, got %d", respD1.Data.Metrics.CancelledOrderCount)
	}

	// Query 2026-09-05 (should have 1 cancelled)
	reqD5, _ := http.NewRequest(http.MethodGet, "/api/analytics/trade?time_range=custom&start_time=2026-09-05 00:00:00&end_time=2026-09-05 23:59:59", nil)
	reqD5.Header.Set("Authorization", "Bearer "+env.adminToken)
	wD5 := httptest.NewRecorder()
	env.router.ServeHTTP(wD5, reqD5)

	var respD5 response.Response[trade.TradeResponse]
	_ = json.Unmarshal(wD5.Body.Bytes(), &respD5)
	if respD5.Data.Metrics.CancelledOrderCount != 1 {
		t.Errorf("expected 1 cancelled order on 2026-09-05, got %d", respD5.Data.Metrics.CancelledOrderCount)
	}
}

// 6. 商品改名后销售聚合：
// 同一商品 ID，曾用名 A（卖出 2 件，20 元），现用名 B（卖出 3 件，30 元）：
// 商品排行榜与商品分析列表中该商品合并为 1 条记录，总销量 = 5 件，总销售额 = 50 元。
func TestProductRenameAggregation(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now()

	o1 := model.OrderMain{OrderNo: "O1", MemberID: 1, Status: model.OrderStatusCompleted, TotalAmount: "20.00", PayAmount: "20.00", CreatedAt: now}
	o2 := model.OrderMain{OrderNo: "O2", MemberID: 2, Status: model.OrderStatusCompleted, TotalAmount: "30.00", PayAmount: "30.00", CreatedAt: now}
	env.db.Create(&o1)
	env.db.Create(&o2)

	env.db.Create(&model.OrderPayment{PaymentNo: "P1", OrderID: o1.ID, Amount: "20.00", Status: model.PaymentStatusSuccess, PaidAt: &now, CreatedAt: now})
	env.db.Create(&model.OrderPayment{PaymentNo: "P2", OrderID: o2.ID, Amount: "30.00", Status: model.PaymentStatusSuccess, PaidAt: &now, CreatedAt: now})

	// Same product ID 99, different historic snapshot names
	env.db.Create(&model.OrderItem{OrderID: o1.ID, ProductID: 99, ProductName: "旧名称A", Quantity: 2, TotalAmount: "20.00"})
	env.db.Create(&model.OrderItem{OrderID: o2.ID, ProductID: 99, ProductName: "现用名B", Quantity: 3, TotalAmount: "30.00"})

	// Check Product Rankings
	reqRank, _ := http.NewRequest(http.MethodGet, "/api/analytics/product/rankings?time_range=today", nil)
	reqRank.Header.Set("Authorization", "Bearer "+env.adminToken)
	wRank := httptest.NewRecorder()
	env.router.ServeHTTP(wRank, reqRank)

	var respRank response.Response[product.ProductRankingsResponse]
	_ = json.Unmarshal(wRank.Body.Bytes(), &respRank)

	if len(respRank.Data.SalesAmountRankings) != 1 {
		t.Fatalf("expected 1 aggregated ranking item for product 99, got %d", len(respRank.Data.SalesAmountRankings))
	}
	if respRank.Data.SalesAmountRankings[0].SalesQuantity != 5 {
		t.Errorf("expected SalesQuantity 5, got %d", respRank.Data.SalesAmountRankings[0].SalesQuantity)
	}
	if respRank.Data.SalesAmountRankings[0].SalesAmount != "50.00" {
		t.Errorf("expected SalesAmount 50.00, got %s", respRank.Data.SalesAmountRankings[0].SalesAmount)
	}

	// Check Product List
	body, _ := json.Marshal(product.ProductListRequest{TimeRange: "today"})
	reqList, _ := http.NewRequest(http.MethodPost, "/api/analytics/product/list", bytes.NewBuffer(body))
	reqList.Header.Set("Authorization", "Bearer "+env.adminToken)
	reqList.Header.Set("Content-Type", "application/json")
	wList := httptest.NewRecorder()
	env.router.ServeHTTP(wList, reqList)

	var respList response.Response[[]product.ProductAnalysisRow]
	_ = json.Unmarshal(wList.Body.Bytes(), &respList)

	if len(respList.Data) != 1 {
		t.Fatalf("expected 1 aggregated row for product 99, got %d", len(respList.Data))
	}
	if respList.Data[0].SalesQuantity != 5 {
		t.Errorf("expected SalesQuantity 5, got %d", respList.Data[0].SalesQuantity)
	}
	if respList.Data[0].SalesAmount != "50.00" {
		t.Errorf("expected SalesAmount 50.00, got %s", respList.Data[0].SalesAmount)
	}
}

// 7. 多笔支付流水防重乘算：
// 订单包含 2 件商品，金额 200 元。该订单存在 2 笔成功支付流水（如拆单或重复记录）：
// 商品分析中该商品销售件数仍为 2，销售额仍为 200 元（不被翻倍为 4 件、400 元）。
// 交易概览中该订单的优惠金额与运费金额不被重复乘算。
func TestMultiplePaymentsNoDuplicateMultiplication(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now()

	order := model.OrderMain{
		OrderNo:        "ORD-SPLIT-PAY",
		MemberID:       1,
		Status:         model.OrderStatusCompleted,
		TotalAmount:    "210.00",
		DiscountAmount: "20.00",
		FreightAmount:  "10.00",
		PayAmount:      "200.00",
		CreatedAt:      now,
	}
	env.db.Create(&order)

	// 2 payment records for same order
	env.db.Create(&model.OrderPayment{PaymentNo: "PAY-SPLIT-1", OrderID: order.ID, Amount: "100.00", Status: model.PaymentStatusSuccess, PaidAt: &now, CreatedAt: now})
	env.db.Create(&model.OrderPayment{PaymentNo: "PAY-SPLIT-2", OrderID: order.ID, Amount: "100.00", Status: model.PaymentStatusSuccess, PaidAt: &now, CreatedAt: now})

	// Order item
	env.db.Create(&model.OrderItem{OrderID: order.ID, ProductID: 101, ProductName: "商品101", Quantity: 2, TotalAmount: "200.00"})

	// Check Product List
	body, _ := json.Marshal(product.ProductListRequest{TimeRange: "today"})
	reqList, _ := http.NewRequest(http.MethodPost, "/api/analytics/product/list", bytes.NewBuffer(body))
	reqList.Header.Set("Authorization", "Bearer "+env.adminToken)
	reqList.Header.Set("Content-Type", "application/json")
	wList := httptest.NewRecorder()
	env.router.ServeHTTP(wList, reqList)

	var respList response.Response[[]product.ProductAnalysisRow]
	_ = json.Unmarshal(wList.Body.Bytes(), &respList)

	if len(respList.Data) != 1 {
		t.Fatalf("expected 1 product row, got %d", len(respList.Data))
	}
	if respList.Data[0].SalesQuantity != 2 {
		t.Errorf("expected SalesQuantity 2, got %d (duplicated!)", respList.Data[0].SalesQuantity)
	}
	if respList.Data[0].SalesAmount != "200.00" {
		t.Errorf("expected SalesAmount 200.00, got %s (duplicated!)", respList.Data[0].SalesAmount)
	}

	// Check Trade Overview discount and freight
	reqTrade, _ := http.NewRequest(http.MethodGet, "/api/analytics/trade?time_range=today", nil)
	reqTrade.Header.Set("Authorization", "Bearer "+env.adminToken)
	wTrade := httptest.NewRecorder()
	env.router.ServeHTTP(wTrade, reqTrade)

	var respTrade response.Response[trade.TradeResponse]
	_ = json.Unmarshal(wTrade.Body.Bytes(), &respTrade)

	if respTrade.Data.Metrics.DiscountAmount != "20.00" {
		t.Errorf("expected DiscountAmount 20.00, got %s (duplicated!)", respTrade.Data.Metrics.DiscountAmount)
	}
	if respTrade.Data.Metrics.FreightAmount != "10.00" {
		t.Errorf("expected FreightAmount 10.00, got %s (duplicated!)", respTrade.Data.Metrics.FreightAmount)
	}
}

// 8. 零基数增长率：
func TestGrowthRateMath(t *testing.T) {
	if r := analytics.CalcGrowthRate(10000, 0); r != "--" {
		t.Errorf("expected '--', got %q", r)
	}
	if r := analytics.CalcGrowthRate(10000, -500); r != "--" {
		t.Errorf("expected '--', got %q", r)
	}
	if r := analytics.CalcGrowthRate(10000, 10000); r != "0.00%" {
		t.Errorf("expected '0.00%%', got %q", r)
	}
	if r := analytics.CalcGrowthRate(15000, 10000); r != "+50.00%" {
		t.Errorf("expected '+50.00%%', got %q", r)
	}
	if r := analytics.CalcGrowthRate(5000, 10000); r != "-50.00%" {
		t.Errorf("expected '-50.00%%', got %q", r)
	}
}

// 9. 退款率极值：
func TestRefundRateExtremes(t *testing.T) {
	if r := analytics.CalcRefundRate(5000, 0); r != "--" {
		t.Errorf("expected '--', got %q", r)
	}
	if r := analytics.CalcRefundRate(0, 10000); r != "0.00%" {
		t.Errorf("expected '0.00%%', got %q", r)
	}
	if r := analytics.CalcRefundRate(3000, 10000); r != "30.00%" {
		t.Errorf("expected '30.00%%', got %q", r)
	}
	if r := analytics.CalcRefundRate(15000, 10000); r != "150.00%" {
		t.Errorf("expected '150.00%%', got %q", r)
	}
}

// 10. 本月环比时间截断：
func TestTimeRangeThisMonthTruncation(t *testing.T) {
	startTime, _, prevStartTime, prevEndTime, _, err := analytics.ParseTimeRange(analytics.TimeRangeThisMonth, "", "")
	if err != nil {
		t.Fatalf("ParseTimeRange failed: %v", err)
	}

	// prevEndTime must never exceed startTime (the beginning of current month)
	if prevEndTime.After(startTime) {
		t.Errorf("prevEndTime %v must not be after startTime %v", prevEndTime, startTime)
	}
	if prevStartTime.Month() == startTime.Month() {
		t.Errorf("prevStartTime must be in previous month")
	}
	if prevEndTime.Month() != prevStartTime.Month() && !prevEndTime.Equal(startTime) {
		t.Errorf("prevEndTime %v should not cross into subsequent months unexpectedly", prevEndTime)
	}
}

// 11. 仅有退款商品列表展示（无 N+1 确认）：
func TestProductOnlyRefundNoSalesNoNPlus1(t *testing.T) {
	env := setupTestEnv(t)
	now := time.Now()

	// Product with no sales, only refund
	as := model.AfterSale{
		AfterSaleNo: "AS-ONLY-REF",
		OrderID:     999,
		MemberID:    1,
		Status:      model.AfterSaleStatusCompleted,
		CreatedAt:   now,
	}
	env.db.Create(&as)

	env.db.Create(&model.AfterSaleItem{
		AfterSaleID:  as.ID,
		OrderItemID:  1,
		ProductID:    888,
		SKUID:        0,
		ProductName:  "仅退款商品快照名",
		Quantity:     1,
		RefundAmount: "80.00",
		CreatedAt:    now,
	})

	env.db.Create(&model.OrderRefund{
		RefundNo:     "REF-888",
		AfterSaleID:  as.ID,
		OrderID:      999,
		RefundAmount: "80.00",
		Status:       model.RefundStatusSuccess,
		RefundedAt:   &now,
		CreatedAt:    now,
	})

	body, _ := json.Marshal(product.ProductListRequest{TimeRange: "today"})
	req, _ := http.NewRequest(http.MethodPost, "/api/analytics/product/list", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var resp response.Response[[]product.ProductAnalysisRow]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 product row, got %d", len(resp.Data))
	}
	item := resp.Data[0]
	if item.ProductID != 888 {
		t.Errorf("expected product ID 888, got %d", item.ProductID)
	}
	if item.ProductName != "仅退款商品快照名" {
		t.Errorf("expected ProductName '仅退款商品快照名', got %q", item.ProductName)
	}
	if item.SalesQuantity != 0 {
		t.Errorf("expected SalesQuantity 0, got %d", item.SalesQuantity)
	}
	if item.SalesAmount != "0.00" {
		t.Errorf("expected SalesAmount '0.00', got %q", item.SalesAmount)
	}
	if item.RefundQuantity != 1 {
		t.Errorf("expected RefundQuantity 1, got %d", item.RefundQuantity)
	}
	if item.RefundAmount != "80.00" {
		t.Errorf("expected RefundAmount '80.00', got %q", item.RefundAmount)
	}
	if item.RefundRate != "--" {
		t.Errorf("expected RefundRate '--', got %q", item.RefundRate)
	}
}

// 12. 自定义时间范围边界：
// 自定义选择 2026-09-01 至 2026-09-02：
// 2026-09-02 23:59:59 发生的数据必须被包含，2026-09-03 00:00:00 发生的数据必须被排除。
func TestCustomTimeRangeBoundaries(t *testing.T) {
	env := setupTestEnv(t)
	tInclusive := time.Date(2026, 9, 2, 23, 59, 59, 0, time.Local)
	tExclusive := time.Date(2026, 9, 3, 0, 0, 0, 0, time.Local)

	env.db.Create(&model.OrderPayment{
		PaymentNo: "PAY-INCL",
		OrderID:   1,
		Amount:    "10.00",
		Status:    model.PaymentStatusSuccess,
		PaidAt:    &tInclusive,
		CreatedAt: tInclusive,
	})
	env.db.Create(&model.OrderPayment{
		PaymentNo: "PAY-EXCL",
		OrderID:   2,
		Amount:    "20.00",
		Status:    model.PaymentStatusSuccess,
		PaidAt:    &tExclusive,
		CreatedAt: tExclusive,
	})

	// Query FlowList with end_time: "2026-09-02 23:59:59"
	body, _ := json.Marshal(flow.FlowListRequest{
		FlowType:  "payment",
		StartTime: "2026-09-01 00:00:00",
		EndTime:   "2026-09-02 23:59:59",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/analytics/flows/list", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+env.adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var resp response.Response[flow.FlowListResponse]
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.Total != 1 {
		t.Fatalf("expected exactly 1 flow within boundary [2026-09-01 00:00:00, 2026-09-02 23:59:59], got %d", resp.Data.Total)
	}
	if resp.Data.Items[0].BizNo != "PAY-INCL" {
		t.Errorf("expected PAY-INCL, got %s", resp.Data.Items[0].BizNo)
	}
}
