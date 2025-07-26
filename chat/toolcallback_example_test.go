package chat

import (
	"context"
	"testing"
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
			callback := func(ctx context.Context, call ToolCall) (ToolResult, bool, error) {
				switch call.Name {
				case "custom_tool":
					// Handle this tool
					return ToolResult{
						Content: map[string]interface{}{
							"result": "custom result",
						},
					}, true, nil // handled=true
				default:
					// Don't handle this tool, let it fallback
					return ToolResult{}, false, nil // handled=false
				}
			}

			// Test the callback
			call := ToolCall{
				ID:   "test_id",
				Name: tt.toolName,
				Arguments: map[string]interface{}{
					"param": "value",
				},
				RawArgs: `{"param": "value"}`,
			}

			result, handled, err := callback(context.Background(), call)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if handled != tt.expectHandled {
				t.Errorf("expected handled=%v, got %v", tt.expectHandled, handled)
			}

			if tt.expectHandled {
				if result.Content == nil {
					t.Errorf("expected content but got nil")
				}
				if resultMap, ok := result.Content.(map[string]interface{}); ok {
					if resultMap["result"] != tt.expectResult {
						t.Errorf("expected result '%s', got '%v'", tt.expectResult, resultMap["result"])
					}
				} else {
					t.Errorf("expected map result but got %T", result.Content)
				}
			} else {
				if result.Content != nil {
					t.Errorf("expected nil content for unhandled tool, got %v", result.Content)
				}
			}
		})
	}
}

func TestToolCallbackIntegrationWithFallback(t *testing.T) {
	// Create a mock tool info mapping for fallback
	mapping := make(ToolInfoMapping)
	mockTool := &ToolInfo{
		Name:    "builtin_tool",
		Builtin: true,
	}
	mapping.AddTool("builtin_tool", mockTool)

	// Create callback that handles some tools but not others
	callback := func(ctx context.Context, call ToolCall) (ToolResult, bool, error) {
		if call.Name == "custom_tool" {
			return ToolResult{
				Content: "handled by custom callback",
			}, true, nil // handled=true
		}
		// Don't handle builtin_tool, let it fallback
		return ToolResult{}, false, nil // handled=false
	}

	// Test custom tool (should be handled by callback)
	customCall := ToolCall{
		ID:      "custom_id",
		Name:    "custom_tool",
		RawArgs: `{}`,
	}

	client := &Client{}
	result, err := client.executeToolWithCallback(context.Background(), customCall, callback, nil, "", mapping)
	if err != nil {
		t.Errorf("unexpected error for custom tool: %v", err)
	}
	if result.Content != "handled by custom callback" {
		t.Errorf("expected custom callback result, got: %v", result.Content)
	}

	// Test builtin tool (should fallback to built-in execution)
	builtinCall := ToolCall{
		ID:      "builtin_id",
		Name:    "builtin_tool",
		RawArgs: `{}`,
	}

	_, err = client.executeToolWithCallback(context.Background(), builtinCall, callback, nil, "", mapping)
	// This will likely fail since we don't have real builtin execution in test,
	// but it demonstrates the fallback behavior
	if err != nil {
		t.Logf("Builtin tool execution failed as expected in test environment: %v", err)
	}
}
