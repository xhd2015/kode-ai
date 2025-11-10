package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xhd2015/llm-tools/tools/batch_read_file"
	"github.com/xhd2015/llm-tools/tools/create_file_with_content"
	"github.com/xhd2015/llm-tools/tools/defs"
	"github.com/xhd2015/llm-tools/tools/delete_file"
	"github.com/xhd2015/llm-tools/tools/file_search"
	"github.com/xhd2015/llm-tools/tools/get_workspace_root"
	"github.com/xhd2015/llm-tools/tools/grep_search"
	"github.com/xhd2015/llm-tools/tools/list_dir"
	"github.com/xhd2015/llm-tools/tools/read_file"
	"github.com/xhd2015/llm-tools/tools/rename_file"
	"github.com/xhd2015/llm-tools/tools/run_bash_script"
	"github.com/xhd2015/llm-tools/tools/run_terminal_cmd"
	"github.com/xhd2015/llm-tools/tools/search_replace"
	"github.com/xhd2015/llm-tools/tools/send_answer"
	"github.com/xhd2015/llm-tools/tools/todo_write"
	"github.com/xhd2015/llm-tools/tools/tree"
	"github.com/xhd2015/llm-tools/tools/write_file"

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
	{
		Name:       "tree",
		Definition: tree.GetToolDefinition(),
		Executor:   TreeExecutor{},
	},
	{
		Name:       "grep_search",
		Definition: grep_search.GetToolDefinition(),
		Executor:   GrepSearchExecutor{},
	},
	{
		Name:       "create_file_with_content",
		Definition: create_file_with_content.GetToolDefinition(),
		Executor:   CreateFileWithContentExecutor{},
	},
	{
		Name:       "read_file",
		Definition: read_file.GetToolDefinition(),
		Executor:   ReadFileExecutor{},
	},
	{
		Name:       "write_file",
		Definition: write_file.GetToolDefinition(),
		Executor:   WriteFileExecutor{},
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
		Name:       "run_bash_script",
		Definition: run_bash_script.GetToolDefinition(),
		Executor:   RunBashScriptExecutor{},
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

func GetBuiltinTools(toolBuiltins []string) ([]*UnifiedTool, error) {
	return getBuiltinTools(toolBuiltins)
}

func GetAllBuiltinTools() ([]*UnifiedTool, error) {
	return getBuiltinTools(allTools)
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
	toolNames := make([]string, 0, len(tools))
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name)
	}
	return toolNames
}

func getBuiltinTools(builtinTools []string) ([]*UnifiedTool, error) {
	tools := make([]*UnifiedTool, 0, len(builtinTools))
	for _, builtinTool := range builtinTools {
		builtinToolInfo := toolMapping[builtinTool]
		if builtinToolInfo == nil {
			return nil, fmt.Errorf("unrecognized builtin tool: %s, available: %s", builtinTool, strings.Join(allTools, ", "))
		}
		toolDef := builtinToolInfo.Definition
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

type RunBashScriptExecutor struct {
}

func (e RunBashScriptExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := run_bash_script.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	req.Cwd = joinDir(opts.DefaultWorkspaceRoot, req.Cwd)
	return run_bash_script.RunBashScript(req)
}

func joinDir(workspaceRoot, dir string) string {
	if workspaceRoot == "" {
		return dir
	}
	if dir == "" {
		return workspaceRoot
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(workspaceRoot, dir)
}

type TreeExecutor struct {
}

func (e TreeExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := tree.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return tree.ExecuteTree(req)
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

type CreateFileWithContentExecutor struct {
}

func (e CreateFileWithContentExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := create_file_with_content.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return create_file_with_content.CreateFileWithContent(req)
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

type WriteFileExecutor struct {
}

func (e WriteFileExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := write_file.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	if req.WorkspaceRoot == "" && opts.DefaultWorkspaceRoot != "" {
		req.WorkspaceRoot = opts.DefaultWorkspaceRoot
	}
	return write_file.WriteFile(req)
}

type SearchReplaceExecutor struct {
}

func (e SearchReplaceExecutor) Execute(arguments string, opts ExecuteOptions) (interface{}, error) {
	req, err := search_replace.ParseJSONRequest(arguments)
	if err != nil {
		return nil, fmt.Errorf("parse args: %v", err)
	}
	return search_replace.SearchReplace(req, opts.DefaultWorkspaceRoot)
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
