package run

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	var_template "github.com/xhd2015/go-var-template"
	"github.com/xhd2015/kode-ai/internal/jsondecode"
	"github.com/xhd2015/kode-ai/tools"
	"github.com/xhd2015/llm-tools/jsonschema"
)

type ToolInfo struct {
	Name   string
	Preset bool

	ToolDefinition *tools.UnifiedTool

	MCPServer string
	MCPClient *client.Client
}

type ToolInfoMapping map[string]*ToolInfo

func listTools() error {
	toolPresets, err := tools.GetAllPresetTools()
	if err != nil {
		return err
	}
	for _, tool := range toolPresets {
		fmt.Println(tool.Name)
	}
	return nil
}

// executeTool checks if the tool name matches a preset and executes it
func executeTool(ctx context.Context, toolName string, arguments string, defaultWorkingDir string, toolInfoMapping map[string]*ToolInfo) (string, bool) {
	toolInfo, ok := toolInfoMapping[toolName]
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", toolName), true
	}

	var res interface{}
	var err error
	if toolInfo.Preset {
		executor := tools.GetExecutor(toolName)
		if executor == nil {
			return fmt.Sprintf("Unknown preset tool: %s", toolName), true
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
	} else if toolInfo.ToolDefinition != nil && len(toolInfo.ToolDefinition.Command) > 0 {
		var m map[string]any
		if err := jsondecode.UnmarshalSafe([]byte(arguments), &m); err != nil {
			return fmt.Sprintf("parse args %s: %v", toolName, err), true
		}
		command, err := interplotList(toolInfo.ToolDefinition.Command, m)
		if err != nil {
			return fmt.Sprintf("interplot %s: %v", toolName, err), true
		}
		cmdOutput, err := exec.CommandContext(ctx, command[0], command[1:]...).Output()
		if err != nil {
			return fmt.Sprintf("execute %s: %v", toolName, err), true
		}
		trimOutput := strings.TrimSpace(string(cmdOutput))
		if strings.HasPrefix(trimOutput, "{") && strings.HasSuffix(trimOutput, "}") {
			return trimOutput, true
		}
		res = map[string]interface{}{
			"output": string(cmdOutput),
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

func (c *ToolInfo) String() string {
	if c.MCPServer != "" {
		return fmt.Sprintf("%s/%s", c.MCPServer, c.Name)
	}
	return c.Name
}

func (c ToolInfoMapping) AddTool(toolName string, toolInfo *ToolInfo) error {
	if prev := c[toolName]; prev != nil {
		return fmt.Errorf("duplicate tool: %s with %s", toolInfo.String(), prev.String())
	}
	c[toolName] = toolInfo
	return nil
}

func interplotList(list []string, args map[string]any) ([]string, error) {
	argsStr := make(map[string]string, len(args))
	for k, v := range args {
		str, err := getStr(v)
		if err != nil {
			return nil, fmt.Errorf("get str %s: %v", k, err)
		}
		argsStr[k] = str
	}

	res := make([]string, len(list))
	for i, v := range list {
		str, err := interplot(v, argsStr)
		if err != nil {
			return nil, fmt.Errorf("interplot %s: %v", v, err)
		}
		res[i] = str
	}
	return res, nil
}

func interplot(tpl string, args map[string]string) (string, error) {
	ctpl := var_template.Compile(tpl)
	return ctpl.Execute(args)
}

func getStr(v interface{}) (string, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	}
	jsonRes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonRes), nil
}

// MCP client functionality
func connectToMCPServer(mcpServerSpec string) (*client.Client, error) {
	if mcpServerSpec == "" {
		return nil, nil
	}

	// Parse MCP server specification
	// Format: ip:port for network connection, or command for CLI
	if strings.Contains(mcpServerSpec, ":") {
		// Network connection (ip:port) - not supported by mark3labs/mcp-go directly
		return nil, fmt.Errorf("network MCP connections not yet supported by this client library")
	} else {
		// CLI connection - use client package
		mcpClient, err := client.NewStdioMCPClient(mcpServerSpec, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create MCP client: %w", err)
		}
		return mcpClient, nil
	}
}

func getMCPTools(ctx context.Context, mcpClient *client.Client) ([]*tools.UnifiedTool, error) {
	// Get tools from MCP server
	toolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}

	var unifiedTools []*tools.UnifiedTool
	for _, tool := range toolsResponse.Tools {
		// Convert MCP tool to unified tool format
		// Since UnifiedTool doesn't have a handler field, we'll create a basic schema representation
		description := tool.Description

		// Convert the MCP tool's input schema to our jsonschema format
		var parameters *jsonschema.JsonSchema
		if tool.InputSchema.Type != "" {
			parameters = &jsonschema.JsonSchema{
				Type:        jsonschema.ParamType(tool.InputSchema.Type),
				Properties:  convertMCPProperties(tool.InputSchema.Properties),
				Required:    tool.InputSchema.Required,
				Description: description,
			}
		}

		unifiedTool := &tools.UnifiedTool{
			Name:        tool.Name,
			Description: description,
			Parameters:  parameters,
		}
		unifiedTools = append(unifiedTools, unifiedTool)
	}

	return unifiedTools, nil
}

// Helper function to convert MCP properties to our jsonschema format
func convertMCPProperties(mcpProps map[string]interface{}) map[string]*jsonschema.JsonSchema {
	if mcpProps == nil {
		return nil
	}

	props := make(map[string]*jsonschema.JsonSchema)
	for name, prop := range mcpProps {
		if propMap, ok := prop.(map[string]interface{}); ok {
			schema := &jsonschema.JsonSchema{}
			if typeVal, exists := propMap["type"]; exists {
				if typeStr, ok := typeVal.(string); ok {
					schema.Type = jsonschema.ParamType(typeStr)
				}
			}
			if desc, exists := propMap["description"]; exists {
				if descStr, ok := desc.(string); ok {
					schema.Description = descStr
				}
			}
			props[name] = schema
		}
	}
	return props
}
