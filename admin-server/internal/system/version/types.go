package version

// VersionData represents the system version and build info.
type VersionData struct {
	Version   string `json:"version" example:"1.3.0"`
	BuildTime string `json:"build_time" example:"20260923.1721"`
	GitCommit string `json:"git_commit" example:"a8f31c2"`
}
