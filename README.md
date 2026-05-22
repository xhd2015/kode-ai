# `kode-ai`
The prime AI tool, including low level completion primitive as well as high level agentic coding.

# Install

`go install`:
```sh
go install github.com/xhd2015/kode-ai/cmd/kode@latest
```

`curl` from github:
```sh
curl -fsSL https://github.com/xhd2015/kode-ai/raw/master/install.sh | bash
```

# Usage

## Command Line Interface

```sh
# chat with gpt-4.1
export OPENAI_API_KEY=...
kode chat 'hello'

# chat with cluade, stop after initial response(1 round)
export ANTHROPIC_API_KEY=...
kode chat --model=claude-sonnet-4 --record=chat.json --system=EXAMPLE_SYSTEM.md --tool batch_read_file "What's in the file?"

# chat with --max-round
kode chat --max-round=10 ...
```

### OpenCode Zen free model smoke test

OpenCode Zen exposes OpenAI-compatible chat completions at the full REST endpoint:

```text
https://opencode.ai/zen/v1/chat/completions
```

When using `kode`, pass the SDK base URL instead. The OpenAI Go SDK appends
`/chat/completions` internally:

```text
https://opencode.ai/zen/v1
```

The live model list is available at:

```sh
curl -sS https://opencode.ai/zen/v1/models
```

For direct API calls, use the raw model id from that list, without the
`opencode/` prefix shown by the OpenCode CLI model picker. Free models observed
there include:

```text
deepseek-v4-flash-free
minimax-m2.5-free
nemotron-3-super-free
qwen3.6-plus-free
```

Run this from the repository root:

```sh
export OPENCODE_API_KEY=...

go run ./cmd/kode chat \
  --api-shape=openai \
  --base-url=https://opencode.ai/zen/v1 \
  --token="$OPENCODE_API_KEY" \
  --model=deepseek-v4-flash-free \
  --log-chat=false \
  'one word of French capital'
```

Expected output:

```text
Paris
```

Equivalent one-liner using the absolute `cmd/kode` package path:

```sh
go run ./cmd/kode chat --api-shape=openai --base-url=https://opencode.ai/zen/v1 --token="$OPENCODE_API_KEY" --model=deepseek-v4-flash-free --log-chat=false 'one word of French capital'
```

Validate tool call with agentic ability:
```sh
go run ./cmd/kode chat --api-shape=openai --base-url=https://opencode.ai/zen/v1 --token="$OPENCODE_API_KEY" --model=deepseek-v4-flash-free --log-chat=false "run ls and tell me what's there" --tool run_bash_script --max-round 10
```

## Library Usage

The `kode-ai` package can also be used as a Go library for programmatic chat interactions:

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/xhd2015/kode-ai/chat"
)

func main() {
    // Create client
    client, err := chat.NewClient(chat.Config{
        Model: "claude-3-7-sonnet",
        Token: "your-api-key",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Simple chat
    response, err := client.Chat(context.Background(), "Hello, how are you?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Response: %s\n", response.Content)
}
```

### Advanced Usage with Options

```go
// Chat with system prompt, tools, and history
response, err := client.Chat(context.Background(), "What files are in the current directory?",
    chat.WithSystemPrompt("You are a helpful assistant"),
    chat.WithTools("file_list", "file_read"),
    chat.WithMaxRounds(3),
    chat.WithCache(true),
)
```

### Custom Tool Callbacks

```go
// Handle tool calls with custom logic
toolCallback := func(ctx context.Context, call chat.ToolCall) (chat.ToolResult, bool, error) {
    switch call.Name {
    case "database_query":
        // Custom database query logic
        result := queryDatabase(call.Arguments["query"].(string))
        return chat.ToolResult{Content: result}, true, nil // handled=true
    case "api_call":
        // Custom API call logic
        response := callExternalAPI(call.Arguments)
        return chat.ToolResult{Content: response}, true, nil // handled=true
    default:
        // Don't handle this tool, fallback to built-in tools
        return chat.ToolResult{}, false, nil // handled=false
    }
}

response, err := client.Chat(context.Background(), "Query the user database",
    chat.WithToolCallback(toolCallback),
)
```

### Event Streaming

```go
// Real-time event handling
eventCallback := func(event chat.Event) {
    switch event.Type {
    case chat.EventTypeMessage:
        fmt.Printf("Message: %s\n", event.Content)
    case chat.EventTypeToolCall:
        fmt.Printf("Tool call: %s\n", event.Metadata["tool_name"])
    case chat.EventTypeTokenUsage:
        usage := event.Metadata["usage"].(chat.TokenUsage)
        fmt.Printf("Tokens used: %d\n", usage.Total)
    }
}

response, err := client.Chat(context.Background(), "Analyze this codebase",
    chat.WithEventCallback(eventCallback),
    chat.WithTools("file_read", "grep_search"),
)
```

# Contribution
