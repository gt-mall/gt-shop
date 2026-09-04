package auth

const (
	SuperAdminRoleCode = "super_admin"

	PermissionSystemUserList   = "system:user:list"
	PermissionSystemUserCreate = "system:user:create"
	PermissionSystemUserUpdate = "system:user:update"
	PermissionSystemUserDelete = "system:user:delete"
	PermissionSystemUserRole   = "system:user:role"

	PermissionSystemRoleList       = "system:role:list"
	PermissionSystemRoleCreate     = "system:role:create"
	PermissionSystemRoleUpdate     = "system:role:update"
	PermissionSystemRoleDelete     = "system:role:delete"
	PermissionSystemRolePermission = "system:role:permission"
)

type PermissionDefinition struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Module string `json:"module"`
} // @name PermissionDefinition

var permissionDefinitions = []PermissionDefinition{
	{Code: PermissionSystemUserList, Name: "查看用户", Module: "用户管理"},
	{Code: PermissionSystemUserCreate, Name: "创建用户", Module: "用户管理"},
	{Code: PermissionSystemUserUpdate, Name: "更新用户", Module: "用户管理"},
	{Code: PermissionSystemUserDelete, Name: "删除用户", Module: "用户管理"},
	{Code: PermissionSystemUserRole, Name: "分配用户角色", Module: "用户管理"},
	{Code: PermissionSystemRoleList, Name: "查看角色", Module: "角色管理"},
	{Code: PermissionSystemRoleCreate, Name: "创建角色", Module: "角色管理"},
	{Code: PermissionSystemRoleUpdate, Name: "更新角色", Module: "角色管理"},
	{Code: PermissionSystemRoleDelete, Name: "删除角色", Module: "角色管理"},
	{Code: PermissionSystemRolePermission, Name: "分配角色权限", Module: "角色管理"},
}

var permissionCodeSet = func() map[string]struct{} {
	result := make(map[string]struct{}, len(permissionDefinitions))
	for _, item := range permissionDefinitions {
		result[item.Code] = struct{}{}
	}
	return result
}()

func AllPermissions() []PermissionDefinition {
	result := make([]PermissionDefinition, len(permissionDefinitions))
	copy(result, permissionDefinitions)
	return result
}

func AllPermissionCodes() []string {
	result := make([]string, 0, len(permissionDefinitions))
	for _, item := range permissionDefinitions {
		result = append(result, item.Code)
	}
	return result
}

func IsPermissionCode(code string) bool {
	_, ok := permissionCodeSet[code]
	return ok
}
