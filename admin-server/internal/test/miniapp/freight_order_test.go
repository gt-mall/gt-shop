package miniapp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	miniapporder "mall/admin_core/internal/miniapp/order"
	"mall/admin_core/internal/model"
	"mall/admin_core/pkg/response"
	"mall/admin_core/pkg/util"
	)

// Scenario 17: 多个不同商品合并结算，运费为各自运费累加，金额计算精确到分，无浮点误差
func TestFreightScenario17_MultiProductFreightSum(t *testing.T) {
	db, router, token, memberID, _, _ := setupOrderTestApp(t)

	addr := model.MemberAddress{
		MemberID:     memberID,
		Name:         "张三",
		Mobile:       "13800138000",
		ProvinceCode: "110000",
		Province:     "北京市",
		CityCode:     "110100",
		City:         "北京市",
		DistrictCode: "110101",
		District:     "东城区",
		Address:      "长安街 1 号",
		IsDefault:    1,
	}
	db.Create(&addr)

	// Prod 1: 固定运费 12.34
	p1 := model.Product{Name: "商品1", CategoryID: 1, Status: 1, FreightType: 2, FreightFee: "12.34"}
	db.Create(&p1)
	sku1 := model.ProductSKU{ProductID: p1.ID, Price: "10.01", Status: 1, SpecHash: "h1"}
	db.Create(&sku1)
	db.Create(&model.ProductSkuInventory{ProductID: p1.ID, SKUID: sku1.ID, Stock: 10, Version: 1})

	// Prod 2: 固定运费 15.67
	p2 := model.Product{Name: "商品2", CategoryID: 1, Status: 1, FreightType: 2, FreightFee: "15.67"}
	db.Create(&p2)
	sku2 := model.ProductSKU{ProductID: p2.ID, Price: "20.02", Status: 1, SpecHash: "h2"}
	db.Create(&sku2)
	db.Create(&model.ProductSkuInventory{ProductID: p2.ID, SKUID: sku2.ID, Stock: 10, Version: 1})

	// Prod 3: 包邮
	p3 := model.Product{Name: "商品3", CategoryID: 1, Status: 1, FreightType: 1, FreightFee: "0.00"}
	db.Create(&p3)
	sku3 := model.ProductSKU{ProductID: p3.ID, Price: "30.03", Status: 1, SpecHash: "h3"}
	db.Create(&sku3)
	db.Create(&model.ProductSkuInventory{ProductID: p3.ID, SKUID: sku3.ID, Stock: 10, Version: 1})

	// 预期总运费: 12.34 + 15.67 + 0.00 = 28.01
	// 预期商品总额: 10.01 + 20.02 + 30.03 = 60.06
	// 预期实付: 60.06 + 28.01 = 88.07

	// 1. 测试 Preview
	previewBody, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
		AddressID: &addr.ID,
		Items: []miniapporder.CheckoutItemRequest{
			{SKUID: sku1.ID, Quantity: 1},
			{SKUID: sku2.ID, Quantity: 1},
			{SKUID: sku3.ID, Quantity: 1},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/checkout/preview", bytes.NewReader(previewBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("preview status: %d", w.Code)
	}
	var previewResp response.Response[miniapporder.CheckoutPreviewResponse]
	json.Unmarshal(w.Body.Bytes(), &previewResp)
	if previewResp.Data.FreightAmount != "28.01" {
		t.Fatalf("expected freight 28.01, got %s", previewResp.Data.FreightAmount)
	}
	if previewResp.Data.TotalAmount != "60.06" {
		t.Fatalf("expected total 60.06, got %s", previewResp.Data.TotalAmount)
	}
	if previewResp.Data.PayAmount != "88.07" {
		t.Fatalf("expected pay 88.07, got %s", previewResp.Data.PayAmount)
	}

	// 2. 测试 Create Order
	createBody, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req_freight_17",
		AddressID: addr.ID,
		Items: []miniapporder.CheckoutItemRequest{
			{SKUID: sku1.ID, Quantity: 1},
			{SKUID: sku2.ID, Quantity: 1},
			{SKUID: sku3.ID, Quantity: 1},
		},
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(createBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("create order status: %d", w2.Code)
	}
	var createResp response.Response[miniapporder.OrderDetailResponse]
	json.Unmarshal(w2.Body.Bytes(), &createResp)
	if createResp.Code != 1 {
		t.Fatalf("create order failed: %s", createResp.Message)
	}

	var orderMain model.OrderMain
	db.Where("id = ?", createResp.Data.ID).First(&orderMain)
	if orderMain.FreightAmount != "28.01" {
		t.Fatalf("expected order freight 28.01, got %s", orderMain.FreightAmount)
	}
	if orderMain.PayAmount != "88.07" {
		t.Fatalf("expected order pay 88.07, got %s", orderMain.PayAmount)
	}
}

// Scenario 18: 同一个商品的多个 SKU 合并结算，运费只计算一次，不因购买数量变化而重复计算
func TestFreightScenario18_SameProductMultipleSKUs(t *testing.T) {
	db, router, token, memberID, _, _ := setupOrderTestApp(t)

	addr := model.MemberAddress{
		MemberID:     memberID,
		Name:         "张三",
		Mobile:       "13800138000",
		ProvinceCode: "110000",
		Province:     "北京市",
		CityCode:     "110100",
		City:         "北京市",
		DistrictCode: "110101",
		District:     "东城区",
		Address:      "长安街 1 号",
		IsDefault:    1,
	}
	db.Create(&addr)

	// 同一商品，固定运费 10.00
	p := model.Product{Name: "同商品多SKU", CategoryID: 1, Status: 1, FreightType: 2, FreightFee: "10.00"}
	db.Create(&p)

	sku1 := model.ProductSKU{ProductID: p.ID, Price: "50.00", Status: 1, SpecHash: "s1"}
	db.Create(&sku1)
	db.Create(&model.ProductSkuInventory{ProductID: p.ID, SKUID: sku1.ID, Stock: 20, Version: 1})

	sku2 := model.ProductSKU{ProductID: p.ID, Price: "80.00", Status: 1, SpecHash: "s2"}
	db.Create(&sku2)
	db.Create(&model.ProductSkuInventory{ProductID: p.ID, SKUID: sku2.ID, Stock: 20, Version: 1})

	// 购买 SKU1 * 3件，SKU2 * 5件 -> 运费仍然仅收一次 10.00
	previewBody, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
		AddressID: &addr.ID,
		Items: []miniapporder.CheckoutItemRequest{
			{SKUID: sku1.ID, Quantity: 3},
			{SKUID: sku2.ID, Quantity: 5},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/checkout/preview", bytes.NewReader(previewBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var previewResp response.Response[miniapporder.CheckoutPreviewResponse]
	json.Unmarshal(w.Body.Bytes(), &previewResp)
	if previewResp.Data.FreightAmount != "10.00" {
		t.Fatalf("expected freight 10.00 for same product multiple SKUs, got %s", previewResp.Data.FreightAmount)
	}

	// 验证创单也是 10.00
	createBody, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req_freight_18",
		AddressID: addr.ID,
		Items: []miniapporder.CheckoutItemRequest{
			{SKUID: sku1.ID, Quantity: 3},
			{SKUID: sku2.ID, Quantity: 5},
		},
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(createBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	var createResp response.Response[miniapporder.OrderDetailResponse]
	json.Unmarshal(w2.Body.Bytes(), &createResp)
	if createResp.Code != 1 {
		t.Fatalf("create order failed: %s", createResp.Message)
	}

	if createResp.Data.FreightAmount != "10.00" {
		t.Fatalf("expected order freight 10.00, got %s", createResp.Data.FreightAmount)
	}
	var orderMain model.OrderMain
	db.Where("id = ?", createResp.Data.ID).First(&orderMain)
	normFreight, _ := util.NormalizeDecimalYuan(orderMain.FreightAmount)
	if normFreight != "10.00" {
		t.Fatalf("expected order freight 10.00, got %s", normFreight)
	}
}

// Scenario 19: Checkout Preview 计算出的运费与结算明细，必须与实际地址完全一致
func TestFreightScenario19_PreviewMatchesAddress(t *testing.T) {
	db, router, token, memberID, _, _ := setupOrderTestApp(t)

	// 地址1: 北京
	addrBJ := model.MemberAddress{
		MemberID: memberID, Name: "北京用户", Mobile: "13800138001",
		ProvinceCode: "110000", Province: "北京市", CityCode: "110100", City: "北京市", Address: "朝阳区",
	}
	db.Create(&addrBJ)

	// 地址2: 广东广州
	addrGZ := model.MemberAddress{
		MemberID: memberID, Name: "广州用户", Mobile: "13800138002",
		ProvinceCode: "440000", Province: "广东省", CityCode: "440100", City: "广州市", Address: "天河区",
	}
	db.Create(&addrGZ)

	// 地址3: 广东深圳
	addrSZ := model.MemberAddress{
		MemberID: memberID, Name: "深圳用户", Mobile: "13800138003",
		ProvinceCode: "440000", Province: "广东省", CityCode: "440300", City: "深圳市", Address: "南山区",
	}
	db.Create(&addrSZ)

	// 运费模板: 基础 10.00, 广东附加 5.00, 深圳城市附加 8.00 (覆盖广东附加，不叠加)
	extraJSON := `[
		{"province_code":"440000","city_code":"","fee":"5.00"},
		{"province_code":"440000","city_code":"440300","fee":"8.00"}
	]`
	tmpl := model.FreightTemplate{
		Name:      "阶梯模板",
		BaseFee:   "10.00",
		ExtraFees: &extraJSON,
		Status:    1,
	}
	db.Create(&tmpl)

	p := model.Product{Name: "模板商品", CategoryID: 1, Status: 1, FreightType: 3, FreightTemplateID: &tmpl.ID}
	db.Create(&p)
	sku := model.ProductSKU{ProductID: p.ID, Price: "100.00", Status: 1, SpecHash: "sh19"}
	db.Create(&sku)
	db.Create(&model.ProductSkuInventory{ProductID: p.ID, SKUID: sku.ID, Stock: 50, Version: 1})

	checkPreview := func(addrID int64, expectedFreight string) {
		body, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
			AddressID: &addrID,
			Items:     []miniapporder.CheckoutItemRequest{{SKUID: sku.ID, Quantity: 1}},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/miniapp/checkout/preview", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp response.Response[miniapporder.CheckoutPreviewResponse]
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.FreightAmount != expectedFreight {
			t.Fatalf("address %d: expected freight %s, got %s", addrID, expectedFreight, resp.Data.FreightAmount)
		}
	}

	// 1. 北京: 10.00
	checkPreview(addrBJ.ID, "10.00")
	// 2. 广州: 10.00 + 5.00 = 15.00
	checkPreview(addrGZ.ID, "15.00")
	// 3. 深圳: 10.00 + 8.00 = 18.00
	checkPreview(addrSZ.ID, "18.00")
}

// Scenario 20: Create Order 服务端重新计算运费，客户端篡改运费参数完全无效；若包含不可配送商品，创单直接失败，不生成脏数据
func TestFreightScenario20_ServerRecalculationAndUndeliverableBlock(t *testing.T) {
	db, router, token, memberID, _, _ := setupOrderTestApp(t)

	// 新疆地址 (不配送地区)
	addrXJ := model.MemberAddress{
		MemberID: memberID, Name: "新疆用户", Mobile: "13800138099",
		ProvinceCode: "650000", Province: "新疆维吾尔自治区", CityCode: "650100", City: "乌鲁木齐市", Address: "天山区",
	}
	db.Create(&addrXJ)

	// 北京地址 (正常配送)
	addrBJ := model.MemberAddress{
		MemberID: memberID, Name: "北京用户", Mobile: "13800138001",
		ProvinceCode: "110000", Province: "北京市", CityCode: "110100", City: "北京市", Address: "朝阳区",
	}
	db.Create(&addrBJ)

	noDelJSON := `[{"province_code":"650000","city_code":""}]`
	tmpl := model.FreightTemplate{
		Name:       "新疆不配送模板",
		BaseFee:    "15.00",
		NoDelivery: &noDelJSON,
		Status:     1,
	}
	db.Create(&tmpl)

	p := model.Product{Name: "限制配送商品", CategoryID: 1, Status: 1, FreightType: 3, FreightTemplateID: &tmpl.ID}
	db.Create(&p)
	sku := model.ProductSKU{ProductID: p.ID, Price: "100.00", Status: 1, SpecHash: "sh20"}
	db.Create(&sku)
	db.Create(&model.ProductSkuInventory{ProductID: p.ID, SKUID: sku.ID, Stock: 10, Version: 1})

	// 1. 客户端篡改运费：即使客户端请求体带有自定义字段或企图绕过，创单时服务端依然严格基于模板计算为 15.00
	tamperedBody := `{"request_id":"req_tamper_1","address_id":` + fmt.Sprintf("%d", addrBJ.ID) + `,"items":[{"sku_id":` + fmt.Sprintf("%d", sku.ID) + `,"quantity":1}],"freight_amount":"0.01","pay_amount":"10.01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader([]byte(tamperedBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp response.Response[miniapporder.OrderDetailResponse]
	json.Unmarshal(w.Body.Bytes(), &createResp)
	if createResp.Code != 1 {
		t.Fatalf("create order failed: %s", createResp.Message)
	}

	if createResp.Data.FreightAmount != "15.00" || createResp.Data.PayAmount != "115.00" {
		t.Fatalf("expected server calculated freight 15.00 and pay 115.00, got freight=%s, pay=%s",
			createResp.Data.FreightAmount, createResp.Data.PayAmount)
	}
	var orderMain model.OrderMain
	db.Where("id = ?", createResp.Data.ID).First(&orderMain)
	normFreight, _ := util.NormalizeDecimalYuan(orderMain.FreightAmount)
	normPay, _ := util.NormalizeDecimalYuan(orderMain.PayAmount)
	if normFreight != "15.00" || normPay != "115.00" {
		t.Fatalf("expected server calculated freight 15.00 and pay 115.00, got freight=%s, pay=%s", normFreight, normPay)
	}

	// 2. 不可配送地址创单：直接失败，不生成脏数据
	var countBefore int64
	db.Model(&model.OrderMain{}).Count(&countBefore)

	undeliverableBody, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req_undeliverable_1",
		AddressID: addrXJ.ID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: sku.ID, Quantity: 1}},
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(undeliverableBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	var undelivResp response.Response[any]
	json.Unmarshal(w2.Body.Bytes(), &undelivResp)
	if undelivResp.Code == 1 {
		t.Fatalf("expected create order to fail for undeliverable address, but succeeded")
	}

	var countAfter int64
	db.Model(&model.OrderMain{}).Count(&countAfter)
	if countAfter != countBefore {
		t.Fatalf("expected no new order created in db, before=%d, after=%d", countBefore, countAfter)
	}

	// 检查库存是否被扣减或锁定
	var inv model.ProductSkuInventory
	db.Where("sku_id = ?", sku.ID).First(&inv)
	// 之前合法订单扣除了1件库存(10 -> 9)，未配送失败订单不应扣除或锁定库存
	if inv.Stock != 9 || inv.LockedStock != 0 {
		t.Fatalf("inventory affected by failed order: stock=%d, locked=%d", inv.Stock, inv.LockedStock)
	}
}

// Scenario 21: 历史订单的运费、应付金额、实付金额在运费模板修改或删除后保持不变
func TestFreightScenario21_HistoricalOrderImmutability(t *testing.T) {
	db, router, token, memberID, _, _ := setupOrderTestApp(t)

	addr := model.MemberAddress{
		MemberID:     memberID,
		Name:         "张三",
		Mobile:       "13800138000",
		ProvinceCode: "110000",
		Province:     "北京市",
		CityCode:     "110100",
		City:         "北京市",
		Address:      "朝阳区",
	}
	db.Create(&addr)

	tmpl := model.FreightTemplate{
		Name:    "历史测试模板",
		BaseFee: "20.00",
		Status:  1,
	}
	db.Create(&tmpl)

	p := model.Product{Name: "模板商品", CategoryID: 1, Status: 1, FreightType: 3, FreightTemplateID: &tmpl.ID}
	db.Create(&p)
	sku := model.ProductSKU{ProductID: p.ID, Price: "100.00", Status: 1, SpecHash: "sh21"}
	db.Create(&sku)
	db.Create(&model.ProductSkuInventory{ProductID: p.ID, SKUID: sku.ID, Stock: 20, Version: 1})

	// 1. 创建订单，运费应为 20.00，实付 120.00
	createBody, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req_history_21",
		AddressID: addr.ID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: sku.ID, Quantity: 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp response.Response[miniapporder.OrderDetailResponse]
	json.Unmarshal(w.Body.Bytes(), &createResp)
	if createResp.Code != 1 {
		t.Fatalf("create order failed: %s", createResp.Message)
	}
	orderID := createResp.Data.ID

	// 2. 篡改/更新运费模板：将 BaseFee 改为 999.00，或者软删除该模板
	db.Model(&model.FreightTemplate{}).Where("id = ?", tmpl.ID).Updates(map[string]any{
		"base_fee":   "999.00",
		"deleted_at": time.Now(),
	})

	// 3. 查询历史订单详情
	reqDetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/miniapp/orders/%d", orderID), nil)
	reqDetail.Header.Set("Authorization", "Bearer "+token)
	wDetail := httptest.NewRecorder()
	router.ServeHTTP(wDetail, reqDetail)

	if wDetail.Code != http.StatusOK {
		t.Fatalf("get order detail status: %d", wDetail.Code)
	}
	var detailResp response.Response[miniapporder.OrderDetailResponse]
	json.Unmarshal(wDetail.Body.Bytes(), &detailResp)

	if detailResp.Data.FreightAmount != "20.00" {
		t.Fatalf("historical order freight changed! expected 20.00, got %s", detailResp.Data.FreightAmount)
	}
	if detailResp.Data.TotalAmount != "100.00" {
		t.Fatalf("historical order total changed! expected 100.00, got %s", detailResp.Data.TotalAmount)
	}
	if detailResp.Data.PayAmount != "120.00" {
		t.Fatalf("historical order pay changed! expected 120.00, got %s", detailResp.Data.PayAmount)
	}

	// 4. 再次直接检查数据库中的 OrderMain 字段
	var savedOrder model.OrderMain
	db.Where("id = ?", orderID).First(&savedOrder)
	normFreight, _ := util.NormalizeDecimalYuan(savedOrder.FreightAmount)
	normPay, _ := util.NormalizeDecimalYuan(savedOrder.PayAmount)
	normTotal, _ := util.NormalizeDecimalYuan(savedOrder.TotalAmount)
	if normFreight != "20.00" || normPay != "120.00" || normTotal != "100.00" {
		t.Fatalf("database historical order record altered! freight=%s, pay=%s, total=%s",
			normFreight, normPay, normTotal)
	}
}

// Scenario: 防止伪造地址编码绕过运费模板限制（如名称填新疆但编码填北京）
func TestFreightScenario_AddressCodeForgeryBlocked(t *testing.T) {
	db, router, token, memberID, _, _ := setupOrderTestApp(t)

	// 运费模板：新疆不配送
	noDelJSON := `[{"province_code":"650000","city_code":""}]`
	tmpl := model.FreightTemplate{
		Name:       "新疆不配送模板",
		BaseFee:    "10.00",
		NoDelivery: &noDelJSON,
		Status:     1,
	}
	db.Create(&tmpl)

	p := model.Product{Name: "新疆限运商品", CategoryID: 1, Status: 1, FreightType: 3, FreightTemplateID: &tmpl.ID}
	db.Create(&p)
	sku := model.ProductSKU{ProductID: p.ID, Price: "100.00", Status: 1, SpecHash: "sh_forgery"}
	db.Create(&sku)
	db.Create(&model.ProductSkuInventory{ProductID: p.ID, SKUID: sku.ID, Stock: 10, Version: 1})

	// 1. 通过接口创建伪造地址（名称填新疆，编码填北京 110000）-> 接口拒绝
	forgeBody := `{"name":"张三","mobile":"13800138000","province_code":"110000","province":"新疆维吾尔自治区","city_code":"110100","city":"乌鲁木齐市","district_code":"110101","district":"天山区","address":"解放路1号"}`
	reqAddr := httptest.NewRequest(http.MethodPost, "/api/miniapp/member/addresses/create", bytes.NewReader([]byte(forgeBody)))
	reqAddr.Header.Set("Content-Type", "application/json")
	reqAddr.Header.Set("Authorization", "Bearer "+token)
	wAddr := httptest.NewRecorder()
	router.ServeHTTP(wAddr, reqAddr)

	var addrResp response.Response[any]
	json.Unmarshal(wAddr.Body.Bytes(), &addrResp)
	if addrResp.Code == 1 {
		t.Fatalf("expected address creation with forged codes to be rejected, but succeeded")
	}

	// 2. 模拟数据库中存在伪造脏数据（直接入库绕过接口）
	forgedAddr := model.MemberAddress{
		MemberID:     memberID,
		Name:         "张三",
		Mobile:       "13800138000",
		ProvinceCode: "110000",
		Province:     "新疆维吾尔自治区",
		CityCode:     "110100",
		City:         "乌鲁木齐市",
		DistrictCode: "110101",
		District:     "天山区",
		Address:      "解放路1号",
	}
	db.Create(&forgedAddr)

	// 3. 在 Preview 阶段尝试使用该伪造地址 -> 应该报错拦截，绝不能放行返回 0 运费
	previewBody, _ := json.Marshal(miniapporder.CheckoutPreviewRequest{
		AddressID: &forgedAddr.ID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: sku.ID, Quantity: 1}},
	})
	reqPrev := httptest.NewRequest(http.MethodPost, "/api/miniapp/checkout/preview", bytes.NewReader(previewBody))
	reqPrev.Header.Set("Content-Type", "application/json")
	reqPrev.Header.Set("Authorization", "Bearer "+token)
	wPrev := httptest.NewRecorder()
	router.ServeHTTP(wPrev, reqPrev)

	var prevResp response.Response[any]
	json.Unmarshal(wPrev.Body.Bytes(), &prevResp)
	if prevResp.Code == 1 {
		t.Fatalf("expected preview to reject forged address, but succeeded")
	}

	// 4. 在 Create Order 阶段尝试使用该伪造地址下单 -> 拦截失败，不能创建订单
	createBody, _ := json.Marshal(miniapporder.OrderCreateRequest{
		RequestID: "req_forge_order_1",
		AddressID: forgedAddr.ID,
		Items:     []miniapporder.CheckoutItemRequest{{SKUID: sku.ID, Quantity: 1}},
	})
	reqOrder := httptest.NewRequest(http.MethodPost, "/api/miniapp/orders/create", bytes.NewReader(createBody))
	reqOrder.Header.Set("Content-Type", "application/json")
	reqOrder.Header.Set("Authorization", "Bearer "+token)
	wOrder := httptest.NewRecorder()
	router.ServeHTTP(wOrder, reqOrder)

	var orderResp response.Response[any]
	json.Unmarshal(wOrder.Body.Bytes(), &orderResp)
	if orderResp.Code == 1 {
		t.Fatalf("expected create order to reject forged address, but succeeded")
	}
}
