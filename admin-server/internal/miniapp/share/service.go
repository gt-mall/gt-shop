package share

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"golang.org/x/sync/singleflight"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mall/admin_core/internal/model"
	systemfile "mall/admin_core/internal/system/file"
	systemsetting "mall/admin_core/internal/system/setting"
	"mall/admin_core/pkg/cache"
)

const (
	defaultEnvVersion   = "release"
	defaultSceneVersion = "v1"
	productDetailPath   = "pages/product/detail"
)

type Service struct {
	db             *gorm.DB
	cache          cache.Cache
	fileService    *systemfile.Service
	settingService *systemsetting.Service
	wxClient       WeChatShareClient
	tokenMgr       *TokenManager
	generation     singleflight.Group
}

func NewService(
	db *gorm.DB,
	c cache.Cache,
	fileService *systemfile.Service,
	settingService *systemsetting.Service,
	wxClient WeChatShareClient,
) *Service {
	if wxClient == nil {
		wxClient = NewDefaultWeChatShareClient()
	}
	return &Service{
		db:             db,
		cache:          c,
		fileService:    fileService,
		settingService: settingService,
		wxClient:       wxClient,
		tokenMgr:       NewTokenManager(wxClient, c),
	}
}

// GetOrCreateProductShareCode 获取或生成商品分享小程序码
func (s *Service) GetOrCreateProductShareCode(ctx context.Context, productID int64) (*ProductShareCodeResponse, error) {
	if s.db == nil {
		return nil, errors.New("数据库不可用")
	}
	if productID <= 0 {
		return nil, errors.New("无效的商品ID")
	}

	// 1. 校验商品是否存在、未被软删除且处于上架状态 (status = 1)
	var product model.Product
	if err := s.db.WithContext(ctx).Where("id = ? AND status = 1", productID).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("商品不存在或已下架")
		}
		return nil, fmt.Errorf("查询商品失败: %w", err)
	}

	// 2. 读取小程序配置
	mpCfg, err := s.settingService.GetMp(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取小程序配置失败: %w", err)
	}
	appID := strings.TrimSpace(mpCfg.AppID)
	appSecret := strings.TrimSpace(mpCfg.AppSecret)
	if appID == "" || appSecret == "" {
		return nil, errors.New("未配置小程序 AppID 或 AppSecret，无法生成商品小程序码")
	}

	env := defaultEnvVersion
	sceneVersion := defaultSceneVersion

	key := fmt.Sprintf("%s:%s:%d:%s", appID, env, productID, sceneVersion)
	result := s.generation.DoChan(key, func() (any, error) {
		// A disconnected caller must not abandon a successfully uploaded file.
		workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 40*time.Second)
		defer cancel()
		unlock, err := s.lockGeneration(workCtx, key)
		if err != nil {
			return nil, err
		}
		defer unlock()
		return s.generate(workCtx, appID, appSecret, env, sceneVersion, productID)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-result:
		if result.Err != nil {
			return nil, result.Err
		}
		copy := *result.Val.(*ProductShareCodeResponse)
		return &copy, nil
	}
}

// MySQL advisory locks coordinate separate server processes without holding product
// or inventory row locks during the external HTTP request. A dedicated connection
// owns the lock, while the normal pool serves file and mapping queries.
func (s *Service) lockGeneration(ctx context.Context, key string) (func(), error) {
	if s.db.Dialector.Name() != "mysql" {
		return func() {}, nil
	}
	pool, err := s.db.DB()
	if err != nil {
		return nil, err
	}
	if pool.Stats().MaxOpenConnections == 1 {
		return nil, errors.New("商品码生成需要至少两个数据库连接")
	}
	conn, err := pool.Conn(ctx)
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("share:%x", sha256.Sum256([]byte(key)))[:64]
	var locked int
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 10)", name).Scan(&locked); err != nil || locked != 1 {
		conn.Close()
		return nil, errors.New("商品码正在生成，请稍后重试")
	}
	return func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(cleanup, "DO RELEASE_LOCK(?)", name)
		_ = conn.Close()
	}, nil
}

func (s *Service) generate(ctx context.Context, appID, appSecret, env, sceneVersion string, productID int64) (*ProductShareCodeResponse, error) {
	// 3. 检查 product_share_code 表是否已有有效缓存映射
	var shareCode model.ProductShareCode
	findErr := s.db.WithContext(ctx).
		Where("app_id = ? AND env = ? AND product_id = ? AND scene_version = ?", appID, env, productID, sceneVersion).
		First(&shareCode).Error

	if findErr == nil && shareCode.FileID > 0 {
		// 校验关联的系统文件是否依然真实存在
		files, resolveErr := s.fileService.Resolve(ctx, systemfile.IDsRequest{IDs: []int64{shareCode.FileID}})
		if resolveErr == nil && len(files) > 0 && files[0].URL != "" {
			return &ProductShareCodeResponse{
				ProductID: productID,
				FileID:    shareCode.FileID,
				URL:       files[0].URL,
			}, nil
		}
		// 若文件记录已丢失，则继续向下重新生成并更新映射
	}

	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, findErr
	}

	// 4. 调用微信接口生成小程序码（带 Token 失效自动重试）
	qrBytes, mimeType, err := s.fetchQRCodeFromWeChat(ctx, appID, appSecret, productID, env)
	if err != nil {
		return nil, err
	}

	// 5. 保存生成的二维码图片到系统文件服务 (SceneSystemGenerated)
	ext := "png"
	if strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg") {
		ext = "jpg"
	}

	fileResp, err := s.fileService.UploadWithParams(ctx, systemfile.UploadParams{
		Filename:     fmt.Sprintf("product_share_%d.%s", productID, ext),
		Scene:        model.SceneSystemGenerated,
		OwnerType:    model.OwnerTypeSystem,
		OwnerID:      0,
		Visibility:   model.VisibilityPublic,
		Reader:       bytes.NewReader(qrBytes),
		DeclaredSize: int64(len(qrBytes)),
	})
	if err != nil {
		return nil, fmt.Errorf("保存小程序码文件失败: %w", err)
	}

	// If persistence loses a race or fails, discard only this request's unreferenced
	// file. Cleanup uses an independent deadline even if the caller has disconnected.
	keepFile := false
	defer func() {
		if !keepFile {
			cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = s.fileService.DiscardGenerated(cleanup, fileResp.ID)
		}
	}()
	newRecord := model.ProductShareCode{AppID: appID, Env: env, ProductID: productID, SceneVersion: sceneVersion, FileID: fileResp.ID}
	if findErr == nil {
		// Repair a broken mapping only if it has not been replaced meanwhile.
		update := s.db.WithContext(ctx).Model(&model.ProductShareCode{}).
			Where("id = ? AND file_id = ?", shareCode.ID, shareCode.FileID).
			Updates(map[string]any{"file_id": fileResp.ID, "updated_at": time.Now()})
		if update.Error != nil {
			return nil, errors.New("保存商品码映射记录失败")
		}
	} else {
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&newRecord).Error; err != nil {
			return nil, errors.New("保存商品码映射记录失败")
		}
	}
	var winner model.ProductShareCode
	if err := s.db.WithContext(ctx).Where("app_id = ? AND env = ? AND product_id = ? AND scene_version = ?", appID, env, productID, sceneVersion).First(&winner).Error; err != nil {
		return nil, errors.New("读取商品码映射失败")
	}
	keepFile = winner.FileID == fileResp.ID
	files, err := s.fileService.Resolve(ctx, systemfile.IDsRequest{IDs: []int64{winner.FileID}})
	if err != nil || len(files) != 1 || files[0].URL == "" {
		return nil, errors.New("读取商品码图片失败")
	}
	return &ProductShareCodeResponse{ProductID: productID, FileID: winner.FileID, URL: files[0].URL}, nil
}

// fetchQRCodeFromWeChat 从微信获取小程序码，支持 token 失效自动重试一次
func (s *Service) fetchQRCodeFromWeChat(
	ctx context.Context,
	appID, appSecret string,
	productID int64,
	env string,
) ([]byte, string, error) {
	token, err := s.tokenMgr.GetToken(ctx, appID, appSecret)
	if err != nil {
		return nil, "", fmt.Errorf("获取微信 access_token 失败: %w", err)
	}

	params := WeChatQRCodeParams{
		Scene:      fmt.Sprintf("p=%d", productID),
		Page:       productDetailPath,
		CheckPath:  false, // 避免开发/体验阶段因页面尚未发布而报错
		EnvVersion: env,
		Width:      430,
	}

	data, mimeType, err := s.wxClient.GetUnlimitedQRCode(ctx, token, params)
	if err != nil {
		var apiErr *WeChatAPIError
		if errors.As(err, &apiErr) && (apiErr.ErrCode == 40001 || apiErr.ErrCode == 42001) {
			// Token 已在微信服务端过期或被重新生成，立即清除缓存重试一次
			s.tokenMgr.InvalidateToken(ctx, appID)
			newToken, tErr := s.tokenMgr.GetToken(ctx, appID, appSecret)
			if tErr == nil {
				data, mimeType, err = s.wxClient.GetUnlimitedQRCode(ctx, newToken, params)
			}
		}
	}

	if err != nil {
		return nil, "", fmt.Errorf("生成商品小程序码失败: %w", err)
	}

	return data, mimeType, nil
}
