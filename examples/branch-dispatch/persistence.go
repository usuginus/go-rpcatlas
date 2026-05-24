package transport

import "context"

type fooStore struct{}

func (s *fooStore) SaveDraft(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	return "draft_123", nil
}

func (s *fooStore) Publish(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	return "foo_123", nil
}
