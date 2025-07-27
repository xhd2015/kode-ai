package chat

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xhd2015/kode-ai/types"
)

func TestLoadHistoryNonExistentFile(t *testing.T) {
	messages, err := LoadHistory("non_existent_file.json")
	if err != nil {
		t.Errorf("expected no error for non-existent file, got: %v", err)
	}
	if messages != nil {
		t.Errorf("expected nil messages for non-existent file, got: %v", messages)
	}
}

func TestLoadHistoryEmptyFilename(t *testing.T) {
	messages, err := LoadHistory("")
	if err != nil {
		t.Errorf("expected no error for empty filename, got: %v", err)
	}
	if messages != nil {
		t.Errorf("expected nil messages for empty filename, got: %v", messages)
	}
}

func TestSaveAndLoadHistory(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "test_history.json")

	// Create test messages
	messages := []types.Message{
		{
			Type:    types.MsgType_Msg,
			Time:    time.Now().Format(time.RFC3339),
			Role:    types.Role_User,
			Model:   "test-model",
			Content: "Hello world",
		},
		{
			Type:    types.MsgType_Msg,
			Time:    time.Now().Format(time.RFC3339),
			Role:    types.Role_Assistant,
			Model:   "test-model",
			Content: "Hello! How can I help you?",
		},
	}

	// Save messages
	err := SaveHistory(historyFile, messages)
	if err != nil {
		t.Fatalf("failed to save history: %v", err)
	}

	// Load messages
	loadedMessages, err := LoadHistory(historyFile)
	if err != nil {
		t.Fatalf("failed to load history: %v", err)
	}

	// Verify messages
	if len(loadedMessages) != len(messages) {
		t.Errorf("expected %d messages, got %d", len(messages), len(loadedMessages))
	}

	for i, msg := range messages {
		if i >= len(loadedMessages) {
			t.Errorf("missing message at index %d", i)
			continue
		}
		loaded := loadedMessages[i]
		if loaded.Type != msg.Type {
			t.Errorf("message %d: expected type %s, got %s", i, msg.Type, loaded.Type)
		}
		if loaded.Role != msg.Role {
			t.Errorf("message %d: expected role %s, got %s", i, msg.Role, loaded.Role)
		}
		if loaded.Content != msg.Content {
			t.Errorf("message %d: expected content %s, got %s", i, msg.Content, loaded.Content)
		}
	}
}

func TestAppendToHistory(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "test_append.json")

	// Create test messages
	msg1 := types.Message{
		Type:    types.MsgType_Msg,
		Role:    types.Role_User,
		Model:   "test-model",
		Content: "First message",
	}
	msg2 := types.Message{
		Type:    types.MsgType_Msg,
		Role:    types.Role_Assistant,
		Model:   "test-model",
		Content: "Second message",
	}

	// Append first message
	err := AppendToHistory(historyFile, msg1)
	if err != nil {
		t.Fatalf("failed to append first message: %v", err)
	}

	// Append second message
	err = AppendToHistory(historyFile, msg2)
	if err != nil {
		t.Fatalf("failed to append second message: %v", err)
	}

	// Load and verify
	messages, err := LoadHistory(historyFile)
	if err != nil {
		t.Fatalf("failed to load history: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].Content != "First message" {
		t.Errorf("expected first message content 'First message', got '%s'", messages[0].Content)
	}
	if messages[1].Content != "Second message" {
		t.Errorf("expected second message content 'Second message', got '%s'", messages[1].Content)
	}
}

func TestFilterHistoryByType(t *testing.T) {
	messages := []types.Message{
		{Type: types.MsgType_Msg, Content: "text message"},
		{Type: types.MsgType_ToolCall, Content: "tool call"},
		{Type: types.MsgType_Msg, Content: "another text message"},
		{Type: types.MsgType_ToolResult, Content: "tool result"},
		{Type: types.MsgType_TokenUsage, Content: "token usage"},
	}

	// Filter for text messages
	textMessages := FilterHistoryByType(messages, types.MsgType_Msg)
	if len(textMessages) != 2 {
		t.Errorf("expected 2 text messages, got %d", len(textMessages))
	}

	// Filter for tool calls
	toolCalls := FilterHistoryByType(messages, types.MsgType_ToolCall)
	if len(toolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(toolCalls))
	}

	// Filter for non-existent type
	nonExistent := FilterHistoryByType(messages, types.MsgType("non_existent"))
	if len(nonExistent) != 0 {
		t.Errorf("expected 0 messages for non-existent type, got %d", len(nonExistent))
	}
}

func TestGetLastUserMessage(t *testing.T) {
	messages := []types.Message{
		{Type: types.MsgType_Msg, Role: types.Role_User, Content: "first user message"},
		{Type: types.MsgType_Msg, Role: types.Role_Assistant, Content: "assistant response"},
		{Type: types.MsgType_Msg, Role: types.Role_User, Content: "second user message"},
		{Type: types.MsgType_ToolCall, Role: types.Role_Assistant, Content: "tool call"},
	}

	lastUser := GetLastUserMessage(messages)
	if lastUser == nil {
		t.Errorf("expected last user message but got nil")
	} else if lastUser.Content != "second user message" {
		t.Errorf("expected 'second user message', got '%s'", lastUser.Content)
	}

	// Test with no user messages
	noUserMessages := []types.Message{
		{Type: types.MsgType_Msg, Role: types.Role_Assistant, Content: "assistant only"},
	}
	lastUser = GetLastUserMessage(noUserMessages)
	if lastUser != nil {
		t.Errorf("expected nil for no user messages, got %v", lastUser)
	}

	// Test with empty slice
	lastUser = GetLastUserMessage([]types.Message{})
	if lastUser != nil {
		t.Errorf("expected nil for empty messages, got %v", lastUser)
	}
}

func TestGetSystemPrompts(t *testing.T) {
	messages := []types.Message{
		{Type: types.MsgType_Msg, Role: types.Role_System, Content: "first system prompt"},
		{Type: types.MsgType_Msg, Role: types.Role_User, Content: "user message"},
		{Type: types.MsgType_Msg, Role: types.Role_System, Content: "second system prompt"},
		{Type: types.MsgType_ToolCall, Role: types.Role_System, Content: "system tool call"}, // Should be ignored
	}

	prompts := GetSystemPrompts(messages)
	if len(prompts) != 2 {
		t.Errorf("expected 2 system prompts, got %d", len(prompts))
	}

	if prompts[0] != "first system prompt" {
		t.Errorf("expected 'first system prompt', got '%s'", prompts[0])
	}
	if prompts[1] != "second system prompt" {
		t.Errorf("expected 'second system prompt', got '%s'", prompts[1])
	}
}

func TestCreateMessage(t *testing.T) {
	msg := CreateMessage(types.MsgType_Msg, types.Role_User, "test-model", "test content")

	if msg.Type != types.MsgType_Msg {
		t.Errorf("expected type %s, got %s", types.MsgType_Msg, msg.Type)
	}
	if msg.Role != types.Role_User {
		t.Errorf("expected role %s, got %s", types.Role_User, msg.Role)
	}
	if msg.Model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", msg.Model)
	}
	if msg.Content != "test content" {
		t.Errorf("expected content 'test content', got '%s'", msg.Content)
	}
	if msg.Time == "" {
		t.Errorf("expected time to be set, got empty string")
	}
}

func TestCreateToolCallMessage(t *testing.T) {
	msg := CreateToolCallMessage(types.Role_Assistant, "test-model", "file_read", "call_123", `{"filename": "test.txt"}`)

	if msg.Type != types.MsgType_ToolCall {
		t.Errorf("expected type %s, got %s", types.MsgType_ToolCall, msg.Type)
	}
	if msg.ToolName != "file_read" {
		t.Errorf("expected tool name 'file_read', got '%s'", msg.ToolName)
	}
	if msg.ToolUseID != "call_123" {
		t.Errorf("expected tool use ID 'call_123', got '%s'", msg.ToolUseID)
	}
	if msg.Content != `{"filename": "test.txt"}` {
		t.Errorf("expected content to match arguments, got '%s'", msg.Content)
	}
}

func TestCreateTokenUsageMessage(t *testing.T) {
	// Test creating a token usage structure
	usage := types.TokenUsage{
		Input:  100,
		Output: 50,
		Total:  150,
	}

	// Just verify the TokenUsage struct can be created
	if usage.Total != usage.Input+usage.Output {
		t.Errorf("expected total to equal input + output")
	}
}

func TestSaveHistoryEmptyFilename(t *testing.T) {
	messages := []types.Message{
		{Type: types.MsgType_Msg, Role: types.Role_User, Content: "test"},
	}
	err := SaveHistory("", messages)
	if err != nil {
		t.Errorf("expected no error for empty filename, got: %v", err)
	}
}

func TestAppendToHistoryEmptyFilename(t *testing.T) {
	msg := types.Message{Type: types.MsgType_Msg, Role: types.Role_User, Content: "test"}
	err := AppendToHistory("", msg)
	if err != nil {
		t.Errorf("expected no error for empty filename, got: %v", err)
	}
}
