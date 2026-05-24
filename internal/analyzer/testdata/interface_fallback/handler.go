package grpc

import "context"

type Server struct {
	worker *workerUsecase
}

type Gateway interface {
	Run(context.Context, string) error
	Reset(context.Context, string) error
}

type workerUsecase struct {
	gateway Gateway
}

type remoteGatewayClient struct{}

var _ Gateway = (*FakeGateway)(nil)

type FakeGateway struct{}

type pb struct{}

func (pb) RunFooRequest() {}

func (pb) RunFooResponse() {}

func (s *Server) RunFoo(ctx context.Context, req *pb.RunFooRequest) (*pb.RunFooResponse, error) {
	if err := s.worker.Run(ctx, req.ID); err != nil {
		return nil, err
	}
	return &pb.RunFooResponse{}, nil
}

func (w *workerUsecase) Run(ctx context.Context, id string) error {
	return w.gateway.Run(ctx, id)
}

func (p *remoteGatewayClient) Run(context.Context, string) error {
	return nil
}

func (f *FakeGateway) Run(context.Context, string) error {
	return nil
}
