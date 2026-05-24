package alpha

import (
	"context"

	"example.com/app/repo"
)

type Runner interface {
	Run(context.Context, string) error
}

type UseCase struct {
	repo repo.AlphaStore
}

var _ Runner = (*UseCase)(nil)

func (uc *UseCase) Run(ctx context.Context, text string) error {
	return uc.repo.Store(ctx, text)
}
