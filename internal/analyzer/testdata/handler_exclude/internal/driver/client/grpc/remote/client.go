package remote

import "context"

type Client struct{}

type CreateRequest struct{}

type CreateResponse struct{}

func (c *Client) Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	return &CreateResponse{}, nil
}
