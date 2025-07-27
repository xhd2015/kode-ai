package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/xhd2015/kode-ai/types"
)

// ExampleClient demonstrates how to use the CLI client
func ExampleClient() {
	// Create a CLI-based client (feels identical to chat.NewClient)
	client, err := NewClient(types.Config{
		Model: "gpt-4o",
		Token: "your-api-token",
	})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// Use the client with functional options (identical to chat package)
	response, err := client.Chat(ctx, "Hello, world!",
		WithSystemPrompt("You are a helpful assistant"),
		WithMaxRounds(3),
		WithTools("list_dir", "read_file"),
		WithEventCallback(func(event types.Message) {
			fmt.Printf("Event: %s - %s\n", event.Type, event.Content)
		}),
		WithToolCallback(func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			// Custom tool handling
			if call.Name == "custom_tool" {
				return types.ToolResult{
					Content: map[string]interface{}{
						"result": "Custom tool executed successfully",
					},
				}, true, nil
			}
			return types.ToolResult{}, false, nil // Let builtin tools handle it
		}),
	)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Response: %+v\n", response)
}

// TestClientInterface demonstrates that the CLI client has the same interface as the chat client
func TestClientInterface(t *testing.T) {
	// This test shows that the CLI package provides the same interface
	// as the chat package, making it a drop-in replacement for binary-only usage

	config := types.Config{
		Model: "gpt-4o",
		Token: "test-token",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify that all the same methods and options are available
	ctx := context.Background()

	// Test that we can create a request with all the same options
	// (This won't actually execute since we don't have a real token/server)
	_, err = client.Chat(ctx, "test message",
		WithSystemPrompt("test system"),
		WithMaxRounds(1),
		WithTools("list_dir"),
		WithToolDefinitions(&types.UnifiedTool{
			Name: "test",
			Handle: func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
				return types.ToolResult{}, false, nil
			},
		}),
		WithDefaultToolCwd("/tmp"),
		WithHistory([]types.Message{{Type: "msg", Role: "user", Content: "previous"}}),
		WithCache(false),
		WithMCPServers("localhost:8080"),
		WithToolCallback(func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			return types.ToolResult{}, false, nil
		}),
		WithEventCallback(func(event types.Message) {
			// Event handling
		}),
	)

	// We expect this to fail since we don't have a real kode binary or token
	// but the interface should be identical to the chat package
	if err == nil {
		t.Log("Chat call succeeded (unexpected in test environment)")
	} else {
		t.Logf("Chat call failed as expected in test environment: %v", err)
	}
}

// TestFunctionalOptions verifies that all functional options work correctly
func TestFunctionalOptions(t *testing.T) {
	req := types.Request{}

	// Test all options
	WithSystemPrompt("test system")(&req)
	WithMaxRounds(5)(&req)
	WithTools("tool1", "tool2")(&req)
	WithToolFiles("file1.json", "file2.json")(&req)
	WithToolJSONs("{\"name\":\"tool1\"}", "{\"name\":\"tool2\"}")(&req)
	WithToolDefinitions(&types.UnifiedTool{
		Name: "tool1",
		Handle: func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			return types.ToolResult{}, false, nil
		},
	}, &types.UnifiedTool{
		Name: "tool2",
		Handle: func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			return types.ToolResult{}, false, nil
		},
	})(&req)
	WithDefaultToolCwd("/test")(&req)
	WithHistory([]types.Message{{Type: "msg", Content: "test"}})(&req)
	WithCache(false)(&req)
	WithMCPServers("server1", "server2")(&req)

	// Verify options were applied
	if req.SystemPrompt != "test system" {
		t.Errorf("Expected system prompt 'test system', got '%s'", req.SystemPrompt)
	}
	if req.MaxRounds != 5 {
		t.Errorf("Expected max rounds 5, got %d", req.MaxRounds)
	}
	if len(req.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(req.Tools))
	}
	if len(req.ToolFiles) != 2 {
		t.Errorf("Expected 2 tool files, got %d", len(req.ToolFiles))
	}
	if len(req.ToolJSONs) != 2 {
		t.Errorf("Expected 2 tool JSONs, got %d", len(req.ToolJSONs))
	}
	if len(req.ToolDefinitions) != 2 {
		t.Errorf("Expected 2 tool definitions, got %d", len(req.ToolDefinitions))
	}
	if req.DefaultToolCwd != "/test" {
		t.Errorf("Expected default tool cwd '/test', got '%s'", req.DefaultToolCwd)
	}
	if len(req.History) != 1 {
		t.Errorf("Expected 1 history message, got %d", len(req.History))
	}
	if !req.NoCache {
		t.Errorf("Expected NoCache to be true")
	}
	if len(req.MCPServers) != 2 {
		t.Errorf("Expected 2 MCP servers, got %d", len(req.MCPServers))
	}
}
