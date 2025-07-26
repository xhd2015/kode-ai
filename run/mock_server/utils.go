package mock_server

import (
	"encoding/json"
	"math/rand"

	"github.com/xhd2015/kode-ai/tools"
)

var builtinToolNames = []string{
	"get_workspace_root",
	"batch_read_file",
	"list_dir",
	"tree",
	"grep_search",
	"create_file_with_content",
	"read_file",
	"rename_file",
	"delete_file",
	"search_replace",
	"send_answer",
	"run_terminal_cmd",
	"file_search",
	"todo_write",
	"web_search",
}

// GetRandomTool returns a random tool name from builtin tools
func GetRandomTool() string {
	return builtinToolNames[rand.Intn(len(builtinToolNames))]
}

// GetRandomToolArgs returns random tool arguments as JSON string based on the tool schema
func GetRandomToolArgs() string {
	toolName := GetRandomTool()
	return GetRandomToolArgsForTool(toolName)
}

// GetRandomToolArgsForTool returns random arguments for a specific tool
func GetRandomToolArgsForTool(toolName string) string {
	switch toolName {
	case "get_workspace_root":
		return `{}`
	case "batch_read_file":
		return `{"files": [{"target_file": "example.txt", "should_read_entire_file": false, "start_line_one_indexed": 1, "end_line_one_indexed_inclusive": 10}]}`
	case "list_dir":
		return `{"relative_workspace_path": "."}`
	case "tree":
		return `{"path": ".", "max_depth": 3}`
	case "grep_search":
		return `{"query": "TODO", "include_pattern": "*.go"}`
	case "create_file_with_content":
		return `{"target_file": "new_file.txt", "content": "Hello World"}`
	case "read_file":
		return `{"target_file": "example.txt", "should_read_entire_file": true}`
	case "rename_file":
		return `{"old_path": "old_file.txt", "new_path": "new_file.txt"}`
	case "delete_file":
		return `{"target_file": "file_to_delete.txt"}`
	case "search_replace":
		return `{"file_path": "example.txt", "old_string": "old text", "new_string": "new text"}`
	case "send_answer":
		return `{"answer": "This is the answer"}`
	case "run_terminal_cmd":
		return `{"command": "ls -la", "is_background": false}`
	case "file_search":
		return `{"query": "example"}`
	case "todo_write":
		return `{"todos": [{"id": "1", "content": "Complete task", "status": "pending"}], "merge": false}`
	case "web_search":
		return `{"search_term": "latest news"}`
	default:
		return `{"query": "mock query"}`
	}
}

// GetRandomResponse returns a random mock response
func GetRandomResponse() string {
	responses := []string{
		"Hello! I'm a mock AI assistant. This is a simulated response for testing purposes.",
		"I understand your request. This is a random response from the mock server.",
		"Thank you for your question! This mock server is working correctly.",
		"This is a test response. The mock server is functioning as expected.",
		"I'm here to help with testing! This is a simulated AI response.",
		"Mock response generated successfully. Integration testing is working!",
		"Hello from the mock server! This response was randomly selected.",
		"Testing, testing... This is a mock AI assistant responding to your query.",
	}
	return responses[rand.Intn(len(responses))]
}

// GetRandomToolFromUserTools returns a random tool from user-provided tools
func GetRandomToolFromUserTools(userTools []*tools.UnifiedTool) string {
	if len(userTools) == 0 {
		return GetRandomTool()
	}
	return userTools[rand.Intn(len(userTools))].Name
}

// GetRandomToolArgsFromUserTools generates random arguments for user-provided tools
func GetRandomToolArgsFromUserTools(userTools []*tools.UnifiedTool) string {
	if len(userTools) == 0 {
		return GetRandomToolArgs()
	}

	tool := userTools[rand.Intn(len(userTools))]
	if tool.Parameters == nil {
		return `{}`
	}

	// Generate simple mock arguments based on the schema
	args := make(map[string]interface{})
	if tool.Parameters.Properties != nil {
		for propName, propSchema := range tool.Parameters.Properties {
			switch propSchema.Type {
			case "string":
				args[propName] = "mock_string_value"
			case "number", "integer":
				args[propName] = 42
			case "boolean":
				args[propName] = true
			case "array":
				args[propName] = []string{"mock_item"}
			case "object":
				args[propName] = map[string]interface{}{"mock_key": "mock_value"}
			default:
				args[propName] = "mock_value"
			}
		}
	}

	jsonData, err := json.Marshal(args)
	if err != nil {
		return `{}`
	}
	return string(jsonData)
}
