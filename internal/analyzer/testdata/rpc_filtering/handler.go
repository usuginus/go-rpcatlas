package grpc

import "context"

type GetFooRequest struct{}

type GetFooResponse struct{}

type debugService struct{}

type userService struct{}

func (s *debugService) GetFoo(ctx context.Context, req *GetFooRequest) (*GetFooResponse, error) {
	return &GetFooResponse{}, nil
}

func (s *userService) GetFoo(ctx context.Context, req *GetFooRequest) (*GetFooResponse, error) {
	return &GetFooResponse{}, nil
}
