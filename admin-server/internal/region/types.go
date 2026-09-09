package region

// RegionNode represents an administrative division node in the tree.
type RegionNode struct {
	Code     string       `json:"code"`
	Name     string       `json:"name"`
	Children []RegionNode `json:"children,omitempty"`
}

// ResolvedRegion contains the standard codes and names resolved from parent-child hierarchy.
type ResolvedRegion struct {
	ProvinceCode string `json:"province_code"`
	Province     string `json:"province"`
	CityCode     string `json:"city_code"`
	City         string `json:"city"`
	DistrictCode string `json:"district_code"`
	District     string `json:"district"`
}

// RegionsData is the data payload returned by the regions API.
type RegionsData struct {
	Version string       `json:"version"`
	Regions []RegionNode `json:"regions"`
}
