package transport

import "context"

type externalClient struct{}

func (c *externalClient) Index(ctx context.Context, fooID string) error {
	return nil
}
