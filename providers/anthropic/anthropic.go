package anthropic

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func NewClient(opts ...option.RequestOption) *anthropic.Client {
	client := anthropic.NewClient(
		opts...,
	)
	return &client
}

func Chat(ctx context.Context, client *anthropic.Client, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	return client.Messages.New(ctx, params)
}
