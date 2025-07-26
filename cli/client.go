package cli

import (
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

func (c *session) writeEvent(event types.Message) error {
	return c.writeEventOpts(event)
}

func (c *session) writeEventNoLock(event types.Message) error {
	return c.writeEventOpts(event)
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
