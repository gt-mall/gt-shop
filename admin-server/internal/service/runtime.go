package service

import (
	"context"

	"mall/admin_core/internal/consts"
	"mall/admin_core/internal/dao"
	"mall/admin_core/internal/models"
)

type HealthService struct {
	runtime *dao.RuntimeDAO
}

func NewHealthService(runtime *dao.RuntimeDAO) *HealthService {
	return &HealthService{runtime: runtime}
}

func (s *HealthService) Check(ctx context.Context) models.HealthData {
	return models.HealthData{
		Service:  consts.AppName,
		Status:   "ok",
		Database: s.runtime.DatabaseState(ctx),
	}
}

type DemoService struct {
	runtime *dao.RuntimeDAO
}

func NewDemoService(runtime *dao.RuntimeDAO) *DemoService {
	return &DemoService{runtime: runtime}
}

func (s *DemoService) Get(ctx context.Context) models.DemoData {
	return models.DemoData{
		Message:  "Mall API is running",
		Database: s.runtime.DatabaseState(ctx),
	}
}
