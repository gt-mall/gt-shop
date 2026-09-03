package hello

// HelloData is exposed by the compile-time hello plugin.
type HelloData struct {
	Plugin  string `json:"plugin" example:"hello"`
	Message string `json:"message" example:"Hello from plugin"`
}
