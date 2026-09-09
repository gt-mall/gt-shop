package member_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mall/admin_core/internal/auth"
	adminmember "mall/admin_core/internal/member/member"
	miniappauth "mall/admin_core/internal/miniapp/auth"
	"mall/admin_core/internal/model"
)

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
		&model.SystemUser{},
		&model.Member{},
		&model.MemberAuth{},
		&model.MemberAddress{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func TestJWTIssuerIsolation(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	sharedSecret := "shared_jwt_secret_key_1234567890"

	// 准备后台管理员与商城会员
	user := model.SystemUser{
		ID:       1,
		Username: "admin",
		Nickname: "管理员",
		Status:   1,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	member := model.Member{
		ID:       1,
		Nickname: "会员一号",
		Status:   1,
	}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}

	adminAuthService := auth.NewService(db, nil, sharedSecret, time.Hour)
	miniappAuthService := miniappauth.NewService(db, sharedSecret, time.Hour)

	// 1. 生成后台 Admin Token (issuer: mall-admin-core)
	adminClaims := jwt.RegisteredClaims{
		Issuer:    "mall-admin-core",
		Subject:   strconv.FormatInt(user.ID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	adminToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims).SignedString([]byte(sharedSecret))
	if err != nil {
		t.Fatal(err)
	}

	// 2. 生成商城 Member Token (issuer: mall-miniapp-member)
	memberClaims := jwt.RegisteredClaims{
		Issuer:    "mall-miniapp-member",
		Subject:   strconv.FormatInt(member.ID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	memberToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, memberClaims).SignedString([]byte(sharedSecret))
	if err != nil {
		t.Fatal(err)
	}

	// 3. 验证 Admin Token 可以在后台认证通过
	authUser, err := adminAuthService.Authenticate(ctx, adminToken)
	if err != nil || authUser.ID != user.ID {
		t.Fatalf("admin token should authenticate on admin service, got user: %+v, err: %v", authUser, err)
	}

	// 4. 验证 Admin Token 不能用于商城会员认证
	_, err = miniappAuthService.Authenticate(ctx, adminToken)
	if err == nil {
		t.Fatalf("admin token must NOT be accepted by miniapp member auth service!")
	}

	// 5. 验证 Member Token 可以在会员服务认证通过
	authMember, err := miniappAuthService.Authenticate(ctx, memberToken)
	if err != nil || authMember.ID != member.ID {
		t.Fatalf("member token should authenticate on member service, got member: %+v, err: %v", authMember, err)
	}

	// 6. 验证 Member Token 不能用于后台管理员认证
	_, err = adminAuthService.Authenticate(ctx, memberToken)
	if err == nil {
		t.Fatalf("member token must NOT be accepted by admin auth service!")
	}
}

func TestAdminMemberListAndDetail(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	mobile1 := "13800138000"
	mobile2 := "13900139000"
	avatarID := int64(10)

	// 插入 system_config (file_storage)
	if err := db.Create(&model.SystemConfig{
		ConfigKey: "file_storage",
		Value:     `{"default_storage":"local","local":{"path":"uploads","domain":"https://img.test"}}`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemFile{
		ID:      10,
		Storage: "local",
		Name:    "avatar.png",
		Path:    "2026/09/00000000-0000-0000-0000-000000000010.png",
		Ext:     "png",
	}).Error; err != nil {
		t.Fatal(err)
	}

	// 插入两个会员：一个正常，一个禁用
	m1 := model.Member{
		ID:       1,
		Nickname: "张三",
		AvatarID: &avatarID,
		Mobile:   &mobile1,
		Status:   1,
	}
	m2 := model.Member{
		ID:       2,
		Nickname: "李四",
		Mobile:   &mobile2,
		Status:   0, // 禁用
	}
	if err := db.Create(&m1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&m2).Error; err != nil {
		t.Fatal(err)
	}

	provData := `{"city":"Guangzhou","province":"Guangdong"}`
	if err := db.Create(&model.MemberAuth{
		MemberID:     1,
		AuthType:     "wechat_miniapp",
		AppID:        "wx_app_test",
		OpenID:       "openid_zhangsan_123",
		ProviderData: &provData,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&model.MemberAddress{
		MemberID:     1,
		Name:         "张三",
		Mobile:       mobile1,
		ProvinceCode: "440000",
		Province:     "广东省",
		CityCode:     "440100",
		City:         "广州市",
		DistrictCode: "440106",
		District:     "天河区",
		Address:      "天河路1号",
		IsDefault:    1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	service := adminmember.NewService(db)

	// 1. 查询全部会员列表
	rows, total, err := service.List(ctx, adminmember.MemberListRequest{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if total != 2 || len(rows) != 2 {
		t.Fatalf("expected 2 members, got total=%d, len=%d", total, len(rows))
	}
	// 验证 avatar_url 解析
	var foundM1 bool
	for _, r := range rows {
		if r.ID == 1 {
			foundM1 = true
			if r.AvatarURL == "" {
				t.Errorf("expected resolved avatar URL for m1")
			}
		}
	}
	if !foundM1 {
		t.Errorf("m1 not found in list")
	}

	// 2. 筛选 status=0（禁用）
	statusZero := int8(0)
	rows0, total0, err := service.List(ctx, adminmember.MemberListRequest{
		Page:     1,
		PageSize: 10,
		Status:   &statusZero,
	})
	if err != nil {
		t.Fatalf("list status=0: %v", err)
	}
	if total0 != 1 || len(rows0) != 1 || rows0[0].ID != 2 {
		t.Fatalf("expected only m2 with status=0, got total=%d", total0)
	}

	// 3. Keyword 搜索 openid
	rowsKW, totalKW, err := service.List(ctx, adminmember.MemberListRequest{
		Page:     1,
		PageSize: 10,
		Keyword:  "zhangsan",
	})
	if err != nil {
		t.Fatalf("list keyword openid: %v", err)
	}
	if totalKW != 1 || len(rowsKW) != 1 || rowsKW[0].ID != 1 {
		t.Fatalf("expected m1 matching keyword zhangsan, got total=%d", totalKW)
	}

	// 4. Keyword 搜索 手机号
	rowsMobile, totalMobile, err := service.List(ctx, adminmember.MemberListRequest{
		Page:     1,
		PageSize: 10,
		Keyword:  "13900139",
	})
	if err != nil {
		t.Fatalf("list keyword mobile: %v", err)
	}
	if totalMobile != 1 || len(rowsMobile) != 1 || rowsMobile[0].ID != 2 {
		t.Fatalf("expected m2 matching mobile, got total=%d", totalMobile)
	}

	// 5. 详情查询
	detail, err := service.Detail(ctx, 1)
	if err != nil {
		t.Fatalf("get detail for m1: %v", err)
	}
	if detail.Member.ID != 1 || detail.Member.Nickname != "张三" {
		t.Errorf("unexpected member detail: %+v", detail.Member)
	}
	if len(detail.Auth) != 1 || detail.Auth[0].OpenID != "openid_zhangsan_123" {
		t.Errorf("unexpected auth in detail: %+v", detail.Auth)
	}
	if detail.Auth[0].ProviderData == nil || *detail.Auth[0].ProviderData == "" {
		t.Errorf("expected formatted provider_data in detail")
	}
	if len(detail.Addresses) != 1 || detail.Addresses[0].Name != "张三" {
		t.Errorf("unexpected addresses in detail: %+v", detail.Addresses)
	}
}
