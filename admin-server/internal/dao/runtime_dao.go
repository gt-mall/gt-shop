package dao

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RuntimeDAO is the small database access boundary used by the demo services.
type RuntimeDAO struct {
	pool       *pgxpool.Pool
	configured bool
}

func NewRuntimeDAO(pool *pgxpool.Pool, configured bool) *RuntimeDAO {
	return &RuntimeDAO{pool: pool, configured: configured}
}

func (d *RuntimeDAO) DatabaseState(ctx context.Context) string {
	if !d.configured {
		return "not_configured"
	}
	if d.pool == nil {
		return "unavailable"
	}
	if err := d.pool.Ping(ctx); err != nil {
		return "unavailable"
	}
	return "connected"
}
