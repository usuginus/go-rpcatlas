package grpc

import (
	"context"

	"example.com/app/alpha"
)

type Server struct {
	alpha alpha.Runner
}

type RunAlphaRequest struct {
	Value string
}

type RunAlphaResponse struct{}

func (s *Server) RunAlpha(ctx context.Context, req *RunAlphaRequest) (*RunAlphaResponse, error) {
	if err := s.alpha.Run(ctx, req.Value); err != nil {
		return nil, err
	}
	return &RunAlphaResponse{}, nil
}
