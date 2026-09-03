package health

// HealthData is the public health payload.
type HealthData struct {
	Service  string `json:"service" example:"mall-admin-core"`
	Status   string `json:"status" example:"ok"`
	Database string `json:"database" example:"not_configured"`
}
