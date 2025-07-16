package run

import "github.com/xhd2015/kode-ai/tools"

// Config represents the configuration structure that can be loaded from a file
type Config struct {
	Token              string               `json:"token,omitempty"`
	MaxRound           int                  `json:"max_round,omitempty"`
	BaseURL            string               `json:"base_url,omitempty"`
	Model              string               `json:"model,omitempty"`
	SystemPrompt       StringOrList         `json:"system,omitempty"` // can be string or a list of strings
	Tools              []string             `json:"tools,omitempty"`
	ToolCustomFiles    []string             `json:"tool_custom_files,omitempty"`
	ToolCustomJSONs    []*tools.UnifiedTool `json:"tool_custom_jsons,omitempty"`
	ToolDefaultCwd     string               `json:"tool_default_cwd,omitempty"`
	RecordFile         string               `json:"record_file,omitempty"`
	NoCache            bool                 `json:"no_cache,omitempty"`
	ShowUsage          bool                 `json:"show_usage,omitempty"`
	IgnoreDuplicateMsg bool                 `json:"ignore_duplicate_msg,omitempty"`
	LogRequest         bool                 `json:"log_request,omitempty"`
	LogChat            bool                 `json:"log_chat,omitempty"`
	Verbose            bool                 `json:"verbose,omitempty"`
	MCPServers         []string             `json:"mcp_servers,omitempty"`
}

type StringOrList = interface{}
