package file

import (
	"errors"
	"io"
	"time"
)

const MaxUploadBytes int64 = 500 << 20 // Max global buffer ceiling (500 MB for video)

var (
	ErrUnavailable      = errors.New("数据库不可用")
	ErrImage            = errors.New("仅支持有效的 JPG、PNG、GIF、WebP 图片，最大 10 MB、2500 万像素")
	ErrConfig           = errors.New("存储配置不完整或地址无效，请检查默认及历史文件使用的存储配置")
	ErrGroupNotFound    = errors.New("素材分组不存在")
	ErrName             = errors.New("请输入有效名称")
	ErrStorage          = errors.New("文件存储失败，请检查存储配置及服务端读写权限")
	ErrFileSizeExceeded = errors.New("文件大小超出限制")
	ErrFileTypeInvalid  = errors.New("不支持的文件类型或格式")
	ErrFileTypeMismatch = errors.New("文件内容与扩展名不匹配")
	ErrFileInUse        = errors.New("文件已被业务引用，禁止删除")
	ErrFileForbidden    = errors.New("无权访问该私有文件")
	ErrInvalidSignature = errors.New("签名无效或已过期")
)

// UploadParams defines server-controlled parameters for uploading a file resource.
type UploadParams struct {
	Filename     string
	GroupID      int64
	Scene        string
	OwnerType    string
	OwnerID      int64
	Visibility   int8
	Reader       io.Reader
	DeclaredSize int64
}

// FileResponse includes a computed URL; only storage and relative path are persisted.
type FileResponse struct {
	ID           int64     `json:"id" binding:"required"`
	GroupID      int64     `json:"group_id" binding:"required"`
	Storage      string    `json:"storage" binding:"required"`
	Name         string    `json:"name" binding:"required"`
	OriginalName string    `json:"original_name"`
	Path         string    `json:"path" binding:"required"`
	Ext          string    `json:"ext" binding:"required"`
	MIMEType     string    `json:"mime_type"`
	FileType     string    `json:"file_type"`
	Scene        string    `json:"scene"`
	OwnerType    string    `json:"owner_type"`
	OwnerID      int64     `json:"owner_id"`
	Visibility   int8      `json:"visibility"`
	Size         int64     `json:"size" binding:"required"`
	Width        int       `json:"width" binding:"required"`
	Height       int       `json:"height" binding:"required"`
	DurationMS   int64     `json:"duration_ms"`
	CoverFileID  int64     `json:"cover_file_id"`
	CoverURL     string    `json:"cover_url"`
	URL          string    `json:"url" binding:"required"`
	CreatedAt    time.Time `json:"created_at" binding:"required"`
} // @name FileRow

type GroupResponse struct {
	ID          int64  `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Count       int64  `json:"count" binding:"required"`
	StorageSize int64  `json:"storage_size" binding:"required"`
} // @name FileGroupRow

type GroupCountsResponse struct {
	Total                int64           `json:"total" binding:"required"`
	TotalStorageSize     int64           `json:"total_storage_size" binding:"required"`
	Ungrouped            int64           `json:"ungrouped" binding:"required"`
	UngroupedStorageSize int64           `json:"ungrouped_storage_size" binding:"required"`
	Groups               []GroupResponse `json:"groups" binding:"required"`
} // @name FileGroupCounts

type ListRequest struct {
	Page     int    `json:"page" binding:"omitempty,min=1"`
	PageSize int    `json:"page_size" binding:"omitempty,min=1,max=100"`
	GroupID  *int64 `json:"group_id" binding:"omitempty,gte=0"`
	Keyword  string `json:"keyword" binding:"max=255"`
	FileType string `json:"file_type" binding:"max=32"`
} // @name FileListRequest

type IDsRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1,max=100,dive,gt=0"`
} // @name FileIDsRequest

type RenameRequest struct {
	ID   int64  `json:"id" binding:"required,gt=0"`
	Name string `json:"name" binding:"required,max=255"`
} // @name FileRenameRequest

type MoveRequest struct {
	IDs     []int64 `json:"ids" binding:"required,min=1,max=100,dive,gt=0"`
	GroupID int64   `json:"group_id" binding:"gte=0"`
} // @name FileMoveRequest

type GroupSaveRequest struct {
	ID   int64  `json:"id" binding:"gte=0"`
	Name string `json:"name" binding:"required,max=64"`
} // @name FileGroupSaveRequest

type GroupIDRequest struct {
	ID int64 `json:"id" binding:"required,gt=0"`
} // @name FileGroupIDRequest

type LocalConfig struct {
	Path   string `json:"path" binding:"required"`
	Domain string `json:"domain"`
} // @name FileLocalConfig

type ObjectConfig struct {
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
	SecretSet       bool   `json:"secret_set"`
	Domain          string `json:"domain"`
	UsePathStyle    bool   `json:"use_path_style"`
} // @name FileObjectConfig

type StorageConfig struct {
	DefaultStorage string       `json:"default_storage" binding:"required,oneof=local oss cos s3"`
	Local          LocalConfig  `json:"local" binding:"required"`
	OSS            ObjectConfig `json:"oss" binding:"required"`
	COS            ObjectConfig `json:"cos" binding:"required"`
	S3             ObjectConfig `json:"s3" binding:"required"`
} // @name FileStorageConfig
