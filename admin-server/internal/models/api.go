package models

// HealthData is the public health payload.
type HealthData struct {
	Service  string `json:"service" example:"mall-admin-core"`
	Status   string `json:"status" example:"ok"`
	Database string `json:"database" example:"not_configured"`
}

// DemoData is the public payload for the backend demo endpoint.
type DemoData struct {
	Message  string `json:"message" example:"Mall API is running"`
	Database string `json:"database" example:"not_configured"`
}

// HelloData is exposed by the compile-time hello plugin.
type HelloData struct {
	Plugin  string `json:"plugin" example:"hello"`
	Message string `json:"message" example:"Hello from plugin"`
}
