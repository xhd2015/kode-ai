package types

import (
	"context"

	"github.com/xhd2015/llm-tools/jsonschema"
)

// UnifiedTool represents a unified tool definition
type UnifiedTool struct {
	Format      string                 `json:"format,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  *jsonschema.JsonSchema `json:"parameters,omitempty"`

	// command to be executed
	Command []string `json:"command"`

	Handle func(ctx context.Context, stream StreamContext, call ToolCall) (ToolResult, bool, error) `json:"-"`
}
