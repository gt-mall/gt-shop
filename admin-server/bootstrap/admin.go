package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"mall/admin_core/internal/auth"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/system/user"
)

var (
	ErrPasswordRequired   = errors.New("ADMIN_PASSWORD must be explicitly provided")
	ErrUsernameRequired   = errors.New("ADMIN_USERNAME must be provided")
	ErrSuperAdminNotFound = errors.New("role 'super_admin' not found; please run init or seed first")
)

// CreateAdmin creates the initial system super administrator safely by reusing existing
// user service validation, password hashing, and Casbin role assignment.
func CreateAdmin(ctx context.Context, db *gorm.DB, authorizer *auth.Service, username, password, nickname string) (*model.SystemUser, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, ErrUsernameRequired
	}
	if password == "" {
		return nil, ErrPasswordRequired
	}
	if len(password) < 6 || len(password) > 72 {
		return nil, fmt.Errorf("%w: password length must be between 6 and 72 bytes", user.ErrInvalidInput)
	}
	if db == nil {
		return nil, errors.New("database handle is nil")
	}
	if authorizer == nil {
		return nil, errors.New("authorizer service is nil")
	}

	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		nickname = "超级管理员"
	}

	// 1. Verify that the super_admin role exists in system_role
	var superAdminRole model.SystemRole
	if err := db.WithContext(ctx).Where("code = ? AND status = 1", auth.SuperAdminRoleCode).First(&superAdminRole).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSuperAdminNotFound
		}
		return nil, fmt.Errorf("lookup super_admin role: %w", err)
	}

	// 2. Reuse user service to validate username, hash password, and insert system_user
	userSvc := user.NewService(db, authorizer)
	userResp, err := userSvc.Create(ctx, user.CreateRequest{
		Username: username,
		Password: password,
		Nickname: nickname,
	})
	if err != nil {
		if errors.Is(err, user.ErrUsernameExists) {
			return nil, fmt.Errorf("user %q already exists", username)
		}
		return nil, fmt.Errorf("create admin user: %w", err)
	}

	// 3. Assign super_admin role to user via Casbin
	if err := userSvc.SaveRoles(ctx, userResp.ID, []int64{superAdminRole.ID}); err != nil {
		return nil, fmt.Errorf("assign super_admin role: %w", err)
	}

	var createdUser model.SystemUser
	if err := db.WithContext(ctx).First(&createdUser, userResp.ID).Error; err != nil {
		return nil, fmt.Errorf("reload created admin: %w", err)
	}

	return &createdUser, nil
}
