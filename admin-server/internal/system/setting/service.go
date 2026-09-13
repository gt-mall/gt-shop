package setting

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/system/file"
	"mall/admin_core/pkg/cache"
)

const (
	ConfigKeyShopBasic = "shop_basic"
	ConfigKeySiteBasic = "site_basic"
	ConfigKeyMpBasic   = "mp_basic"
)

var (
	ErrUnavailable = errors.New("database unavailable")
)

type Service struct {
	db    *gorm.DB
	cache cache.Cache
}

func NewService(db *gorm.DB, caches ...cache.Cache) *Service {
	var c cache.Cache = cache.NewNoop()
	if len(caches) > 0 && caches[0] != nil {
		c = caches[0]
	}
	return &Service{db: db, cache: c}
}

func DefaultShopConfig() ShopBasicConfig {
	return ShopBasicConfig{
		Name:        "Mall 商城",
		ServiceName: "商城客服",
		Description: "全场景高性能电商系统",
	}
}

func DefaultSiteConfig() SiteBasicConfig {
	return SiteBasicConfig{
		Title:       "Mall 管理后台",
		Keywords:    "Mall,电商系统,管理后台",
		Description: "Mall 电商管理系统",
		Copyright:   "© 2026 Mall. All rights reserved.",
	}
}

func DefaultMpConfig() MpConfig {
	return MpConfig{
		Name: "Mall 小程序",
	}
}

func (s *Service) query(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx)
}

func (s *Service) resolveFiles(ctx context.Context, config *ShopBasicConfig) {
	if s.db == nil || config == nil {
		return
	}
	ids := make([]int64, 0, 3)
	if config.LogoID > 0 {
		ids = append(ids, config.LogoID)
	}
	if config.FaviconID > 0 {
		ids = append(ids, config.FaviconID)
	}
	if config.ServiceWechatQRCodeID > 0 {
		ids = append(ids, config.ServiceWechatQRCodeID)
	}
	if len(ids) == 0 {
		return
	}
	files, err := file.NewService(s.db).Resolve(ctx, file.IDsRequest{IDs: ids})
	if err != nil {
		return
	}
	for _, f := range files {
		if config.LogoID > 0 && f.ID == config.LogoID {
			config.Logo = f.URL
		}
		if config.FaviconID > 0 && f.ID == config.FaviconID {
			config.Favicon = f.URL
		}
		if config.ServiceWechatQRCodeID > 0 && f.ID == config.ServiceWechatQRCodeID {
			config.ServiceWechatQRCode = f.URL
		}
	}
}

func (s *Service) GetShop(ctx context.Context) (ShopBasicConfig, error) {
	key := cache.SystemConfigKey(ConfigKeyShopBasic)
	var cached ShopBasicConfig
	if s.cache != nil {
		if hit, err := s.cache.Get(ctx, key, &cached); err != nil {
			log.Printf("[Cache] get %s error: %v", key, err)
		} else if hit {
			return cached, nil
		}
	}

	config := DefaultShopConfig()
	if s.db == nil {
		return config, ErrUnavailable
	}
	var row model.SystemConfig
	err := s.query(ctx).Where("config_key = ?", ConfigKeyShopBasic).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if s.cache != nil {
			_ = s.cache.Set(ctx, key, config, cache.TTLSystemConfig)
		}
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
		return config, err
	}
	s.resolveFiles(ctx, &config)
	if s.cache != nil {
		if err := s.cache.Set(ctx, key, config, cache.TTLSystemConfig); err != nil {
			log.Printf("[Cache] set %s error: %v", key, err)
		}
	}
	return config, nil
}

func (s *Service) SaveShop(ctx context.Context, config ShopBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	config.Name = strings.TrimSpace(config.Name)
	config.ServiceName = strings.TrimSpace(config.ServiceName)
	config.ServicePhone = strings.TrimSpace(config.ServicePhone)
	config.ServiceWechat = strings.TrimSpace(config.ServiceWechat)
	config.Description = strings.TrimSpace(config.Description)
	if config.LogoID == 0 {
		config.Logo = ""
	}
	if config.FaviconID == 0 {
		config.Favicon = ""
	}
	if config.ServiceWechatQRCodeID == 0 {
		config.ServiceWechatQRCode = ""
	}
	s.resolveFiles(ctx, &config)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyShopBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil {
		key := cache.SystemConfigKey(ConfigKeyShopBasic)
		if err := s.cache.Delete(ctx, key); err != nil {
			log.Printf("[Cache] delete %s error: %v", key, err)
		}
	}
	return nil
}

func (s *Service) GetSite(ctx context.Context) (SiteBasicConfig, error) {
	key := cache.SystemConfigKey(ConfigKeySiteBasic)
	var cached SiteBasicConfig
	if s.cache != nil {
		if hit, err := s.cache.Get(ctx, key, &cached); err != nil {
			log.Printf("[Cache] get %s error: %v", key, err)
		} else if hit {
			return cached, nil
		}
	}

	config := DefaultSiteConfig()
	if s.db == nil {
		return config, ErrUnavailable
	}
	var row model.SystemConfig
	err := s.query(ctx).Where("config_key = ?", ConfigKeySiteBasic).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if s.cache != nil {
			_ = s.cache.Set(ctx, key, config, cache.TTLSystemConfig)
		}
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
		return config, err
	}
	if s.cache != nil {
		if err := s.cache.Set(ctx, key, config, cache.TTLSystemConfig); err != nil {
			log.Printf("[Cache] set %s error: %v", key, err)
		}
	}
	return config, nil
}

func (s *Service) SaveSite(ctx context.Context, config SiteBasicConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	config.Title = strings.TrimSpace(config.Title)
	config.Keywords = strings.TrimSpace(config.Keywords)
	config.Description = strings.TrimSpace(config.Description)
	config.ICP = strings.TrimSpace(config.ICP)
	config.Copyright = strings.TrimSpace(config.Copyright)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeySiteBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil {
		key := cache.SystemConfigKey(ConfigKeySiteBasic)
		if err := s.cache.Delete(ctx, key); err != nil {
			log.Printf("[Cache] delete %s error: %v", key, err)
		}
	}
	return nil
}

func (s *Service) resolveMpFiles(ctx context.Context, config *MpConfig) {
	if s.db == nil || config == nil || config.QRCodeID <= 0 {
		return
	}
	files, err := file.NewService(s.db).Resolve(ctx, file.IDsRequest{IDs: []int64{config.QRCodeID}})
	if err != nil {
		return
	}
	for _, f := range files {
		if f.ID == config.QRCodeID {
			config.QRCode = f.URL
		}
	}
}

func (s *Service) GetMp(ctx context.Context) (MpConfig, error) {
	key := cache.SystemConfigKey(ConfigKeyMpBasic)
	var cached MpConfig
	if s.cache != nil {
		if hit, err := s.cache.Get(ctx, key, &cached); err != nil {
			log.Printf("[Cache] get %s error: %v", key, err)
		} else if hit {
			return cached, nil
		}
	}

	config := DefaultMpConfig()
	if s.db == nil {
		return config, ErrUnavailable
	}
	var row model.SystemConfig
	err := s.query(ctx).Where("config_key = ?", ConfigKeyMpBasic).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if s.cache != nil {
			_ = s.cache.Set(ctx, key, config, cache.TTLSystemConfig)
		}
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
		return config, err
	}
	s.resolveMpFiles(ctx, &config)
	if s.cache != nil {
		if err := s.cache.Set(ctx, key, config, cache.TTLSystemConfig); err != nil {
			log.Printf("[Cache] set %s error: %v", key, err)
		}
	}
	return config, nil
}

func (s *Service) SaveMp(ctx context.Context, config MpConfig) error {
	if s.db == nil {
		return ErrUnavailable
	}
	config.Name = strings.TrimSpace(config.Name)
	config.OriginalID = strings.TrimSpace(config.OriginalID)
	config.AppID = strings.TrimSpace(config.AppID)
	config.AppSecret = strings.TrimSpace(config.AppSecret)
	config.RequestDomain = strings.TrimSpace(config.RequestDomain)
	config.SocketDomain = strings.TrimSpace(config.SocketDomain)
	config.UploadDomain = strings.TrimSpace(config.UploadDomain)
	config.DownloadDomain = strings.TrimSpace(config.DownloadDomain)
	config.UDPDomain = strings.TrimSpace(config.UDPDomain)
	config.TCPDomain = strings.TrimSpace(config.TCPDomain)
	config.BusinessDomain = strings.TrimSpace(config.BusinessDomain)
	config.UploadPrivateKey = strings.TrimSpace(config.UploadPrivateKey)
	if config.QRCodeID == 0 {
		config.QRCode = ""
	}
	s.resolveMpFiles(ctx, &config)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyMpBasic,
		Value:     string(data),
	}
	if err := s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error; err != nil {
		return err
	}

	if s.cache != nil {
		key := cache.SystemConfigKey(ConfigKeyMpBasic)
		if err := s.cache.Delete(ctx, key); err != nil {
			log.Printf("[Cache] delete %s error: %v", key, err)
		}
	}
	return nil
}
