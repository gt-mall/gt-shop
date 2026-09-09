package miniapp_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	miniappauth "mall/admin_core/internal/miniapp/auth"
	miniappmember "mall/admin_core/internal/miniapp/member"
	"mall/admin_core/internal/model"
	systemsetting "mall/admin_core/internal/system/setting"
)

type mockWeChatClient struct {
	openID     string
	sessionKey string
	unionID    string
	err        error
}

func (m *mockWeChatClient) Code2Session(ctx context.Context, appID, appSecret, code string) (*miniappauth.WeChatSessionResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &miniappauth.WeChatSessionResult{
		OpenID:     m.openID,
		SessionKey: m.sessionKey,
		UnionID:    m.unionID,
	}, nil
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
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
		&model.SystemConfig{},
		&model.SystemFile{},
		&model.Member{},
		&model.MemberAuth{},
		&model.MemberAddress{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	settingService := systemsetting.NewService(db)
	if err := settingService.SaveMp(context.Background(), systemsetting.MpConfig{
		Name:      "Test MP",
		AppID:     "wx_test_appid",
		AppSecret: "wx_test_secret",
	}); err != nil {
		t.Fatalf("save mp config: %v", err)
	}

	return db
}

func TestWechatLogin_FirstAndSecondTime(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	authService := miniappauth.NewService(db, "test_jwt_secret_key_1234567890", 24*time.Hour)
	mockClient := &mockWeChatClient{
		openID:     "openid_test_1",
		sessionKey: "session_key_sensitive_12345",
		unionID:    "unionid_test_1",
	}
	authService.SetWeChatClient(mockClient)

	// 1. 首次登录自动创建会员
	res1, err := authService.WechatLogin(ctx, "mock_code_1")
	if err != nil {
		t.Fatalf("first login failed: %v", err)
	}
	if !res1.IsNewMember {
		t.Errorf("expected is_new_member=true, got false")
	}
	if res1.Member.ID == 0 {
		t.Errorf("expected valid member ID, got 0")
	}
	if res1.Token == "" {
		t.Errorf("expected non-empty token")
	}

	// 验证 member_auth 记录
	var authRow model.MemberAuth
	if err := db.Where("member_id = ?", res1.Member.ID).First(&authRow).Error; err != nil {
		t.Fatalf("find auth row: %v", err)
	}
	if authRow.OpenID != "openid_test_1" {
		t.Errorf("expected openid openid_test_1, got %s", authRow.OpenID)
	}
	if authRow.UnionID == nil || *authRow.UnionID != "unionid_test_1" {
		t.Errorf("expected unionid unionid_test_1, got %v", authRow.UnionID)
	}
	// 验证 provider_data 不得包含 session_key、app_secret
	if authRow.ProviderData != nil {
		if strings.Contains(*authRow.ProviderData, "session_key_sensitive_12345") {
			t.Fatalf("provider_data must NOT contain session_key!")
		}
		if strings.Contains(*authRow.ProviderData, "wx_test_secret") {
			t.Fatalf("provider_data must NOT contain app_secret!")
		}
	}

	firstLoginAt := res1.Member.LastLoginAt

	// 2. 第二次相同 app_id + openid 登录，返回原会员
	time.Sleep(10 * time.Millisecond)
	res2, err := authService.WechatLogin(ctx, "mock_code_2")
	if err != nil {
		t.Fatalf("second login failed: %v", err)
	}
	if res2.IsNewMember {
		t.Errorf("expected is_new_member=false on second login, got true")
	}
	if res2.Member.ID != res1.Member.ID {
		t.Errorf("expected same member ID %d, got %d", res1.Member.ID, res2.Member.ID)
	}
	if res2.Member.LastLoginAt == nil || (firstLoginAt != nil && !res2.Member.LastLoginAt.After(*firstLoginAt) && !res2.Member.LastLoginAt.Equal(*firstLoginAt)) {
		t.Errorf("expected updated last_login_at")
	}
}

func TestWechatLogin_DisabledMemberFails(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	authService := miniappauth.NewService(db, "test_jwt_secret_key_1234567890", 24*time.Hour)
	mockClient := &mockWeChatClient{
		openID:     "openid_disabled",
		sessionKey: "session_key_disabled",
	}
	authService.SetWeChatClient(mockClient)

	res, err := authService.WechatLogin(ctx, "mock_code")
	if err != nil {
		t.Fatalf("first login failed: %v", err)
	}

	// 禁用会员
	if err := db.Model(&model.Member{}).Where("id = ?", res.Member.ID).Update("status", 0).Error; err != nil {
		t.Fatalf("disable member: %v", err)
	}

	// 再次登录应该失败
	_, err = authService.WechatLogin(ctx, "mock_code")
	if err == nil {
		t.Fatalf("expected error for disabled member, got nil")
	}
	if !strings.Contains(err.Error(), "禁用") {
		t.Errorf("expected disabled error message, got %v", err)
	}
}

func TestMemberAuth_UniqueConstraintAndNullableUnionID(t *testing.T) {
	db := setupTestDB(t)

	// unionid 为空
	auth1 := model.MemberAuth{
		MemberID: 1,
		AuthType: "wechat_miniapp",
		AppID:    "wx_app_1",
		OpenID:   "openid_x",
		UnionID:  nil,
	}
	if err := db.Create(&auth1).Error; err != nil {
		t.Fatalf("create auth with nil unionid: %v", err)
	}

	// 重复插入相同 auth_type + app_id + openid 应该唯一冲突
	auth2 := model.MemberAuth{
		MemberID: 2,
		AuthType: "wechat_miniapp",
		AppID:    "wx_app_1",
		OpenID:   "openid_x",
		UnionID:  nil,
	}
	if err := db.Create(&auth2).Error; err == nil {
		t.Fatalf("expected error on duplicate (auth_type, app_id, openid), got nil")
	}
}

func TestAddressOperations_DefaultAndIsolation(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	memberService := miniappmember.NewService(db)

	memberA := int64(10)
	memberB := int64(20)

	// 1. 会员 A 创建第一个地址，应该自动成为默认地址
	addr1, err := memberService.CreateAddress(ctx, memberA, miniappmember.AddressCreateRequest{
		Name:         "张三",
		Mobile:       "13800000001",
		ProvinceCode: "110000",
		Province:     "北京市",
		CityCode:     "110100",
		City:         "北京市",
		DistrictCode: "110101",
		District:     "东城区",
		Address:      "长安街1号",
	})
	if err != nil {
		t.Fatalf("create addr1: %v", err)
	}
	if addr1.IsDefault != 1 {
		t.Errorf("expected first address to be default (1), got %d", addr1.IsDefault)
	}

	// 2. 会员 A 创建第二个地址并指定为默认地址，addr1 应被取消默认
	addr2, err := memberService.CreateAddress(ctx, memberA, miniappmember.AddressCreateRequest{
		Name:         "李四",
		Mobile:       "13800000002",
		ProvinceCode: "310000",
		Province:     "上海市",
		CityCode:     "310100",
		City:         "上海市",
		DistrictCode: "310104",
		District:     "徐汇区",
		Address:      "淮海路1号",
		IsDefault:    &[]int8{1}[0],
	})
	if err != nil {
		t.Fatalf("create addr2: %v", err)
	}
	if addr2.IsDefault != 1 {
		t.Errorf("expected addr2 to be default, got %d", addr2.IsDefault)
	}

	// 验证 addr1 现在 is_default=0
	listA, err := memberService.ListAddresses(ctx, memberA)
	if err != nil {
		t.Fatalf("list memberA addresses: %v", err)
	}
	if len(listA) != 2 {
		t.Fatalf("expected 2 addresses, got %d", len(listA))
	}
	for _, a := range listA {
		if a.ID == addr1.ID && a.IsDefault != 0 {
			t.Errorf("addr1 should no longer be default")
		}
		if a.ID == addr2.ID && a.IsDefault != 1 {
			t.Errorf("addr2 should be default")
		}
	}

	// 3. 跨会员隔离：会员 B 不能更新、删除、设默认 会员 A 的地址
	_, err = memberService.UpdateAddress(ctx, memberB, miniappmember.AddressUpdateRequest{
		ID:           addr1.ID,
		Name:         "黑客",
		Mobile:       "13999999999",
		ProvinceCode: "110000",
		Province:     "北京市",
		CityCode:     "110100",
		City:         "北京市",
		DistrictCode: "110101",
		District:     "东城区",
		Address:      "尝试篡改",
	})
	if err == nil {
		t.Fatalf("expected cross-member update to fail, got nil")
	}

	if err := memberService.SetDefaultAddress(ctx, memberB, addr1.ID); err == nil {
		t.Fatalf("expected cross-member set default to fail, got nil")
	}

	if err := memberService.DeleteAddress(ctx, memberB, addr1.ID); err == nil {
		t.Fatalf("expected cross-member delete to fail, got nil")
	}

	// 4. 删除默认地址 (addr2)，addr1 应该自动被设为默认地址
	if err := memberService.DeleteAddress(ctx, memberA, addr2.ID); err != nil {
		t.Fatalf("delete addr2: %v", err)
	}

	listAAfterDelete, err := memberService.ListAddresses(ctx, memberA)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(listAAfterDelete) != 1 {
		t.Fatalf("expected 1 address remaining, got %d", len(listAAfterDelete))
	}
	if listAAfterDelete[0].ID != addr1.ID {
		t.Fatalf("expected remaining address to be addr1")
	}
	if listAAfterDelete[0].IsDefault != 1 {
		t.Errorf("expected remaining addr1 to automatically become default, got %d", listAAfterDelete[0].IsDefault)
	}
}

func TestMemberAvatarResolve(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// 插入 system_config (file_storage)
	if err := db.Create(&model.SystemConfig{
		ConfigKey: "file_storage",
		Value:     `{"default_storage":"local","local":{"path":"uploads","domain":"https://img.example.com"}}`,
	}).Error; err != nil {
		t.Fatal(err)
	}

	// 插入 system_file
	fileRow := model.SystemFile{
		ID:      201,
		Storage: "local",
		Name:    "avatar.png",
		Path:    "2026/09/00000000-0000-0000-0000-000000000201.png",
		Ext:     "png",
	}
	if err := db.Create(&fileRow).Error; err != nil {
		t.Fatal(err)
	}

	avatarID := int64(201)
	m := model.Member{
		ID:       100,
		Nickname: "头像测试用户",
		AvatarID: &avatarID,
		Status:   1,
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}

	memberService := miniappmember.NewService(db)
	profile, err := memberService.GetProfile(ctx, 100)
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.AvatarURL == "" {
		t.Errorf("expected resolved avatar URL, got empty")
	}
	if !strings.Contains(profile.AvatarURL, "000000000201.png") {
		t.Errorf("expected avatar URL to contain filename, got %s", profile.AvatarURL)
	}
}
