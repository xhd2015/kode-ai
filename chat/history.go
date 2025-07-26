package chat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/xhd2015/kode-ai/types"
)

// LoadHistory loads historical messages from a file
func LoadHistory(filename string) ([]types.Message, error) {
	if filename == "" {
		return nil, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, return empty history
		}
		return nil, fmt.Errorf("open history file: %w", err)
	}
	defer file.Close()

	var messages []types.Message
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg types.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, fmt.Errorf("parse history message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read history file: %w", err)
	}

	return messages, nil
}

// SaveHistory saves messages to a file (overwrites existing file)
func SaveHistory(filename string, messages []types.Message) error {
	if filename == "" {
		return nil
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create history file: %w", err)
	}
	defer file.Close()

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal message: %w", err)
		}
		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("write message: %w", err)
		}
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("write newline: %w", err)
		}
	}

	return nil
}

// AppendToHistory appends a single message to a history file
func AppendToHistory(filename string, message types.Message) error {
	if filename == "" {
		return nil
	}

	// Set timestamp if not already set
	if message.Time == "" {
		message.Time = time.Now().Format(time.RFC3339)
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open history file for append: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}

	return nil
}

// FilterHistoryByType filters messages by type
func FilterHistoryByType(messages []types.Message, msgType types.MsgType) []types.Message {
	var filtered []types.Message
	for _, msg := range messages {
		if msg.Type == msgType {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// GetLastUserMessage returns the last user message from history
func GetLastUserMessage(messages []types.Message) *types.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Type == types.MsgType_Msg && msg.Role == types.Role_User {
			return &msg
		}
	}
	return nil
}

// GetSystemPrompts extracts all system prompts from message history
func GetSystemPrompts(messages []types.Message) []string {
	var prompts []string
	for _, msg := range messages {
		if msg.Type == types.MsgType_Msg && msg.Role == types.Role_System {
			prompts = append(prompts, msg.Content)
		}
	}
	return prompts
}

// CreateMessage creates a new message with timestamp
func CreateMessage(msgType types.MsgType, role types.Role, model, content string) types.Message {
	return types.Message{
		Type:    msgType,
		Time:    time.Now().Format(time.RFC3339),
		Role:    role,
		Model:   model,
		Content: content,
	}
}

// CreateToolCallMessage creates a tool call message
func CreateToolCallMessage(role types.Role, model, toolName, toolUseID, content string) types.Message {
	return types.Message{
		Type:      types.MsgType_ToolCall,
		Time:      time.Now().Format(time.RFC3339),
		Role:      role,
		Model:     model,
		Content:   content,
		ToolUseID: toolUseID,
		ToolName:  toolName,
	}
}

// CreateToolResultMessage creates a tool result message
func CreateToolResultMessage(role types.Role, model, toolName, toolUseID, content string) types.Message {
	return types.Message{
		Type:      types.MsgType_ToolResult,
		Time:      time.Now().Format(time.RFC3339),
		Role:      role,
		Model:     model,
		Content:   content,
		ToolUseID: toolUseID,
		ToolName:  toolName,
	}
}
