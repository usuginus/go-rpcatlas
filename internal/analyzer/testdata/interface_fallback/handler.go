package grpc

import "context"

type Server struct {
	billing *billingUsecase
}

type Gateway interface {
	Charge(context.Context, string) error
	Refund(context.Context, string) error
}

type billingUsecase struct {
	gateway Gateway
}

type paymentGatewayClient struct{}

var _ Gateway = (*FakeGateway)(nil)

type FakeGateway struct{}

type pb struct{}

func (pb) BillFooRequest() {}

func (pb) BillFooResponse() {}

func (s *Server) BillFoo(ctx context.Context, req *pb.BillFooRequest) (*pb.BillFooResponse, error) {
	if err := s.billing.Bill(ctx, req.ID); err != nil {
		return nil, err
	}
	return &pb.BillFooResponse{}, nil
}

func (b *billingUsecase) Bill(ctx context.Context, id string) error {
	return b.gateway.Charge(ctx, id)
}

func (p *paymentGatewayClient) Charge(context.Context, string) error {
	return nil
}

func (f *FakeGateway) Charge(context.Context, string) error {
	return nil
}
