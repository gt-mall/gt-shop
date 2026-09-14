package product_test

import (
	"context"
	"testing"

	"mall/admin_core/internal/model"
	productfreight "mall/admin_core/internal/product/freight"
	productproduct "mall/admin_core/internal/product/product"
)

// Scenario 1: 全国包邮商品，无论收货地在哪里，运费都是 0.00
func TestFreightScenario1_NationwideFreeShipping(t *testing.T) {
	prod := &model.Product{
		FreightType: 1, // 包邮
	}

	testCases := []struct {
		province string
		city     string
	}{
		{"110000", "110100"}, // 北京
		{"440000", "440100"}, // 广东广州
		{"650000", "650100"}, // 新疆乌鲁木齐
		{"", ""},             // 未选地址
	}

	for _, tc := range testCases {
		feeCents, deliverable, err := productfreight.CalculateProductFreight(prod, nil, tc.province, tc.city)
		if err != nil {
			t.Fatalf("unexpected error for %s/%s: %v", tc.province, tc.city, err)
		}
		if !deliverable {
			t.Errorf("expected deliverable for %s/%s", tc.province, tc.city)
		}
		if feeCents != 0 {
			t.Errorf("expected fee 0 for %s/%s, got: %d", tc.province, tc.city, feeCents)
		}
	}
}

// Scenario 2: 固定运费商品，无论收货地在哪里，运费都是该固定金额
func TestFreightScenario2_FixedFreight(t *testing.T) {
	prod := &model.Product{
		FreightType: 2,
		FreightFee:  "15.50",
	}

	testCases := []struct {
		province string
		city     string
	}{
		{"110000", "110100"},
		{"440000", "440100"},
		{"650000", "650100"},
		{"", ""},
	}

	for _, tc := range testCases {
		feeCents, deliverable, err := productfreight.CalculateProductFreight(prod, nil, tc.province, tc.city)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !deliverable {
			t.Errorf("expected deliverable")
		}
		if feeCents != 1550 {
			t.Errorf("expected 1550 cents, got: %d", feeCents)
		}
	}
}

// Scenario 3: 模板默认包邮，目标收货地没有特殊规则，运费是 0.00
func TestFreightScenario3_TemplateDefaultFree(t *testing.T) {
	tmpl := &model.FreightTemplate{
		BaseFee: "0.00",
		Status:  1,
	}

	feeCents, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "310000", "310100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 0 {
		t.Fatalf("expected deliverable with fee 0, got deliverable=%v, fee=%d", deliverable, feeCents)
	}
}

// Scenario 4: 模板默认固定运费，目标收货地没有特殊规则，运费是默认运费
func TestFreightScenario4_TemplateDefaultFixed(t *testing.T) {
	tmpl := &model.FreightTemplate{
		BaseFee: "12.00",
		Status:  1,
	}

	feeCents, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "310000", "310100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 1200 {
		t.Fatalf("expected deliverable with fee 1200, got deliverable=%v, fee=%d", deliverable, feeCents)
	}
}

// Scenario 5: 模板针对某省配置了附加运费，收货地在该省的任意普通城市，运费正确加上该省附加运费
func TestFreightScenario5_TemplateProvinceExtraFee(t *testing.T) {
	extraJSON := `[{"province_code":"650000","province_name":"新疆维吾尔自治区","city_code":"","city_name":"","fee":"20.00"}]`
	tmpl := &model.FreightTemplate{
		BaseFee:   "10.00",
		ExtraFees: &extraJSON,
		Status:    1,
	}

	// 新疆乌鲁木齐 (650100) -> 10.00 + 20.00 = 30.00 (3000分)
	feeCents, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "650000", "650100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 3000 {
		t.Fatalf("expected 3000 cents, got %d", feeCents)
	}

	// 其他省份例如北京 (110000) -> 仅基础运费 10.00 (1000分)
	feeCents, deliverable, err = productfreight.CalculateTemplateFreight(tmpl, "110000", "110100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 1000 {
		t.Fatalf("expected 1000 cents, got %d", feeCents)
	}
}

// Scenario 6: 模板针对某市配置了附加运费，收货地在该市，运费正确加上该市附加运费
func TestFreightScenario6_TemplateCityExtraFee(t *testing.T) {
	extraJSON := `[{"province_code":"440000","province_name":"广东省","city_code":"440300","city_name":"深圳市","fee":"8.00"}]`
	tmpl := &model.FreightTemplate{
		BaseFee:   "10.00",
		ExtraFees: &extraJSON,
		Status:    1,
	}

	// 广东深圳 (440300) -> 10.00 + 8.00 = 18.00 (1800分)
	feeCents, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "440000", "440300")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 1800 {
		t.Fatalf("expected 1800 cents, got %d", feeCents)
	}

	// 广东广州 (440100) -> 仅基础运费 10.00 (1000分)
	feeCents, deliverable, err = productfreight.CalculateTemplateFreight(tmpl, "440000", "440100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 1000 {
		t.Fatalf("expected 1000 cents, got %d", feeCents)
	}
}

// Scenario 7: 模板针对某省和该省下某市都配置了附加运费，收货地在该市时，优先采用城市附加运费，且不与省份附加运费叠加
func TestFreightScenario7_TemplateCityOverridesProvinceExtraFee(t *testing.T) {
	extraJSON := `[
		{"province_code":"440000","province_name":"广东省","city_code":"","city_name":"","fee":"15.00"},
		{"province_code":"440000","province_name":"广东省","city_code":"440300","city_name":"深圳市","fee":"5.00"}
	]`
	tmpl := &model.FreightTemplate{
		BaseFee:   "10.00",
		ExtraFees: &extraJSON,
		Status:    1,
	}

	// 广东深圳 (440300): 优先城市附加5.00，不与省份15.00叠加 -> 10.00 + 5.00 = 15.00 (1500分)
	feeCents, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "440000", "440300")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 1500 {
		t.Fatalf("expected 1500 cents (base 10 + city 5), got %d", feeCents)
	}

	// 广东广州 (440100): 采用省份附加15.00 -> 10.00 + 15.00 = 25.00 (2500分)
	feeCents, deliverable, err = productfreight.CalculateTemplateFreight(tmpl, "440000", "440100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 2500 {
		t.Fatalf("expected 2500 cents (base 10 + prov 15), got %d", feeCents)
	}
}

// Scenario 8: 模板针对某省配置了不配送，收货地在该省，返回不可配送与对应原因
func TestFreightScenario8_TemplateProvinceNoDelivery(t *testing.T) {
	noDelJSON := `[{"province_code":"540000","province_name":"西藏自治区","city_code":"","city_name":""}]`
	tmpl := &model.FreightTemplate{
		BaseFee:    "0.00",
		NoDelivery: &noDelJSON,
		Status:     1,
	}

	// 西藏拉萨 (540100) -> 不可配送
	_, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "540000", "540100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deliverable {
		t.Fatalf("expected undeliverable for Tibet")
	}

	// 北京 (110000) -> 正常可配送
	feeCents, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "110000", "110100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 0 {
		t.Fatalf("expected deliverable for Beijing, got %v, fee: %d", deliverable, feeCents)
	}
}

// Scenario 9: 模板针对某市配置了不配送，收货地在该市，返回不可配送与对应原因
func TestFreightScenario9_TemplateCityNoDelivery(t *testing.T) {
	noDelJSON := `[{"province_code":"440000","province_name":"广东省","city_code":"440100","city_name":"广州市"}]`
	tmpl := &model.FreightTemplate{
		BaseFee:    "0.00",
		NoDelivery: &noDelJSON,
		Status:     1,
	}

	// 广东广州 (440100) -> 不可配送
	_, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "440000", "440100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deliverable {
		t.Fatalf("expected undeliverable for Guangzhou")
	}

	// 广东深圳 (440300) -> 可配送
	_, deliverable, err = productfreight.CalculateTemplateFreight(tmpl, "440000", "440300")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable {
		t.Fatalf("expected deliverable for Shenzhen")
	}
}

// Scenario 10: 模板针对某市配置了附加运费，但针对该市所在省配置了不配送，收货地在该市时，返回不可配送，不配送优先于附加运费
func TestFreightScenario10_CityExtraProvinceNoDelivery(t *testing.T) {
	extraJSON := `[{"province_code":"650000","province_name":"新疆维吾尔自治区","city_code":"650100","city_name":"乌鲁木齐市","fee":"30.00"}]`
	noDelJSON := `[{"province_code":"650000","province_name":"新疆维吾尔自治区","city_code":"","city_name":""}]`
	tmpl := &model.FreightTemplate{
		BaseFee:    "10.00",
		ExtraFees:  &extraJSON,
		NoDelivery: &noDelJSON,
		Status:     1,
	}

	// 收货地在新疆乌鲁木齐 (650000, 650100) -> 全省不配送优先，返回不可配送
	_, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "650000", "650100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deliverable {
		t.Fatalf("expected undeliverable due to province no-delivery priority over city extra fee")
	}
}

// Scenario 11: 模板针对某省配置了附加运费，但针对该省某市配置了不配送，收货地在该市时，返回不可配送，不配送优先于附加运费
func TestFreightScenario11_ProvinceExtraCityNoDelivery(t *testing.T) {
	extraJSON := `[{"province_code":"440000","province_name":"广东省","city_code":"","city_name":"","fee":"10.00"}]`
	noDelJSON := `[{"province_code":"440000","province_name":"广东省","city_code":"440300","city_name":"深圳市"}]`
	tmpl := &model.FreightTemplate{
		BaseFee:    "10.00",
		ExtraFees:  &extraJSON,
		NoDelivery: &noDelJSON,
		Status:     1,
	}

	// 广东深圳 (440300) -> 市级不配送生效，返回不可配送
	_, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "440000", "440300")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deliverable {
		t.Fatalf("expected undeliverable for Shenzhen")
	}

	// 广东广州 (440100) -> 正常可配送，运费 10 + 10 = 20.00 (2000分)
	feeCents, deliverable, err := productfreight.CalculateTemplateFreight(tmpl, "440000", "440100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deliverable || feeCents != 2000 {
		t.Fatalf("expected 2000 cents for Guangzhou, got %d", feeCents)
	}
}

// Scenario 12: 模板配置规则重复校验：同一个省份或城市在同类型规则中重复出现，保存或更新报错
func TestFreightScenario12_DuplicateRulesValidation(t *testing.T) {
	db := productTestDB(t)
	freightSvc := productfreight.NewService(db)
	ctx := context.Background()

	// 1. 同一省份在附加运费中重复
	req1 := productfreight.FreightTemplateCreateRequest{
		Name:    "重复省份附加费",
		BaseFee: "0.00",
		ExtraFees: []productfreight.ExtraFeeRule{
			{ProvinceCode: "110000", Fee: "5.00"},
			{ProvinceCode: "110000", Fee: "10.00"},
		},
	}
	_, err := freightSvc.Create(ctx, req1)
	if err == nil {
		t.Fatalf("expected error for duplicate province in extra fees, got nil")
	}

	// 2. 同一城市在附加运费中重复
	req2 := productfreight.FreightTemplateCreateRequest{
		Name:    "重复城市附加费",
		BaseFee: "0.00",
		ExtraFees: []productfreight.ExtraFeeRule{
			{ProvinceCode: "440000", CityCode: "440100", Fee: "5.00"},
			{ProvinceCode: "440000", CityCode: "440100", Fee: "8.00"},
		},
	}
	_, err = freightSvc.Create(ctx, req2)
	if err == nil {
		t.Fatalf("expected error for duplicate city in extra fees, got nil")
	}

	// 3. 同一地区在不配送中重复
	req3 := productfreight.FreightTemplateCreateRequest{
		Name:    "重复不配送地区",
		BaseFee: "0.00",
		NoDelivery: []productfreight.NoDeliveryRule{
			{ProvinceCode: "540000"},
			{ProvinceCode: "540000"},
		},
	}
	_, err = freightSvc.Create(ctx, req3)
	if err == nil {
		t.Fatalf("expected error for duplicate area in no delivery, got nil")
	}
}

// Scenario 13: 模板配置规则冲突校验：同一个省份或城市同时出现在附加运费和不配送中，保存或更新报错
func TestFreightScenario13_ConflictRulesValidation(t *testing.T) {
	db := productTestDB(t)
	freightSvc := productfreight.NewService(db)
	ctx := context.Background()

	// 1. 同一省份既配置附加运费又配置不配送
	req1 := productfreight.FreightTemplateCreateRequest{
		Name:    "省份冲突",
		BaseFee: "0.00",
		ExtraFees: []productfreight.ExtraFeeRule{
			{ProvinceCode: "650000", Fee: "20.00"},
		},
		NoDelivery: []productfreight.NoDeliveryRule{
			{ProvinceCode: "650000"},
		},
	}
	_, err := freightSvc.Create(ctx, req1)
	if err == nil {
		t.Fatalf("expected error for province conflict between extra and no-delivery, got nil")
	}

	// 2. 同一城市既配置附加运费又配置不配送
	req2 := productfreight.FreightTemplateCreateRequest{
		Name:    "城市冲突",
		BaseFee: "0.00",
		ExtraFees: []productfreight.ExtraFeeRule{
			{ProvinceCode: "440000", CityCode: "440300", Fee: "5.00"},
		},
		NoDelivery: []productfreight.NoDeliveryRule{
			{ProvinceCode: "440000", CityCode: "440300"},
		},
	}
	_, err = freightSvc.Create(ctx, req2)
	if err == nil {
		t.Fatalf("expected error for city conflict between extra and no-delivery, got nil")
	}
}

// Scenario 14: 模板配置非法金额校验：负数运费、小数点超过 2 位、非数字金额，保存或更新报错
func TestFreightScenario14_IllegalAmountValidation(t *testing.T) {
	db := productTestDB(t)
	freightSvc := productfreight.NewService(db)
	ctx := context.Background()

	invalidBaseFees := []string{"-1.00", "-0.01", "12.345", "abc", "12.3.4", "", " "}
	for _, fee := range invalidBaseFees {
		req := productfreight.FreightTemplateCreateRequest{
			Name:    "非法基础金额测试",
			BaseFee: fee,
		}
		_, err := freightSvc.Create(ctx, req)
		if err == nil {
			t.Errorf("expected error for invalid base fee %q, got nil", fee)
		}
	}

	invalidExtraFees := []string{"-5.00", "5.123", "nan"}
	for _, extra := range invalidExtraFees {
		req := productfreight.FreightTemplateCreateRequest{
			Name:    "非法附加金额测试",
			BaseFee: "0.00",
			ExtraFees: []productfreight.ExtraFeeRule{
				{ProvinceCode: "110000", Fee: extra},
			},
		}
		_, err := freightSvc.Create(ctx, req)
		if err == nil {
			t.Errorf("expected error for invalid extra fee %q, got nil", extra)
		}
	}
}

// Scenario 15: 商品绑定不存在的运费模板，创建或更新报错
func TestFreightScenario15_NonExistentFreightTemplate(t *testing.T) {
	db := productTestDB(t)
	productSvc := productproduct.NewService(db)
	ctx := context.Background()

	cat := model.ProductCategory{Name: "测试分类", Status: 1}
	db.Create(&cat)

	nonExistentTmplID := int64(99999)
	freightType3 := int8(3)
	createReq := productproduct.ProductCreateRequest{
		CategoryID:        cat.ID,
		Name:              "绑定不存在模板商品",
		FreightType:       &freightType3,
		FreightTemplateID: &nonExistentTmplID,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "100.00"},
		},
	}

	_, err := productSvc.Create(ctx, createReq)
	if err == nil {
		t.Fatalf("expected error binding non-existent template, got nil")
	}

	// 接下来测试固定运费但金额未提供
	freightType2 := int8(2)
	createReqFixedInvalid := createReq
	createReqFixedInvalid.FreightType = &freightType2
	createReqFixedInvalid.FreightTemplateID = nil
	createReqFixedInvalid.FreightFee = nil
	_, err = productSvc.Create(ctx, createReqFixedInvalid)
	if err == nil {
		t.Fatalf("expected error for missing freight_fee when freight_type=2, got nil")
	}
}

// Scenario 16: 商品绑定已禁用的运费模板，创建或更新报错；下架商品上架时若模板已被禁用，上架报错
func TestFreightScenario16_DisabledTemplateValidation(t *testing.T) {
	db := productTestDB(t)
	freightSvc := productfreight.NewService(db)
	productSvc := productproduct.NewService(db)
	ctx := context.Background()

	cat := model.ProductCategory{Name: "测试分类", Status: 1}
	db.Create(&cat)

	// 1. 创建一个正常启用的运费模板
	tmplID, err := freightSvc.Create(ctx, productfreight.FreightTemplateCreateRequest{
		Name:    "可禁用模板",
		BaseFee: "0.00",
	})
	if err != nil {
		t.Fatalf("create template failed: %v", err)
	}

	// 2. 禁用该模板
	if err := freightSvc.Status(ctx, tmplID, 0); err != nil {
		t.Fatalf("disable template failed: %v", err)
	}

	// 3. 创建商品绑定该已禁用模板 -> 应报错
	freightType3 := int8(3)
	createReq := productproduct.ProductCreateRequest{
		CategoryID:        cat.ID,
		Name:              "绑定已禁用模板商品",
		FreightType:       &freightType3,
		FreightTemplateID: &tmplID,
		SKUs: []productproduct.ProductSKURequest{
			{Price: "100.00"},
		},
	}
	_, err = productSvc.Create(ctx, createReq)
	if err == nil {
		t.Fatalf("expected error when binding disabled template, got nil")
	}

	// 4. 重新启用模板
	if err := freightSvc.Status(ctx, tmplID, 1); err != nil {
		t.Fatalf("enable template failed: %v", err)
	}

	// 5. 成功创建商品
	statusDraft := int8(0)
	createReq.Status = &statusDraft
	prodDetail, err := productSvc.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("create product with enabled template failed: %v", err)
	}

	// 6. 再次将模板禁用
	if err := freightSvc.Status(ctx, tmplID, 0); err != nil {
		t.Fatalf("disable template failed: %v", err)
	}

	// 7. 尝试将商品上架 (status=1) -> 应当报错拒绝上架
	err = productSvc.Status(ctx, productproduct.ProductStatusRequest{ID: prodDetail.ID, Status: 1})
	if err == nil {
		t.Fatalf("expected error when publishing product whose freight template is disabled, got nil")
	}

	// 8. 尝试删除该被引用的模板 -> 应当报错拒绝删除
	err = freightSvc.Delete(ctx, tmplID)
	if err == nil {
		t.Fatalf("expected error when deleting template referenced by product, got nil")
	}
}

// Scenario: 创建禁用模板，实际应保存为禁用 (status = 0) 而非被 GORM 替换为默认值 1
func TestFreightScenario_CreateDisabledTemplateDirectly(t *testing.T) {
	db := productTestDB(t)
	freightSvc := productfreight.NewService(db)
	ctx := context.Background()

	statusZero := int8(0)
	tmplID, err := freightSvc.Create(ctx, productfreight.FreightTemplateCreateRequest{
		Name:    "直接创建为禁用的模板",
		BaseFee: "10.00",
		Status:  &statusZero,
	})
	if err != nil {
		t.Fatalf("create template failed: %v", err)
	}

	detail, err := freightSvc.Detail(ctx, tmplID)
	if err != nil {
		t.Fatalf("get template detail failed: %v", err)
	}

	if detail.Status != 0 {
		t.Fatalf("expected template status 0, but got %d", detail.Status)
	}

	var modelTmpl model.FreightTemplate
	if err := db.Where("id = ?", tmplID).First(&modelTmpl).Error; err != nil {
		t.Fatalf("query db failed: %v", err)
	}
	if modelTmpl.Status != 0 {
		t.Fatalf("expected DB model status 0, but got %d", modelTmpl.Status)
	}
}

// Scenario: 地区编码带空格时，规则能够去除空格保存并正常生效
func TestFreightScenario_RuleCodesWithWhitespace(t *testing.T) {
	db := productTestDB(t)
	freightSvc := productfreight.NewService(db)
	ctx := context.Background()

	// 带空格的规则
	req := productfreight.FreightTemplateCreateRequest{
		Name:    "带空格规则模板",
		BaseFee: "10.00",
		ExtraFees: []productfreight.ExtraFeeRule{
			{ProvinceCode: " 440000 ", CityCode: " 440300 ", Fee: " 8.00 "},
		},
		NoDelivery: []productfreight.NoDeliveryRule{
			{ProvinceCode: " 650000 ", CityCode: " "},
		},
	}

	tmplID, err := freightSvc.Create(ctx, req)
	if err != nil {
		t.Fatalf("create template with whitespace failed: %v", err)
	}

	var saved model.FreightTemplate
	if err := db.Where("id = ?", tmplID).First(&saved).Error; err != nil {
		t.Fatalf("query db failed: %v", err)
	}

	// 1. 验证新疆 (650000) 触发不可配送
	_, deliverable, err := productfreight.CalculateTemplateFreight(&saved, "650000", "650100")
	if err != nil {
		t.Fatalf("calc freight: %v", err)
	}
	if deliverable {
		t.Fatalf("expected Xinjiang to be undeliverable even when rule had whitespace")
	}

	// 2. 验证深圳 (440000, 440300) 附加运费 10 + 8 = 18.00
	feeCents, deliverable, err := productfreight.CalculateTemplateFreight(&saved, "440000", "440300")
	if err != nil {
		t.Fatalf("calc freight: %v", err)
	}
	if !deliverable || feeCents != 1800 {
		t.Fatalf("expected 1800 cents for Shenzhen, got fee=%d, deliverable=%v", feeCents, deliverable)
	}
}
