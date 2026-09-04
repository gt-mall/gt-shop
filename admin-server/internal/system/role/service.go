package role

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"mall/admin_core/internal/model"
)

var (
	ErrDatabaseUnavailable = errors.New("database unavailable")
	ErrInvalidInput        = errors.New("invalid role input")
	ErrRoleNotFound        = errors.New("role not found")
	ErrRoleCodeExists      = errors.New("role code already exists")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, request CreateRequest) (RoleResponse, error) {
	db, err := s.database()
	if err != nil {
		return RoleResponse{}, err
	}

	name, err := validateName(request.Name)
	if err != nil {
		return RoleResponse{}, err
	}
	code, err := validateCode(request.Code)
	if err != nil {
		return RoleResponse{}, err
	}
	status, err := normalizeStatus(request.Status)
	if err != nil {
		return RoleResponse{}, err
	}
	sort, err := normalizeSort(request.Sort)
	if err != nil {
		return RoleResponse{}, err
	}
	remark, err := validateRemark(request.Remark)
	if err != nil {
		return RoleResponse{}, err
	}

	var existingCount int64
	if err := db.WithContext(ctx).Model(&model.SystemRole{}).Where("code = ?", code).Count(&existingCount).Error; err != nil {
		return RoleResponse{}, fmt.Errorf("check role code: %w", err)
	}
	if existingCount > 0 {
		return RoleResponse{}, ErrRoleCodeExists
	}

	item := model.SystemRole{
		Name:   name,
		Code:   code,
		Status: status,
		Sort:   sort,
		Remark: remark,
	}
	if err := db.WithContext(ctx).Create(&item).Error; err != nil {
		if isDuplicateKey(err) {
			return RoleResponse{}, ErrRoleCodeExists
		}
		return RoleResponse{}, fmt.Errorf("create role: %w", err)
	}

	return toResponse(item), nil
}

func (s *Service) List(ctx context.Context, request ListRequest) ([]RoleResponse, int, error) {
	db, err := s.database()
	if err != nil {
		return nil, 0, err
	}

	page := request.Page
	if page < 1 {
		page = 1
	}
	pageSize := request.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := db.WithContext(ctx).Model(&model.SystemRole{})
	if keyword := strings.TrimSpace(request.Keyword); keyword != "" {
		keyword = "%" + keyword + "%"
		query = query.Where("(name LIKE ? OR code LIKE ?)", keyword, keyword)
	}
	if request.Status != nil {
		if *request.Status != 0 && *request.Status != 1 {
			return nil, 0, fmt.Errorf("%w: status must be 0 or 1", ErrInvalidInput)
		}
		query = query.Where("status = ?", *request.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count roles: %w", err)
	}

	var items []model.SystemRole
	if err := query.Order("sort ASC").Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list roles: %w", err)
	}

	result := make([]RoleResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toResponse(item))
	}
	return result, int(total), nil
}

func (s *Service) Get(ctx context.Context, id int64) (RoleResponse, error) {
	db, err := s.database()
	if err != nil {
		return RoleResponse{}, err
	}

	var item model.SystemRole
	if err := db.WithContext(ctx).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return RoleResponse{}, ErrRoleNotFound
		}
		return RoleResponse{}, fmt.Errorf("get role: %w", err)
	}
	return toResponse(item), nil
}

func (s *Service) Update(ctx context.Context, id int64, request UpdateRequest) (RoleResponse, error) {
	db, err := s.database()
	if err != nil {
		return RoleResponse{}, err
	}

	var item model.SystemRole
	if err := db.WithContext(ctx).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return RoleResponse{}, ErrRoleNotFound
		}
		return RoleResponse{}, fmt.Errorf("find role: %w", err)
	}

	updates := make(map[string]any)
	if request.Name != nil {
		name, err := validateName(*request.Name)
		if err != nil {
			return RoleResponse{}, err
		}
		updates["name"] = name
	}
	if request.Code != nil {
		code, err := validateCode(*request.Code)
		if err != nil {
			return RoleResponse{}, err
		}
		updates["code"] = code
	}
	if request.Status != nil {
		status, err := normalizeStatus(request.Status)
		if err != nil {
			return RoleResponse{}, err
		}
		updates["status"] = status
	}
	if request.Sort != nil {
		sort, err := normalizeSort(request.Sort)
		if err != nil {
			return RoleResponse{}, err
		}
		updates["sort"] = sort
	}
	if request.Remark != nil {
		remark, err := validateRemark(*request.Remark)
		if err != nil {
			return RoleResponse{}, err
		}
		updates["remark"] = remark
	}

	if len(updates) > 0 {
		if err := db.WithContext(ctx).Model(&item).Updates(updates).Error; err != nil {
			if isDuplicateKey(err) {
				return RoleResponse{}, ErrRoleCodeExists
			}
			return RoleResponse{}, fmt.Errorf("update role: %w", err)
		}
	}

	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	db, err := s.database()
	if err != nil {
		return err
	}

	result := db.WithContext(ctx).Delete(&model.SystemRole{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrRoleNotFound
	}
	return nil
}

func (s *Service) database() (*gorm.DB, error) {
	if s == nil || s.db == nil {
		return nil, ErrDatabaseUnavailable
	}
	return s.db, nil
}

func validateName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if len(name) > 64 {
		return "", fmt.Errorf("%w: name is too long", ErrInvalidInput)
	}
	return name, nil
}

func validateCode(value string) (string, error) {
	code := strings.TrimSpace(value)
	if code == "" {
		return "", fmt.Errorf("%w: code is required", ErrInvalidInput)
	}
	if len(code) > 64 {
		return "", fmt.Errorf("%w: code is too long", ErrInvalidInput)
	}
	return code, nil
}

func validateRemark(value string) (string, error) {
	remark := strings.TrimSpace(value)
	if len(remark) > 255 {
		return "", fmt.Errorf("%w: remark is too long", ErrInvalidInput)
	}
	return remark, nil
}

func normalizeStatus(value *int8) (int8, error) {
	if value == nil {
		return 1, nil
	}
	if *value != 0 && *value != 1 {
		return 0, fmt.Errorf("%w: status must be 0 or 1", ErrInvalidInput)
	}
	return *value, nil
}

func normalizeSort(value *int) (int, error) {
	if value == nil {
		return 0, nil
	}
	if *value < 0 {
		return 0, fmt.Errorf("%w: sort must be greater than or equal to 0", ErrInvalidInput)
	}
	return *value, nil
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func toResponse(item model.SystemRole) RoleResponse {
	return RoleResponse{
		ID:        item.ID,
		Name:      item.Name,
		Code:      item.Code,
		Status:    item.Status,
		Sort:      item.Sort,
		Remark:    item.Remark,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
