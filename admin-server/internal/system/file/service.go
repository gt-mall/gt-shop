package file

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
	"gorm.io/gorm"
	"mall/admin_core/internal/model"
)

var (
	ErrUnavailable   = errors.New("数据库不可用")
	ErrImage         = errors.New("仅支持有效的 JPG、PNG、GIF、WebP 图片，最大 10 MB、2500 万像素")
	ErrConfig        = errors.New("存储配置不完整或地址无效，请检查默认及历史文件使用的存储配置")
	ErrGroupNotFound = errors.New("素材分组不存在")
	ErrName          = errors.New("请输入有效名称")
	ErrStorage       = errors.New("文件存储失败，请检查存储配置及服务端读写权限")
)

type Service struct{ dao *DAO }

func NewService(db *gorm.DB) *Service { return &Service{dao: &DAO{db: db}} }

func (s *Service) Upload(ctx context.Context, name string, groupID int64, reader io.Reader) (FileResponse, error) {
	if s.dao.db == nil {
		return FileResponse{}, ErrUnavailable
	}
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 255 || groupID < 0 {
		return FileResponse{}, ErrName
	}
	data, err := io.ReadAll(io.LimitReader(reader, MaxUploadBytes+1))
	if err != nil {
		return FileResponse{}, ErrImage
	}
	meta, format, err := image.DecodeConfig(bytes.NewReader(data))
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
	formats := map[string]string{"jpg": "jpeg", "jpeg": "jpeg", "png": "png", "gif": "gif", "webp": "webp"}
	if err != nil || int64(len(data)) > MaxUploadBytes || meta.Width <= 0 || meta.Height <= 0 || meta.Width > 25000000/meta.Height || formats[ext] != format {
		return FileResponse{}, ErrImage
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return FileResponse{}, ErrImage
	}
	config, err := s.loadConfig(ctx)
	if err != nil {
		return FileResponse{}, err
	}
	driver, err := NewStorage(config, config.DefaultStorage)
	if err != nil {
		return FileResponse{}, ErrConfig
	}
	key := time.Now().Format("2006/01") + "/" + uuid.NewString() + "." + ext
	if err := driver.Put(ctx, key, data, "image/"+format); err != nil {
		return FileResponse{}, ErrStorage
	}
	row := model.SystemFile{GroupID: groupID, Storage: config.DefaultStorage, Name: name, Path: key, Ext: ext, Size: int64(len(data)), Width: meta.Width, Height: meta.Height}
	if err := s.dao.Create(ctx, &row); err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = driver.Delete(cleanup, key)
		return FileResponse{}, err
	}
	return s.toResponse(ctx, row, config), nil
}
func (s *Service) toResponse(ctx context.Context, row model.SystemFile, config StorageConfig) FileResponse {
	var imageURL string
	if driver, err := NewStorage(config, row.Storage); err == nil {
		imageURL, _ = driver.URL(ctx, row.Path)
	}
	return FileResponse{ID: row.ID, GroupID: row.GroupID, Storage: row.Storage, Name: row.Name, Path: row.Path, Ext: row.Ext, Size: row.Size, Width: row.Width, Height: row.Height, URL: imageURL, CreatedAt: row.CreatedAt}
}
func (s *Service) List(ctx context.Context, req ListRequest) ([]FileResponse, int64, error) {
	config, err := s.loadConfig(ctx)
	if err != nil {
		return nil, 0, err
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 25
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	rows, total, err := s.dao.List(ctx, req)
	if err != nil {
		return nil, 0, err
	}
	result := make([]FileResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, s.toResponse(ctx, row, config))
	}
	return result, total, nil
}
func (s *Service) Resolve(ctx context.Context, req IDsRequest) ([]FileResponse, error) {
	config, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.dao.Resolve(ctx, req.IDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]model.SystemFile, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	result := make([]FileResponse, 0, len(rows))
	for _, id := range req.IDs {
		if row, ok := byID[id]; ok {
			result = append(result, s.toResponse(ctx, row, config))
		}
	}
	return result, nil
}
func (s *Service) Groups(ctx context.Context) ([]GroupResponse, error) {
	if s.dao.db == nil {
		return nil, ErrUnavailable
	}
	return s.dao.Groups(ctx)
}
func (s *Service) GroupCounts(ctx context.Context) (GroupCountsResponse, error) {
	if s.dao.db == nil {
		return GroupCountsResponse{}, ErrUnavailable
	}
	return s.dao.GroupCounts(ctx)
}
func (s *Service) SaveGroup(ctx context.Context, req GroupSaveRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 64 {
		return ErrName
	}
	return s.dao.SaveGroup(ctx, req)
}
func (s *Service) DeleteGroup(ctx context.Context, req GroupIDRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	return s.dao.DeleteGroup(ctx, req.ID)
}
func (s *Service) Rename(ctx context.Context, req RenameRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || utf8.RuneCountInString(req.Name) > 255 {
		return ErrName
	}
	var file model.SystemFile
	if err := s.dao.query(ctx).Select("id, ext").First(&file, req.ID).Error; err != nil {
		return err
	}
	if file.Ext != "" {
		suffix := "." + strings.ToLower(file.Ext)
		baseName := req.Name
		if ext := path.Ext(baseName); ext != "" {
			baseName = strings.TrimSuffix(baseName, ext)
		}
		baseName = strings.TrimSpace(baseName)
		if baseName == "" {
			return ErrName
		}
		req.Name = baseName + suffix
	}
	return s.dao.Rename(ctx, req)
}
func (s *Service) Move(ctx context.Context, req MoveRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	return s.dao.Move(ctx, req)
}
func (s *Service) Delete(ctx context.Context, req IDsRequest) error {
	if s.dao.db == nil {
		return ErrUnavailable
	}
	return s.dao.Delete(ctx, req.IDs)
}
func (s *Service) OpenLocal(ctx context.Context, key string) (*os.File, error) {
	if !ValidPath(key) {
		return nil, os.ErrNotExist
	}
	config, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	var row model.SystemFile
	if err := s.dao.query(ctx).Unscoped().Where("storage = ? AND path = ?", "local", key).First(&row).Error; err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(config.Local.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.Open(key)
}
