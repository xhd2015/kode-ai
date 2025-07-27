package chat

import (
	"context"
	"fmt"

	"github.com/xhd2015/kode-ai/types"
)

func Chat(ctx context.Context, req types.Request) (*types.Response, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("requires model")
	}
	if req.Token == "" {
		return nil, fmt.Errorf("requires token")
	}
	if req.BaseURL == "" {
		return nil, fmt.Errorf("requires base url")
	}

	client, err := NewClient(Config{
		Model:   req.Model,
		Token:   req.Token,
		BaseURL: req.BaseURL,
	})
	if err != nil {
		return nil, err
	}

	return client.ChatRequest(ctx, req)
}
