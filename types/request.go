package types

import (
	"context"
	"io"
)

// Request represents a chat request
type Request struct {
	Model   string `json:"model"`
	Token   string `json:"token"`
	BaseURL string `json:"base_url"`

	SystemPrompt string    `json:"system_prompt"`
	Message      string    `json:"message"`
	History      []Message `json:"history"`

	MaxRounds       int            `json:"max_rounds"`
	Tools           []string       `json:"tools"`
	ToolFiles       []string       `json:"tool_files"`
	ToolJSONs       []string       `json:"tool_jsons"`
	ToolDefinitions []*UnifiedTool `json:"tool_definitions"`
	DefaultToolCwd  string         `json:"default_tool_cwd"`

	NoCache    bool     `json:"no_cache"`
	MCPServers []string `json:"mcp_servers"`

	Logger Logger `json:"-"`

	// functional options
	EventCallback    EventCallback    `json:"-"` // Cannot be serialized
	ToolCallback     ToolCallback     `json:"-"` // Cannot be serialized
	FollowUpCallback FollowUpCallback `json:"-"` // Cannot be serialized

	// Stream fields for bidirectional tool callback communication
	StreamPair *StreamPair `json:"-"` // Cannot be serialized
}

type LogType string

const (
	LogType_Info  LogType = "info"
	LogType_Error LogType = "error"
)

type Logger interface {
	Log(ctx context.Context, logType LogType, format string, args ...interface{})
}

type StreamPair struct {
	Input  io.Reader `json:"-"` // Cannot be serialized - for reading tool callback responses
	Output io.Writer `json:"-"` // Cannot be serialized - for writing tool callback requests
}

// Response represents a chat response
type Response struct {
	// TODO: populate these fields
	TokenUsage TokenUsage `json:"token_usage"` // Token consumption details
	Cost       *TokenCost `json:"cost"`        // Cost information if available
	StopReason string     `json:"stop_reason"` // Why the conversation stopped
	RoundsUsed int        `json:"rounds_used"` // Number of conversation rounds used

	// last response message of the chat
	// assitant respones including msg and tool calls
	// LastAssistantMsg in the exactly the last assistant msg, excluding
	// tool calls
	LastAssistantMsg string `json:"last_assistant_response"`
}

type LoggerFunc func(ctx context.Context, logType LogType, format string, args ...interface{})

func (l LoggerFunc) Log(ctx context.Context, logType LogType, format string, args ...interface{}) {
	l(ctx, logType, format, args...)
}
