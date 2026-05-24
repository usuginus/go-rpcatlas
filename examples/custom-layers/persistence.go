package transport

import "context"

type fooStore struct{}

func (s *fooStore) Insert(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	return "foo_123", nil
}
