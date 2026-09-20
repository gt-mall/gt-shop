package model

import "time"

// ProductShareCode 商品分享小程序码持久化映射表
type ProductShareCode struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	AppID        string    `gorm:"size:64;not null;uniqueIndex:uk_share_code_identity,priority:1"`
	Env          string    `gorm:"size:32;not null;default:'release';uniqueIndex:uk_share_code_identity,priority:2"`
	ProductID    int64     `gorm:"not null;uniqueIndex:uk_share_code_identity,priority:3;index:idx_share_code_product"`
	SceneVersion string    `gorm:"size:32;not null;default:'v1';uniqueIndex:uk_share_code_identity,priority:4"`
	FileID       int64     `gorm:"not null;index:idx_share_code_file"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (ProductShareCode) TableName() string {
	return "product_share_code"
}
