package grpc

import "context"

type Server struct {
	Workflow
}

type Workflow interface {
	Run(context.Context) error
}

type workflowImpl struct{}

type CreateRequest struct{}

type CreateResponse struct{}

func BuildServer() *Server {
	workflow := NewWorkflow()
	return NewServer(workflow)
}

func NewServer(workflow Workflow) *Server {
	return &Server{Workflow: workflow}
}

func NewWorkflow() Workflow {
	return &workflowImpl{}
}

func (s *Server) Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	if err := s.Workflow.Run(ctx); err != nil {
		return nil, err
	}
	return &CreateResponse{}, nil
}

func (w *workflowImpl) Run(context.Context) error {
	return nil
}
