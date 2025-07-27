package cli

import (
	"context"
	"fmt"

	"github.com/xhd2015/kode-ai/types"
)

// Client represents the CLI-based chat client
type Client struct {
	config types.Config
}

// NewClient creates a new CLI-based chat client
func NewClient(config types.Config) (*Client, error) {
	if config.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if config.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	return &Client{
		config: config,
	}, nil
}

// Chat performs a chat conversation using the CLI binary
func (c *Client) Chat(ctx context.Context, message string, opts ...types.ChatOption) (*types.Response, error) {
	req := types.Request{
		Model:   c.config.Model,
		Token:   c.config.Token,
		BaseURL: c.config.BaseURL,
		Message: message,
	}

	// Apply options
	for _, opt := range opts {
		opt(&req)
	}

	return Chat(ctx, req)
}

func (c *session) writeEventOpts(event types.Message) error {
	event = event.TimeFilled()
	if c.eventCallback != nil {
		c.eventCallback(event)
	}
	if c.stream != nil {
		// when writing new user message, reset the last assistant response
		if event.Role == types.Role_User {
			c.lastAssistantMsg = ""
		}
		err := c.stream.Write(event)
		if err != nil {
			return fmt.Errorf("failed to write event: %w", err)
		}
	}
	return nil
}
