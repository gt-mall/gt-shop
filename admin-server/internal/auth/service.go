package auth

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"mall/admin_core/internal/model"
)

var (
	ErrUnavailable        = errors.New("authentication service unavailable")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUserDisabled       = errors.New("user is disabled")
	ErrRoleNotFound       = errors.New("role not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidPermission  = errors.New("invalid permission code")
	ErrSuperAdminRole     = errors.New("super_admin permissions are built in")
)

const (
	tokenIssuer      = "mall-admin-core"
	rememberTokenTTL = 30 * 24 * time.Hour
)

type tokenClaims struct {
	jwt.RegisteredClaims
}

type Service struct {
	db        *gorm.DB
	enforcer  *casbin.SyncedEnforcer
	jwtSecret []byte
	tokenTTL  time.Duration
	policyMu  sync.Mutex
}

func NewService(db *gorm.DB, enforcer *casbin.SyncedEnforcer, jwtSecret string, tokenTTL time.Duration) *Service {
	return &Service{
		db:        db,
		enforcer:  enforcer,
		jwtSecret: []byte(strings.TrimSpace(jwtSecret)),
		tokenTTL:  tokenTTL,
	}
}

func (s *Service) Login(ctx context.Context, request LoginRequest) (LoginResponse, error) {
	if s == nil || s.db == nil || len(s.jwtSecret) == 0 {
		return LoginResponse{}, ErrUnavailable
	}

	var user model.SystemUser
	username := strings.TrimSpace(request.Username)
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return LoginResponse{}, ErrInvalidCredentials
		}
		return LoginResponse{}, fmt.Errorf("find login user: %w", err)
	}
	if user.Status != 1 {
		return LoginResponse{}, ErrUserDisabled
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	now := time.Now()
	tokenTTL := s.tokenTTL
	if request.Remember {
		tokenTTL = rememberTokenTTL
	}
	expiresAt := now.Add(tokenTTL)
	claims := tokenClaims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer:    tokenIssuer,
		Subject:   strconv.FormatInt(user.ID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("sign login token: %w", err)
	}
	return LoginResponse{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (model.SystemUser, error) {
	if s == nil || s.db == nil || len(s.jwtSecret) == 0 {
		return model.SystemUser{}, ErrUnavailable
	}

	claims := &tokenClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(tokenIssuer))
	if err != nil || !parsed.Valid {
		return model.SystemUser{}, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return model.SystemUser{}, ErrInvalidToken
	}

	var user model.SystemUser
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SystemUser{}, ErrInvalidToken
		}
		return model.SystemUser{}, fmt.Errorf("find authenticated user: %w", err)
	}
	if user.Status != 1 {
		return model.SystemUser{}, ErrUserDisabled
	}
	return user, nil
}

func (s *Service) Can(ctx context.Context, userID int64, permission string) (bool, error) {
	if s == nil || s.enforcer == nil {
		return false, ErrUnavailable
	}
	subject := UserSubject(userID)
	roleCodes, err := s.enforcer.GetRolesForUser(subject)
	if err != nil {
		return false, fmt.Errorf("get user roles: %w", err)
	}
	if s.db == nil {
		for _, code := range roleCodes {
			if code == SuperAdminRoleCode {
				return true, nil
			}
		}
		allowed, err := s.enforcer.Enforce(subject, permission)
		if err != nil {
			return false, fmt.Errorf("enforce permission: %w", err)
		}
		return allowed, nil
	}

	if len(roleCodes) == 0 {
		return false, nil
	}
	var activeRoles []model.SystemRole
	if err := s.db.WithContext(ctx).Where("code IN ? AND status = 1", roleCodes).Find(&activeRoles).Error; err != nil {
		return false, fmt.Errorf("load active roles: %w", err)
	}
	for _, role := range activeRoles {
		if role.Code == SuperAdminRoleCode {
			return true, nil
		}
		allowed, err := s.enforcer.HasPermissionForUser(role.Code, permission)
		if err != nil {
			return false, fmt.Errorf("check role permission: %w", err)
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) CurrentUser(ctx context.Context, user model.SystemUser) (CurrentUserResponse, error) {
	roles, err := s.rolesForSubject(ctx, UserSubject(user.ID))
	if err != nil {
		return CurrentUserResponse{}, err
	}
	permissions, err := s.permissionsForSubject(UserSubject(user.ID), roles)
	if err != nil {
		return CurrentUserResponse{}, err
	}
	return CurrentUserResponse{
		User:        toUserInfo(user),
		Roles:       roles,
		Permissions: permissions,
	}, nil
}

func (s *Service) RolePermissions(ctx context.Context, roleID int64) ([]string, error) {
	role, err := s.findRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role.Code == SuperAdminRoleCode {
		return AllPermissionCodes(), nil
	}
	return s.directPermissions(role.Code)
}

func (s *Service) ReplaceRolePermissions(ctx context.Context, roleID int64, permissions []string) error {
	if s == nil || s.enforcer == nil {
		return ErrUnavailable
	}
	role, err := s.findRole(ctx, roleID)
	if err != nil {
		return err
	}
	if role.Code == SuperAdminRoleCode {
		return ErrSuperAdminRole
	}

	permissions, err = normalizePermissionCodes(permissions)
	if err != nil {
		return err
	}

	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	previous, err := s.directPermissions(role.Code)
	if err != nil {
		return err
	}
	if _, err := s.enforcer.DeletePermissionsForUser(role.Code); err != nil {
		return fmt.Errorf("clear role permissions: %w", err)
	}
	if len(permissions) == 0 {
		return nil
	}
	rules := make([][]string, 0, len(permissions))
	for _, code := range permissions {
		rules = append(rules, []string{code})
	}
	if _, err := s.enforcer.AddPermissionsForUser(role.Code, rules...); err != nil {
		_, _ = s.enforcer.DeletePermissionsForUser(role.Code)
		restorePermissions(s.enforcer, role.Code, previous)
		return fmt.Errorf("save role permissions: %w", err)
	}
	return nil
}

func (s *Service) UserRoles(ctx context.Context, userID int64) ([]RoleInfo, error) {
	if err := s.ensureUser(ctx, userID); err != nil {
		return nil, err
	}
	return s.rolesForSubject(ctx, UserSubject(userID))
}

func (s *Service) ReplaceUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	if s == nil || s.enforcer == nil {
		return ErrUnavailable
	}
	if err := s.ensureUser(ctx, userID); err != nil {
		return err
	}
	roleIDs = uniquePositiveIDs(roleIDs)

	roles := make([]model.SystemRole, 0, len(roleIDs))
	if len(roleIDs) > 0 {
		if err := s.db.WithContext(ctx).Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return fmt.Errorf("find assigned roles: %w", err)
		}
		if len(roles) != len(roleIDs) {
			return ErrRoleNotFound
		}
	}
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		codes = append(codes, role.Code)
	}
	sort.Strings(codes)

	subject := UserSubject(userID)
	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	previous, err := s.enforcer.GetRolesForUser(subject)
	if err != nil {
		return fmt.Errorf("get current user roles: %w", err)
	}
	if _, err := s.enforcer.DeleteRolesForUser(subject); err != nil {
		return fmt.Errorf("clear user roles: %w", err)
	}
	if len(codes) == 0 {
		return nil
	}
	if _, err := s.enforcer.AddRolesForUser(subject, codes); err != nil {
		_, _ = s.enforcer.DeleteRolesForUser(subject)
		if len(previous) > 0 {
			_, _ = s.enforcer.AddRolesForUser(subject, previous)
		}
		return fmt.Errorf("save user roles: %w", err)
	}
	return nil
}

func (s *Service) DeleteRolePolicies(roleCode string) error {
	if s == nil || s.enforcer == nil {
		return ErrUnavailable
	}
	_, err := s.enforcer.DeleteRole(roleCode)
	if err != nil {
		return fmt.Errorf("delete casbin role: %w", err)
	}
	return nil
}

func (s *Service) DeleteUserRoles(userID int64) error {
	if s == nil || s.enforcer == nil {
		return ErrUnavailable
	}
	_, err := s.enforcer.DeleteUser(UserSubject(userID))
	if err != nil {
		return fmt.Errorf("delete casbin user: %w", err)
	}
	return nil
}

func UserSubject(userID int64) string {
	return "user:" + strconv.FormatInt(userID, 10)
}

func (s *Service) rolesForSubject(ctx context.Context, subject string) ([]RoleInfo, error) {
	if s == nil || s.db == nil || s.enforcer == nil {
		return nil, ErrUnavailable
	}
	codes, err := s.enforcer.GetRolesForUser(subject)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	if len(codes) == 0 {
		return []RoleInfo{}, nil
	}
	var roles []model.SystemRole
	if err := s.db.WithContext(ctx).Where("code IN ?", codes).Order("sort ASC").Order("id ASC").Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("load user roles: %w", err)
	}
	result := make([]RoleInfo, 0, len(roles))
	for _, role := range roles {
		result = append(result, RoleInfo{ID: role.ID, Name: role.Name, Code: role.Code, Status: role.Status})
	}
	return result, nil
}

func (s *Service) permissionsForSubject(subject string, roles []RoleInfo) ([]string, error) {
	if s == nil || s.enforcer == nil {
		return nil, ErrUnavailable
	}
	activeRoleCodes := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		if role.Status != 1 {
			continue
		}
		if role.Code == SuperAdminRoleCode {
			return AllPermissionCodes(), nil
		}
		activeRoleCodes[role.Code] = struct{}{}
	}
	policies, err := s.enforcer.GetImplicitPermissionsForUser(subject)
	if err != nil {
		return nil, fmt.Errorf("get user permissions: %w", err)
	}
	found := make(map[string]struct{}, len(policies))
	for _, policy := range policies {
		if len(policy) < 2 {
			continue
		}
		_, active := activeRoleCodes[policy[0]]
		if active && IsPermissionCode(policy[1]) {
			found[policy[1]] = struct{}{}
		}
	}
	result := make([]string, 0, len(found))
	for _, definition := range permissionDefinitions {
		if _, ok := found[definition.Code]; ok {
			result = append(result, definition.Code)
		}
	}
	return result, nil
}

func (s *Service) directPermissions(roleCode string) ([]string, error) {
	if s == nil || s.enforcer == nil {
		return nil, ErrUnavailable
	}
	policies, err := s.enforcer.GetPermissionsForUser(roleCode)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}
	found := make(map[string]struct{}, len(policies))
	for _, policy := range policies {
		if len(policy) >= 2 && IsPermissionCode(policy[1]) {
			found[policy[1]] = struct{}{}
		}
	}
	result := make([]string, 0, len(found))
	for _, definition := range permissionDefinitions {
		if _, ok := found[definition.Code]; ok {
			result = append(result, definition.Code)
		}
	}
	return result, nil
}

func (s *Service) findRole(ctx context.Context, roleID int64) (model.SystemRole, error) {
	if s == nil || s.db == nil {
		return model.SystemRole{}, ErrUnavailable
	}
	var role model.SystemRole
	if err := s.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SystemRole{}, ErrRoleNotFound
		}
		return model.SystemRole{}, fmt.Errorf("find role: %w", err)
	}
	return role, nil
}

func (s *Service) ensureUser(ctx context.Context, userID int64) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.SystemUser{}).Where("id = ?", userID).Count(&count).Error; err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if count == 0 {
		return ErrUserNotFound
	}
	return nil
}

func normalizePermissionCodes(values []string) ([]string, error) {
	found := make(map[string]struct{}, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value)
		if !IsPermissionCode(code) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidPermission, code)
		}
		found[code] = struct{}{}
	}
	result := make([]string, 0, len(found))
	for _, definition := range permissionDefinitions {
		if _, ok := found[definition.Code]; ok {
			result = append(result, definition.Code)
		}
	}
	return result, nil
}

func uniquePositiveIDs(values []int64) []int64 {
	found := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value > 0 {
			found[value] = struct{}{}
		}
	}
	result := make([]int64, 0, len(found))
	for value := range found {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func restorePermissions(enforcer *casbin.SyncedEnforcer, roleCode string, permissions []string) {
	rules := make([][]string, 0, len(permissions))
	for _, code := range permissions {
		rules = append(rules, []string{code})
	}
	if len(rules) > 0 {
		_, _ = enforcer.AddPermissionsForUser(roleCode, rules...)
	}
}

func toUserInfo(user model.SystemUser) UserInfo {
	return UserInfo{
		ID: user.ID, Username: user.Username, Nickname: user.Nickname, Status: user.Status,
		CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}
