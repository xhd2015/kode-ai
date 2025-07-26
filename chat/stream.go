package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xhd2015/kode-ai/internal/jsondecode"
	"github.com/xhd2015/kode-ai/types"
)

// const TOOL_ACK_TIMEOUT = 1 * time.Second

// executeToolWithStream executes a tool using bidirectional stream communication
// Enhanced protocol with background stdin reader:
// 1. Write tool_call_request to stdout
// 2. Wait for tool_call_handle_begin from stdin (within 1s timeout)
// 3. Wait for tool_call_response from stdin (after tool execution)
func executeToolWithStream(ctx context.Context, call types.ToolCall, stdout io.Writer, reader types.StdinReader, defaultWorkingDir string) (types.ToolResult, bool, error) {
	// Validate call.ID is not empty
	if call.ID == "" {
		return types.ToolResult{}, false, fmt.Errorf("tool call ID cannot be empty")
	}

	jsonArgs, err := json.Marshal(call.Arguments)
	if err != nil {
		return types.ToolResult{}, false, fmt.Errorf("marshal tool call arguments: %w", err)
	}

	toolCallRequest := types.Message{
		Type:     types.MsgType_StreamRequestTool,
		StreamID: call.ID,
		ToolName: call.Name,
		Content:  string(jsonArgs),
		Metadata: types.Metadata{
			StreamRequestTool: &types.StreamRequestToolMetadata{
				DefaultWorkingDir: defaultWorkingDir,
			},
		},
	}

	respMsg, err := types.StreamRequest(ctx, stdout, reader, toolCallRequest, types.MsgType_StreamResponseTool)
	if err != nil {
		return types.ToolResult{}, false, err
	}
	return processToolResponse(respMsg)
}

// processToolResponse processes a tool_call_response message
func processToolResponse(msg types.Message) (types.ToolResult, bool, error) {
	if msg.Metadata.StreamResponseTool == nil || !msg.Metadata.StreamResponseTool.OK {
		return types.ToolResult{}, false, nil
	}

	var res interface{}
	content := msg.Content
	// check if res is a valid json, if not, wrap it
	if !strings.HasPrefix(content, "{") || !strings.HasSuffix(content, "}") {
		res = map[string]interface{}{
			"result": content,
		}
	} else {
		var err error
		res, err = jsondecode.UnmarshalSafeAny([]byte(msg.Content))
		if err != nil {
			return types.ToolResult{
				Error: err.Error(),
			}, false, nil
		}
	}

	return types.ToolResult{
		Content: res,
		Error:   msg.Error,
	}, true, nil
}
