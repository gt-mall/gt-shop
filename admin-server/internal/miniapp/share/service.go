package share

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

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

	// 6. 写入或更新持久化映射记录 (并发安全处理)
	newRecord := model.ProductShareCode{
		AppID:        appID,
		Env:          env,
		ProductID:    productID,
		SceneVersion: sceneVersion,
		FileID:       fileResp.ID,
	}

	// 使用 Upsert 机制保证高并发或唯一索引冲突时的安全性
	err = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "app_id"},
			{Name: "env"},
			{Name: "product_id"},
			{Name: "scene_version"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"file_id", "updated_at"}),
	}).Create(&newRecord).Error
	if err != nil {
		// 若写入失败，记录补偿或重新查询
		var existing model.ProductShareCode
		if qErr := s.db.WithContext(ctx).
			Where("app_id = ? AND env = ? AND product_id = ? AND scene_version = ?", appID, env, productID, sceneVersion).
			First(&existing).Error; qErr == nil {
			return &ProductShareCodeResponse{
				ProductID: productID,
				FileID:    existing.FileID,
				URL:       fileResp.URL,
			}, nil
		}
		return nil, fmt.Errorf("保存商品码映射记录失败: %w", err)
	}

	return &ProductShareCodeResponse{
		ProductID: productID,
		FileID:    fileResp.ID,
		URL:       fileResp.URL,
	}, nil
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
