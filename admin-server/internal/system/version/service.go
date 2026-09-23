package version

import (
	pkgversion "mall/admin_core/pkg/version"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Get() VersionData {
	info := pkgversion.Get()
	return VersionData{
		Version:   info.Version,
		BuildTime: info.BuildTime,
		GitCommit: info.GitCommit,
	}
}
