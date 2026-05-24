package repo

import "context"

type AlphaStore interface {
	Store(context.Context, string) error
}

type BetaStore interface {
	Store(context.Context, string) error
}
