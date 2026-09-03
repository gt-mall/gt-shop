package health

import (
	"context"

	"mall/admin_core/internal/system"
)

type Service struct {
	runtime *system.Runtime
}

func NewService(runtime *system.Runtime) *Service {
	return &Service{runtime: runtime}
}

func (s *Service) Check(ctx context.Context) HealthData {
	return HealthData{
		Service:  system.AppName,
		Status:   "ok",
		Database: s.runtime.DatabaseState(ctx),
	}
}
