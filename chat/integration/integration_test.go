package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/xhd2015/kode-ai/chat/server"
	"github.com/xhd2015/kode-ai/cli"
	"github.com/xhd2015/kode-ai/types"
	"github.com/xhd2015/llm-tools/jsonschema"
)

func logMsg(t *testing.T, msg types.Message) {
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
		t.Logf("# event: %s%s", msg.Type, suffix)
	}
}

// TestIntegration_ServerWithCLI tests the server with CLI ChatWithServer function
func TestIntegration_ServerWithCLI(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") != "true" {
		t.Skip("Integration tests skipped. Set TEST_INTEGRATION=true to run.")
	}

	// Start the server on a random port
	port := findFreePort(t)
	serverOpts := server.ServerOptions{
		Verbose: false,
	}

	// Start server in background
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Start(port, serverOpts)
	}()

	// Wait a bit for server to start
	time.Sleep(500 * time.Millisecond)

	// Test basic communication using CLI
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serverURL := fmt.Sprintf("http://localhost:%d", port)

	var receivedEvents []types.Message
	req := types.Request{
		Message:      "Hello, how are you?",
		SystemPrompt: "You are a helpful test assistant. Always respond with 'Test response: ' followed by your actual response.",
		Model:        "gpt-4o-mini",
		Token:        "mock-token",            // Mock server doesn't validate tokens
		BaseURL:      "http://localhost:8080", // Use mock server
		ToolDefinitions: []*types.UnifiedTool{
			{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters: &jsonschema.JsonSchema{
					Type: "object",
					Properties: map[string]*jsonschema.JsonSchema{
						"name": {
							Type: "string",
						},
					},
				},
			},
		},
		EventCallback: func(event types.Message) {
			receivedEvents = append(receivedEvents, event)
			logMsg(t, event)
		},
	}

	response, err := cli.ChatWithServer(ctx, serverURL, req)
	if err != nil {
		t.Fatalf("ChatWithServer failed: %v", err)
	}

	// Verify we got a response
	if response.LastAssistantMsg == "" {
		t.Error("Expected to receive an assistant message")
	}

	// Verify we received events
	if len(receivedEvents) == 0 {
		t.Error("Expected to receive events from server")
	}

	t.Logf("Integration test completed successfully. Last response: %s", response.LastAssistantMsg)
}

// TestIntegration_ServerWithTools tests tool functionality with the actual server
func TestIntegration_ServerWithTools(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") != "true" {
		t.Skip("Integration tests skipped. Set TEST_INTEGRATION=true to run.")
	}

	// Start the server with tools
	port := findFreePort(t)
	serverOpts := server.ServerOptions{
		Verbose: false,
	}

	// Start server in background
	go func() {
		server.Start(port, serverOpts)
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Test with tool callback
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serverURL := fmt.Sprintf("http://localhost:%d", port)

	var receivedEvents []types.Message

	req := types.Request{
		Message:        "What files are in the current directory?",
		SystemPrompt:   "You are a helpful assistant. When asked about the current directory, use the list_dir tool.",
		Tools:          []string{"list_dir"},
		DefaultToolCwd: ".",
		Model:          "gpt-4o-mini",
		Token:          "mock-token",            // Mock server doesn't validate tokens
		BaseURL:        "http://localhost:8080", // Use mock server
		EventCallback: func(event types.Message) {
			receivedEvents = append(receivedEvents, event)
			logMsg(t, event)
		},
		ToolCallback: func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			t.Logf("Tool callback called: %s with args: %v", call.Name, call.Arguments)

			// Let the built-in tool handle it
			return types.ToolResult{}, false, nil
		},
	}

	response, err := cli.ChatWithServer(ctx, serverURL, req)
	if err != nil {
		t.Fatalf("ChatWithServer with tools failed: %v", err)
	}

	// Verify we got a response
	if response.LastAssistantMsg == "" {
		t.Error("Expected to receive an assistant message")
	}

	t.Logf("Integration test with tools completed. Response: %s", response.LastAssistantMsg)
}

// TestIntegration_ServerWithSleepTool tests tool functionality with a long-running sleep tool
func TestIntegration_ServerWithSleepTool(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") != "true" {
		t.Skip("Integration tests skipped. Set TEST_INTEGRATION=true to run.")
	}

	// Start the server with tools - ENABLE VERBOSE LOGGING
	port := findFreePort(t)
	serverOpts := server.ServerOptions{
		Verbose: true, // Enable verbose logging to see channel management
	}

	// Start server in background
	go func() {
		server.Start(port, serverOpts)
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Test with sleep tool callback - use longer timeout to accommodate 2min sleep
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second) // 3 minutes timeout
	defer cancel()

	serverURL := fmt.Sprintf("http://localhost:%d", port)

	var receivedEvents []types.Message
	toolCallReceived := false
	toolResultReceived := false

	req := types.Request{
		Message:        "Please sleep for 2 minutes",
		SystemPrompt:   "You are a helpful assistant. When asked to sleep, use the sleep_tool.",
		MaxRounds:      2, // Changed from default 1 to 2 to see what happens
		DefaultToolCwd: ".",
		Model:          "gpt-4o-mini",
		Token:          "mock-token",            // Mock server doesn't validate tokens
		BaseURL:        "http://localhost:8080", // Use mock server
		ToolDefinitions: []*types.UnifiedTool{
			{
				Name:        "sleep_tool",
				Description: "A tool that sleeps for a specified number of seconds",
				Parameters: &jsonschema.JsonSchema{
					Type: "object",
					Properties: map[string]*jsonschema.JsonSchema{
						"seconds": {
							Type: "integer",
						},
					},
				},
			},
		},
		EventCallback: func(event types.Message) {
			receivedEvents = append(receivedEvents, event)
			logMsg(t, event)

			if event.Type == types.MsgType_ToolCall {
				toolCallReceived = true
			}
			if event.Type == types.MsgType_ToolResult {
				toolResultReceived = true
			}
		},
		ToolCallback: func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			t.Logf("Tool callback called: %s with args: %v", call.Name, call.Arguments)

			if call.Name == "sleep_tool" {
				// Extract seconds from arguments - default to 2 minutes (120 seconds)
				seconds := 120 // default 2 minutes
				if secsVal, ok := call.Arguments["seconds"]; ok {
					if secsInt, ok := secsVal.(float64); ok {
						seconds = int(secsInt)
					}
				}

				t.Logf("Sleeping for %d seconds (%d minutes)...", seconds, seconds/60)
				time.Sleep(time.Duration(seconds) * time.Second)
				t.Logf("Sleep completed after %d seconds (%d minutes)", seconds, seconds/60)

				return types.ToolResult{
					Content: fmt.Sprintf("Slept for %d seconds (%d minutes)", seconds, seconds/60),
				}, true, nil
			}

			// Let other tools be handled by built-in handlers
			return types.ToolResult{}, false, nil
		},
	}

	startTime := time.Now()
	_, err := cli.ChatWithServer(ctx, serverURL, req)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("ChatWithServer with sleep tool failed: %v", err)
	}

	// Verify tool call and result were received
	if !toolCallReceived {
		t.Error("Expected to receive a tool call event")
	}
	if !toolResultReceived {
		t.Error("Expected to receive a tool result event")
	}

	// Verify the call took at least 2 minutes (with some tolerance)
	if duration < 119*time.Second {
		t.Errorf("Expected call to take at least 2 minutes (120 seconds), but took %v", duration)
	}

	// Verify no errors occurred
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	t.Logf("Sleep tool integration test completed successfully in %v. Tool call and result received.", duration)
}

// TestIntegration_ServerWithHistory tests history functionality
func TestIntegration_ServerWithHistory(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") != "true" {
		t.Skip("Integration tests skipped. Set TEST_INTEGRATION=true to run.")
	}

	// Start the server
	port := findFreePort(t)
	serverOpts := server.ServerOptions{
		Verbose: false,
	}

	// Start server in background
	go func() {
		server.Start(port, serverOpts)
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Test with history
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serverURL := fmt.Sprintf("http://localhost:%d", port)

	req := types.Request{
		Message:      "What did I just tell you my name was?",
		SystemPrompt: "You are a helpful assistant.",
		History: []types.Message{
			{Type: types.MsgType_Msg, Role: types.Role_User, Content: "My name is Alice"},
			{Type: types.MsgType_Msg, Role: types.Role_Assistant, Content: "Nice to meet you, Alice!"},
		},
		Model:   "gpt-4o-mini",
		Token:   "mock-token",            // Mock server doesn't validate tokens
		BaseURL: "http://localhost:8080", // Use mock server
		EventCallback: func(event types.Message) {
			logMsg(t, event)
		},
	}

	response, err := cli.ChatWithServer(ctx, serverURL, req)
	if err != nil {
		t.Fatalf("ChatWithServer with history failed: %v", err)
	}

	// Verify we got a response
	if response.LastAssistantMsg == "" {
		t.Error("Expected to receive an assistant message")
	}

	t.Logf("Integration test with history completed. Response: %s", response.LastAssistantMsg)
}

// TestIntegration_ServerError tests error handling
func TestIntegration_ServerError(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") != "true" {
		t.Skip("Integration tests skipped. Set TEST_INTEGRATION=true to run.")
	}

	// Test connecting to non-existent server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := types.Request{
		Message: "This should fail",
	}

	_, err := cli.ChatWithServer(ctx, "http://localhost:99999", req)
	if err == nil {
		t.Fatal("Expected error when connecting to non-existent server")
	}

	t.Logf("Error handling test passed: %v", err)
}

// TestIntegration_ServerMultipleRounds tests multiple conversation rounds
func TestIntegration_ServerMultipleRounds(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") != "true" {
		t.Skip("Integration tests skipped. Set TEST_INTEGRATION=true to run.")
	}

	// Start the server
	port := findFreePort(t)
	serverOpts := server.ServerOptions{
		Verbose: false,
	}

	// Start server in background
	go func() {
		server.Start(port, serverOpts)
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Test multiple rounds
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	serverURL := fmt.Sprintf("http://localhost:%d", port)

	roundCount := 0
	req := types.Request{
		Message:      "Let's have a conversation. Ask me a question.",
		SystemPrompt: "You are a helpful assistant. Keep your responses brief.",
		MaxRounds:    3,
		EventCallback: func(event types.Message) {
			logMsg(t, event)
		},
		FollowUpCallback: func(ctx context.Context) (*types.Message, error) {
			roundCount++
			if roundCount >= 3 {
				return nil, nil // End conversation
			}
			return &types.Message{
				Content: fmt.Sprintf("This is my response to round %d. Ask me another question.", roundCount),
			}, nil
		},
	}

	response, err := cli.ChatWithServer(ctx, serverURL, req)
	if err != nil {
		t.Fatalf("ChatWithServer multiple rounds failed: %v", err)
	}

	// Verify we got a response
	if response.LastAssistantMsg == "" {
		t.Error("Expected to receive an assistant message")
	}

	if roundCount == 0 {
		t.Error("Expected at least one follow-up round")
	}

	t.Logf("Multiple rounds integration test completed after %d rounds. Final response: %s", roundCount, response.LastAssistantMsg)
}

// TestIntegration_ServerVerboseLogging tests the verbose logging functionality
func TestIntegration_ServerVerboseLogging(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") != "true" {
		t.Skip("Integration tests skipped. Set TEST_INTEGRATION=true to run.")
	}

	// Start the server with verbose logging enabled
	port := findFreePort(t)
	serverOpts := server.ServerOptions{
		Verbose: true, // Enable verbose logging
	}

	// Start server in background
	go func() {
		server.Start(port, serverOpts)
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Test basic communication with verbose logging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverURL := fmt.Sprintf("http://localhost:%d", port)

	req := types.Request{
		Message:      "Hello server with verbose logging",
		SystemPrompt: "You are a helpful assistant.",
		Model:        "gpt-4o-mini",
		Token:        "mock-token",
		BaseURL:      "http://localhost:8080",
		EventCallback: func(event types.Message) {
			logMsg(t, event)
		},
	}

	_, err := cli.ChatWithServer(ctx, serverURL, req)
	if err != nil {
		t.Fatalf("ChatWithServer with verbose logging failed: %v", err)
	}

	t.Logf("Verbose logging test completed successfully")
}

func findFreePort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}
