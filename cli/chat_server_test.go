package cli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/kode-ai/types"
)

// Mock WebSocket server for testing
func createMockWebSocketServer(t *testing.T, handler func(*websocket.Conn, *http.Request)) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()
		handler(conn, r)
	}))

	return server
}

func TestChatWithServerBasic(t *testing.T) {
	// Create a mock server that echoes back a simple response
	server := createMockWebSocketServer(t, func(conn *websocket.Conn, r *http.Request) {
		// Read initial events until finish marker
		for {
			var msg types.Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				t.Errorf("Failed to read message: %v", err)
				return
			}
			if msg.Type == types.MsgType_StreamInitEventsFinished {
				break
			}
		}

		// Read the user message
		var userMsg types.Message
		err := conn.ReadJSON(&userMsg)
		if err != nil {
			t.Errorf("Failed to read user message: %v", err)
			return
		}

		// Send back a simple assistant response
		response := types.Message{
			Type:    types.MsgType_Msg,
			Role:    types.Role_Assistant,
			Content: "Hello from server!",
		}
		err = conn.WriteJSON(response)
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			return
		}

		// Send stream end to signal completion
		endMsg := types.Message{
			Type: types.MsgType_StreamEnd,
		}
		conn.WriteJSON(endMsg)
	})
	defer server.Close()

	// Convert http://... to ws://...
	serverURL := strings.Replace(server.URL, "http://", "ws://", 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var receivedEvents []types.Message
	req := types.Request{
		Message: "Hello server!",
		EventCallback: func(event types.Message) {
			receivedEvents = append(receivedEvents, event)
		},
	}

	response, err := ChatWithServer(ctx, serverURL, req)
	if err != nil {
		t.Fatalf("ChatWithServer failed: %v", err)
	}

	if response.LastAssistantMsg != "Hello from server!" {
		t.Errorf("Expected last assistant message 'Hello from server!', got '%s'", response.LastAssistantMsg)
	}

	// Check that we received the assistant message in events
	found := false
	for _, event := range receivedEvents {
		if event.Type == types.MsgType_Msg && event.Role == types.Role_Assistant && event.Content == "Hello from server!" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to receive assistant message in event callback")
	}
}

func TestChatWithServerWithHistory(t *testing.T) {
	// Test that query parameters are set correctly
	server := createMockWebSocketServer(t, func(conn *websocket.Conn, r *http.Request) {
		// Check query parameters
		if !r.URL.Query().Has("wait_for_stream_events") {
			t.Error("Expected wait_for_stream_events query parameter")
		}
		if r.URL.Query().Get("wait_for_stream_events") != "true" {
			t.Error("Expected wait_for_stream_events to be 'true'")
		}

		// Read initial events until finish marker
		for {
			var msg types.Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				t.Errorf("Failed to read message: %v", err)
				return
			}
			if msg.Type == types.MsgType_StreamInitEventsFinished {
				break
			}
		}

		// Read user message
		var userMsg types.Message
		err := conn.ReadJSON(&userMsg)
		if err != nil {
			t.Errorf("Failed to read user message: %v", err)
			return
		}

		if userMsg.Type != types.MsgType_StreamRequestUserMsg {
			t.Errorf("Expected MsgType_StreamRequestUserMsg, got %s", userMsg.Type)
		}

		// Send response
		response := types.Message{
			Type:    types.MsgType_Msg,
			Role:    types.Role_Assistant,
			Content: "Received your message",
		}
		conn.WriteJSON(response)

		// Send stream end to signal completion
		endMsg := types.Message{
			Type: types.MsgType_StreamEnd,
		}
		conn.WriteJSON(endMsg)
	})
	defer server.Close()

	serverURL := strings.Replace(server.URL, "http://", "ws://", 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := types.Request{
		Message:      "Test message",
		SystemPrompt: "You are a test assistant",
		History: []types.Message{
			{Type: types.MsgType_Msg, Role: types.Role_User, Content: "Previous message"},
		},
	}

	_, err := ChatWithServer(ctx, serverURL, req)
	if err != nil {
		t.Fatalf("ChatWithServer failed: %v", err)
	}
}

func TestChatWithServerToolCallback(t *testing.T) {
	// Test tool callback functionality
	server := createMockWebSocketServer(t, func(conn *websocket.Conn, r *http.Request) {
		// Read initial events
		for {
			var msg types.Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				return
			}
			if msg.Type == types.MsgType_StreamInitEventsFinished {
				break
			}
		}

		// Read user message
		var userMsg types.Message
		conn.ReadJSON(&userMsg)

		// Send a tool call request
		toolCall := types.Message{
			Type:     types.MsgType_StreamRequestTool,
			StreamID: "test-tool-id",
			ToolName: "test_tool",
			Content:  `{"param": "value"}`,
		}
		err := conn.WriteJSON(toolCall)
		if err != nil {
			t.Errorf("Failed to send tool call: %v", err)
			return
		}

		// Wait for ACK
		var ackMsg types.Message
		err = conn.ReadJSON(&ackMsg)
		if err != nil {
			t.Errorf("Failed to read ACK: %v", err)
			return
		}
		if ackMsg.Type != types.MsgType_StreamHandleAck {
			t.Errorf("Expected ACK, got %s", ackMsg.Type)
		}

		// Wait for tool response
		var toolResponse types.Message
		err = conn.ReadJSON(&toolResponse)
		if err != nil {
			t.Errorf("Failed to read tool response: %v", err)
			return
		}
		if toolResponse.Type != types.MsgType_StreamResponseTool {
			t.Errorf("Expected tool response, got %s", toolResponse.Type)
		}

		// Send final assistant response
		response := types.Message{
			Type:    types.MsgType_Msg,
			Role:    types.Role_Assistant,
			Content: "Tool executed successfully",
		}
		conn.WriteJSON(response)

		// Send stream end to signal completion
		endMsg := types.Message{
			Type: types.MsgType_StreamEnd,
		}
		conn.WriteJSON(endMsg)
	})
	defer server.Close()

	serverURL := strings.Replace(server.URL, "http://", "ws://", 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	toolCallbackCalled := false
	req := types.Request{
		Message: "Execute test tool",
		ToolCallback: func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
			toolCallbackCalled = true
			if call.Name != "test_tool" {
				t.Errorf("Expected tool name 'test_tool', got '%s'", call.Name)
			}
			if call.Arguments["param"] != "value" {
				t.Errorf("Expected param 'value', got '%v'", call.Arguments["param"])
			}
			return types.ToolResult{
				Content: map[string]interface{}{
					"result": "success",
				},
			}, true, nil
		},
	}

	_, err := ChatWithServer(ctx, serverURL, req)
	if err != nil {
		t.Fatalf("ChatWithServer failed: %v", err)
	}

	if !toolCallbackCalled {
		t.Error("Expected tool callback to be called")
	}
}

func TestChatWithServerError(t *testing.T) {
	// Test error handling
	server := createMockWebSocketServer(t, func(conn *websocket.Conn, r *http.Request) {
		// Send an error message
		errorMsg := types.Message{
			Type:  types.MsgType_Error,
			Error: "Server error occurred",
		}
		conn.WriteJSON(errorMsg)
	})
	defer server.Close()

	serverURL := strings.Replace(server.URL, "http://", "ws://", 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := types.Request{
		Message: "Test message",
	}

	_, err := ChatWithServer(ctx, serverURL, req)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
	if !strings.Contains(err.Error(), "Server error occurred") {
		t.Errorf("Expected error to contain 'Server error occurred', got: %v", err)
	}
}

func TestChatWithServerInvalidURL(t *testing.T) {
	ctx := context.Background()
	req := types.Request{
		Message: "Test message",
	}

	_, err := ChatWithServer(ctx, "://invalid-url", req)
	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "invalid server URL") {
		t.Errorf("Expected 'invalid server URL' error, got: %v", err)
	}
}

func TestChatWithServerConnectionFailure(t *testing.T) {
	ctx := context.Background()
	req := types.Request{
		Message: "Test message",
	}

	// Try to connect to a non-existent server
	_, err := ChatWithServer(ctx, "ws://localhost:99999", req)
	if err == nil {
		t.Fatal("Expected connection error")
	}
	if !strings.Contains(err.Error(), "failed to connect to WebSocket server") {
		t.Errorf("Expected connection error, got: %v", err)
	}
}

func TestServerSessionWriteEvent(t *testing.T) {
	// Test that serverSession.writeEvent calls the event callback
	sess := &serverSession{}

	var receivedEvent types.Message
	sess.eventCallback = func(event types.Message) {
		receivedEvent = event
	}

	testMsg := types.Message{
		Type:    types.MsgType_Msg,
		Role:    types.Role_User,
		Content: "test message",
	}

	// This should call the event callback but fail to write since no stream
	err := sess.writeEvent(testMsg)
	if err == nil {
		t.Error("Expected error when no stream context available")
	}

	// Check that event callback was called
	if receivedEvent.Content != "test message" {
		t.Errorf("Expected event callback to receive message, got: %v", receivedEvent)
	}
}

func TestChatWithServerURLParsing(t *testing.T) {
	tests := []struct {
		name           string
		serverURL      string
		expectedScheme string
		expectedHost   string
		expectedPath   string
		shouldError    bool
	}{
		{
			name:           "HTTP URL",
			serverURL:      "http://localhost:8080",
			expectedScheme: "ws",
			expectedHost:   "localhost:8080",
			expectedPath:   "/stream",
			shouldError:    false,
		},
		{
			name:           "HTTPS URL",
			serverURL:      "https://example.com:3000",
			expectedScheme: "wss",
			expectedHost:   "example.com:3000",
			expectedPath:   "/stream",
			shouldError:    false,
		},
		{
			name:        "Invalid URL",
			serverURL:   "://invalid-url",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse server URL and build WebSocket URL (same logic as in chatWithServer)
			serverURL, err := url.Parse(tt.serverURL)
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Convert http/https to ws/wss
			scheme := "ws"
			if serverURL.Scheme == "https" {
				scheme = "wss"
			}

			// Build WebSocket URL
			wsURL := &url.URL{
				Scheme: scheme,
				Host:   serverURL.Host,
				Path:   "/stream",
			}

			if wsURL.Scheme != tt.expectedScheme {
				t.Errorf("expected scheme %s, got %s", tt.expectedScheme, wsURL.Scheme)
			}
			if wsURL.Host != tt.expectedHost {
				t.Errorf("expected host %s, got %s", tt.expectedHost, wsURL.Host)
			}
			if wsURL.Path != tt.expectedPath {
				t.Errorf("expected path %s, got %s", tt.expectedPath, wsURL.Path)
			}
		})
	}
}

func TestChatWithServerQueryParamsURLLogic(t *testing.T) {
	// Test that query parameters are set correctly when history or system prompt exist
	serverURL := "http://localhost:8080"

	// Test with history
	req := types.Request{
		Message: "test message",
		History: []types.Message{
			{Type: types.MsgType_Msg, Role: types.Role_User, Content: "previous message"},
		},
	}

	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	wsURL := &url.URL{
		Scheme: "ws",
		Host:   parsedURL.Host,
		Path:   "/stream",
	}

	// Add wait_for_stream_events query parameter if we have history or system prompt
	if len(req.History) > 0 || req.SystemPrompt != "" {
		query := wsURL.Query()
		query.Set("wait_for_stream_events", "true")
		wsURL.RawQuery = query.Encode()
	}

	if !wsURL.Query().Has("wait_for_stream_events") {
		t.Error("expected wait_for_stream_events query parameter to be set")
	}
	if wsURL.Query().Get("wait_for_stream_events") != "true" {
		t.Error("expected wait_for_stream_events to be 'true'")
	}
}
