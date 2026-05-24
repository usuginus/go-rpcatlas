package transport

import "context"

type FooApplication interface {
	ProcessFoo(context.Context, ProcessFooCommand) (string, error)
}

type ProcessFooCommand struct {
	Title string
	Body  string
}

type fooApplication struct {
	policy *fooPolicy
	store  *fooStore
	index  *externalClient
}

var _ FooApplication = (*fooApplication)(nil)

func (a *fooApplication) ProcessFoo(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	if err := a.policy.Validate(cmd); err != nil {
		return "", err
	}
	fooID, err := a.store.Insert(ctx, cmd)
	if err != nil {
		return "", err
	}
	if err := a.index.Index(ctx, fooID); err != nil {
		return "", err
	}
	return fooID, nil
}
