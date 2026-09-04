package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
)

var (
	ErrDatabaseUnavailable = errors.New("database unavailable")
	ErrInvalidInput        = errors.New("invalid user input")
	ErrUserNotFound        = errors.New("user not found")
	ErrUsernameExists      = errors.New("username already exists")
)

type Service struct {
	db         *gorm.DB
	authorizer *auth.Service
}

func NewService(db *gorm.DB, authorizer *auth.Service) *Service {
	return &Service{db: db, authorizer: authorizer}
}

func (s *Service) Create(ctx context.Context, request CreateRequest) (UserResponse, error) {
	db, err := s.database()
	if err != nil {
		return UserResponse{}, err
	}

	username, err := validateUsername(request.Username)
	if err != nil {
		return UserResponse{}, err
	}
	if len(request.Password) < 6 || len(request.Password) > 72 {
		return UserResponse{}, fmt.Errorf("%w: password length must be between 6 and 72 bytes", ErrInvalidInput)
	}
	nickname, err := validateNickname(request.Nickname)
	if err != nil {
		return UserResponse{}, err
	}
	status, err := normalizeStatus(request.Status)
	if err != nil {
		return UserResponse{}, err
	}

	var existingCount int64
	if err := db.WithContext(ctx).Model(&model.SystemUser{}).Where("username = ?", username).Count(&existingCount).Error; err != nil {
		return UserResponse{}, fmt.Errorf("check username: %w", err)
	}
	if existingCount > 0 {
		return UserResponse{}, ErrUsernameExists
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return UserResponse{}, fmt.Errorf("hash password: %w", err)
	}

	newUser := model.SystemUser{
		Username:     username,
		PasswordHash: string(passwordHash),
		Nickname:     nickname,
		Status:       status,
	}
	if err := db.WithContext(ctx).Create(&newUser).Error; err != nil {
		if isDuplicateKey(err) {
			return UserResponse{}, ErrUsernameExists
		}
		return UserResponse{}, fmt.Errorf("create user: %w", err)
	}

	return toResponse(newUser), nil
}

func (s *Service) List(ctx context.Context, request ListRequest) ([]UserResponse, int, error) {
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

	query := db.WithContext(ctx).Model(&model.SystemUser{})
	if username := strings.TrimSpace(request.Username); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	var users []model.SystemUser
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	result := make([]UserResponse, 0, len(users))
	for _, item := range users {
		result = append(result, toResponse(item))
	}
	return result, int(total), nil
}

func (s *Service) Get(ctx context.Context, id int64) (UserResponse, error) {
	db, err := s.database()
	if err != nil {
		return UserResponse{}, err
	}

	var item model.SystemUser
	if err := db.WithContext(ctx).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserResponse{}, ErrUserNotFound
		}
		return UserResponse{}, fmt.Errorf("get user: %w", err)
	}
	return toResponse(item), nil
}

func (s *Service) Update(ctx context.Context, id int64, request UpdateRequest) (UserResponse, error) {
	db, err := s.database()
	if err != nil {
		return UserResponse{}, err
	}

	var item model.SystemUser
	if err := db.WithContext(ctx).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserResponse{}, ErrUserNotFound
		}
		return UserResponse{}, fmt.Errorf("find user: %w", err)
	}

	updates := make(map[string]any)
	if request.Username != nil {
		username, err := validateUsername(*request.Username)
		if err != nil {
			return UserResponse{}, err
		}
		updates["username"] = username
	}
	if request.Password != nil {
		if len(*request.Password) < 6 || len(*request.Password) > 72 {
			return UserResponse{}, fmt.Errorf("%w: password length must be between 6 and 72 bytes", ErrInvalidInput)
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(*request.Password), bcrypt.DefaultCost)
		if err != nil {
			return UserResponse{}, fmt.Errorf("hash password: %w", err)
		}
		updates["password_hash"] = string(passwordHash)
	}
	if request.Nickname != nil {
		nickname, err := validateNickname(*request.Nickname)
		if err != nil {
			return UserResponse{}, err
		}
		updates["nickname"] = nickname
	}
	if request.Status != nil {
		status, err := normalizeStatus(request.Status)
		if err != nil {
			return UserResponse{}, err
		}
		updates["status"] = status
	}

	if len(updates) > 0 {
		if err := db.WithContext(ctx).Model(&item).Updates(updates).Error; err != nil {
			if isDuplicateKey(err) {
				return UserResponse{}, ErrUsernameExists
			}
			return UserResponse{}, fmt.Errorf("update user: %w", err)
		}
	}

	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	db, err := s.database()
	if err != nil {
		return err
	}

	var item model.SystemUser
	if err := db.WithContext(ctx).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("find user: %w", err)
	}
	if s.authorizer == nil {
		return auth.ErrUnavailable
	}
	if err := s.authorizer.DeleteUserRoles(id); err != nil {
		return err
	}
	result := db.WithContext(ctx).Delete(&item)
	if result.Error != nil {
		return fmt.Errorf("delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *Service) Roles(ctx context.Context, userID int64) ([]auth.RoleInfo, error) {
	if s == nil || s.authorizer == nil {
		return nil, auth.ErrUnavailable
	}
	return s.authorizer.UserRoles(ctx, userID)
}

func (s *Service) SaveRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	if s == nil || s.authorizer == nil {
		return auth.ErrUnavailable
	}
	return s.authorizer.ReplaceUserRoles(ctx, userID, roleIDs)
}

func (s *Service) database() (*gorm.DB, error) {
	if s == nil || s.db == nil {
		return nil, ErrDatabaseUnavailable
	}
	return s.db, nil
}

func validateUsername(value string) (string, error) {
	username := strings.TrimSpace(value)
	if username == "" {
		return "", fmt.Errorf("%w: username is required", ErrInvalidInput)
	}
	if len(username) > 64 {
		return "", fmt.Errorf("%w: username is too long", ErrInvalidInput)
	}
	return username, nil
}

func validateNickname(value string) (string, error) {
	nickname := strings.TrimSpace(value)
	if len(nickname) > 100 {
		return "", fmt.Errorf("%w: nickname is too long", ErrInvalidInput)
	}
	return nickname, nil
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

func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func toResponse(item model.SystemUser) UserResponse {
	return UserResponse{
		ID:        item.ID,
		Username:  item.Username,
		Nickname:  item.Nickname,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
