package common

// HelloResponse is the response payload for miniapp hello world.
type HelloResponse struct {
	Text string `json:"text" example:"Hello World"`
}
