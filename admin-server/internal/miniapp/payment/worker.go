package payment

import (
	"context"
	"log"
	"time"
)

const (
	DefaultOrderExpiry   = 30 * time.Minute
	DefaultSweepInterval = time.Minute
)

// RunTimeoutWorker sweeps on startup and on each tick until ctx is cancelled.
// Start it once from main, not during router construction or in request handlers.
func (s *Service) RunTimeoutWorker(ctx context.Context, expiry, interval time.Duration) {
	if expiry <= 0 || interval <= 0 {
		log.Print("order timeout worker requires positive expiry and interval")
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		closed, err := s.CloseExpiredOrders(ctx, expiry)
		if err != nil && ctx.Err() == nil {
			log.Printf("order timeout sweep failed (closed %d): %v", closed, err)
		} else if closed > 0 {
			log.Printf("order timeout sweep closed %d orders", closed)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
