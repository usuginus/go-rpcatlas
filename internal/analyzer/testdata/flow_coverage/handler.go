package grpc

import "context"

type Server struct {
	usecase *sampleUsecase
}

type Kind string

const (
	KindAlpha Kind = "alpha"
	KindBeta  Kind = "beta"
)

type Command struct {
	Kind Kind
	Fast bool
}

type pb struct{}

func (pb) ProcessRequest() {}

func (pb) ProcessResponse() {}

func (s *Server) Process(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	return s.usecase.Process(ctx, Command{Kind: KindAlpha, Fast: true})
}

type processor interface {
	Process(context.Context, Command) string
}

type sampleUsecase struct {
	processors map[Kind]processor
}

var packageProcessors = map[Kind]processor{
	KindAlpha: alphaUsecase{},
	KindBeta:  newBetaUsecase(),
}

func newSampleUsecase() *sampleUsecase {
	return &sampleUsecase{
		processors: map[Kind]processor{
			KindAlpha: alphaUsecase{},
			KindBeta:  newBetaUsecase(),
		},
	}
}

func (u *sampleUsecase) Process(ctx context.Context, cmd Command) (*pb.ProcessResponse, error) {
	if cmd.Fast {
		u.fastPath(ctx, cmd)
	} else if cmd.Kind == KindBeta {
		u.betaPath(ctx, cmd)
	} else {
		u.slowPath(ctx, cmd)
	}

	packageProcessors[cmd.Kind].Process(ctx, cmd)

	localProcessors := map[Kind]processor{
		KindAlpha: alphaUsecase{},
		KindBeta:  newBetaUsecase(),
	}
	localProcessors[cmd.Kind].Process(ctx, cmd)

	result := u.processors[cmd.Kind].Process(ctx, cmd)
	return &pb.ProcessResponse{Result: result}, nil
}

func (u *sampleUsecase) fastPath(context.Context, Command) {}

func (u *sampleUsecase) betaPath(context.Context, Command) {}

func (u *sampleUsecase) slowPath(context.Context, Command) {}

type alphaUsecase struct{}

func (alphaUsecase) Process(context.Context, Command) string {
	return "alpha"
}

type betaUsecase struct{}

func newBetaUsecase() processor {
	return betaUsecase{}
}

func (betaUsecase) Process(context.Context, Command) string {
	return "beta"
}
