package miniapp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	miniappshare "mall/admin_core/internal/miniapp/share"
	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	systemsetting "mall/admin_core/internal/system/setting"
)

var testPngBytes = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

type mockShareWeChatClient struct {
	tokenCalls   int32
	qrCodeCalls  int32
	tokenErr     error
	qrErr        error
	tokenReturn  string
	qrDataReturn []byte
	qrMimeReturn string
	mu           sync.Mutex
}

func (m *mockShareWeChatClient) GetAccessToken(ctx context.Context, appID, appSecret string) (string, int, error) {
	atomic.AddInt32(&m.tokenCalls, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tokenErr != nil {
		return "", 0, m.tokenErr
	}
	if m.tokenReturn != "" {
		return m.tokenReturn, 7200, nil
	}
	return "mock-access-token", 7200, nil
}

func (m *mockShareWeChatClient) GetUnlimitedQRCode(ctx context.Context, accessToken string, params miniappshare.WeChatQRCodeParams) ([]byte, string, error) {
	atomic.AddInt32(&m.qrCodeCalls, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.qrErr != nil {
		return nil, "", m.qrErr
	}
	if len(m.qrDataReturn) > 0 {
		return m.qrDataReturn, m.qrMimeReturn, nil
	}
	return testPngBytes, "image/png", nil
}

func setupShareCodeTestEnv(t *testing.T) (*gorm.DB, *miniappshare.Service, *mockShareWeChatClient, *systemsetting.Service) {
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
		&model.SystemFileGroup{},
		&model.ProductCategory{},
		&model.Product{},
		&model.ProductShareCode{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	fileSvc := systemfile.NewService(db)
	cfg := systemfile.DefaultConfig()
	cfg.Local.Path = t.TempDir()
	if err := fileSvc.SaveConfig(context.Background(), cfg); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	settingSvc := systemsetting.NewService(db, nil)
	// 初始化小程序设置
	_ = settingSvc.SaveMp(context.Background(), systemsetting.MpConfig{
		AppID:     "wx1234567890abcdef",
		AppSecret: "secret_1234567890abcdef",
	})

	mockClient := &mockShareWeChatClient{
		tokenReturn:  "test-token",
		qrDataReturn: testPngBytes,
		qrMimeReturn: "image/png",
	}

	shareSvc := miniappshare.NewService(db, nil, fileSvc, settingSvc, mockClient)
	return db, shareSvc, mockClient, settingSvc
}

// 1. 商品不存在或已下架测试
func TestShareCode_ProductNotFoundOrOffShelf(t *testing.T) {
	ctx := context.Background()
	db, shareSvc, _, _ := setupShareCodeTestEnv(t)

	// 不存在的商品
	_, err := shareSvc.GetOrCreateProductShareCode(ctx, 99999)
	if err == nil || err.Error() != "商品不存在或已下架" {
		t.Fatalf("expected '商品不存在或已下架', got: %v", err)
	}

	// 下架商品 (status = 0)
	offShelfProd := model.Product{
		ID:     101,
		Name:   "已下架商品",
		Status: 0,
	}
	if err := db.Create(&offShelfProd).Error; err != nil {
		t.Fatalf("create off-shelf product: %v", err)
	}

	_, err = shareSvc.GetOrCreateProductShareCode(ctx, 101)
	if err == nil || err.Error() != "商品不存在或已下架" {
		t.Fatalf("expected '商品不存在或已下架', got: %v", err)
	}
}

// 2. 未配置 AppID / AppSecret 测试
func TestShareCode_MissingMpConfig(t *testing.T) {
	ctx := context.Background()
	db, shareSvc, _, settingSvc := setupShareCodeTestEnv(t)

	// 清空小程序配置
	_ = settingSvc.SaveMp(ctx, systemsetting.MpConfig{
		AppID:     "",
		AppSecret: "",
	})

	prod := model.Product{
		ID:     102,
		Name:   "正常商品",
		Status: 1,
	}
	db.Create(&prod)

	_, err := shareSvc.GetOrCreateProductShareCode(ctx, 102)
	if err == nil || !errors.Is(err, errors.New("未配置小程序 AppID 或 AppSecret，无法生成商品小程序码")) &&
		!strings.Contains(err.Error(), "未配置小程序 AppID 或 AppSecret") {
		t.Fatalf("expected missing mp config error, got: %v", err)
	}
}

// 3. 正常生成、持久化映射与二次调用复用缓存测试
func TestShareCode_SuccessAndCacheReuse(t *testing.T) {
	ctx := context.Background()
	db, shareSvc, mockClient, _ := setupShareCodeTestEnv(t)

	prod := model.Product{
		ID:     201,
		Name:   "测试上架手机",
		Status: 1,
	}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	// 第一次调用：应调用微信生成，保存文件与映射表
	resp1, err := shareSvc.GetOrCreateProductShareCode(ctx, 201)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if resp1.ProductID != 201 || resp1.FileID <= 0 || resp1.URL == "" {
		t.Fatalf("invalid resp1: %+v", resp1)
	}
	if atomic.LoadInt32(&mockClient.qrCodeCalls) != 1 {
		t.Fatalf("expected 1 qrCodeCall, got %d", mockClient.qrCodeCalls)
	}

	// 验证数据表存在映射
	var count int64
	db.Model(&model.ProductShareCode{}).Where("product_id = ?", 201).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 record in product_share_code, got %d", count)
	}

	// 第二次调用：应直接从数据库/文件服务复用，不调用微信
	resp2, err := shareSvc.GetOrCreateProductShareCode(ctx, 201)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if resp2.FileID != resp1.FileID || resp2.URL != resp1.URL {
		t.Fatalf("expected resp2 to match resp1, got resp1=%+v, resp2=%+v", resp1, resp2)
	}
	if atomic.LoadInt32(&mockClient.qrCodeCalls) != 1 {
		t.Fatalf("expected qrCodeCalls to remain 1 after cache hit, got %d", mockClient.qrCodeCalls)
	}
}

func TestShareCode_ConcurrentRequestsShareOneGeneration(t *testing.T) {
	ctx := context.Background()
	db, _, mockClient, settingSvc := setupShareCodeTestEnv(t)
	if err := db.Create(&model.Product{ID: 202, Name: "并发商品", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	var qrCalls int32
	client := &retryMockClient{mock: mockClient, qrFunc: func(context.Context, string, miniappshare.WeChatQRCodeParams) ([]byte, string, error) {
		atomic.AddInt32(&qrCalls, 1)
		once.Do(func() { close(started) })
		<-release
		return testPngBytes, "image/png", nil
	}}
	fileSvc := systemfile.NewService(db)
	service := miniappshare.NewService(db, nil, fileSvc, settingSvc, client)

	type result struct {
		resp *miniappshare.ProductShareCodeResponse
		err  error
	}
	results := make(chan result, 2)
	go func() { resp, err := service.GetOrCreateProductShareCode(ctx, 202); results <- result{resp, err} }()
	<-started
	go func() { resp, err := service.GetOrCreateProductShareCode(ctx, 202); results <- result{resp, err} }()
	time.Sleep(20 * time.Millisecond)
	close(release)
	a, b := <-results, <-results
	if a.err != nil || b.err != nil {
		t.Fatalf("concurrent generation failed: %v / %v", a.err, b.err)
	}
	if a.resp.FileID != b.resp.FileID || a.resp.URL != b.resp.URL {
		t.Fatalf("callers received different winners: %+v / %+v", a.resp, b.resp)
	}
	if calls := atomic.LoadInt32(&qrCalls); calls != 1 {
		t.Fatalf("expected one merged QR generation, got %d", calls)
	}
}

// 4. 微信 Token 失效重试 (40001)
func TestShareCode_TokenInvalidRetry(t *testing.T) {
	ctx := context.Background()
	db, _, mockClient, _ := setupShareCodeTestEnv(t)

	prod := model.Product{
		ID:     301,
		Name:   "测试商品",
		Status: 1,
	}
	db.Create(&prod)

	var qrAttempts int32
	mockClient.mu.Lock()
	mockClient.qrErr = nil
	mockClient.mu.Unlock()

	// 第一次调用返回 40001 无效凭据，重试时返回正常图片
	wrappedClient := &retryMockClient{
		mock: mockClient,
		qrFunc: func(ctx context.Context, token string, params miniappshare.WeChatQRCodeParams) ([]byte, string, error) {
			val := atomic.AddInt32(&qrAttempts, 1)
			if val == 1 {
				return nil, "", &miniappshare.WeChatAPIError{ErrCode: 40001, ErrMsg: "invalid credential"}
			}
			return testPngBytes, "image/png", nil
		},
	}

	fileSvc := systemfile.NewService(db)
	settingSvc := systemsetting.NewService(db, nil)
	_ = settingSvc.SaveMp(ctx, systemsetting.MpConfig{
		AppID:     "wx123456",
		AppSecret: "sec123456",
	})
	retryShareSvc := miniappshare.NewService(db, nil, fileSvc, settingSvc, wrappedClient)

	resp, err := retryShareSvc.GetOrCreateProductShareCode(ctx, 301)
	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if resp.FileID <= 0 {
		t.Fatalf("expected valid fileID, got %d", resp.FileID)
	}
	if atomic.LoadInt32(&qrAttempts) != 2 {
		t.Fatalf("expected 2 qr attempts (1 fail + 1 retry), got %d", qrAttempts)
	}
}

// 5. 微信接口返回业务错误时，禁止保存假图片
func TestShareCode_WeChatErrorDoesNotSaveImage(t *testing.T) {
	ctx := context.Background()
	db, shareSvc, mockClient, _ := setupShareCodeTestEnv(t)

	prod := model.Product{
		ID:     401,
		Name:   "测试商品",
		Status: 1,
	}
	db.Create(&prod)

	mockClient.mu.Lock()
	mockClient.qrErr = &miniappshare.WeChatAPIError{ErrCode: 40013, ErrMsg: "invalid appid"}
	mockClient.mu.Unlock()

	_, err := shareSvc.GetOrCreateProductShareCode(ctx, 401)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// 验证未保存任何文件
	var fileCount int64
	db.Model(&model.SystemFile{}).Where("scene = ?", model.SceneSystemGenerated).Count(&fileCount)
	if fileCount != 0 {
		t.Fatalf("expected 0 system files saved, got %d", fileCount)
	}

	// 验证未写入映射表
	var shareCount int64
	db.Model(&model.ProductShareCode{}).Where("product_id = ?", 401).Count(&shareCount)
	if shareCount != 0 {
		t.Fatalf("expected 0 share codes saved, got %d", shareCount)
	}
}

type retryMockClient struct {
	mock   *mockShareWeChatClient
	qrFunc func(ctx context.Context, token string, params miniappshare.WeChatQRCodeParams) ([]byte, string, error)
}

func (r *retryMockClient) GetAccessToken(ctx context.Context, appID, appSecret string) (string, int, error) {
	return r.mock.GetAccessToken(ctx, appID, appSecret)
}

func (r *retryMockClient) GetUnlimitedQRCode(ctx context.Context, accessToken string, params miniappshare.WeChatQRCodeParams) ([]byte, string, error) {
	if r.qrFunc != nil {
		return r.qrFunc(ctx, accessToken, params)
	}
	return r.mock.GetUnlimitedQRCode(ctx, accessToken, params)
}

// 6. 接口鉴权测试：生成小程序码接口必须登录
func TestShareCode_API_AuthRequired(t *testing.T) {
	_, handler, _, _, _, _ := setupOrderTestApp(t)

	// 未提供 Authorization Header
	req := httptest.NewRequest(http.MethodPost, "/api/miniapp/products/1/share-code", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// 应返回未授权错误 (401 或 code == -1)
	if w.Code == http.StatusOK {
		var resp struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Code != -1 {
			t.Fatalf("expected code -1 for unauthenticated request, got %d", resp.Code)
		}
	} else if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 or 200 with code -1, got status %d", w.Code)
	}
}
