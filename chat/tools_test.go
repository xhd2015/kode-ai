package chat

import (
	"context"
	"testing"

	"github.com/xhd2015/kode-ai/tools"
)

func TestToolInfoMapping(t *testing.T) {
	mapping := make(ToolInfoMapping)

	// Test adding a tool
	tool1 := &ToolInfo{
		Name:    "test-tool",
		Builtin: true,
	}
	err := mapping.AddTool("test-tool", tool1)
	if err != nil {
		t.Errorf("unexpected error adding tool: %v", err)
	}

	// Test duplicate tool
	tool2 := &ToolInfo{
		Name:    "test-tool",
		Builtin: false,
	}
	err = mapping.AddTool("test-tool", tool2)
	if err == nil {
		t.Errorf("expected error for duplicate tool but got none")
	}

	// Test tool retrieval
	retrieved := mapping["test-tool"]
	if retrieved == nil {
		t.Errorf("expected to retrieve tool but got nil")
	}
	if retrieved.Name != "test-tool" {
		t.Errorf("expected tool name 'test-tool' but got '%s'", retrieved.Name)
	}
	if !retrieved.Builtin {
		t.Errorf("expected builtin tool but got non-builtin")
	}
}

func TestToolInfoString(t *testing.T) {
	tests := []struct {
		name     string
		toolInfo ToolInfo
		expected string
	}{
		{
			name: "builtin tool",
			toolInfo: ToolInfo{
				Name:    "file_read",
				Builtin: true,
			},
			expected: "file_read",
		},
		{
			name: "mcp tool",
			toolInfo: ToolInfo{
				Name:      "custom_tool",
				MCPServer: "server1",
			},
			expected: "server1/custom_tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.toolInfo.String()
			if result != tt.expected {
				t.Errorf("expected '%s' but got '%s'", tt.expected, result)
			}
		})
	}
}

func TestParseToolCall(t *testing.T) {
	tests := []struct {
		name      string
		toolName  string
		toolID    string
		arguments string
		expectErr bool
	}{
		{
			name:      "valid tool call",
			toolName:  "file_read",
			toolID:    "call_123",
			arguments: `{"filename": "test.txt"}`,
			expectErr: false,
		},
		{
			name:      "empty arguments",
			toolName:  "simple_tool",
			toolID:    "call_456",
			arguments: "",
			expectErr: false,
		},
		{
			name:      "invalid json",
			toolName:  "bad_tool",
			toolID:    "call_789",
			arguments: `{"invalid": json}`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call, err := parseToolCall(tt.toolName, tt.toolID, tt.arguments, "")
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if call.Name != tt.toolName {
					t.Errorf("expected tool name '%s' but got '%s'", tt.toolName, call.Name)
				}
				if call.ID != tt.toolID {
					t.Errorf("expected tool ID '%s' but got '%s'", tt.toolID, call.ID)
				}
				if call.RawArgs != tt.arguments {
					t.Errorf("expected raw args '%s' but got '%s'", tt.arguments, call.RawArgs)
				}
			}
		})
	}
}

func TestExecuteToolWithCallback(t *testing.T) {
	// Create a mock tool info mapping
	mapping := make(ToolInfoMapping)
	mockTool := &ToolInfo{
		Name:    "mock_tool",
		Builtin: true,
		ToolDefinition: &tools.UnifiedTool{
			Name:        "mock_tool",
			Description: "A mock tool for testing",
		},
	}
	mapping.AddTool("mock_tool", mockTool)

	// Test with custom callback
	customCallback := func(ctx context.Context, call ToolCall) (ToolResult, bool, error) {
		if call.Name == "custom_tool" {
			return ToolResult{
				Content: map[string]interface{}{
					"result": "custom result",
				},
			}, true, nil // handled=true
		}
		// Don't handle this tool, fallback to built-in execution
		return ToolResult{}, false, nil // handled=false
	}

	// Test custom tool execution
	call := ToolCall{
		ID:   "test_id",
		Name: "custom_tool",
		Arguments: map[string]interface{}{
			"param": "value",
		},
		RawArgs: `{"param": "value"}`,
	}

	client := &Client{}
	result, err := client.executeToolWithCallback(context.Background(), call, customCallback, nil, "", mapping)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Content == nil {
		t.Errorf("expected content but got nil")
	}
	if resultMap, ok := result.Content.(map[string]interface{}); ok {
		if resultMap["result"] != "custom result" {
			t.Errorf("expected 'custom result' but got '%v'", resultMap["result"])
		}
	} else {
		t.Errorf("expected map result but got %T", result.Content)
	}

	// Test fallback to built-in (this will fail since we don't have real built-in tools in test)
	builtinCall := ToolCall{
		ID:      "builtin_id",
		Name:    "mock_tool",
		RawArgs: `{}`,
	}

	_, err = client.executeToolWithCallback(context.Background(), builtinCall, customCallback, nil, "", mapping)
	// We expect this to fail since we don't have real tool executors in test
	if err == nil {
		t.Logf("Note: builtin tool execution would normally fail in test environment")
	}
}

func TestExecuteBuiltinTool(t *testing.T) {
	// Test with a non-existent tool (should return error)
	call := ToolCall{
		Name:    "non_existent_tool",
		RawArgs: `{}`,
	}

	result, err := ExecuteBuiltinTool(context.Background(), call)
	if err != nil {
		t.Errorf("ExecuteBuiltinTool should not return error, but got: %v", err)
	}
	if result.Error == nil {
		t.Errorf("expected result.Error to be set for non-existent tool")
	}
}
