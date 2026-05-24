package transport

import "context"

type previewClient struct{}

func (c *previewClient) RenderBeta(ctx context.Context, cmd ProcessFooCommand) (string, error) {
	return "beta-preview", nil
}
