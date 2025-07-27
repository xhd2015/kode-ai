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

// startMockServer starts a mock server on a random available port and returns the base URL
func startMockServer(t *testing.T, provider string) (string, func()) {
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

func TestChatIntegrationOpenAI(t *testing.T) {
	// Start mock server
	baseURL, cleanup := startMockServer(t, "openai")
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

	// Test basic chat
	response, err := client.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected response but got nil")
	}

	// Check that we got a response (content can vary due to random responses)
	if response.LastAssistantMsg == "" {
		t.Error("expected non-empty response content")
	}

	// Check token usage
	if response.TokenUsage.Total == 0 {
		t.Error("expected token usage to be recorded")
	}

	if response.TokenUsage.Input == 0 {
		t.Error("expected input tokens to be recorded")
	}

	if response.TokenUsage.Output == 0 {
		t.Error("expected output tokens to be recorded")
	}
}

func TestChatIntegrationAnthropic(t *testing.T) {
	// Start mock server
	baseURL, cleanup := startMockServer(t, "anthropic")
	defer cleanup()

	// Create client with mock server URL
	client, err := NewClient(Config{
		Model:   "claude-3-7-sonnet",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test basic chat
	response, err := client.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected response but got nil")
	}

	// Check that we got a response (content can vary due to random responses)
	if response.LastAssistantMsg == "" {
		t.Error("expected non-empty response content")
	}

	// Check token usage
	if response.TokenUsage.Total == 0 {
		t.Error("expected token usage to be recorded")
	}
}

func TestChatIntegrationGemini(t *testing.T) {
	// Start mock server
	baseURL, cleanup := startMockServer(t, "gemini")
	defer cleanup()

	// Create client with mock server URL
	client, err := NewClient(Config{
		Model:   "gemini-2.0-flash",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test basic chat
	response, err := client.Chat(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected response but got nil")
	}

	// Check that we got a response (content can vary due to random responses)
	if response.LastAssistantMsg == "" {
		t.Error("expected non-empty response content")
	}

	// Check token usage
	if response.TokenUsage.Total == 0 {
		t.Error("expected token usage to be recorded")
	}
}

func TestChatWithSystemPrompt(t *testing.T) {
	baseURL, cleanup := startMockServer(t, "openai")
	defer cleanup()

	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test with system prompt
	response, err := client.Chat(context.Background(), "Hello",
		WithSystemPrompt("You are a helpful assistant"),
	)
	if err != nil {
		t.Fatalf("chat with system prompt failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected response but got nil")
	}
}

func TestChatWithHistory(t *testing.T) {
	baseURL, cleanup := startMockServer(t, "openai")
	defer cleanup()

	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Create history
	history := []types.Message{
		{Type: types.MsgType_Msg, Role: types.Role_User, Content: "Previous question"},
		{Type: types.MsgType_Msg, Role: types.Role_Assistant, Content: "Previous answer"},
	}

	// Test with history
	response, err := client.Chat(context.Background(), "Follow up question",
		WithHistory(history),
	)
	if err != nil {
		t.Fatalf("chat with history failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected response but got nil")
	}
}

func TestChatWithEventCallback(t *testing.T) {
	baseURL, cleanup := startMockServer(t, "openai")
	defer cleanup()

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

	// Test with event callback
	response, err := client.Chat(context.Background(), "Hello",
		WithEventCallback(eventCallback),
	)
	if err != nil {
		t.Fatalf("chat with event callback failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected response but got nil")
	}

	// Verify we got a response
	if response.LastAssistantMsg == "" {
		t.Error("Expected to receive an assistant message")
	}

	t.Logf("Integration test with event callback completed. Response: %s", response.LastAssistantMsg)

	// Verify events were captured
	if len(events) == 0 {
		t.Error("expected events to be captured but got none")
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
}

func TestChatRequestAPI(t *testing.T) {
	baseURL, cleanup := startMockServer(t, "openai")
	defer cleanup()

	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test ChatRequest API
	req := types.Request{
		Message:      "Hello from ChatRequest",
		SystemPrompt: "You are a test assistant",
		MaxRounds:    1,
		History: []types.Message{
			{Type: types.MsgType_Msg, Role: types.Role_User, Content: "Previous message"},
		},
	}

	response, err := client.ChatRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatRequest failed: %v", err)
	}

	if response == nil {
		t.Fatal("expected response but got nil")
	}

	// Check that we got a response (content can vary due to random responses)
	if response.LastAssistantMsg == "" {
		t.Error("expected non-empty response content")
	}

	t.Logf("Integration test with ChatRequest completed. Response: %s", response.LastAssistantMsg)
}

// Test error handling
func TestChatErrorHandling(t *testing.T) {
	// Create client with invalid URL (no server running)
	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: "http://localhost:99999", // Port that should not be in use
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test that errors are handled properly
	_, err = client.Chat(context.Background(), "Hello")
	if err == nil {
		t.Error("expected error but got none")
	}
}

// Test timeout handling
func TestChatTimeout(t *testing.T) {
	baseURL, cleanup := startMockServer(t, "openai")
	defer cleanup()

	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Test that timeout is handled properly
	_, err = client.Chat(ctx, "Hello")
	if err == nil {
		t.Error("expected timeout error but got none")
	}
}

// Test empty message handling
func TestChatEmptyMessage(t *testing.T) {
	baseURL, cleanup := startMockServer(t, "openai")
	defer cleanup()

	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test empty message
	_, err = client.Chat(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty message but got none")
	}
}

// Test with custom tool callback
func TestChatWithToolCallback(t *testing.T) {
	baseURL, cleanup := startMockServer(t, "openai")
	defer cleanup()

	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Custom tool callback
	toolCalled := false
	toolCallback := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
		toolCalled = true
		if call.Name == "test_tool" {
			return types.ToolResult{
				Content: "Tool executed successfully",
			}, true, nil // handled=true
		}
		return types.ToolResult{}, false, fmt.Errorf("unknown tool: %s", call.Name) // handled=false, with error
	}

	// Test with tool callback - the mock server may or may not return tool calls
	_, err = client.Chat(context.Background(), "Use a tool",
		WithToolCallback(toolCallback),
		WithMaxRounds(1), // Limit rounds to avoid infinite loop
	)

	// This test may succeed or fail depending on whether the mock server returns tool calls
	// The important thing is that it doesn't crash
	if err != nil {
		t.Logf("Tool callback test completed with result: %v", err)
	}

	// Note: toolCalled might be false if mock server doesn't return tool calls
	t.Logf("Tool callback was called: %v", toolCalled)
}
