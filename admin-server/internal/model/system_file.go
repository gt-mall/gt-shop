package model

import (
	"gorm.io/gorm"
	"time"
)

type SystemFile struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	GroupID   int64          `gorm:"not null;default:0;index:idx_file_group_stat,priority:2;index"`
	Storage   string         `gorm:"size:16;not null"`
	Name      string         `gorm:"size:255;not null"`
	Path      string         `gorm:"size:255;not null;uniqueIndex:idx_file_storage_path"`
	Ext       string         `gorm:"size:10;not null"`
	Size      int64          `gorm:"not null;index:idx_file_group_stat,priority:3"`
	Width     int            `gorm:"not null"`
	Height    int            `gorm:"not null"`
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index:idx_file_group_stat,priority:1;index"`
}

func (SystemFile) TableName() string { return "system_file" }

type SystemFileGroup struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"size:64;not null;uniqueIndex"`
	CreatedAt time.Time
}

func (SystemFileGroup) TableName() string { return "system_file_group" }

type SystemConfig struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	ConfigKey string `gorm:"size:64;not null;uniqueIndex"`
	Value     string `gorm:"type:json;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (SystemConfig) TableName() string { return "system_config" }
