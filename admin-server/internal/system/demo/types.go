package demo

// DemoData is the public payload for the backend demo endpoint.
type DemoData struct {
	Message  string `json:"message" example:"Mall API is running"`
	Database string `json:"database" example:"not_configured"`
}
