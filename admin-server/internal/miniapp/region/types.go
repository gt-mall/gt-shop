package region

import commonregion "mall/admin_core/internal/region"

// MiniappRegionsData represents the data in miniapp regions response.
type MiniappRegionsData struct {
	Version string                    `json:"version"`
	Regions []commonregion.RegionNode `json:"regions"`
}
