package grpc

import "context"

type Server struct {
	processor *sampleUsecase
}

type sampleUsecase struct {
	repo *sampleRepository
}

type sampleRepository struct{}

type pb struct{}

func (pb) CreateFooRequest() {}

func (pb) CreateFooResponse() {}

func (s *Server) CreateFoo(ctx context.Context, req *pb.CreateFooRequest) (*pb.CreateFooResponse, error) {
	if err := InvokeWithToken(ctx, s.processor.Run); err != nil {
		return nil, err
	}
	if err := InvokeWithToken(ctx, runStandalone); err != nil {
		return nil, err
	}
	return &pb.CreateFooResponse{}, nil
}

func InvokeWithToken(context.Context, func(context.Context, string) error) error {
	return nil
}

func (u *sampleUsecase) Run(ctx context.Context, token string) error {
	return u.repo.Save(ctx, token)
}

func runStandalone(context.Context, string) error {
	return nil
}

func (r *sampleRepository) Save(context.Context, string) error {
	return nil
}
