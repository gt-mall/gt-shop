package demo

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

func (s *Service) Get(ctx context.Context) DemoData {
	return DemoData{
		Message:  "Mall API is running",
		Database: s.runtime.DatabaseState(ctx),
	}
}
