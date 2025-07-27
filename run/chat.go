package run

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/xhd2015/kode-ai/chat"
	"github.com/xhd2015/kode-ai/providers"
	"github.com/xhd2015/kode-ai/types"
)

type ChatOptions struct {
	maxRound int

	systemPrompt string
	toolBuiltins []string
	toolFiles    []string
	toolJSONs    []string
	recordFile   string

	toolDefaultCwd string

	ignoreDuplicateMsg bool
	noCache            bool

	logRequest          bool
	verbose             bool
	logChat             bool
	jsonOutput          bool
	stdStream           bool
	waitForStreamEvents bool

	// MCP server configuration
	mcpServers []string
}

// Getter methods for ChatOptions to allow external access
func (c ChatOptions) SystemPrompt() string {
	return c.systemPrompt
}

func (c ChatOptions) MaxRound() int {
	return c.maxRound
}

func (c ChatOptions) ToolBuiltins() []string {
	return c.toolBuiltins
}

func (c ChatOptions) ToolFiles() []string {
	return c.toolFiles
}

func (c ChatOptions) ToolJSONs() []string {
	return c.toolJSONs
}

func (c ChatOptions) ToolDefaultCwd() string {
	return c.toolDefaultCwd
}

func (c ChatOptions) NoCache() bool {
	return c.noCache
}

func (c ChatOptions) MCPServers() []string {
	return c.mcpServers
}

type ChatHandler struct {
	APIShape providers.APIShape

	// openAIKeepMultipleSystemPrompts_DO_NOT_SET only for testing purpose
	// please do not set it.
	//
	// to test open ai behavior when multiple system prompts exists, which will win?
	// the result shows the last one wins:
	//  > system: Your name is Lubby, your age is 18
	//  > user: who are you?
	//  > assistant: Hi! I'm Lubby, an 18-year-old here to help you out. What can I do for you today?
	//  > system: Your name is Chuk, your age is 12
	//  > user: how old are you?
	//  > assistant: I'm 12 years old! How about you?
	openAIKeepMultipleSystemPrompts_DO_NOT_SET bool
}

func (c *ChatHandler) Handle(model string, baseUrl string, token string, msg string, opts ChatOptions) error {
	// Convert to new library format
	config := chat.Config{
		Model:   model,
		Token:   token,
		BaseURL: baseUrl,
	}

	// Set log level based on existing options
	if opts.logRequest {
		config.LogLevel = types.LogLevelRequest
	}

	// Convert existing options to new library options
	var coreOpts []types.ChatOption
	if opts.systemPrompt != "" {
		coreOpts = append(coreOpts, chat.WithSystemPrompt(opts.systemPrompt))
	}
	if opts.maxRound > 0 {
		coreOpts = append(coreOpts, chat.WithMaxRounds(opts.maxRound))
	}
	if len(opts.toolBuiltins) > 0 {
		coreOpts = append(coreOpts, chat.WithTools(opts.toolBuiltins...))
	}
	if len(opts.toolFiles) > 0 {
		coreOpts = append(coreOpts, chat.WithToolFiles(opts.toolFiles...))
	}
	if len(opts.toolJSONs) > 0 {
		coreOpts = append(coreOpts, chat.WithToolJSONs(opts.toolJSONs...))
	}
	if opts.toolDefaultCwd != "" {
		coreOpts = append(coreOpts, chat.WithDefaultToolCwd(opts.toolDefaultCwd))
	}
	if opts.noCache {
		coreOpts = append(coreOpts, chat.WithCache(false))
	}
	if len(opts.mcpServers) > 0 {
		coreOpts = append(coreOpts, chat.WithMCPServers(opts.mcpServers...))
	}

	// Add stdin/stdout streams for bidirectional tool callback communication
	if opts.stdStream {
		// If waiting for stream events, load historical events from stdin first
		if opts.waitForStreamEvents {
			messages, err := loadMessagesFromStdin(30 * time.Second) // Default 30 second timeout
			if err != nil {
				return fmt.Errorf("failed to load messages from stdin: %w", err)
			}

			// Convert messages to history format and apply to chat options
			history := convertMessagesToHistory(messages)
			if len(history) > 0 {
				coreOpts = append(coreOpts, chat.WithHistory(history))
			}
		}
		coreOpts = append(coreOpts, chat.WithStdStream(os.Stdin, os.Stdout))
	}

	// Create client
	client, err := chat.NewClient(config)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// Create CLI handler with existing CLI-specific options
	cliHandler := chat.NewCliHandler(client, chat.CliOptions{
		RecordFile:         opts.recordFile,
		IgnoreDuplicateMsg: opts.ignoreDuplicateMsg,
		LogRequest:         opts.logRequest,
		LogChat:            opts.logChat,
		Verbose:            opts.verbose,
		JSONOutput:         opts.jsonOutput || opts.stdStream,
	})

	// Execute using CLI handler
	return cliHandler.HandleCli(context.Background(), msg, coreOpts...)
}
