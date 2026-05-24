package transport

import "context"

type fooStore struct{}

func (s *fooStore) SaveAlpha(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	return "alpha-foo", nil
}
