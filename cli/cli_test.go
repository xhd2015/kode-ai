package cli

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/xhd2015/kode-ai/types"
	"github.com/xhd2015/llm-tools/jsonschema"
)

func TestCliWithToolCallback(t *testing.T) {
	t.Skipf("only for debugging")
	// start the server with:
	//  kode mock-server --first-msg-tool-call
	client, err := NewClient(types.Config{
		Model:   "gpt-4o-mini",
		Token:   "sk-proj-1234567890",
		BaseURL: "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	resp, err := client.Chat(context.Background(), "Hello, how are you?",
		WithToolDefinitions(&types.UnifiedTool{
			Name:        "test_tool",
			Description: "test_tool description",
			Parameters: &jsonschema.JsonSchema{
				Type: "object",
				Properties: map[string]*jsonschema.JsonSchema{
					"name": {Type: "string"},
				},
			},
			Handle: func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
				time.Sleep(3 * time.Second)
				return types.ToolResult{
					Content: map[string]interface{}{
						"result": "world",
					},
				}, true, nil
			},
		}),
		WithEventCallback(func(msg types.Message) {
			if msg.Type == types.MsgType_ToolCall {
				t.Logf("%s: %s(%s)", msg.Role, msg.ToolName, msg.Content)
			} else if msg.Type == types.MsgType_ToolResult {
				t.Logf("%s: %s => %s", msg.Role, msg.ToolName, msg.Content)
			} else if msg.Role != "" {
				t.Logf("%s: %s", msg.Role, msg.Content)
			} else {
				suffix := ""
				if msg.Content != "" {
					suffix = fmt.Sprintf(", %s", msg.Content)
				}
				if false {
					t.Logf("# event from kode: %s%s", msg.Type, suffix)
				}
			}
		}),
		WithFollowUpCallback(func(ctx context.Context) (*types.Message, error) {
			time.Sleep(3 * time.Second)
			return &types.Message{
				Content: "What is your name?",
			}, nil
		}),
		WithMaxRounds(10),
	)

	if err != nil {
		t.Fatalf("Failed to chat: %v", err)
	}

	t.Logf("total rounds used: %d", resp.RoundsUsed)
}

func TestCliWithWeatherTool(t *testing.T) {
	t.Skipf("only for debugging")
	// start the server with:
	//  kode mock-server --first-msg-tool-call
	client, err := NewClient(types.Config{
		Model:   "gpt-4.1",
		Token:   os.Getenv("OPENAI_API_KEY"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	var followUp int
	resp, err := client.Chat(context.Background(), "Hello, how are you?",
		WithToolDefinitions(&types.UnifiedTool{
			Name:        "get_weather",
			Description: "get the weather of a city",
			Parameters: &jsonschema.JsonSchema{
				Type: "object",
				Properties: map[string]*jsonschema.JsonSchema{
					"city": {Type: "string"},
				},
			},
			Handle: func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
				time.Sleep(3 * time.Second)
				return types.ToolResult{
					Content: map[string]interface{}{
						"result": "43.5C",
					},
				}, true, nil
			},
		}),
		WithEventCallback(func(msg types.Message) {
			if msg.Type == types.MsgType_ToolCall {
				t.Logf("%s: %s(%s)", msg.Role, msg.ToolName, msg.Content)
			} else if msg.Type == types.MsgType_ToolResult {
				t.Logf("%s: %s => %s", msg.Role, msg.ToolName, msg.Content)
			} else if msg.Role != "" {
				t.Logf("%s: %s", msg.Role, msg.Content)
			} else {
				suffix := ""
				if msg.Content != "" {
					suffix = fmt.Sprintf(", %s", msg.Content)
				}
				if false {
					t.Logf("# event from kode: %s%s", msg.Type, suffix)
				}
			}
		}),
		WithFollowUpCallback(func(ctx context.Context) (*types.Message, error) {
			followUp++
			questions := []string{
				"What is the weather in Tokyo?",
				"What is the weather in London?",
				"What is the weather in Paris?",
				"What is the weather in Berlin?",
				"What is the weather in Rome?",
				"What is the weather in Madrid?",
				"What is the weather in Amsterdam?",
			}
			question := questions[rand.Intn(len(questions))]
			time.Sleep(3 * time.Second)
			return &types.Message{
				Content: question,
			}, nil
		}),
		WithMaxRounds(10),
	)

	if err != nil {
		t.Fatalf("Failed to chat: %v", err)
	}

	t.Logf("total rounds used: %d", resp.RoundsUsed)
}

// TestWithToolCallback tests the WithToolCallback functionality
func TestWithToolCallback(t *testing.T) {
	t.Run("ToolCallbackOption", func(t *testing.T) {
		// Test that WithToolCallback option sets the callback correctly
		var callbackCalled bool
		callback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			callbackCalled = true
			return types.ToolResult{}, false, nil
		}

		req := types.Request{}
		WithToolCallback(callback)(&req)

		if req.ToolCallback == nil {
			t.Error("Expected ToolCallback to be set")
		}

		// Test that the callback can be called
		req.ToolCallback(context.Background(), nil, types.ToolCall{})
		if !callbackCalled {
			t.Error("Expected callback to be called")
		}
	})

	t.Run("ToolCallbackWithRequest", func(t *testing.T) {
		// Test that tool callback can be set and retrieved from a request
		var callbackCalled bool
		var receivedCall types.ToolCall

		callback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			callbackCalled = true
			receivedCall = call
			return types.ToolResult{
				Content: map[string]interface{}{
					"custom": "response",
				},
			}, true, nil
		}

		req := types.Request{
			Message: "test message",
		}

		// Apply the tool callback option
		WithToolCallback(callback)(&req)

		// Verify the callback is set
		if req.ToolCallback == nil {
			t.Error("Expected ToolCallback to be set in request")
		}

		// Test that the callback can be invoked
		testCall := types.ToolCall{
			ID:   "test-id",
			Name: "test_tool",
			Arguments: map[string]interface{}{
				"param": "value",
			},
			RawArgs: `{"param": "value"}`,
		}

		result, handled, err := req.ToolCallback(context.Background(), nil, testCall)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !handled {
			t.Error("Expected callback to handle the tool")
		}
		if !callbackCalled {
			t.Error("Expected callback to be called")
		}
		if receivedCall.ID != "test-id" {
			t.Errorf("Expected call ID 'test-id', got '%s'", receivedCall.ID)
		}
		if receivedCall.Name != "test_tool" {
			t.Errorf("Expected call name 'test_tool', got '%s'", receivedCall.Name)
		}

		// Verify result
		content, ok := result.Content.(map[string]interface{})
		if !ok {
			t.Errorf("Expected result content to be a map, got %T", result.Content)
		} else if content["custom"] != "response" {
			t.Errorf("Expected custom response, got '%v'", content["custom"])
		}
	})

	t.Run("ToolCallbackHandlesError", func(t *testing.T) {
		// Test that tool callback can return errors
		callback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			return types.ToolResult{}, true, fmt.Errorf("tool execution failed")
		}

		req := types.Request{}
		WithToolCallback(callback)(&req)

		testCall := types.ToolCall{
			ID:   "error-test-id",
			Name: "error_tool",
		}

		result, handled, err := req.ToolCallback(context.Background(), nil, testCall)

		if err == nil {
			t.Error("Expected error to be returned")
		}
		if err.Error() != "tool execution failed" {
			t.Errorf("Expected error 'tool execution failed', got '%v'", err)
		}
		if !handled {
			t.Error("Expected callback to handle the tool even with error")
		}
		if result.Content != nil {
			t.Errorf("Expected nil content with error, got %v", result.Content)
		}
	})

	t.Run("ToolCallbackNotHandled", func(t *testing.T) {
		// Test that tool callback can indicate it doesn't handle a tool
		callback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			// Only handle specific tools
			if call.Name == "handled_tool" {
				return types.ToolResult{Content: "handled"}, true, nil
			}
			return types.ToolResult{}, false, nil // Not handled
		}

		req := types.Request{}
		WithToolCallback(callback)(&req)

		// Test handled tool
		handledCall := types.ToolCall{ID: "1", Name: "handled_tool"}
		result, handled, err := req.ToolCallback(context.Background(), nil, handledCall)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !handled {
			t.Error("Expected handled_tool to be handled")
		}
		if result.Content != "handled" {
			t.Errorf("Expected content 'handled', got %v", result.Content)
		}

		// Test unhandled tool
		unhandledCall := types.ToolCall{ID: "2", Name: "unhandled_tool"}
		result, handled, err = req.ToolCallback(context.Background(), nil, unhandledCall)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if handled {
			t.Error("Expected unhandled_tool to not be handled")
		}
		if result.Content != nil {
			t.Errorf("Expected nil content for unhandled tool, got %v", result.Content)
		}
	})
}
