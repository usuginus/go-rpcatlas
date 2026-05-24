package transport

import "context"

type Service struct {
	fooApplication FooApplication
}

type ProcessFooRequest struct {
	Mode    string
	Payload FooPayload
}

type ProcessFooResponse struct {
	FooID string
}

func (s *Service) ProcessFoo(ctx context.Context, req *ProcessFooRequest) (*ProcessFooResponse, error) {
	fooID, err := s.fooApplication.ProcessFoo(ctx, ProcessFooCommand{
		Mode:    req.Mode,
		Payload: req.Payload,
	})
	if err != nil {
		return nil, err
	}
	return &ProcessFooResponse{FooID: fooID}, nil
}
