package tools

import (
	"fmt"

	"github.com/xhd2015/llm-tools/tools/batch_read_file"
	"github.com/xhd2015/llm-tools/tools/defs"
	"github.com/xhd2015/llm-tools/tools/delete_file"
	"github.com/xhd2015/llm-tools/tools/file_search"
	"github.com/xhd2015/llm-tools/tools/get_workspace_root"
	"github.com/xhd2015/llm-tools/tools/grep_search"
	"github.com/xhd2015/llm-tools/tools/list_dir"
	"github.com/xhd2015/llm-tools/tools/read_file"
	"github.com/xhd2015/llm-tools/tools/rename_file"
	"github.com/xhd2015/llm-tools/tools/run_terminal_cmd"
	"github.com/xhd2015/llm-tools/tools/search_replace"
	"github.com/xhd2015/llm-tools/tools/send_answer"
	"github.com/xhd2015/llm-tools/tools/todo_write"

	// "github.com/xhd2015/llm-tools/tools/tree"
	"github.com/xhd2015/llm-tools/tools/web_search"
)

type ExecutorInfo struct {
	Name       string
	Definition defs.ToolDefinition
	Executor   Executor
}

// TODO: add tree

var tools = []*ExecutorInfo{
	{
		Name:       "get_workspace_root",
		Definition: get_workspace_root.GetToolDefinition(),
		Executor:   GetWorkspaceRootExecutor{},
	},
	{
		Name:       "batch_read_file",
		Definition: batch_read_file.GetToolDefinition(),
		Executor:   BatchReadFileExecutor{},
	},
	{
		Name:       "list_dir",
		Definition: list_dir.GetToolDefinition(),
		Executor:   ListDirExecutor{},
	},
	// {
	// 	Name:       "tree",
	// 	Definition: tree.GetToolDefinition(),
	// 	Executor:   TreeExecutor{},
	// },
	{
		Name:       "grep_search",
		Definition: grep_search.GetToolDefinition(),
		Executor:   GrepSearchExecutor{},
	},
	{
		Name:       "read_file",
		Definition: read_file.GetToolDefinition(),
		Executor:   ReadFileExecutor{},
	},
	{
		Name:       "rename_file",
		Definition: rename_file.GetToolDefinition(),
		Executor:   RenameFileExecutor{},
	},
	{
		Name:       "delete_file",
		Definition: delete_file.GetToolDefinition(),
		Executor:   DeleteFileExecutor{},
	},
	{
		Name:       "search_replace",
		Definition: search_replace.GetToolDefinition(),
		Executor:   SearchReplaceExecutor{},
	},
	{
		Name:       "send_answer",
		Definition: send_answer.GetToolDefinition(),
		Executor:   SendAnswerExecutor{},
	},
	{
		Name:       "run_terminal_cmd",
		Definition: run_terminal_cmd.GetToolDefinition(),
		Executor:   RunTerminalCmdExecutor{},
	},
	{
		Name:       "file_search",
		Definition: file_search.GetToolDefinition(),
		Executor:   FileSearchExecutor{},
	},
	{
		Name:       "todo_write",
		Definition: todo_write.GetToolDefinition(),
		Executor:   TodoWriteExecutor{},
	},
	{
		Name:       "web_search",
		Definition: web_search.GetToolDefinition(),
		Executor:   WebSearchExecutor{},
	},
}

func GetExecutor(toolName string) Executor {
	toolInfo := toolMapping[toolName]
	if toolInfo == nil {
		return nil
	}
	return toolInfo.Executor
}

type ExecuteOptions struct {
	DefaultWorkspaceRoot string
}

type Executor interface {
	Execute(arguments string, opts ExecuteOptions) (interface{}, error)
}

func GetPresetTools(toolPresets []string) ([]*UnifiedTool, error) {
	return getPresetTools(toolPresets)
}

func GetAllPresetTools() ([]*UnifiedTool, error) {
	return getPresetTools(allTools)
}

var toolMapping = buildToolMapping(tools)
var allTools = buildToolNames(tools)

func buildToolMapping(tools []*ExecutorInfo) map[string]*ExecutorInfo {
	toolMapping := make(map[string]*ExecutorInfo)
	for _, tool := range tools {
		toolMapping[tool.Name] = tool
	}
	return toolMapping
}
func buildToolNames(tools []*ExecutorInfo) []string {
	toolNames := make([]string, len(tools))
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name)
	}
	return toolNames
}

func getPresetTools(toolPresets []string) ([]*UnifiedTool, error) {
	tools := make([]*UnifiedTool, 0, len(toolPresets))
	for _, preset := range toolPresets {
		presetTool := toolMapping[preset]
		if presetTool == nil {
			return nil, fmt.Errorf("unrecognized tool preset: %s, available: get_workspace_root, batch_read_file, list_dir, run_terminal_cmd, grep_search, read_file", preset)
		}
		toolDef := presetTool.Definition
		tools = append(tools, &UnifiedTool{
			Name:        toolDef.Name,
			Description: toolDef.Description,
			Parameters:  toolDef.Parameters,
		})
	}
	return tools, nil
}

type GetWorkspaceRootExecutor struct {
}

func (e GetWorkspaceRootExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	return get_workspace_root.GetWorkspaceRoot(get_workspace_root.GetWorkspaceRootRequest{}, opts.DefaultWorkspaceRoot)
}

type BatchReadFileExecutor struct {
}

func (e BatchReadFileExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := batch_read_file.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return batch_read_file.BatchReadFile(req)
}

type ListDirExecutor struct {
}

func (e ListDirExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := list_dir.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return list_dir.ListDir(req)
}

type RunTerminalCmdExecutor struct {
}

func (e RunTerminalCmdExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := run_terminal_cmd.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return run_terminal_cmd.RunTerminalCmd(req)
}

type GrepSearchExecutor struct {
}

func (e GrepSearchExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := grep_search.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return grep_search.GrepSearch(req)
}

type ReadFileExecutor struct {
}

func (e ReadFileExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := read_file.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return read_file.ReadFile(req)
}

type SearchReplaceExecutor struct {
}

func (e SearchReplaceExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := search_replace.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return search_replace.SearchReplace(req)
}

type SendAnswerExecutor struct {
}

func (e SendAnswerExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := send_answer.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	return send_answer.SendAnswer(req)
}

type FileSearchExecutor struct {
}

func (e FileSearchExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := file_search.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return file_search.FileSearch(req)
}

type TodoWriteExecutor struct {
}

func (e TodoWriteExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := todo_write.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return todo_write.TodoWrite(req)
}

type WebSearchExecutor struct {
}

func (e WebSearchExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := web_search.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	return web_search.WebSearch(req)
}

type RenameFileExecutor struct {
}

func (e RenameFileExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := rename_file.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return rename_file.RenameFile(req)
}

type DeleteFileExecutor struct {
}

func (e DeleteFileExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := delete_file.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return delete_file.DeleteFile(req)
}
