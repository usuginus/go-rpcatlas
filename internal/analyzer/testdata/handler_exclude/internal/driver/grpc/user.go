package grpc

import "context"

type Server struct{}

type CreateRequest struct{}

type CreateResponse struct{}

func (s *Server) Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	return &CreateResponse{}, nil
}
