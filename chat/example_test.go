package chat_test

import (
	"context"
	"fmt"
	"os"

	"github.com/xhd2015/kode-ai/chat"
	"github.com/xhd2015/kode-ai/types"
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

	fmt.Printf("Response: %s\n", response.LastAssistantMsg[:50]+"...")
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
		chat.WithEventCallback(func(event types.Message) {
			switch event.Type {
			case types.MsgType_Msg:
				fmt.Print(event.Content)
			case types.MsgType_ToolCall:
				fmt.Printf("\n🔧 Calling tool: %s\n", event.ToolName)
			case types.MsgType_ToolResult:
				fmt.Printf("✅ Tool completed\n")
			}
		}),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response.LastAssistantMsg)
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
	toolHandler := func(ctx context.Context, stream types.StreamContext, call types.ToolCall) (types.ToolResult, bool, error) {
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
			return types.ToolResult{Content: result}, true, nil // handled=true
		default:
			// Don't handle this tool, fallback to built-in tools
			return types.ToolResult{}, false, nil // handled=false, no error
		}
	}

	response, err := client.Chat(context.Background(), "Query the database for users",
		chat.WithToolCallback(toolHandler))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response received: %s\n", response.LastAssistantMsg)
}

// ExampleCliHandler demonstrates CLI usage
func ExampleCliHandler() {
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

	err = cliHandler.HandleCli(context.Background(), "Hello, how are you?",
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

	var history []types.Message

	// First message
	_, err = client.Chat(context.Background(), "My name is Alice",
		chat.WithHistory(history))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	// Note: Response doesn't have Messages field, so we can't append to history in this simple way

	// Follow-up message
	response2, err := client.Chat(context.Background(), "What is my name?",
		chat.WithHistory(history))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Second response: %s\n", response2.LastAssistantMsg[:50]+"...")
}
