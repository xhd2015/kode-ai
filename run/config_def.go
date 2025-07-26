package run

import (
	"github.com/xhd2015/kode-ai/tools"
	"github.com/xhd2015/kode-ai/types"
)

// Config represents the configuration structure that can be loaded from a file
type Config struct {
	types.Config                         // Embed the base config
	ToolCustomJSONs []*tools.UnifiedTool `json:"tool_custom_jsons,omitempty"` // Tool-specific field that needs UnifiedTool
}

// Legacy type alias for compatibility
type StringOrList = types.StringOrList
