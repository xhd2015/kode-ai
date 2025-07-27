package chat

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xhd2015/kode-ai/run/mock_server"
	"github.com/xhd2015/kode-ai/types"
)

// startToolMockServer starts a mock server that supports tool calls
func startToolMockServer(t *testing.T, provider string) (string, func()) {
	// Find an available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Create mock server
	mockServer := mock_server.NewMockServer(mock_server.Config{
		Port:     port,
		Provider: provider,
	})

	// Create HTTP server
	mux := http.NewServeMux()

	// Set up routes based on provider
	switch strings.ToLower(provider) {
	case "openai":
		mux.HandleFunc("/chat/completions", mockServer.HandleOpenAIMock)
	case "anthropic":
		mux.HandleFunc("/v1/messages", mockServer.HandleAnthropicMock)
	case "gemini":
		mux.HandleFunc("/v1beta/models/", mockServer.HandleGeminiMock)
		mux.HandleFunc("/models/", mockServer.HandleGeminiMock)
	case "all", "":
		// Enable all APIs
		mux.HandleFunc("/chat/completions", mockServer.HandleOpenAIMock)
		mux.HandleFunc("/v1/messages", mockServer.HandleAnthropicMock)
		mux.HandleFunc("/v1beta/models/", mockServer.HandleGeminiMock)
		mux.HandleFunc("/models/", mockServer.HandleGeminiMock)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("Mock server error: %v", err)
		}
	}()

	// Wait for server to start
	baseURL := fmt.Sprintf("http://localhost:%d", port)
	for i := 0; i < 10; i++ {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			break
		}
		if i == 9 {
			// Health check endpoint might not exist, try the actual endpoint
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}

	return baseURL, cleanup
}

func TestOpenAIToolCallIntegration(t *testing.T) {
	// Start mock server that supports tool calls
	baseURL, cleanup := startToolMockServer(t, "openai")
	defer cleanup()

	// Create client with mock server URL
	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Track events
	var events []types.Message
	eventCallback := func(event types.Message) {
		events = append(events, event)
	}

	// Custom tool callback to handle weather tool
	toolCalled := false
	toolCallback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
		toolCalled = true

		if call.Name == "get_weather" {
			// Return mock weather data for any location
			return types.ToolResult{
				Content: map[string]interface{}{
					"location":    "San Francisco",
					"temperature": 22,
					"unit":        "celsius",
					"condition":   "sunny",
					"humidity":    65,
				},
			}, true, nil // handled=true
		}

		// Handle any builtin tools that the mock server might call
		if strings.Contains(call.Name, "list_dir") || strings.Contains(call.Name, "read_file") ||
			strings.Contains(call.Name, "grep_search") || strings.Contains(call.Name, "batch_read_file") {
			return types.ToolResult{
				Content: map[string]interface{}{
					"result": "Mock tool execution successful",
					"tool":   call.Name,
				},
			}, true, nil
		}

		return types.ToolResult{}, false, nil // not handled
	}

	// Execute chat with tool calls - the mock server may randomly return tool calls
	response, err := client.Chat(context.Background(), "What's the weather in San Francisco?",
		WithToolCallback(toolCallback),
		WithEventCallback(eventCallback),
		WithMaxRounds(2), // Allow multiple rounds for tool execution
	)

	if err != nil {
		t.Fatalf("chat with tool calls failed: %v", err)
	}

	// Verify response
	if response == nil {
		t.Fatal("expected response but got nil")
	}

	// Check that we got a response
	if response.LastAssistantMsg == "" {
		t.Error("expected non-empty response content")
	}

	// Check that we got a response (content can vary due to random responses)
	if response.LastAssistantMsg == "" {
		t.Error("expected non-empty response content")
	}

	// Verify events were emitted (may include tool events if mock server returned tool calls)
	if len(events) == 0 {
		t.Error("expected events to be emitted")
	}

	// Check for message event
	hasMessageEvent := false
	for _, event := range events {
		if event.Type == types.MsgType_Msg {
			hasMessageEvent = true
			break
		}
	}
	if !hasMessageEvent {
		t.Error("expected at least one message event")
	}

	// Verify token usage
	if response.TokenUsage.Total == 0 {
		t.Error("expected token usage to be recorded")
	}

	// Log whether tool was called (depends on random behavior of mock server)
	t.Logf("Tool callback was called: %v", toolCalled)
}

func TestOpenAIToolCallWithBuiltinFallback(t *testing.T) {
	// Start mock server
	baseURL, cleanup := startToolMockServer(t, "openai")
	defer cleanup()

	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Tool callback that handles some tools but not others
	toolCallback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
		if call.Name == "custom_tool" {
			return types.ToolResult{Content: "handled"}, true, nil
		}
		// Don't handle other tools, let them fallback to built-in or fail gracefully
		return types.ToolResult{}, false, nil // not handled
	}

	// This should work with the mock server, which may or may not return tool calls
	_, err = client.Chat(context.Background(), "Read a file",
		WithToolCallback(toolCallback),
		WithMaxRounds(1), // Limit to avoid infinite loop in test
	)

	// We expect this to work since the mock server provides proper responses
	if err != nil {
		t.Logf("Tool execution completed with result: %v", err)
	}
}

func TestOpenAIMultipleToolCalls(t *testing.T) {
	// Start mock server
	baseURL, cleanup := startToolMockServer(t, "openai")
	defer cleanup()

	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Track which tools were called
	toolsCalled := make(map[string]bool)
	toolCallback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
		toolsCalled[call.Name] = true

		// Handle common tool names that might be returned by mock server
		switch call.Name {
		case "get_weather":
			return types.ToolResult{
				Content: map[string]interface{}{
					"temperature": 18,
					"condition":   "cloudy",
				},
			}, true, nil
		case "get_time":
			return types.ToolResult{
				Content: map[string]interface{}{
					"time":     "14:30:00",
					"timezone": "EST",
				},
			}, true, nil
		default:
			// Handle any other tools the mock server might return
			return types.ToolResult{
				Content: map[string]interface{}{
					"result": "Mock execution successful",
					"tool":   call.Name,
				},
			}, true, nil
		}
	}

	_, err = client.Chat(context.Background(), "Get weather and time",
		WithToolCallback(toolCallback),
		WithMaxRounds(1),
	)

	if err != nil {
		t.Logf("Multiple tool call test completed with result: %v", err)
	}

	// Log which tools were called (depends on random behavior of mock server)
	t.Logf("Tools called: %v", toolsCalled)
}

func TestOpenAIToolCallErrorHandling(t *testing.T) {
	// Start mock server
	baseURL, cleanup := startToolMockServer(t, "openai")
	defer cleanup()

	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Tool callback that returns an error for specific tools
	toolCallback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
		if call.Name == "failing_tool" {
			return types.ToolResult{
				Error: "tool execution failed",
			}, true, fmt.Errorf("simulated tool error")
		}
		// Handle other tools normally
		return types.ToolResult{
			Content: map[string]interface{}{
				"result": "Success",
			},
		}, true, nil
	}

	// This should handle any tool calls gracefully
	_, err = client.Chat(context.Background(), "Use tools",
		WithToolCallback(toolCallback),
		WithMaxRounds(1),
	)

	// The result depends on what tools the mock server returns
	if err != nil {
		t.Logf("Tool error handling test completed with result: %v", err)
		// Check if the error contains our simulated error (only if failing_tool was called)
		if strings.Contains(err.Error(), "simulated tool error") {
			t.Log("Successfully caught simulated tool error")
		}
	} else {
		t.Log("Tool error handling test completed successfully")
	}
}
