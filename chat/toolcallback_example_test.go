package chat

import (
	"context"
	"testing"

	"github.com/xhd2015/kode-ai/types"
)

func TestToolCallbackFallbackBehavior(t *testing.T) {
	// Test the new ToolCallback signature with different fallback scenarios
	tests := []struct {
		name          string
		toolName      string
		expectHandled bool
		expectResult  string
	}{
		{
			name:          "custom tool handled",
			toolName:      "custom_tool",
			expectHandled: true,
			expectResult:  "custom result",
		},
		{
			name:          "unknown tool not handled",
			toolName:      "unknown_tool",
			expectHandled: false,
			expectResult:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create callback that demonstrates the new signature
			callback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
				switch call.Name {
				case "custom_tool":
					// Handle this tool
					return types.ToolResult{
						Content: map[string]interface{}{
							"result": "custom result",
						},
					}, true, nil // handled=true
				default:
					// Don't handle this tool, let it fallback
					return types.ToolResult{}, false, nil // handled=false
				}
			}

			// Test the callback
			call := types.ToolCall{
				ID:   "test_id",
				Name: tt.toolName,
				Arguments: map[string]interface{}{
					"param": "value",
				},
				RawArgs: `{"param": "value"}`,
			}

			result, handled, err := callback(context.Background(), nil, call)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if handled != tt.expectHandled {
				t.Errorf("expected handled=%v, got %v", tt.expectHandled, handled)
			}

			if tt.expectHandled {
				// Check result content
				if result.Content == nil {
					t.Error("expected result content but got nil")
				} else if content, ok := result.Content.(map[string]interface{}); ok {
					if content["result"] != tt.expectResult {
						t.Errorf("expected result=%s, got %v", tt.expectResult, content["result"])
					}
				} else {
					t.Errorf("expected map content, got %T", result.Content)
				}
			}
		})
	}
}

func TestToolCallbackComplexScenarios(t *testing.T) {
	// Test more complex tool callback scenarios
	tests := []struct {
		name         string
		toolName     string
		expectError  bool
		expectResult bool
	}{
		{
			name:         "successful tool execution",
			toolName:     "success_tool",
			expectError:  false,
			expectResult: true,
		},
		{
			name:         "tool with error",
			toolName:     "error_tool",
			expectError:  true,
			expectResult: true, // handled but with error
		},
		{
			name:         "unhandled tool",
			toolName:     "unhandled_tool",
			expectError:  false,
			expectResult: false, // not handled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
				switch call.Name {
				case "success_tool":
					return types.ToolResult{
						Content: "success",
					}, true, nil
				case "error_tool":
					return types.ToolResult{
						Error: "tool error",
					}, true, nil // handled but with error in result
				default:
					return types.ToolResult{}, false, nil // not handled
				}
			}

			call := types.ToolCall{
				ID:   "test_id",
				Name: tt.toolName,
			}

			result, handled, err := callback(context.Background(), nil, call)

			if tt.expectError {
				// For error_tool, we expect no callback error but error in result
				if err != nil {
					t.Errorf("unexpected callback error: %v", err)
				}
				if result.Error == "" {
					t.Error("expected error in result but got none")
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if handled != tt.expectResult {
				t.Errorf("expected handled=%v, got %v", tt.expectResult, handled)
			}
		})
	}
}
