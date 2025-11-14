package types

import (
	"context"
	"encoding/json"
	"io"
	"time"

	_ "embed"

	"github.com/xhd2015/llm-tools/jsonschema"
)

//go:embed tool.go
var UnifiedToolDef string

type JsonSchema = jsonschema.JsonSchema
type ParamType = jsonschema.ParamType

// LogLevel represents the logging level for the client
type LogLevel int

const (
	LogLevelNone LogLevel = iota
	LogLevelRequest
	LogLevelResponse
	LogLevelDebug
)

// MsgType represents the type of message
type MsgType string

const (
	// for both logs and input
	MsgType_Msg        MsgType = "msg"
	MsgType_ToolCall   MsgType = "tool_call"
	MsgType_ToolResult MsgType = "tool_result"

	// for logs only
	MsgType_Info       MsgType = "info"
	MsgType_Error      MsgType = "error"
	MsgType_CacheInfo  MsgType = "cache_info"
	MsgType_StopReason MsgType = "stop_reason"
	MsgType_TokenUsage MsgType = "token_usage"

	// for stream
	MsgType_StreamRequestTool    MsgType = "stream_request_tool"
	MsgType_StreamResponseTool   MsgType = "stream_response_tool"
	MsgType_StreamRequestUserMsg MsgType = "stream_request_user_msg"
	MsgType_StreamHandleAck      MsgType = "stream_handle_ack"
	MsgType_StreamEnd            MsgType = "stream_end" // cannot handle message

	// for initial stream
	MsgType_StreamInitRequest        MsgType = "stream_init_request"
	MsgType_StreamInitEventsFinished MsgType = "stream_init_events_finished"
)

func (m MsgType) HistorySendable() bool {
	return m == MsgType_Msg || m == MsgType_ToolCall || m == MsgType_ToolResult
}

// Role represents the role of a message sender
type Role string

const (
	Role_User      Role = "user"
	Role_Assistant Role = "assistant"
	Role_System    Role = "system"
)

// Message represents a message in the chat conversation
type Message struct {
	Type MsgType `json:"type"`
	// Annotation for Timestamp
	Time  string `json:"time"`
	Role  Role   `json:"role"`
	Model string `json:"model"`

	// general content
	Content string `json:"content"`
	Error   string `json:"error,omitempty"` // meaningful when: Type == MsgType_StreamEnd, MsgType_Error, MsgType_ToolCall

	// for tool call
	ToolUseID string `json:"tool_use_id,omitempty"`
	ToolName  string `json:"tool_name,omitempty"`

	// StreamID for stream
	StreamID string `json:"stream_id,omitempty"`

	// for message token usage record
	TokenUsage *TokenUsage `json:"token_usage,omitempty"`

	// for message token cost record
	TokenCost *TokenCost `json:"token_cost,omitempty"`

	// Extended structured metadata
	Metadata Metadata `json:"metadata,omitempty"`

	// unix timestamp, accurate
	Timestamp int64 `json:"timestamp,omitempty"`
}

type Metadata struct {
	CacheInfo          *CacheInfoMetadata          `json:"cache_info,omitempty"`
	RoundStart         *RoundStartMetadata         `json:"round_start,omitempty"`
	RoundEnd           *RoundEndMetadata           `json:"round_end,omitempty"`
	StreamRequestTool  *StreamRequestToolMetadata  `json:"stream_request_tool,omitempty"`
	StreamResponseTool *StreamResponseToolMetadata `json:"stream_response_tool,omitempty"`
}

func (c Message) TimeFilled() Message {
	if c.Timestamp == 0 {
		now := time.Now()
		c.Timestamp = now.Unix()
		c.Time = now.Format(time.RFC3339)
	} else if c.Time == "" {
		t := time.Unix(c.Timestamp, 0)
		c.Time = t.Format(time.RFC3339)
	}
	return c
}

// Messages represents a slice of messages
type Messages []Message

// TokenUsageInputBreakdown represents input token breakdown
type TokenUsageInputBreakdown struct {
	CacheWrite   int64 `json:"cache_write"`
	CacheRead    int64 `json:"cache_read"`
	NonCacheRead int64 `json:"non_cache_read"`
}

// Add adds two TokenUsageInputBreakdown together
func (t TokenUsageInputBreakdown) Add(other TokenUsageInputBreakdown) TokenUsageInputBreakdown {
	return TokenUsageInputBreakdown{
		CacheWrite:   t.CacheWrite + other.CacheWrite,
		CacheRead:    t.CacheRead + other.CacheRead,
		NonCacheRead: t.NonCacheRead + other.NonCacheRead,
	}
}

// TokenUsageOutputBreakdown represents output token breakdown
type TokenUsageOutputBreakdown struct {
	CacheOutput int64 `json:"cache_output"`
}

// Add adds two TokenUsageOutputBreakdown together
func (t TokenUsageOutputBreakdown) Add(other TokenUsageOutputBreakdown) TokenUsageOutputBreakdown {
	return TokenUsageOutputBreakdown{
		CacheOutput: t.CacheOutput + other.CacheOutput,
	}
}

// Anthropic:
//   - how to: https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching
//   - when: https://www.anthropic.com/news/prompt-caching
//   - summary:
//     . seems anthropic only caches for long enough texts
//     . The minimum cacheable prompt length is:
//     . 1024 tokens for Claude Opus 4, Claude Sonnet 4, Claude Sonnet 3.7, Claude Sonnet 3.5 and Claude Opus 3
//     . The cache is invalidated after 5 minutes
//
// TokenUsage represents token usage information
type TokenUsage struct {
	Input           int64                     `json:"input"`
	Output          int64                     `json:"output"`
	Total           int64                     `json:"total"`
	InputBreakdown  TokenUsageInputBreakdown  `json:"input_breakdown"`
	OutputBreakdown TokenUsageOutputBreakdown `json:"output_breakdown"`
}

// Add adds two TokenUsage together
func (t TokenUsage) Add(other TokenUsage) TokenUsage {
	return TokenUsage{
		Input:           t.Input + other.Input,
		Output:          t.Output + other.Output,
		Total:           t.Total + other.Total,
		InputBreakdown:  t.InputBreakdown.Add(other.InputBreakdown),
		OutputBreakdown: t.OutputBreakdown.Add(other.OutputBreakdown),
	}
}

// TokenCostInputBreakdown represents input token cost breakdown
type TokenCostInputBreakdown struct {
	CacheWriteUSD   string `json:"cache_write_usd"`
	CacheReadUSD    string `json:"cache_read_usd"`
	NonCacheReadUSD string `json:"non_cache_read_usd"`
}

// Add adds two TokenCostInputBreakdown together
func (t TokenCostInputBreakdown) Add(other TokenCostInputBreakdown) TokenCostInputBreakdown {
	return TokenCostInputBreakdown{
		CacheWriteUSD:   addDecimals(t.CacheWriteUSD, other.CacheWriteUSD),
		CacheReadUSD:    addDecimals(t.CacheReadUSD, other.CacheReadUSD),
		NonCacheReadUSD: addDecimals(t.NonCacheReadUSD, other.NonCacheReadUSD),
	}
}

// TokenCost represents cost information
type TokenCost struct {
	InputUSD       string                  `json:"input_usd"`
	OutputUSD      string                  `json:"output_usd"`
	TotalUSD       string                  `json:"total_usd"`
	InputBreakdown TokenCostInputBreakdown `json:"input_breakdown"`
}

// Add adds two TokenCost together
func (t TokenCost) Add(other TokenCost) TokenCost {
	return TokenCost{
		InputUSD:       addDecimals(t.InputUSD, other.InputUSD),
		OutputUSD:      addDecimals(t.OutputUSD, other.OutputUSD),
		TotalUSD:       addDecimals(t.TotalUSD, other.TotalUSD),
		InputBreakdown: t.InputBreakdown.Add(other.InputBreakdown),
	}
}

type StreamContext interface {
	// ACK will handle it
	ACK(id string) error

	Write(msg Message) error
}

type streamContext struct {
	out io.Writer
}

func NewStreamContext(out io.Writer) StreamContext {
	return &streamContext{
		out: out,
	}
}

func (s *streamContext) ACK(id string) error {
	return s.Write(Message{
		Type:      MsgType_StreamHandleAck,
		ToolUseID: id,
	})
}

func (s *streamContext) Write(msg Message) error {
	msg = msg.TimeFilled()
	return json.NewEncoder(s.out).Encode(msg)
}

// TokenUsageCost combines usage and cost information
type TokenUsageCost struct {
	Usage TokenUsage `json:"usage"`
	Cost  TokenCost  `json:"cost"`
}

// EventCallback is called for each message during chat processing
type EventCallback func(msg Message)

// ToolCall represents a tool call
type ToolCall struct {
	ID         string                 `json:"id"`          // Unique identifier for this tool call
	Name       string                 `json:"name"`        // Tool name
	Arguments  map[string]interface{} `json:"arguments"`   // Parsed JSON arguments
	RawArgs    string                 `json:"raw_args"`    // Raw JSON string arguments
	WorkingDir string                 `json:"working_dir"` // Working directory for tool execution
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content interface{} `json:"content"`         // Result data (must be JSON serializable)
	Error   string      `json:"error,omitempty"` // Tool execution error (if any)
}

// ToolCallback allows custom tool execution
// Returns: (result, handled, error)
// - result: Tool execution result
// - handled: true if tool was handled by callback, false to fallback to built-in tools
// - error: Any execution error
type ToolCallback func(ctx context.Context, stream StreamContext, call ToolCall) (ToolResult, bool, error)

// FollowUpCallback allows custom follow-up tool execution
// Returns: (result, handled, error)
// - result: Follow-up message, nil to indicate end of conversation
// - error: Any execution error
type FollowUpCallback func(ctx context.Context) (*Message, error)

// StringOrList represents a value that can be either a string or a list of strings
type StringOrList = interface{}
