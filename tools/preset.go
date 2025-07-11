package tools

import (
	"fmt"

	"github.com/xhd2015/llm-tools/tools/batch_read_file"
	"github.com/xhd2015/llm-tools/tools/get_workspace_root"
	"github.com/xhd2015/llm-tools/tools/grep_search"
	"github.com/xhd2015/llm-tools/tools/list_dir"
	"github.com/xhd2015/llm-tools/tools/read_file"
	"github.com/xhd2015/llm-tools/tools/run_terminal_cmd"
)

func GetPresetTools(toolPresets []string) ([]*UnifiedTool, error) {
	var tools []*UnifiedTool
	for _, preset := range toolPresets {
		switch preset {
		case "get_workspace_root":
			getWorkspaceRoot := get_workspace_root.GetToolDefinition()
			tools = append(tools, &UnifiedTool{
				Name:        getWorkspaceRoot.Name,
				Description: getWorkspaceRoot.Description,
				Parameters:  getWorkspaceRoot.Parameters,
			})
		case "batch_read_file":
			batchReadFile := batch_read_file.GetToolDefinition()
			tools = append(tools, &UnifiedTool{
				Name:        batchReadFile.Name,
				Description: batchReadFile.Description,
				Parameters:  batchReadFile.Parameters,
			})
		case "list_dir":
			listDir := list_dir.GetToolDefinition()
			tools = append(tools, &UnifiedTool{
				Name:        listDir.Name,
				Description: listDir.Description,
				Parameters:  listDir.Parameters,
			})
		case "run_terminal_cmd":
			runTerminalCmd := run_terminal_cmd.GetToolDefinition()
			tools = append(tools, &UnifiedTool{
				Name:        runTerminalCmd.Name,
				Description: runTerminalCmd.Description,
				Parameters:  runTerminalCmd.Parameters,
			})
		case "grep_search":
			grepSearch := grep_search.GetToolDefinition()
			tools = append(tools, &UnifiedTool{
				Name:        grepSearch.Name,
				Description: grepSearch.Description,
				Parameters:  grepSearch.Parameters,
			})
		case "read_file":
			readFile := read_file.GetToolDefinition()
			tools = append(tools, &UnifiedTool{
				Name:        readFile.Name,
				Description: readFile.Description,
				Parameters:  readFile.Parameters,
			})
		default:
			return nil, fmt.Errorf("unrecognized tool preset: %s, available: get_workspace_root, batch_read_file, list_dir, run_terminal_cmd, grep_search, read_file", preset)
		}
	}
	return tools, nil
}
