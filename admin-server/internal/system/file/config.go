package file

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"mall/admin_core/internal/model"
)

func DefaultConfig() StorageConfig {
	return StorageConfig{DefaultStorage: "local", Local: LocalConfig{Path: "uploads"}}
}
func (s *Service) loadConfig(ctx context.Context) (StorageConfig, error) {
	config := DefaultConfig()
	if s.dao.db == nil {
		return config, ErrUnavailable
	}
	var row model.SystemConfig
	err := s.dao.query(ctx).Where("config_key = ?", "file_storage").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	err = json.Unmarshal([]byte(row.Value), &config)
	return config, err
}
func (s *Service) Config(ctx context.Context) (StorageConfig, error) {
	config, err := s.loadConfig(ctx)
	for _, p := range []*ObjectConfig{&config.OSS, &config.COS, &config.S3} {
		p.SecretSet = p.AccessKeySecret != ""
		p.AccessKeySecret = ""
	}
	return config, err
}
func validURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}
func (s *Service) SaveConfig(ctx context.Context, config StorageConfig) error {
	old, err := s.loadConfig(ctx)
	if err != nil {
		return err
	}
	config.Local.Path = strings.TrimSpace(config.Local.Path)
	config.Local.Domain = strings.TrimRight(strings.TrimSpace(config.Local.Domain), "/")
	if config.Local.Path == "" || (config.Local.Domain != "" && !validURL(config.Local.Domain)) {
		return ErrConfig
	}
	previous := []*ObjectConfig{&old.OSS, &old.COS, &old.S3}
	for i, p := range []*ObjectConfig{&config.OSS, &config.COS, &config.S3} {
		p.Endpoint = strings.TrimRight(strings.TrimSpace(p.Endpoint), "/")
		p.Domain = strings.TrimRight(strings.TrimSpace(p.Domain), "/")
		p.Bucket = strings.TrimSpace(p.Bucket)
		p.Region = strings.TrimSpace(p.Region)
		p.AccessKeyID = strings.TrimSpace(p.AccessKeyID)
		if p.AccessKeySecret == "" {
			p.AccessKeySecret = previous[i].AccessKeySecret
		}
		p.SecretSet = false
		if (p.Domain != "" && !validURL(p.Domain)) || (p.Endpoint != "" && !validURL(p.Endpoint)) {
			return ErrConfig
		}
	}
	used, err := s.dao.UsedStorages(ctx)
	if err != nil {
		return err
	}
	// 历史文件使用的存储配置必须仍然完整，不能被默认存储的切换覆盖。
	for _, kind := range append(used, config.DefaultStorage) {
		if _, err := NewStorage(config, kind); err != nil {
			return ErrConfig
		}
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := model.SystemConfig{ConfigKey: "file_storage", Value: string(data)}
	return s.dao.query(ctx).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "config_key"}}, DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"})}).Create(&row).Error
}
