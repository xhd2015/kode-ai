package chat_test

import (
	"context"
	"fmt"
	"os"

	"github.com/xhd2015/kode-ai/chat"
)

// ExampleClient demonstrates basic usage of the chat library
func ExampleClient() {
	// Create a client
	client, err := chat.NewClient(chat.Config{
		Model: "claude-3-7-sonnet",
		Token: os.Getenv("ANTHROPIC_API_KEY"),
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	// Simple chat
	response, err := client.Chat(context.Background(), "What is Go programming language?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response.Messages[0].Content[:50]+"...")
	fmt.Printf("Token usage: %d\n", response.TokenUsage.Total)
}

// ExampleClient_withTools demonstrates chat with tools
func ExampleClient_withTools() {
	client, err := chat.NewClient(chat.Config{
		Model: "gpt-4o",
		Token: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	// Chat with tools and custom callback
	response, err := client.Chat(context.Background(), "List files in current directory",
		chat.WithTools("file_list"),
		chat.WithEventCallback(func(event chat.Event) {
			switch event.Type {
			case chat.EventTypeMessage:
				fmt.Print(event.Content)
			case chat.EventTypeToolCall:
				fmt.Printf("\n🔧 Calling tool: %s\n", event.Metadata["tool_name"])
			case chat.EventTypeToolResult:
				fmt.Printf("✅ Tool completed\n")
			}
		}),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Tool calls made: %d\n", len(response.ToolCalls))
}

// ExampleClient_withCustomToolCallback demonstrates custom tool handling
func ExampleClient_withCustomToolCallback() {
	client, err := chat.NewClient(chat.Config{
		Model: "claude-3-7-sonnet",
		Token: os.Getenv("ANTHROPIC_API_KEY"),
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	// Custom tool handler
	toolHandler := func(ctx context.Context, call chat.ToolCall) (chat.ToolResult, bool, error) {
		switch call.Name {
		case "custom_database_query":
			sql := call.Arguments["sql"].(string)
			// Simulate database query
			result := map[string]interface{}{
				"rows":    []string{"user1", "user2"},
				"count":   2,
				"query":   sql,
				"message": "Query executed successfully",
			}
			return chat.ToolResult{Content: result}, true, nil // handled=true
		default:
			// Don't handle this tool, fallback to built-in tools
			return chat.ToolResult{}, false, nil // handled=false, no error
		}
	}

	response, err := client.Chat(context.Background(), "Query the database for users",
		chat.WithToolCallback(toolHandler))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response received with %d messages\n", len(response.Messages))
}

// ExampleCLIHandler demonstrates CLI usage
func ExampleCLIHandler() {
	client, err := chat.NewClient(chat.Config{
		Model: "claude-3-7-sonnet",
		Token: os.Getenv("ANTHROPIC_API_KEY"),
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	// CLI wrapper for command-line usage
	cliHandler := chat.NewCliHandler(client, chat.CliOptions{
		RecordFile: "session.json",
		LogChat:    true,
		Verbose:    false,
	})

	err = cliHandler.HandleCLI(context.Background(), "Hello, how are you?",
		chat.WithTools("file_read"),
		chat.WithMaxRounds(2),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("CLI chat completed")
}

// ExampleClient_multiRound demonstrates multi-round conversation
func ExampleClient_multiRound() {
	client, err := chat.NewClient(chat.Config{
		Model: "gpt-4o",
		Token: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	var history []chat.Message

	// First message
	response1, err := client.Chat(context.Background(), "My name is Alice",
		chat.WithHistory(history))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	history = append(history, response1.Messages...)

	// Follow-up message
	response2, err := client.Chat(context.Background(), "What is my name?",
		chat.WithHistory(history))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Second response: %s\n", response2.Messages[0].Content[:50]+"...")
}
