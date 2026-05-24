package grpc

import "context"

type fooRepository struct{}

type Foo struct {
	ID    string
	Title string
}

func (r *fooRepository) FindFoo(ctx context.Context, fooID string) (*Foo, error) {
	return &Foo{
		ID:    fooID,
		Title: "example",
	}, nil
}
