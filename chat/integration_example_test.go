package chat

import (
	"context"
	"fmt"

	"github.com/xhd2015/kode-ai/providers"
	"github.com/xhd2015/kode-ai/types"
)

// This file shows how the existing run/chat.go could be refactored to use the new library

// Example integration function that could replace the existing ChatHandler.Handle method
func IntegrateWithExistingCLI(model, baseUrl, token, msg string, opts ExistingChatOptions) error {
	// Convert existing options to new library format
	config := Config{
		Model:   model,
		Token:   token,
		BaseURL: baseUrl,
	}

	// Set log level based on existing options
	if opts.LogRequest {
		config.LogLevel = types.LogLevelRequest
	}

	// Create client
	client, err := NewClient(config)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// Create CLI handler with existing CLI-specific options
	cliHandler := NewCliHandler(client, CliOptions{
		RecordFile:         opts.RecordFile,
		IgnoreDuplicateMsg: opts.IgnoreDuplicateMsg,
		LogRequest:         opts.LogRequest,
		LogChat:            opts.LogChat,
		Verbose:            opts.Verbose,
	})

	// Convert existing options to new library options
	var coreOpts []types.ChatOption
	if opts.SystemPrompt != "" {
		coreOpts = append(coreOpts, WithSystemPrompt(opts.SystemPrompt))
	}
	if opts.MaxRound > 0 {
		coreOpts = append(coreOpts, WithMaxRounds(opts.MaxRound))
	}
	if len(opts.ToolBuiltins) > 0 {
		coreOpts = append(coreOpts, WithTools(opts.ToolBuiltins...))
	}
	if len(opts.ToolFiles) > 0 {
		coreOpts = append(coreOpts, WithToolFiles(opts.ToolFiles...))
	}
	if len(opts.ToolJSONs) > 0 {
		coreOpts = append(coreOpts, WithToolJSONs(opts.ToolJSONs...))
	}
	if opts.ToolDefaultCwd != "" {
		coreOpts = append(coreOpts, WithDefaultToolCwd(opts.ToolDefaultCwd))
	}
	if opts.NoCache {
		coreOpts = append(coreOpts, WithCache(false))
	}
	if len(opts.MCPServers) > 0 {
		coreOpts = append(coreOpts, WithMCPServers(opts.MCPServers...))
	}

	// Execute using CLI handler
	return cliHandler.HandleCli(context.Background(), msg, coreOpts...)
}

// ExistingChatOptions represents the existing ChatOptions structure
// This shows how the old options map to the new library
type ExistingChatOptions struct {
	MaxRound           int
	SystemPrompt       string
	ToolBuiltins       []string
	ToolFiles          []string
	ToolJSONs          []string
	RecordFile         string
	ToolDefaultCwd     string
	IgnoreDuplicateMsg bool
	NoCache            bool
	LogRequest         bool
	Verbose            bool
	LogChat            bool
	MCPServers         []string
}

// Example of how to migrate existing ChatHandler to use new library
type LegacyChatHandler struct {
	APIShape providers.APIShape
}

func (c *LegacyChatHandler) Handle(model string, baseUrl string, token string, msg string, opts ExistingChatOptions) error {
	// This is how the existing Handle method could be replaced
	return IntegrateWithExistingCLI(model, baseUrl, token, msg, opts)
}

// Migration guide:
//
// 1. Replace the existing ChatHandler.Handle method with IntegrateWithExistingCLI
// 2. Update imports to use the new chat package
// 3. Existing CLI behavior remains the same through CLIHandler
// 4. All existing functionality is preserved
//
// Benefits of migration:
// - Clean separation of library vs CLI concerns
// - Programmatic access to chat functionality
// - Custom tool callbacks for advanced integrations
// - Event system for real-time updates
// - Maintained backward compatibility for CLI usage
