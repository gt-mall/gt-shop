package common

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Hello() HelloResponse {
	return HelloResponse{
		Text: "Hello World",
	}
}
