package transport

import "context"

type previewClient struct{}

func (c *previewClient) Index(ctx context.Context, fooID string) error {
	return nil
}
