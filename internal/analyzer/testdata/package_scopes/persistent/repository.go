package persistent

import (
	"context"

	"example.com/app/repo"
)

type AlphaStore struct{}

var _ repo.AlphaStore = (*AlphaStore)(nil)

func (r *AlphaStore) Store(context.Context, string) error {
	return nil
}

type BetaStore struct{}

var _ repo.BetaStore = (*BetaStore)(nil)

func (r *BetaStore) Store(context.Context, string) error {
	return nil
}
