package setting

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"mall/admin_core/internal/model"
)

const (
	ConfigKeyShopBasic = "shop_basic"
	ConfigKeySiteBasic = "site_basic"
)

var (
	ErrUnavailable = errors.New("database unavailable")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
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

func (s *Service) query(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx)
}

func (s *Service) GetShop(ctx context.Context) (ShopBasicConfig, error) {
	config := DefaultShopConfig()
	if s.db == nil {
		return config, ErrUnavailable
	}
	var row model.SystemConfig
	err := s.query(ctx).Where("config_key = ?", ConfigKeyShopBasic).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
		return config, err
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

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{
		ConfigKey: ConfigKeyShopBasic,
		Value:     string(data),
	}
	return s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error
}

func (s *Service) GetSite(ctx context.Context) (SiteBasicConfig, error) {
	config := DefaultSiteConfig()
	if s.db == nil {
		return config, ErrUnavailable
	}
	var row model.SystemConfig
	err := s.query(ctx).Where("config_key = ?", ConfigKeySiteBasic).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
		return config, err
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
	return s.query(ctx).
		Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "config_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&row).Error
}
