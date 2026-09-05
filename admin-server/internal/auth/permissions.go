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

	PermissionSystemFileList   = "system:file:list"
	PermissionSystemFileUpload = "system:file:upload"
	PermissionSystemFileUpdate = "system:file:update"
	PermissionSystemFileDelete = "system:file:delete"
	PermissionSystemFileGroup  = "system:file:group"
	PermissionSystemFileConfig = "system:file:config"
)

type PermissionDefinition struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Module string `json:"module"`
} // @name PermissionDefinition

var permissionDefinitions = []PermissionDefinition{
	{Code: PermissionSystemFileList, Name: "查看图片素材", Module: "图片素材"},
	{Code: PermissionSystemFileUpload, Name: "上传图片", Module: "图片素材"},
	{Code: PermissionSystemFileUpdate, Name: "重命名及移动图片", Module: "图片素材"},
	{Code: PermissionSystemFileDelete, Name: "删除图片素材", Module: "图片素材"},
	{Code: PermissionSystemFileGroup, Name: "管理图片分组", Module: "图片素材"},
	{Code: PermissionSystemFileConfig, Name: "配置图片存储", Module: "图片素材"},
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
