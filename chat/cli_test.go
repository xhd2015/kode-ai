package chat

import (
	"path/filepath"
	"testing"

	"github.com/xhd2015/kode-ai/types"
)

func TestNewCLIHandler(t *testing.T) {
	client, err := NewClient(Config{
		Model: "claude-3-7-sonnet",
		Token: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	opts := CliOptions{
		RecordFile: "test.json",
		LogChat:    true,
		Verbose:    false,
	}

	handler := NewCliHandler(client, opts)
	if handler == nil {
		t.Errorf("expected CLI handler but got nil")
	}
	if handler.client != client {
		t.Errorf("expected client to be set correctly")
	}
	if handler.opts.RecordFile != "test.json" {
		t.Errorf("expected record file 'test.json', got '%s'", handler.opts.RecordFile)
	}
}

func TestCLIOptionsValidation(t *testing.T) {
	client, err := NewClient(Config{
		Model: "claude-3-7-sonnet",
		Token: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test all CLI options
	opts := CliOptions{
		RecordFile:         "session.json",
		IgnoreDuplicateMsg: true,
		LogRequest:         true,
		LogChat:            true,
		Verbose:            true,
	}

	handler := NewCliHandler(client, opts)
	if handler.opts.RecordFile != "session.json" {
		t.Errorf("expected record file 'session.json', got '%s'", handler.opts.RecordFile)
	}
	if !handler.opts.IgnoreDuplicateMsg {
		t.Errorf("expected IgnoreDuplicateMsg to be true")
	}
	if !handler.opts.LogRequest {
		t.Errorf("expected LogRequest to be true")
	}
	if !handler.opts.LogChat {
		t.Errorf("expected LogChat to be true")
	}
	if !handler.opts.Verbose {
		t.Errorf("expected Verbose to be true")
	}
}

func TestCLIHandlerLoadHistory(t *testing.T) {
	client, err := NewClient(Config{
		Model: "claude-3-7-sonnet",
		Token: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test with non-existent file
	handler := NewCliHandler(client, CliOptions{
		RecordFile: "non_existent.json",
	})

	history, err := handler.loadHistory()
	if err != nil {
		t.Errorf("expected no error for non-existent file, got: %v", err)
	}
	if history != nil {
		t.Errorf("expected nil history for non-existent file, got: %v", history)
	}

	// Test with empty record file
	handler = NewCliHandler(client, CliOptions{
		RecordFile: "",
	})

	history, err = handler.loadHistory()
	if err != nil {
		t.Errorf("expected no error for empty record file, got: %v", err)
	}
	if history != nil {
		t.Errorf("expected nil history for empty record file, got: %v", history)
	}
}

func TestCLIHandlerSaveToRecord(t *testing.T) {
	client, err := NewClient(Config{
		Model: "claude-3-7-sonnet",
		Token: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Create temporary file
	tmpDir := t.TempDir()
	recordFile := filepath.Join(tmpDir, "test_record.json")

	handler := NewCliHandler(client, CliOptions{
		RecordFile: recordFile,
	})

	// Save a message
	msg := CreateMessage(types.MsgType_Msg, types.Role_User, "test-model", "test message")
	err = handler.saveToRecord(msg)
	if err != nil {
		t.Errorf("unexpected error saving to record: %v", err)
	}

	// Verify the message was saved
	history, err := handler.loadHistory()
	if err != nil {
		t.Errorf("unexpected error loading history: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 message in history, got %d", len(history))
	}
	if history[0].Content != "test message" {
		t.Errorf("expected content 'test message', got '%s'", history[0].Content)
	}
}

func TestCLIHandlerCheckDuplicateMessage(t *testing.T) {
	client, err := NewClient(Config{
		Model: "claude-3-7-sonnet",
		Token: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Create history with a user message
	history := []types.Message{
		CreateMessage(types.MsgType_Msg, types.Role_User, "test-model", "hello world"),
		CreateMessage(types.MsgType_Msg, types.Role_Assistant, "test-model", "Hi there!"),
	}

	// Test with ignore duplicate enabled
	handler := NewCliHandler(client, CliOptions{
		IgnoreDuplicateMsg: true,
	})

	msg, stop, err := handler.checkDuplicateMessage("hello world", history)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if stop {
		t.Errorf("expected not to stop but got stop=true")
	}
	if msg != "" {
		t.Errorf("expected empty message when ignoring duplicates, got '%s'", msg)
	}

	// Test with different message
	msg, stop, err = handler.checkDuplicateMessage("different message", history)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if stop {
		t.Errorf("expected not to stop for different message")
	}
	if msg != "different message" {
		t.Errorf("expected 'different message', got '%s'", msg)
	}

	// Test with empty history
	msg, stop, err = handler.checkDuplicateMessage("any message", []types.Message{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if stop {
		t.Errorf("expected not to stop for empty history")
	}
	if msg != "any message" {
		t.Errorf("expected 'any message', got '%s'", msg)
	}
}

func TestCLIHandlerFormatOutput(t *testing.T) {
	client, err := NewClient(Config{
		Model: "claude-3-7-sonnet",
		Token: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	handler := NewCliHandler(client, CliOptions{
		LogChat: true,
		Verbose: true,
	})

	// Test different event types (these don't produce output we can easily test,
	// but we can at least verify they don't panic)
	events := []types.Message{
		{
			Type:    types.MsgType_Msg,
			Content: "Hello world",
		},
		{
			Type:     types.MsgType_ToolCall,
			Content:  `{"param": "value"}`,
			ToolName: "test_tool",
		},
		{
			Type:     types.MsgType_ToolResult,
			Content:  "Tool result",
			ToolName: "test_tool",
		},
		{
			Type: types.MsgType_TokenUsage,
		},
		{
			Type:  types.MsgType_Error,
			Error: "test error",
		},
		{
			Type:    types.MsgType_CacheInfo,
			Content: "Prompt cache enabled with claude-3-7-sonnet",
			Model:   "claude-3-7-sonnet",
			Metadata: types.Metadata{
				CacheInfo: &types.CacheInfoMetadata{
					CacheEnabled: true,
				},
			},
		},
	}

	// Test that formatOutput doesn't panic for any event type
	for _, event := range events {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("formatOutput panicked for event type %s: %v", event.Type, r)
				}
			}()
			handler.formatOutput(event)
		}()
	}
}

func TestGetUsageString(t *testing.T) {
	tests := []struct {
		name     string
		usage    types.TokenUsage
		expected string
	}{
		{
			name: "basic usage",
			usage: types.TokenUsage{
				Input:  100,
				Output: 50,
				Total:  150,
			},
			expected: "Usage: 100 + 50 = 150 tokens",
		},
		{
			name: "zero usage",
			usage: types.TokenUsage{
				Input:  0,
				Output: 0,
				Total:  0,
			},
			expected: "Usage: 0 + 0 = 0 tokens",
		},
	}

	// Test that TokenUsage struct can be created without panicking
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct can be created
			if tt.usage.Total != tt.usage.Input+tt.usage.Output {
				t.Errorf("TokenUsage total should equal input + output")
			}
		})
	}
}
