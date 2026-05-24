package transport

import "context"

type Service struct {
	app FooApplication
}

type ProcessFooRequest struct {
	Kind FooKind
	Body string
}

type ProcessFooResponse struct {
	Result string
}

func (s *Service) ProcessFoo(ctx context.Context, req *ProcessFooRequest) (*ProcessFooResponse, error) {
	result, err := s.app.ProcessFoo(ctx, ProcessFooCommand{
		Kind: req.Kind,
		Body: req.Body,
	})
	if err != nil {
		return nil, err
	}
	return &ProcessFooResponse{Result: result}, nil
}
