package transport

import "context"

type Service struct {
	fooApplication FooApplication
}

type ProcessFooRequest struct {
	Title string
	Body  string
}

type ProcessFooResponse struct {
	FooID string
}

func (s *Service) ProcessFoo(ctx context.Context, req *ProcessFooRequest) (*ProcessFooResponse, error) {
	fooID, err := s.fooApplication.ProcessFoo(ctx, ProcessFooCommand{
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		return nil, err
	}
	return &ProcessFooResponse{FooID: fooID}, nil
}
