package dict_service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"mall/admin_core/internal/model"
)

var (
	ErrDatabaseUnavailable = errors.New("数据库不可用")
	ErrNotFound            = errors.New("服务保障不存在")
	ErrNameRequired        = errors.New("服务保障名称不能为空")
	ErrNameTooLong         = errors.New("服务保障名称最多64个字符")
	ErrNameExists          = errors.New("服务保障名称已存在")
	ErrDescriptionTooLong  = errors.New("服务保障说明最多255个字符")
	ErrInvalidStatus       = errors.New("状态值无效")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, req DictServiceListRequest) ([]ProductDictServiceRow, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, ErrDatabaseUnavailable
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}

	query := s.db.WithContext(ctx).Model(&model.ProductDictService{})

	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		likePattern := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", likePattern, likePattern)
	}

	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	var list []model.ProductDictService
	if err := query.Order("sort ASC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]ProductDictServiceRow, 0, len(list))
	for _, item := range list {
		rows = append(rows, toRow(item))
	}

	return rows, total, nil
}

func (s *Service) Create(ctx context.Context, req DictServiceCreateRequest) (ProductDictServiceRow, error) {
	if s == nil || s.db == nil {
		return ProductDictServiceRow{}, ErrDatabaseUnavailable
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ProductDictServiceRow{}, ErrNameRequired
	}
	if utf8.RuneCountInString(name) > 64 {
		return ProductDictServiceRow{}, ErrNameTooLong
	}

	desc := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(desc) > 255 {
		return ProductDictServiceRow{}, ErrDescriptionTooLong
	}

	status := int8(1)
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			return ProductDictServiceRow{}, ErrInvalidStatus
		}
		status = *req.Status
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&model.ProductDictService{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return ProductDictServiceRow{}, err
	}
	if count > 0 {
		return ProductDictServiceRow{}, ErrNameExists
	}

	record := model.ProductDictService{
		Name:        name,
		Description: desc,
		Sort:        req.Sort,
		Status:      status,
	}

	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return ProductDictServiceRow{}, err
	}

	return toRow(record), nil
}

func (s *Service) Update(ctx context.Context, id int64, req DictServiceUpdateRequest) (ProductDictServiceRow, error) {
	if s == nil || s.db == nil {
		return ProductDictServiceRow{}, ErrDatabaseUnavailable
	}
	if id <= 0 {
		return ProductDictServiceRow{}, ErrNotFound
	}

	var existing model.ProductDictService
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ProductDictServiceRow{}, ErrNotFound
		}
		return ProductDictServiceRow{}, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ProductDictServiceRow{}, ErrNameRequired
	}
	if utf8.RuneCountInString(name) > 64 {
		return ProductDictServiceRow{}, ErrNameTooLong
	}

	desc := strings.TrimSpace(req.Description)
	if utf8.RuneCountInString(desc) > 255 {
		return ProductDictServiceRow{}, ErrDescriptionTooLong
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&model.ProductDictService{}).
		Where("name = ? AND id != ?", name, id).Count(&count).Error; err != nil {
		return ProductDictServiceRow{}, err
	}
	if count > 0 {
		return ProductDictServiceRow{}, ErrNameExists
	}

	existing.Name = name
	existing.Description = desc
	if req.Sort != nil {
		existing.Sort = *req.Sort
	}
	if req.Status != nil {
		if *req.Status != 0 && *req.Status != 1 {
			return ProductDictServiceRow{}, ErrInvalidStatus
		}
		existing.Status = *req.Status
	}

	if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return ProductDictServiceRow{}, err
	}

	return toRow(existing), nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if s == nil || s.db == nil {
		return ErrDatabaseUnavailable
	}
	if id <= 0 {
		return ErrNotFound
	}

	var existing model.ProductDictService
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}

	return s.db.WithContext(ctx).Delete(&existing).Error
}

func toRow(item model.ProductDictService) ProductDictServiceRow {
	return ProductDictServiceRow{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Sort:        item.Sort,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}
