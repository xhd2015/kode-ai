package run

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/xhd2015/kode-ai/types"
)

// loadMessagesFromStdin reads JSON messages from stdin until it encounters a message with type "events_loaded"
func loadMessagesFromStdin(timeout time.Duration) ([]types.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var messages []types.Message
	scanner := bufio.NewScanner(os.Stdin)

	// Create a channel to handle the scanning in a goroutine
	resultChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- fmt.Errorf("panic during message loading: %v", r)
			}
		}()

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var msg types.Message
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				resultChan <- fmt.Errorf("failed to parse JSON message: %w", err)
				return
			}

			// Check for termination event
			if msg.Type == types.MsgType_StreamInitEventsFinished {
				resultChan <- nil
				return
			}

			messages = append(messages, msg)
		}

		if err := scanner.Err(); err != nil {
			resultChan <- fmt.Errorf("error reading from stdin: %w", err)
			return
		}

		resultChan <- fmt.Errorf("reached end of input without events_loaded signal")
	}()

	select {
	case err := <-resultChan:
		return messages, err
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for events from stdin")
	}
}

// convertMessagesToHistory converts a slice of messages to the history format expected by chat options
func convertMessagesToHistory(messages []types.Message) []types.Message {
	var history []types.Message

	for _, msg := range messages {
		if !msg.Type.HistorySendable() {
			continue
		}
		history = append(history, msg)
	}

	return history
}
