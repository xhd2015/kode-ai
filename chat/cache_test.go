package chat

import (
	"context"
	"testing"

	"github.com/xhd2015/kode-ai/types"
)

func TestCacheInfoEvent(t *testing.T) {
	// Start mock server
	baseURL, cleanup := startMockServer(t, "openai")
	defer cleanup()

	// Create client with mock server URL
	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test cache enabled
	t.Run("cache enabled", func(t *testing.T) {
		var events []types.Message
		eventCallback := func(event types.Message) {
			events = append(events, event)
		}

		_, err := client.Chat(context.Background(), "Hello",
			WithCache(true), // Enable cache (this sets NoCache=false)
			WithEventCallback(eventCallback),
		)
		if err != nil {
			t.Fatalf("chat failed: %v", err)
		}

		// Check for cache info event
		var cacheEvent *types.Message
		for _, event := range events {
			if event.Type == types.MsgType_CacheInfo {
				cacheEvent = &event
				break
			}
		}

		if cacheEvent == nil {
			t.Fatal("expected cache info event but got none")
		}

		if cacheEvent.Content != "Prompt cache enabled with gpt-4o" {
			t.Errorf("expected 'Prompt cache enabled with gpt-4o', got '%s'", cacheEvent.Content)
		}

		if cacheEvent.Metadata.CacheInfo == nil || !cacheEvent.Metadata.CacheInfo.CacheEnabled {
			t.Errorf("expected cache_enabled=true in metadata, got %v", cacheEvent.Metadata.CacheInfo)
		}

		if cacheEvent.Model != "gpt-4o" {
			t.Errorf("expected model='gpt-4o', got %v", cacheEvent.Model)
		}
	})

	// Test cache disabled
	t.Run("cache disabled", func(t *testing.T) {
		var events []types.Message
		eventCallback := func(event types.Message) {
			events = append(events, event)
		}

		_, err := client.Chat(context.Background(), "Hello",
			WithCache(false), // Disable cache (this sets NoCache=true)
			WithEventCallback(eventCallback),
		)
		if err != nil {
			t.Fatalf("chat failed: %v", err)
		}

		// Check for cache info event
		var cacheEvent *types.Message
		for _, event := range events {
			if event.Type == types.MsgType_CacheInfo {
				cacheEvent = &event
				break
			}
		}

		if cacheEvent == nil {
			t.Fatal("expected cache info event but got none")
		}

		if cacheEvent.Content != "Prompt cache disabled with gpt-4o" {
			t.Errorf("expected 'Prompt cache disabled with gpt-4o', got '%s'", cacheEvent.Content)
		}

		if cacheEvent.Metadata.CacheInfo == nil || cacheEvent.Metadata.CacheInfo.CacheEnabled {
			t.Errorf("expected cache_enabled=false in metadata, got %v", cacheEvent.Metadata.CacheInfo)
		}
	})
}

func TestCacheInfoCLILogging(t *testing.T) {
	// Start mock server
	baseURL, cleanup := startMockServer(t, "openai")
	defer cleanup()

	// Create client with mock server URL
	client, err := NewClient(Config{
		Model:   "gpt-4o",
		Token:   "test-token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Create CLI handler with LogChat enabled
	cliHandler := NewCliHandler(client, CliOptions{
		LogChat: true,
	})

	// Test that CLI handler processes cache info events
	// We can't easily capture stdout in tests, but we can verify the formatOutput doesn't panic
	cacheEvent := types.Message{
		Type:    types.MsgType_CacheInfo,
		Content: "Prompt cache enabled with gpt-4o",
		Model:   "gpt-4o",
		Metadata: types.Metadata{
			CacheInfo: &types.CacheInfoMetadata{
				CacheEnabled: true,
			},
		},
	}

	// This should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("formatOutput panicked for cache info event: %v", r)
			}
		}()
		cliHandler.formatOutput(cacheEvent)
	}()
}
