package grpc

import "context"

type FooUsecase interface {
	GetFoo(context.Context, string) (*Foo, error)
}

type fooUsecase struct {
	repositories *Repositories
}

type Repositories struct {
	Foos *fooRepository
}

var _ FooUsecase = (*fooUsecase)(nil)

func (u *fooUsecase) GetFoo(ctx context.Context, fooID string) (*Foo, error) {
	foo, err := u.repositories.Foos.FindFoo(ctx, fooID)
	if err != nil {
		return nil, err
	}
	return foo, nil
}
