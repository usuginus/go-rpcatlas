package beta

import (
	"context"

	"example.com/app/repo"
)

type UseCase struct {
	repo repo.BetaStore
}

func (uc *UseCase) Run(ctx context.Context, id string) error {
	return uc.repo.Store(ctx, id)
}
