package system

import (
	"context"

	"gorm.io/gorm"
)

// Runtime is the shared runtime boundary used by system submodules.
type Runtime struct {
	db         *gorm.DB
	configured bool
}

func NewRuntime(db *gorm.DB, configured bool) *Runtime {
	return &Runtime{db: db, configured: configured}
}

func (r *Runtime) DatabaseState(ctx context.Context) string {
	if !r.configured {
		return "not_configured"
	}
	if r.db == nil {
		return "unavailable"
	}
	sqlDB, err := r.db.DB()
	if err != nil {
		return "unavailable"
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return "unavailable"
	}
	return "connected"
}
