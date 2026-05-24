package grpc

import "context"

type Server struct {
	workflow Workflow
}

type Workflow interface {
	Run(context.Context, string) (*Foo, error)
}

type Repository interface {
	Save(context.Context, *Foo) error
}

type createUsecase struct {
	repo Repository
}

type archiveUsecase struct {
	repo Repository
}

type sqlRepository struct{}

type memoryRepository struct{}

type Foo struct{}

type pb struct{}

func (pb) CreateFooRequest() {}

func (pb) CreateFooResponse() {}

func NewServer() *Server {
	return &Server{
		workflow: NewCreateUsecase(NewSQLRepository()),
	}
}

func NewCreateUsecase(repo *sqlRepository) Workflow {
	return &createUsecase{repo}
}

func NewSQLRepository() *sqlRepository {
	return &sqlRepository{}
}

func (s *Server) CreateFoo(ctx context.Context, req *pb.CreateFooRequest) (*pb.CreateFooResponse, error) {
	foo, err := s.workflow.Run(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return convertFoo(foo), nil
}

func convertFoo(*Foo) *pb.CreateFooResponse {
	return nil
}

func (c *createUsecase) Run(ctx context.Context, id string) (*Foo, error) {
	foo := &Foo{}
	if err := c.repo.Save(ctx, foo); err != nil {
		return nil, err
	}
	return foo, nil
}

func (a *archiveUsecase) Run(context.Context, string) (*Foo, error) {
	return &Foo{}, nil
}

func (r *sqlRepository) Save(context.Context, *Foo) error {
	return nil
}

func (r *memoryRepository) Save(context.Context, *Foo) error {
	return nil
}
