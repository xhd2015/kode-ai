package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xhd2015/kode-ai/internal/terminal"
	"github.com/xhd2015/kode-ai/providers"
	"github.com/xhd2015/kode-ai/types"
)

// JSONLogEntry represents a structured log entry for JSON output
type JSONLogEntry struct {
	Type      string      `json:"type"`
	Content   string      `json:"content,omitempty"`
	Metadata  interface{} `json:"metadata,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}

// CliOptions represents CLI-specific options that don't belong in core library
type CliOptions struct {
	RecordFile         string // File recording for session persistence
	IgnoreDuplicateMsg bool   // Interactive duplicate message handling
	LogRequest         bool   // Debug request logging
	LogChat            bool   // Chat progress logging
	Verbose            bool   // Verbose output
	JSONOutput         bool   // Output response as JSON

	StreamPair *types.StreamPair
}

// CliHandler wraps the core client with CLI-specific functionality
type CliHandler struct {
	client *Client
	opts   CliOptions
}

// NewCliHandler creates a new CLI handler
func NewCliHandler(client *Client, opts CliOptions) *CliHandler {
	return &CliHandler{
		client: client,
		opts:   opts,
	}
}

// HandleCLI handles a chat request with CLI-specific behavior
func (h *CliHandler) HandleCli(ctx context.Context, message string, coreOpts ...types.ChatOption) error {
	// Load history if record file is specified
	var loadedHistory []types.Message
	if h.opts.RecordFile != "" {
		var err error
		loadedHistory, err = h.loadHistory()
		if err != nil {
			return fmt.Errorf("load history: %w", err)
		}
	}

	// Check for duplicate messages
	message, stop, err := h.checkDuplicateMessage(message, loadedHistory)
	if err != nil {
		return fmt.Errorf("check duplicate message: %w", err)
	}
	if stop {
		return nil
	}

	// Create event callback for output formatting
	var eventCallback types.EventCallback
	if h.opts.JSONOutput {
		var stdout io.Writer
		if h.opts.StreamPair != nil {
			stdout = h.opts.StreamPair.Output
		} else {
			stdout = os.Stdout
		}
		encoder := json.NewEncoder(stdout)
		eventCallback = func(event types.Message) {
			// will automatically add newline
			event = event.TimeFilled()
			encoder.Encode(event)
		}
	} else {
		eventCallback = func(event types.Message) {
			h.formatOutput(event)
		}
	}

	// Prepare core options
	allOpts := append(coreOpts, WithHistory(loadedHistory))
	allOpts = append(allOpts, WithEventCallback(eventCallback))

	// Log chat start if enabled
	if h.opts.LogChat {
		eventCallback(types.Message{
			Type:      types.MsgType_Info,
			Content:   "Request...",
			Timestamp: time.Now().Unix(),
		})
	}
	req := types.Request{
		Message: message,
	}

	// Apply options
	for _, opt := range allOpts {
		opt(&req)
	}

	h.opts.StreamPair = req.StreamPair
	return h.handleCliRequest(ctx, req)
}

func (h *CliHandler) handleCliRequest(ctx context.Context, req types.Request) error {
	// Execute chat
	response, err := h.client.ChatRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("chat request: %w", err)
	}

	// Record messages if record file is specified
	if h.opts.RecordFile != "" && req.Message != "" {
		// Record user message first
		userMsg := CreateMessage(types.MsgType_Msg, types.Role_User, h.client.config.Model, req.Message)
		if err := h.saveToRecord(userMsg); err != nil {
			return fmt.Errorf("record user message: %w", err)
		}

		// TODO: record all response messages
		// // Record all response messages
		// for _, msg := range response.Messages {
		// 	if err := h.saveToRecord(msg); err != nil {
		// 		return fmt.Errorf("record response message: %w", err)
		// 	}
		// }
	}

	// Log token usage if enabled
	if h.opts.LogChat && (!h.opts.JSONOutput && h.opts.StreamPair == nil) {
		var costUSD string
		if response.Cost != nil {
			costUSD = "$" + response.Cost.TotalUSD
		}
		h.printTokenUsage("Token Usage", response.TokenUsage, costUSD)
	}

	return nil
}

// loadHistory loads historical messages from the record file
func (h *CliHandler) loadHistory() ([]types.Message, error) {
	return LoadHistory(h.opts.RecordFile)
}

// saveToRecord saves a message to the record file
func (h *CliHandler) saveToRecord(msg types.Message) error {
	return AppendToHistory(h.opts.RecordFile, msg)
}

// checkDuplicateMessage checks for duplicate messages and handles user interaction
func (h *CliHandler) checkDuplicateMessage(msg string, history []types.Message) (string, bool, error) {
	lastUserMsg := GetLastUserMessage(history)

	if msg != "" && lastUserMsg != nil && msg == lastUserMsg.Content {
		if h.opts.IgnoreDuplicateMsg {
			msg = ""
		} else {
			if !terminal.IsStdinTTY() {
				return "", true, fmt.Errorf("duplicate user msg, either clear the msg or run with --ignore-duplicate-msg")
			}
			// prompt user: duplicate msg, continue?
			prompt := fmt.Sprintf("Duplicate input with last msg created at %s, proceed?\n  c:proceed with duplicate, x:proceed without duplicate, q:quit, a:ask with a different question", lastUserMsg.Time)
			for {
				reader := bufio.NewReader(os.Stdin)
				fmt.Println(prompt)
				fmt.Print("user> ")
				response, err := reader.ReadString('\n')
				if err != nil {
					return "", false, fmt.Errorf("failed to read response: %v", err)
				}
				decision := strings.TrimSuffix(response, "\n")
				if suffix, ok := strings.CutPrefix(decision, "a:"); ok {
					if suffix == "" {
						continue
					}
					msg = suffix
					break
				}
				if decision == "c" {
					break
				}
				if decision == "q" {
					return "", true, nil
				}
				if decision == "x" {
					msg = ""
					break
				}
			}
		}
	}
	return msg, false, nil
}

// formatOutput formats events for CLI output
func (h *CliHandler) formatOutput(event types.Message) {
	event = event.TimeFilled()
	switch event.Type {
	case types.MsgType_Msg:
		// Print message content directly (streaming)
		fmt.Print(event.Content)

	case types.MsgType_ToolCall:
		toolCallStr := fmt.Sprintf("<tool_call>%s(%s)</tool_call>", event.ToolName, event.Content)
		fmt.Println(toolCallStr)

	case types.MsgType_ToolResult:
		toolResultStr := fmt.Sprintf("<tool_result>%s</tool_result>", event.Content)
		fmt.Println(toolResultStr)

	case types.MsgType_TokenUsage:
		if h.opts.Verbose {
			tokenUsage := event.TokenUsage
			if tokenUsage != nil {
				h.printTokenUsage("Token Usage", *tokenUsage, "")
			}
		}

	case types.MsgType_Error:
		fmt.Printf("Error: %v\n", event.Error)

	case types.MsgType_CacheInfo:
		if h.opts.LogChat {
			fmt.Println(event.Content)
		}
	}
}

// printTokenUsage prints token usage information
func (h *CliHandler) printTokenUsage(title string, tokenUsage types.TokenUsage, cost string) {
	if cost == "" {
		cost = h.getTotalTokenCost(tokenUsage)
	}
	fmt.Fprintf(os.Stderr, "%s - Input: %d, Cache/R: %d, Cache/W: %d, NonCache/R: %d, Output: %d, Total: %d, Cost: %s\n",
		title,
		tokenUsage.Input,
		tokenUsage.InputBreakdown.CacheRead,
		tokenUsage.InputBreakdown.CacheWrite,
		tokenUsage.InputBreakdown.NonCacheRead,
		tokenUsage.Output,
		tokenUsage.Total,
		cost,
	)
}

func (h *CliHandler) getTotalTokenCost(tokenUsage types.TokenUsage) string {
	provider, err := providers.GetModelAPIShape(h.client.config.Model)
	if err != nil {
		return ""
	}
	cost, costOK := providers.ComputeCost(provider, h.client.config.Model, tokenUsage)
	if !costOK {
		return ""
	}
	return "$" + cost.TotalUSD
}
