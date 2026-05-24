package grpc

import "context"

type Server struct {
	fooUsecase   FooUsecase
	fooConverter fooConverter
}

type GetFooRequest struct {
	FooID string
}

type FooResponse struct {
	ID    string
	Title string
}

func (s *Server) GetFoo(ctx context.Context, req *GetFooRequest) (*FooResponse, error) {
	foo, err := s.fooUsecase.GetFoo(ctx, req.FooID)
	if err != nil {
		return nil, err
	}
	return s.fooConverter.ToResponse(foo), nil
}
