package bootstrap

import (
	"errors"
	"fmt"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

const casbinModel = `
[request_definition]
r = sub, perm

[policy_definition]
p = sub, perm

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.perm == p.perm
`

// NewCasbinEnforcer initializes Casbin from the existing GORM connection.
// SQL migrations own the casbin_rule schema, so the adapter's AutoMigrate is disabled.
func NewCasbinEnforcer(db *gorm.DB) (*casbin.SyncedEnforcer, error) {
	if db == nil {
		return nil, errors.New("database unavailable")
	}

	model, err := casbinmodel.NewModelFromString(casbinModel)
	if err != nil {
		return nil, fmt.Errorf("create casbin model: %w", err)
	}

	adapterDB := db.Session(&gorm.Session{})
	gormadapter.TurnOffAutoMigrate(adapterDB)
	adapter, err := gormadapter.NewAdapterByDB(adapterDB)
	if err != nil {
		return nil, fmt.Errorf("create casbin adapter: %w", err)
	}

	enforcer, err := casbin.NewSyncedEnforcer(model, adapter)
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}
	return enforcer, nil
}
