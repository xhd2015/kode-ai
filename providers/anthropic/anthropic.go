package anthropic

import (
	"context"
	"fmt"

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

// stream
func Stream(ctx context.Context, client *anthropic.Client, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	stream := client.Messages.NewStreaming(ctx, params)
	message := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			return nil, fmt.Errorf("accumulate event: %w", err)
		}
		// TODO: add callback to send as soon as possible
		if false {
			eventAny := event.AsAny()
			fmt.Printf("Received event: %T\n", eventAny)
			switch eventVariant := eventAny.(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch deltaVariant := eventVariant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					fmt.Printf("Streaming received: %s\n", deltaVariant.Text)
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("stream error: %w", err)
	}
	return &message, nil
}
