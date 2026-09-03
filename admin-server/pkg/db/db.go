package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ping is shared by future database-backed modules that need a small readiness check.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return nil
	}
	return pool.Ping(ctx)
}
