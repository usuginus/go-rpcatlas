package grpc

import (
	"context"

	"example.com/app/domain"
	"example.com/app/state"
)

type Server struct {
	worker *state.Worker
}

type CreateRequest struct{}

type CreateResponse struct{}

func (s *Server) Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	payload := &domain.Payload{}
	_ = payload.Token()
	s.worker.Run(ctx)
	return &CreateResponse{}, nil
}
