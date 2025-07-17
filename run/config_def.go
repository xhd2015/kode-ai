package run

import "github.com/xhd2015/kode-ai/tools"

// Config represents the configuration structure that can be loaded from a file
type Config struct {
	Token           string               `json:"token,omitempty"`
	MaxRound        int                  `json:"max_round,omitempty"`
	BaseURL         string               `json:"base_url,omitempty"`
	Model           string               `json:"model,omitempty"`
	SystemPrompt    StringOrList         `json:"system,omitempty"` // can be string or a list of strings
	Tools           []string             `json:"tools,omitempty"`
	ToolCustomFiles []string             `json:"tool_custom_files,omitempty"`
	ToolCustomJSONs []*tools.UnifiedTool `json:"tool_custom_jsons,omitempty"`
	ToolDefaultCwd  string               `json:"tool_default_cwd,omitempty"`
	MCPServers      []string             `json:"mcp_servers,omitempty"`
	Examples        []string             `json:"examples,omitempty"` // a list of example questions this agent can assist with
}

type StringOrList = interface{}
