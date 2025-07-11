package run

import (
	"encoding/json"
	"fmt"

	"github.com/xhd2015/llm-tools/tools/batch_read_file"
	"github.com/xhd2015/llm-tools/tools/get_workspace_root"
	"github.com/xhd2015/llm-tools/tools/grep_search"
	"github.com/xhd2015/llm-tools/tools/list_dir"
	"github.com/xhd2015/llm-tools/tools/read_file"
	"github.com/xhd2015/llm-tools/tools/run_terminal_cmd"
)

// executePresetTool checks if the tool name matches a preset and executes it
func executePresetTool(toolName string, arguments string, defaultWorkingDir string, toolPresets []string) (string, bool) {
	// Check if the tool name is in the preset list
	isPreset := false
	for _, preset := range toolPresets {
		if preset == toolName {
			isPreset = true
			break
		}
	}

	if !isPreset {
		return "", false
	}

	var res interface{}
	var err error

	// Execute the tool based on its name
	switch toolName {
	case "get_workspace_root":
		res, err = get_workspace_root.GetWorkspaceRoot(get_workspace_root.GetWorkspaceRootRequest{}, defaultWorkingDir)
		if err != nil {
			return fmt.Sprintf("Error executing %s: %v", toolName, err), true
		}
	case "batch_read_file":
		req, err := batch_read_file.ParseJSONRequest(arguments)
		if err != nil {
			return fmt.Sprintf("Error parsing %s: %v", toolName, err), true
		}
		if req.WorkspaceRoot == "" && defaultWorkingDir != "" {
			req.WorkspaceRoot = defaultWorkingDir
		}
		res, err = batch_read_file.BatchReadFile(req)
		if err != nil {
			return fmt.Sprintf("Error executing %s: %v", toolName, err), true
		}
	case "list_dir":
		req, err := list_dir.ParseJSONRequest(arguments)
		if err != nil {
			return fmt.Sprintf("Error parsing %s: %v", toolName, err), true
		}
		if req.WorkspaceRoot == "" && defaultWorkingDir != "" {
			req.WorkspaceRoot = defaultWorkingDir
		}
		res, err = list_dir.ListDir(req)
		if err != nil {
			return fmt.Sprintf("Error executing %s: %v", toolName, err), true
		}
	case "run_terminal_cmd":
		req, err := run_terminal_cmd.ParseJSONRequest(arguments)
		if err != nil {
			return fmt.Sprintf("Error executing %s: %v", toolName, err), true
		}
		res, err = run_terminal_cmd.RunTerminalCmd(req)
		if err != nil {
			return fmt.Sprintf("Error executing %s: %v", toolName, err), true
		}
	case "grep_search":
		req, err := grep_search.ParseJSONRequest(arguments)
		if err != nil {
			return fmt.Sprintf("Error executing %s: %v", toolName, err), true
		}
		res, err = grep_search.GrepSearch(req)
		if err != nil {
			return fmt.Sprintf("Error executing %s: %v", toolName, err), true
		}
	case "read_file":
		req, err := read_file.ParseJSONRequest(arguments)
		if err != nil {
			return fmt.Sprintf("Error executing %s: %v", toolName, err), true
		}
		if req.WorkspaceRoot == "" && defaultWorkingDir != "" {
			req.WorkspaceRoot = defaultWorkingDir
		}
		res, err = read_file.ReadFile(req)
		if err != nil {
			return fmt.Sprintf("Error executing %s: %v", toolName, err), true
		}
	default:
		return fmt.Sprintf("Unknown preset tool: %s", toolName), true
	}
	jsonRes, err := json.Marshal(res)
	if err != nil {
		return fmt.Sprintf("marshalling result %s: %v", toolName, err), true
	}
	return string(jsonRes), true
}
