package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/xhd2015/kode-ai/internal/jsondecode"
	"github.com/xhd2015/kode-ai/tools"
	"github.com/xhd2015/kode-ai/types"
)

// ToolInfo represents information about a tool
type ToolInfo struct {
	Name           string
	Builtin        bool
	ToolDefinition *tools.UnifiedTool
	MCPServer      string
	MCPClient      *client.Client
}

// ToolInfoMapping maps tool names to their information
type ToolInfoMapping map[string]*ToolInfo

// AddTool adds a tool to the mapping
func (c ToolInfoMapping) AddTool(toolName string, toolInfo *ToolInfo) error {
	if prev := c[toolName]; prev != nil {
		return fmt.Errorf("duplicate tool: %s with %s", toolInfo.String(), prev.String())
	}
	c[toolName] = toolInfo
	return nil
}

// String returns a string representation of the tool info
func (c *ToolInfo) String() string {
	if c.MCPServer != "" {
		return fmt.Sprintf("%s/%s", c.MCPServer, c.Name)
	}
	return c.Name
}

// ExecuteBuiltinTool executes a builtin tool with the given call
func ExecuteBuiltinTool(ctx context.Context, call types.ToolCall) (types.ToolResult, error) {
	executor := tools.GetExecutor(call.Name)
	if executor == nil {
		return types.ToolResult{
			Error: fmt.Sprintf("unknown builtin tool: %s", call.Name),
		}, nil
	}

	// Execute the tool with compile-time type safety
	res, err := executor.Execute(call.RawArgs, tools.ExecuteOptions{
		DefaultWorkspaceRoot: "", // This would need to be passed in
	})
	if err != nil {
		return types.ToolResult{
			Error: fmt.Sprintf("execute %s: %v", call.Name, err),
		}, nil
	}

	return types.ToolResult{
		Content: res,
	}, nil
}

// executeTool executes a tool using the tool info mapping
func executeTool(ctx context.Context, toolName string, arguments string, defaultWorkingDir string, toolInfoMapping ToolInfoMapping) (string, bool) {
	toolInfo, ok := toolInfoMapping[toolName]
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", toolName), false
	}

	var res interface{}
	var err error
	if toolInfo.Builtin {
		executor := tools.GetExecutor(toolName)
		if executor == nil {
			return fmt.Sprintf("Unknown builtin tool: %s", toolName), true
		}

		// Execute the tool with compile-time type safety
		res, err = executor.Execute(arguments, tools.ExecuteOptions{
			DefaultWorkspaceRoot: defaultWorkingDir,
		})
		if err != nil {
			return fmt.Sprintf("execute %s: %v", toolName, err), true
		}
	} else if toolInfo.MCPClient != nil {
		res, err = toolInfo.MCPClient.CallTool(ctx, mcp.CallToolRequest{
			Request: mcp.Request{},
			Params: mcp.CallToolParams{
				Name:      toolName,
				Arguments: json.RawMessage(arguments),
			},
		})
		if err != nil {
			return fmt.Sprintf("execute mcp %s/%s: %v", toolInfo.MCPServer, toolName, err), true
		}
	} else if toolInfo.ToolDefinition != nil {
		if len(toolInfo.ToolDefinition.Command) > 0 {
			// Handle custom command-based tools
			var m map[string]any
			if err := jsondecode.UnmarshalSafe([]byte(arguments), &m); err != nil {
				return fmt.Sprintf("parse args %s: %v", toolName, err), true
			}
			// This would need the command execution logic from run/tools.go
			return fmt.Sprintf("Custom command tools not yet implemented: %s", toolName), true
		} else {
			// Handle function-based tools
			return fmt.Sprintf("Custom command unable to execute: %s", toolName), false
		}
	} else {
		return fmt.Sprintf("Unable to execute tool: %s", toolName), true
	}

	jsonRes, err := json.Marshal(res)
	if err != nil {
		return fmt.Sprintf("marshalling result %s: %v", toolName, err), true
	}
	return string(jsonRes), true
}

// parseToolCall parses a tool call from provider-specific format to our unified format
func parseToolCall(toolName, toolID, arguments string, defaultWorkingDir string) (types.ToolCall, error) {
	var args map[string]interface{}
	if arguments != "" {
		if err := jsondecode.UnmarshalSafe([]byte(arguments), &args); err != nil {
			return types.ToolCall{}, fmt.Errorf("parse tool arguments: %w", err)
		}
	}

	return types.ToolCall{
		ID:         toolID,
		Name:       toolName,
		Arguments:  args,
		RawArgs:    arguments,
		WorkingDir: defaultWorkingDir,
	}, nil
}

// executeToolWithCallback executes a tool using either custom callback, stream communication, or built-in execution
func (c *Client) executeToolWithCallback(ctx context.Context, stream types.StreamContext, call types.ToolCall, callback types.ToolCallback, stdout io.Writer, defaultWorkingDir string, toolInfoMapping ToolInfoMapping) (types.ToolResult, error) {
	// If custom callback is provided, use it first
	if callback != nil {
		result, handled, err := callback(ctx, stream, call)
		if err != nil {
			return result, err
		}
		// If callback handled the tool (regardless of result), use it
		if handled {
			return result, nil
		}
		// If callback didn't handle the tool (handled=false), fall through to built-in execution
	}

	// Fall back to built-in tool execution
	resultStr, ok := executeTool(ctx, call.Name, call.RawArgs, defaultWorkingDir, toolInfoMapping)
	if !ok {
		// If streams are provided, use bidirectional stream communication
		if c.stdinReader != nil {
			result, handled, err := executeToolWithStream(ctx, call, stdout, c.stdinReader, defaultWorkingDir)
			if err != nil {
				return result, err
			}
			if handled {
				return result, nil
			}
		}

		return types.ToolResult{
			Error: fmt.Sprintf("tool execution failed: %s", call.Name),
		}, nil
	}

	// Try to parse as JSON, otherwise return as string
	var content interface{}
	if err := json.Unmarshal([]byte(resultStr), &content); err != nil {
		// If not valid JSON, return as string content
		content = map[string]interface{}{
			"output": resultStr,
		}
	}

	return types.ToolResult{
		Content: content,
	}, nil
}
