package chat

import (
	"context"
	"testing"

	"github.com/xhd2015/kode-ai/providers"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				Model: "claude-3-7-sonnet",
				Token: "test-token",
			},
			expectError: false,
		},
		{
			name: "missing model",
			config: Config{
				Token: "test-token",
			},
			expectError: true,
		},
		{
			name: "missing token",
			config: Config{
				Model: "claude-3-7-sonnet",
			},
			expectError: true,
		},
		{
			name: "invalid model",
			config: Config{
				Model: "invalid-model",
				Token: "test-token",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if client != nil {
					t.Errorf("expected nil client but got %v", client)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("expected client but got nil")
				}
			}
		})
	}
}

func TestClientAPIShapeDetection(t *testing.T) {
	tests := []struct {
		model            string
		expectedAPIShape providers.APIShape
	}{
		{"claude-3-7-sonnet", providers.APIShapeAnthropic},
		{"gpt-4o", providers.APIShapeOpenAI},
		{"gemini-2.0-flash", providers.APIShapeGemini},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			client, err := NewClient(Config{
				Model: tt.model,
				Token: "test-token",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if client.apiShape != tt.expectedAPIShape {
				t.Errorf("expected API shape %v but got %v", tt.expectedAPIShape, client.apiShape)
			}
		})
	}
}

func TestChatRequestValidation(t *testing.T) {
	client, err := NewClient(Config{
		Model: "claude-3-7-sonnet",
		Token: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test empty message
	_, err = client.Chat(context.Background(), "")
	if err == nil {
		t.Errorf("expected error for empty message but got none")
	}
}

func TestChatOptions(t *testing.T) {
	req := &Request{}

	// Test WithSystemPrompt
	WithSystemPrompt("test prompt")(req)
	if req.SystemPrompt != "test prompt" {
		t.Errorf("expected system prompt 'test prompt' but got '%s'", req.SystemPrompt)
	}

	// Test WithMaxRounds
	WithMaxRounds(5)(req)
	if req.MaxRounds != 5 {
		t.Errorf("expected max rounds 5 but got %d", req.MaxRounds)
	}

	// Test WithTools
	WithTools("tool1", "tool2")(req)
	if len(req.Tools) != 2 || req.Tools[0] != "tool1" || req.Tools[1] != "tool2" {
		t.Errorf("expected tools [tool1, tool2] but got %v", req.Tools)
	}

	// Test WithCache
	WithCache(false)(req)
	if !req.NoCache {
		t.Errorf("expected NoCache to be true when cache is disabled")
	}

	// Test WithHistory
	history := []Message{
		{Type: MsgType_Msg, Role: Role_User, Content: "test"},
	}
	WithHistory(history)(req)
	if len(req.History) != 1 || req.History[0].Content != "test" {
		t.Errorf("expected history with one message but got %v", req.History)
	}
}

func TestEventTypes(t *testing.T) {
	// Test that all event types are defined
	eventTypes := []EventType{
		EventTypeMessage,
		EventTypeToolCall,
		EventTypeToolResult,
		EventTypeTokenUsage,
		EventTypeRoundStart,
		EventTypeRoundEnd,
		EventTypeError,
		EventTypeCacheInfo,
	}

	for _, et := range eventTypes {
		if string(et) == "" {
			t.Errorf("event type should not be empty")
		}
	}
}

func TestMessageTypes(t *testing.T) {
	// Test that all message types are defined
	msgTypes := []MsgType{
		MsgType_Msg,
		MsgType_ToolCall,
		MsgType_ToolResult,
		MsgType_TokenUsage,
		MsgType_StopReason,
	}

	for _, mt := range msgTypes {
		if string(mt) == "" {
			t.Errorf("message type should not be empty")
		}
	}
}

func TestRoles(t *testing.T) {
	// Test that all roles are defined
	roles := []Role{
		Role_User,
		Role_Assistant,
		Role_System,
	}

	for _, r := range roles {
		if string(r) == "" {
			t.Errorf("role should not be empty")
		}
	}
}

func TestTokenUsageAdd(t *testing.T) {
	usage1 := TokenUsage{
		Input:  100,
		Output: 50,
		Total:  150,
		InputBreakdown: TokenUsageInputBreakdown{
			CacheRead:    10,
			CacheWrite:   5,
			NonCacheRead: 85,
		},
	}

	usage2 := TokenUsage{
		Input:  200,
		Output: 100,
		Total:  300,
		InputBreakdown: TokenUsageInputBreakdown{
			CacheRead:    20,
			CacheWrite:   10,
			NonCacheRead: 170,
		},
	}

	result := usage1.Add(usage2)

	if result.Input != 300 {
		t.Errorf("expected input 300 but got %d", result.Input)
	}
	if result.Output != 150 {
		t.Errorf("expected output 150 but got %d", result.Output)
	}
	if result.Total != 450 {
		t.Errorf("expected total 450 but got %d", result.Total)
	}
	if result.InputBreakdown.CacheRead != 30 {
		t.Errorf("expected cache read 30 but got %d", result.InputBreakdown.CacheRead)
	}
	if result.InputBreakdown.CacheWrite != 15 {
		t.Errorf("expected cache write 15 but got %d", result.InputBreakdown.CacheWrite)
	}
	if result.InputBreakdown.NonCacheRead != 255 {
		t.Errorf("expected non-cache read 255 but got %d", result.InputBreakdown.NonCacheRead)
	}
}
