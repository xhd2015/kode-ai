package types

import (
	"io"
)

// ChatOption represents a functional option for chat configuration
type ChatOption func(*Request)

// WithSystemPrompt sets the system prompt for the conversation
func WithSystemPrompt(prompt string) ChatOption {
	return func(req *Request) {
		req.SystemPrompt = prompt
	}
}

// WithMaxRounds sets the maximum number of conversation rounds
func WithMaxRounds(rounds int) ChatOption {
	return func(req *Request) {
		req.MaxRounds = rounds
	}
}

// WithTools specifies the builtin tools to make available
func WithTools(tools ...string) ChatOption {
	return func(req *Request) {
		req.Tools = append(req.Tools, tools...)
	}
}

// WithToolFiles specifies custom tool definition files to load
func WithToolFiles(files ...string) ChatOption {
	return func(req *Request) {
		req.ToolFiles = append(req.ToolFiles, files...)
	}
}

// WithToolJSONs specifies custom tool definitions as JSON strings
func WithToolJSONs(jsons ...string) ChatOption {
	return func(req *Request) {
		req.ToolJSONs = append(req.ToolJSONs, jsons...)
	}
}

func WithToolDefinitions(tool ...*UnifiedTool) ChatOption {
	return func(req *Request) {
		req.ToolDefinitions = append(req.ToolDefinitions, tool...)
	}
}

// WithDefaultToolCwd sets the default working directory for tool execution
func WithDefaultToolCwd(cwd string) ChatOption {
	return func(req *Request) {
		req.DefaultToolCwd = cwd
	}
}

// WithHistory provides historical messages for conversation context
func WithHistory(messages []Message) ChatOption {
	return func(req *Request) {
		req.History = append(req.History, messages...)
	}
}

// WithCache controls whether caching is enabled (default: true)
func WithCache(enabled bool) ChatOption {
	return func(req *Request) {
		req.NoCache = !enabled
	}
}

// WithMCPServers specifies MCP servers to connect to
func WithMCPServers(servers ...string) ChatOption {
	return func(req *Request) {
		req.MCPServers = append(req.MCPServers, servers...)
	}
}

// WithEventCallback sets a callback for receiving events during chat processing
func WithEventCallback(callback EventCallback) ChatOption {
	return func(req *Request) {
		req.EventCallback = callback
	}
}

// WithToolCallback sets a custom tool execution callback
func WithToolCallback(callback ToolCallback) ChatOption {
	return func(req *Request) {
		req.ToolCallback = callback
	}
}

// WithFollowUpCallback sets a callback for follow-up tool execution
func WithFollowUpCallback(callback FollowUpCallback) ChatOption {
	return func(req *Request) {
		req.FollowUpCallback = callback
	}
}

// WithStdStream sets stdin and stdout for bidirectional tool callback communication
func WithStdStream(stdin io.Reader, stdout io.Writer) ChatOption {
	return func(req *Request) {
		req.StreamPair = &StreamPair{
			Input:  stdin,
			Output: stdout,
		}
	}
}
