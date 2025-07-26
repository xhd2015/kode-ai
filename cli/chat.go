// Package cli provides a client interface that wraps the kode CLI binary
// to provide the same programming interface as the chat package.
//
// This package allows clients to interact with the kode CLI as a pure binary
// dependency via stdin/stdout using --json mode. The CLI converts events to
// Go function callbacks, making it feel identical to using the chat package
// directly.
//
// Usage Example:
//
//	import "github.com/xhd2015/kode-ai/cli"
//
//	// Create a client (identical interface to chat.NewClient)
//	client, err := cli.NewClient(cli.Config{
//		Model: "gpt-4o",
//		Token: "your-api-token",
//	})
//	if err != nil {
//		panic(err)
//	}
//
//	// Use the client with functional options (identical to chat package)
//	response, err := client.Chat(ctx, "Hello, world!",
//		cli.WithSystemPrompt("You are a helpful assistant"),
//		cli.WithMaxRounds(3),
//		cli.WithTools("list_dir", "read_file"),
//		cli.WithEventCallback(func(event cli.Event) {
//			fmt.Printf("Event: %s - %s\n", event.Type, event.Content)
//		}),
//		cli.WithToolCallback(func(ctx context.Context, call cli.ToolCall) (cli.ToolResult, bool, error) {
//			// Custom tool handling - this now works with the stream protocol!
//			if call.Name == "custom_tool" {
//				return cli.ToolResult{
//					Content: map[string]interface{}{
//						"result": "Custom tool executed successfully",
//					},
//				}, true, nil
//			}
//			return cli.ToolResult{}, false, nil // Let builtin tools handle it
//		}),
//	)
//
// Key Features:
//
// - Drop-in replacement for the chat package
// - Uses kode CLI binary with --json mode for communication
// - Supports all the same functional options (WithSystemPrompt, WithTools, etc.)
// - Provides event callbacks for real-time event processing
// - Supports custom tool callbacks via bidirectional stream protocol
// - Pure binary dependency - no need to import chat package dependencies
//
// The client communicates with the kode CLI process via stdin/stdout using JSON
// messages. Events from the CLI are parsed and converted to Go function callbacks.
// Tool callbacks use a bidirectional stream protocol where the CLI sends tool_call_request
// messages and waits for tool_call_response messages, providing a seamless integration experience.
package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/xhd2015/kode-ai/types"
)

type CliOption func(cfg *CliOptionConfig)

type CliOptionConfig struct {
	cli     string
	cliArgs []string
	envs    []string
	dir     string
}

func WithCli(cli string, args ...string) CliOption {
	return func(cfg *CliOptionConfig) {
		cfg.cli = cli
		cfg.cliArgs = args
	}
}

func WithEnv(envs ...string) CliOption {
	return func(cfg *CliOptionConfig) {
		cfg.envs = append(cfg.envs, envs...)
	}
}

func WithDir(dir string) CliOption {
	return func(cfg *CliOptionConfig) {
		cfg.dir = dir
	}
}

func Chat(ctx context.Context, req types.Request, opts ...CliOption) (*types.Response, error) {
	sess := &session{}
	return sess.chat(ctx, req, opts...)
}

type session struct {
	stream        types.StreamContext
	eventCallback types.EventCallback
	logger        types.Logger

	lastAssistantMsg string
}

// ChatRequest performs a chat conversation using a direct request
func (c *session) chat(ctx context.Context, req types.Request, opts ...CliOption) (*types.Response, error) {
	cfg := &CliOptionConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	c.eventCallback = req.EventCallback
	c.logger = getLogger(req.Logger)
	if req.StreamPair != nil {
		return nil, fmt.Errorf("stream pair is not supported")
	}

	// Build command arguments
	args := []string{"chat", "--std-stream", "--wait-for-stream-events"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.Token != "" {
		args = append(args, "--token", req.Token)
	}

	if req.BaseURL != "" {
		args = append(args, "--base-url", req.BaseURL)
	}

	if req.MaxRounds > 0 {
		args = append(args, "--max-round", strconv.Itoa(req.MaxRounds))
	}

	for _, tool := range req.Tools {
		args = append(args, "--tool", tool)
	}

	for _, toolFile := range req.ToolFiles {
		args = append(args, "--tool-custom", toolFile)
	}

	for _, toolJSON := range req.ToolJSONs {
		args = append(args, "--tool-custom-json", toolJSON)
	}

	for _, toolDefinition := range req.ToolDefinitions {
		json, err := json.Marshal(toolDefinition)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tool definition: %w", err)
		}
		args = append(args, "--tool-custom-json", string(json))
	}

	if req.DefaultToolCwd != "" {
		args = append(args, "--tool-default-cwd", req.DefaultToolCwd)
	}

	for _, mcpServer := range req.MCPServers {
		args = append(args, "--mcp", mcpServer)
	}

	if req.NoCache {
		args = append(args, "--no-cache")
	}

	cli := "kode"
	if cfg.cli != "" {
		cli = cfg.cli
	}

	if len(cfg.cliArgs) > 0 {
		combinedArgs := make([]string, len(cfg.cliArgs)+len(args))
		copy(combinedArgs, cfg.cliArgs)
		copy(combinedArgs[len(cfg.cliArgs):], args)
		args = combinedArgs
	}

	// Create command
	cmd := exec.CommandContext(ctx, cli, args...)
	if len(cfg.envs) > 0 {
		cmd.Env = append(os.Environ(), cfg.envs...)
	}
	if cfg.dir != "" {
		cmd.Dir = cfg.dir
	}

	// Set up pipes for stdin/stdout communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	c.stream = types.NewStreamContext(stdin)

	// wait stream events
	// send all and prompts history and message to stdin
	// history
	for _, msg := range req.History {
		if !msg.Type.HistorySendable() {
			continue
		}
		if err := c.writeEventNoLock(msg); err != nil {
			return nil, fmt.Errorf("failed to write history: %w", err)
		}
	}

	// system prompt
	if req.SystemPrompt != "" {
		if err := c.writeEventNoLock(types.Message{
			Type:    types.MsgType_Msg,
			Role:    types.Role_System,
			Content: req.SystemPrompt,
		}); err != nil {
			return nil, fmt.Errorf("failed to write system prompt: %w", err)
		}
	}
	if req.Message != "" {
		if err := c.writeEventNoLock(types.Message{
			Type:    types.MsgType_Msg,
			Role:    types.Role_User,
			Content: req.Message,
		}); err != nil {
			return nil, fmt.Errorf("failed to write message: %w", err)
		}
	}
	// send a finish
	if err := c.writeEventNoLock(types.Message{
		Type: types.MsgType_StreamInitEventsFinished,
	}); err != nil {
		return nil, fmt.Errorf("failed to write finish: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Handle stderr
	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			// Log stderr output if needed
			c.logger.Log(ctx, types.LogType_Info, "%s\n", scanner.Text())
		}
		close(done)
	}()

	// Process stdout with both events and tool callbacks
	response, err := c.processStdoutWithToolCallbacks(ctx, stdin, stdout, req.ToolCallback, req.FollowUpCallback, req.ToolDefinitions)
	if err != nil {
		return nil, fmt.Errorf("failed to process stdout: %w", err)
	}

	// Wait for command to finish
	err = cmd.Wait()
	<-done
	if err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	}

	return response, nil
}

// handleSingleToolCallback handles a single tool callback request using the stream protocol
func (c *session) handleSingleToolCallback(ctx context.Context, toolCallRequest types.Message, stdin io.Writer, toolCallback types.ToolCallback) {
	if c.stream == nil {
		return
	}
	streamID := toolCallRequest.StreamID
	if streamID == "" || toolCallRequest.ToolName == "" {
		return
	}

	// ack
	err := c.stream.ACK(streamID)
	if err != nil {
		c.logger.Log(ctx, types.LogType_Error, "failed to ack stream: %v\n", err)
		return
	}

	argsJSON := toolCallRequest.Content
	var args map[string]interface{}
	if err := unmarshalSafe([]byte(argsJSON), &args); err != nil {
		c.logger.Log(ctx, types.LogType_Error, "failed to unmarshal arguments: %v\n", err)
		return
	}

	var workingDir string
	if toolCallRequest.Metadata.StreamRequestTool != nil {
		workingDir = toolCallRequest.Metadata.StreamRequestTool.DefaultWorkingDir
	}

	// Create ToolCall struct
	call := types.ToolCall{
		ID:         streamID,
		Name:       toolCallRequest.ToolName,
		Arguments:  args,
		RawArgs:    argsJSON,
		WorkingDir: workingDir,
	}

	// Execute the tool callback
	result, handled, err := toolCallback(ctx, c.stream, call)

	var toolError string

	if err != nil {
		toolError = err.Error()
	}

	var contentAsStr string
	switch res := result.Content.(type) {
	case string:
		contentAsStr = res
	case []byte:
		contentAsStr = string(res)
	}
	var resultJSON string
	if contentAsStr != "" {
		if !strings.HasPrefix(contentAsStr, "{") || !strings.HasSuffix(contentAsStr, "}") {
			resultJSON = fmt.Sprintf("{\"result\": %q}", contentAsStr)
		} else {
			resultJSON = contentAsStr
		}
	} else {
		json, err := json.Marshal(result.Content)
		if err != nil {
			resultJSON = fmt.Sprintf("failed to marshal result: %v", result.Content)
		} else {
			resultJSON = string(json)
		}
	}

	response := types.Message{
		Type:     types.MsgType_StreamResponseTool,
		StreamID: streamID,
		ToolName: toolCallRequest.ToolName,
		Content:  resultJSON,
		Error:    toolError,
		Metadata: types.Metadata{
			StreamResponseTool: &types.StreamResponseToolMetadata{
				OK: handled,
			},
		},
	}

	err = c.writeEvent(response)
	if err != nil {
		c.logger.Log(ctx, types.LogType_Error, "failed to write response: %v\n", err)
	}
}

// processStdoutWithToolCallbacks processes stdout handling both events and tool callbacks
func (c *session) processStdoutWithToolCallbacks(ctx context.Context, stdin io.WriteCloser, stdout io.ReadCloser, toolCallback types.ToolCallback, followUpCallback types.FollowUpCallback, toolDefs []*types.UnifiedTool) (*types.Response, error) {
	defer stdin.Close()
	defer stdout.Close()

	scanner := bufio.NewScanner(stdout)
	var response types.Response

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse as event for regular event processing
		var msg types.Message
		if err := unmarshalSafe([]byte(line), &msg); err != nil {
			c.logger.Log(ctx, types.LogType_Error, "malformed event: %s\n", line)
			continue // Skip malformed events
		}

		if msg.Type == types.MsgType_Msg && msg.Role == types.Role_Assistant {
			c.lastAssistantMsg = msg.Content
		}

		if c.eventCallback != nil {
			c.eventCallback(msg)
		}

		// requires i/o stream
		if c.stream != nil {
			var unableToHandle bool
			switch msg.Type {
			case types.MsgType_StreamRequestTool:
				// Validate tool call data and log debug info
				var foundToolCallback types.ToolCallback
				if msg.ToolName != "" && msg.Content != "" {
					for _, toolDef := range toolDefs {
						if toolDef.Name == msg.ToolName {
							if toolDef.Handle != nil {
								foundToolCallback = toolDef.Handle
							} else if len(toolDef.Command) > 0 {
								foundToolCallback = makeCmdToolCallback(toolDef)
							}
							break
						}
					}
				}
				if foundToolCallback == nil {
					foundToolCallback = toolCallback
				}
				if foundToolCallback != nil {
					c.handleSingleToolCallback(ctx, msg, stdin, foundToolCallback)
					continue
				}
				unableToHandle = true
			case types.MsgType_StreamRequestUserMsg:
				if followUpCallback != nil {
					// ack
					err := c.stream.ACK(msg.StreamID)
					if err != nil {
						c.logger.Log(ctx, types.LogType_Error, "failed to ack stream: %v\n", err)
						continue
					}
					followUpResult, err := followUpCallback(ctx)
					if err != nil {
						c.writeEvent(types.Message{
							Type:     types.MsgType_Error,
							StreamID: msg.StreamID,
							Error:    err.Error(),
						})
						continue
					}
					if followUpResult != nil {
						fmsg := *followUpResult
						if fmsg.Type == "" {
							fmsg.Type = types.MsgType_Msg
						}
						if fmsg.Role == "" {
							fmsg.Role = types.Role_User
						}
						fmsg.StreamID = msg.StreamID
						c.writeEvent(fmsg)
						continue
					}
				}
				unableToHandle = true
			}
			if unableToHandle {
				writeErr := c.writeEvent(types.Message{
					Type:     types.MsgType_StreamEnd,
					StreamID: msg.StreamID,
				})
				if writeErr != nil {
					c.logger.Log(ctx, types.LogType_Error, "failed to write stream end: %v\n", writeErr)
				}
				continue
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading from stdout: %w", err)
	}
	response.LastAssistantMsg = c.lastAssistantMsg

	return &response, nil
}

func makeCmdToolCallback(toolDef *types.UnifiedTool) func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
	return func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
		toolResult := types.ToolResult{}

		var stdout bytes.Buffer

		cmd := exec.CommandContext(ctx, toolDef.Command[0], toolDef.Command[1:]...)
		cmd.Stdout = &stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = call.WorkingDir

		if err := cmd.Start(); err != nil {
			return types.ToolResult{}, true, fmt.Errorf("failed to start command: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			return types.ToolResult{}, true, fmt.Errorf("failed to wait for command: %w", err)
		}

		toolResult.Content = stdout
		return toolResult, true, nil
	}
}
