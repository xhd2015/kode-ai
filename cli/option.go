package cli

import (
	"io"

	"github.com/xhd2015/kode-ai/types"
)

// WithSystemPrompt sets the system prompt for the conversation
func WithSystemPrompt(prompt string) types.ChatOption {
	return types.WithSystemPrompt(prompt)
}

// WithMaxRounds sets the maximum number of conversation rounds
func WithMaxRounds(rounds int) types.ChatOption {
	return types.WithMaxRounds(rounds)
}

// WithTools specifies the builtin tools to make available
func WithTools(tools ...string) types.ChatOption {
	return types.WithTools(tools...)
}

// WithToolFiles specifies custom tool definition files to load
func WithToolFiles(files ...string) types.ChatOption {
	return types.WithToolFiles(files...)
}

// WithToolJSONs specifies custom tool definitions as JSON strings
func WithToolJSONs(jsons ...string) types.ChatOption {
	return types.WithToolJSONs(jsons...)
}

func WithToolDefinitions(tool ...*types.UnifiedTool) types.ChatOption {
	return types.WithToolDefinitions(tool...)
}

// WithDefaultToolCwd sets the default working directory for tool execution
func WithDefaultToolCwd(cwd string) types.ChatOption {
	return types.WithDefaultToolCwd(cwd)
}

// WithHistory provides historical messages for conversation context
func WithHistory(messages []types.Message) types.ChatOption {
	return types.WithHistory(messages)
}

// WithCache controls whether caching is enabled (default: true)
func WithCache(enabled bool) types.ChatOption {
	return types.WithCache(enabled)
}

// WithMCPServers specifies MCP servers to connect to
func WithMCPServers(servers ...string) types.ChatOption {
	return types.WithMCPServers(servers...)
}

// WithToolCallback sets a custom tool execution callback
func WithToolCallback(callback types.ToolCallback) types.ChatOption {
	return types.WithToolCallback(callback)
}

// WithEventCallback sets a callback for receiving events during chat processing
func WithEventCallback(callback types.EventCallback) types.ChatOption {
	return types.WithEventCallback(callback)
}

// WithFollowUpCallback sets a callback for follow-up tool execution
func WithFollowUpCallback(callback types.FollowUpCallback) types.ChatOption {
	return types.WithFollowUpCallback(callback)
}

// WithStdStream sets stdin and stdout for bidirectional tool callback communication
func WithStdStream(stdin io.Reader, stdout io.Writer) types.ChatOption {
	return types.WithStdStream(stdin, stdout)
}
