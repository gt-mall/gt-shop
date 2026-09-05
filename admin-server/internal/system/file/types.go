package file

import "time"

const MaxUploadBytes int64 = 10 << 20

// FileResponse includes a computed URL; only storage and relative path are persisted.
type FileResponse struct {
	ID        int64     `json:"id" binding:"required"`
	GroupID   int64     `json:"group_id" binding:"required"`
	Storage   string    `json:"storage" binding:"required"`
	Name      string    `json:"name" binding:"required"`
	Path      string    `json:"path" binding:"required"`
	Ext       string    `json:"ext" binding:"required"`
	Size      int64     `json:"size" binding:"required"`
	Width     int       `json:"width" binding:"required"`
	Height    int       `json:"height" binding:"required"`
	URL       string    `json:"url" binding:"required"`
	CreatedAt time.Time `json:"created_at" binding:"required"`
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
