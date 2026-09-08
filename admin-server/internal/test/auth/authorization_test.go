package auth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"

	"mall/admin_core/internal/auth"
)

func TestRBACAndSuperAdmin(t *testing.T) {
	model := casbinmodel.NewModel()
	model.AddDef("r", "r", "sub, perm")
	model.AddDef("p", "p", "sub, perm")
	model.AddDef("g", "g", "_, _")
	model.AddDef("e", "e", "some(where (p_eft == allow))")
	model.AddDef("m", "m", "g(r_sub, p_sub) && r_perm == p_perm")
	enforcer, err := casbin.NewSyncedEnforcer(model)
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	service := auth.NewService(nil, enforcer, "", time.Hour)

	if _, err := enforcer.AddPermissionForUser("role_test", auth.PermissionSystemUserList); err != nil {
		t.Fatalf("add permission: %v", err)
	}
	if _, err := enforcer.AddRoleForUser(auth.UserSubject(1), "role_test"); err != nil {
		t.Fatalf("add role: %v", err)
	}
	allowed, err := service.Can(context.Background(), 1, auth.PermissionSystemUserList)
	if err != nil || !allowed {
		t.Fatalf("assigned permission allowed = %v, err = %v", allowed, err)
	}
	allowed, err = service.Can(context.Background(), 1, auth.PermissionSystemUserDelete)
	if err != nil || allowed {
		t.Fatalf("unassigned permission allowed = %v, err = %v", allowed, err)
	}

	if _, err := enforcer.AddRoleForUser(auth.UserSubject(2), auth.SuperAdminRoleCode); err != nil {
		t.Fatalf("add super admin role: %v", err)
	}
	allowed, err = service.Can(context.Background(), 2, "future:module:permission")
	if err != nil || !allowed {
		t.Fatalf("super admin permission allowed = %v, err = %v", allowed, err)
	}
}

func TestPermissionDefinitionsAreUniqueAndStructured(t *testing.T) {
	seen := make(map[string]struct{})
	for _, definition := range auth.AllPermissions() {
		if len(strings.Split(definition.Code, ":")) != 3 {
			t.Fatalf("permission code %q does not have three segments", definition.Code)
		}
		if _, exists := seen[definition.Code]; exists {
			t.Fatalf("duplicate permission code %q", definition.Code)
		}
		seen[definition.Code] = struct{}{}
	}
}
