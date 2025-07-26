package cli

import (
	"context"
	"testing"
	"time"

	"github.com/xhd2015/kode-ai/types"
)

func TestToolCallback(t *testing.T) {
	// Skip this test if kode binary is not available
	t.Skip("Integration test - requires kode binary and valid API token")

	client, err := NewClient(types.Config{
		Model: "gpt-4o-mini",
		Token: "test-token", // This would need to be a real token for actual testing
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with a tool callback
	response, err := client.Chat(ctx, "What's the current directory?",
		WithTools("list_dir"),
		WithToolCallback(func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			t.Logf("Tool callback called: %s with args: %v", call.Name, call.Arguments)

			// Let built-in tools handle it
			return types.ToolResult{}, false, nil
		}),
		WithEventCallback(func(event types.Message) {
			t.Logf("Event: %s - %s", event.Type, event.Content)
		}),
	)

	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	t.Logf("Response: %+v", response)
}

func TestToolCallbackHandling(t *testing.T) {
	// Test the tool callback handling logic without requiring the actual kode binary
	client := &Client{}

	// Test handleSingleToolCallback
	toolCallRequest := types.Message{
		Type:     types.MsgType_StreamRequestTool,
		StreamID: "test-id",
		ToolName: "test_tool",
		Content:  "{}",
	}

	callbackCalled := false
	toolCallback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
		callbackCalled = true

		if call.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got '%s'", call.ID)
		}
		if call.Name != "test_tool" {
			t.Errorf("Expected Name 'test_tool', got '%s'", call.Name)
		}
		if call.Arguments["param1"] != "value1" {
			t.Errorf("Expected param1 'value1', got '%v'", call.Arguments["param1"])
		}

		return types.ToolResult{
			Content: map[string]interface{}{
				"result": "success",
			},
		}, true, nil
	}

	// Create a mock writer that captures output
	mockWriter := &mockWriter{}

	ctx := context.Background()
	client.handleSingleToolCallback(ctx, toolCallRequest, mockWriter, toolCallback)

	if !callbackCalled {
		t.Error("Tool callback was not called")
	}

	// Verify that acknowledgment and response were written
	if len(mockWriter.writes) != 2 {
		t.Errorf("Expected 2 writes (ack + response), got %d", len(mockWriter.writes))
	}
}

// mockWriter captures all writes for testing
type mockWriter struct {
	writes [][]byte
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	w.writes = append(w.writes, append([]byte(nil), p...))
	return len(p), nil
}

func (w *mockWriter) Close() error {
	return nil
}
