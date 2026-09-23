package version

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	// Version is the application semver version, injected via -ldflags.
	Version = "dev"
	// BuildTime is the build timestamp, injected via -ldflags (format: yyyyMMdd.HHmm).
	BuildTime = "unknown"
	// GitCommit is the short git commit hash, injected via -ldflags.
	GitCommit = "unknown"

	resolveOnce sync.Once
)

// Info holds version and build metadata.
type Info struct {
	Version   string `json:"version" example:"1.3.0"`
	BuildTime string `json:"build_time" example:"20260923.1721"`
	GitCommit string `json:"git_commit" example:"a8f31c2"`
}

// Get returns the resolved version metadata.
// In development mode (Version == "dev" or empty), it safely searches for the VERSION file
// at candidate paths without failing if not found.
func Get() Info {
	resolveOnce.Do(func() {
		if Version == "dev" || Version == "" {
			if v := readVersionFile(); v != "" {
				Version = v
			} else {
				Version = "dev"
			}
		}
	})
	return Info{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
	}
}

// String returns formatted version summary: GT-Shop v1.3.0 (20260923.1721-a8f31c2)
func String() string {
	info := Get()
	return "GT-Shop v" + info.Version + " (" + info.BuildTime + "-" + info.GitCommit + ")"
}

func readVersionFile() string {
	candidates := []string{
		"VERSION",
		"../VERSION",
		"../../VERSION",
	}
	if root := os.Getenv("PROJECT_ROOT"); root != "" {
		candidates = append([]string{filepath.Join(root, "VERSION")}, candidates...)
	}

	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			v := strings.TrimSpace(string(data))
			if v != "" {
				return v
			}
		}
	}
	return ""
}
