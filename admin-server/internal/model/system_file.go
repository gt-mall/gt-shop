package model

import (
	"time"

	"gorm.io/gorm"
)

// File Types
const (
	FileTypeImage    = "image"
	FileTypeVideo    = "video"
	FileTypeAudio    = "audio"
	FileTypeDocument = "document"
	FileTypeArchive  = "archive"
	FileTypeOther    = "other"
)

// Scenes
const (
	SceneAdminMaterial   = "admin_material"
	SceneAfterSaleProof  = "after_sale_proof"
	SceneMemberAvatar    = "member_avatar"
	SceneSystemGenerated = "system_generated"
	SceneReview          = "review"
)

// Owner Types
const (
	OwnerTypeAdmin  = "admin"
	OwnerTypeMember = "member"
	OwnerTypeSystem = "system"
)

// Visibility
const (
	VisibilityPublic  int8 = 1
	VisibilityPrivate int8 = 2
)

type SystemFile struct {
	ID           int64          `gorm:"primaryKey;autoIncrement"`
	GroupID      int64          `gorm:"not null;default:0;index:idx_file_group_stat,priority:2;index"`
	Storage      string         `gorm:"size:16;not null"`
	Name         string         `gorm:"size:255;not null"`
	OriginalName string         `gorm:"size:255;not null;default:''"`
	Path         string         `gorm:"size:255;not null;uniqueIndex:idx_file_storage_path"`
	Ext          string         `gorm:"size:10;not null"`
	MIMEType     string         `gorm:"column:mime_type;size:128;not null;default:''"`
	FileType     string         `gorm:"column:file_type;size:32;not null;default:'image';index:idx_file_scene_type,priority:2"`
	Scene        string         `gorm:"size:64;not null;default:'admin_material';index:idx_file_scene_type,priority:1"`
	OwnerType    string         `gorm:"column:owner_type;size:32;not null;default:'admin';index:idx_file_owner,priority:1"`
	OwnerID      int64          `gorm:"column:owner_id;not null;default:0;index:idx_file_owner,priority:2"`
	Visibility   int8           `gorm:"not null;default:1"`
	Size         int64          `gorm:"not null;index:idx_file_group_stat,priority:3"`
	Width        int            `gorm:"not null;default:0"`
	Height       int            `gorm:"not null;default:0"`
	DurationMS   int64          `gorm:"column:duration_ms;not null;default:0"`
	CoverFileID  int64          `gorm:"column:cover_file_id;not null;default:0"`
	CreatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index:idx_file_group_stat,priority:1;index"`
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
