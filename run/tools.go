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
	"github.com/xhd2015/kode-ai/internal/jsondecode"
	"github.com/xhd2015/kode-ai/tools"
	"github.com/xhd2015/llm-tools/tools/batch_read_file"
	"github.com/xhd2015/llm-tools/tools/get_workspace_root"
	"github.com/xhd2015/llm-tools/tools/grep_search"
	"github.com/xhd2015/llm-tools/tools/list_dir"
	"github.com/xhd2015/llm-tools/tools/read_file"
	"github.com/xhd2015/llm-tools/tools/run_terminal_cmd"
	"github.com/xhd2015/llm-tools/tools/search_replace"
	"github.com/xhd2015/llm-tools/tools/send_answer"
)

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

type ToolInfo struct {
	Name   string
	Preset bool

	ToolDefinition *tools.UnifiedTool

	MCPServer string
	MCPClient *client.Client
}

type ToolInfoMapping map[string]*ToolInfo

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

// executeTool checks if the tool name matches a preset and executes it
func executeTool(ctx context.Context, toolName string, arguments string, defaultWorkingDir string, toolPresets []string, toolInfoMapping map[string]*ToolInfo) (string, bool) {
	toolInfo, ok := toolInfoMapping[toolName]
	if !ok {
		return fmt.Sprintf("Unknown tool: %s", toolName), true
	}

	var res interface{}
	var err error
	if toolInfo.Preset {
		// Execute the tool based on its name
		switch toolName {
		case "get_workspace_root":
			res, err = get_workspace_root.GetWorkspaceRoot(get_workspace_root.GetWorkspaceRootRequest{}, defaultWorkingDir)
			if err != nil {
				return fmt.Sprintf("execute %s: %v", toolName, err), true
			}
		case "batch_read_file":
			req, err := batch_read_file.ParseJSONRequest(arguments)
			if err != nil {
				return fmt.Sprintf("parse args %s: %v", toolName, err), true
			}
			if req.WorkspaceRoot == "" && defaultWorkingDir != "" {
				req.WorkspaceRoot = defaultWorkingDir
			}
			res, err = batch_read_file.BatchReadFile(req)
			if err != nil {
				return fmt.Sprintf("execute %s: %v", toolName, err), true
			}
		case "list_dir":
			req, err := list_dir.ParseJSONRequest(arguments)
			if err != nil {
				return fmt.Sprintf("parse args %s: %v", toolName, err), true
			}
			if req.WorkspaceRoot == "" && defaultWorkingDir != "" {
				req.WorkspaceRoot = defaultWorkingDir
			}
			res, err = list_dir.ListDir(req)
			if err != nil {
				return fmt.Sprintf("execute %s: %v", toolName, err), true
			}
		case "run_terminal_cmd":
			req, err := run_terminal_cmd.ParseJSONRequest(arguments)
			if err != nil {
				return fmt.Sprintf("parse args %s: %v", toolName, err), true
			}
			res, err = run_terminal_cmd.RunTerminalCmd(req)
			if err != nil {
				return fmt.Sprintf("execute %s: %v", toolName, err), true
			}
		case "grep_search":
			req, err := grep_search.ParseJSONRequest(arguments)
			if err != nil {
				return fmt.Sprintf("parse args %s: %v", toolName, err), true
			}
			res, err = grep_search.GrepSearch(req)
			if err != nil {
				return fmt.Sprintf("execute %s: %v", toolName, err), true
			}
		case "read_file":
			req, err := read_file.ParseJSONRequest(arguments)
			if err != nil {
				return fmt.Sprintf("parse args %s: %v", toolName, err), true
			}
			if req.WorkspaceRoot == "" && defaultWorkingDir != "" {
				req.WorkspaceRoot = defaultWorkingDir
			}
			res, err = read_file.ReadFile(req)
			if err != nil {
				return fmt.Sprintf("Error executing %s: %v", toolName, err), true
			}
		case "search_replace":
			req, err := search_replace.ParseJSONRequest(arguments)
			if err != nil {
				return fmt.Sprintf("parse args %s: %v", toolName, err), true
			}
			res, err = search_replace.SearchReplace(req)
			if err != nil {
				return fmt.Sprintf("execute %s: %v", toolName, err), true
			}
		case "send_answer":
			req, err := send_answer.ParseJSONRequest(arguments)
			if err != nil {
				return fmt.Sprintf("parse args %s: %v", toolName, err), true
			}
			res, err = send_answer.SendAnswer(req)
			if err != nil {
				return fmt.Sprintf("executing %s: %v", toolName, err), true
			}
		default:
			return fmt.Sprintf("Unknown preset tool: %s", toolName), true
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
	for k, v := range args {
		tpl = strings.ReplaceAll(tpl, "$"+k, v)
		tpl = strings.ReplaceAll(tpl, "${"+k+"}", v)
	}
	return tpl, nil
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
