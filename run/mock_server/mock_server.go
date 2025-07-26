package mock_server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go"
	"github.com/xhd2015/kode-ai/tools"
	"github.com/xhd2015/llm-tools/jsonschema"
	"google.golang.org/genai"
)

// GeminiAPIRequest represents the minimal Gemini API request structure for parsing
// (SDK doesn't provide request parsing types, only response types)
type GeminiAPIRequest struct {
	Contents []*genai.Content `json:"contents"`
	Tools    []*genai.Tool    `json:"tools,omitempty"`
}

// Config holds the configuration for the mock server
type Config struct {
	Port             int
	Provider         string // "openai", "anthropic", "gemini", "all"
	FirstMsgToolCall bool   // if true, always respond with tool call instead of random
}

type MockServer struct {
	rand   *rand.Rand
	config Config
}

func NewMockServer(config Config) *MockServer {
	rd := rand.New(rand.NewSource(time.Now().UnixMicro()))
	return &MockServer{
		rand:   rd,
		config: config,
	}
}

// Start starts the mock HTTP server
func Start(config Config) error {
	m := NewMockServer(config)
	// Create HTTP server
	mux := http.NewServeMux()

	// Set up routes based on provider configuration
	switch strings.ToLower(config.Provider) {
	case "openai":
		mux.HandleFunc("/chat/completions", m.HandleOpenAIMock)
	case "anthropic":
		mux.HandleFunc("/v1/messages", m.HandleAnthropicMock)
	case "gemini":
		mux.HandleFunc("/v1beta/models/", m.HandleGeminiMock)
		mux.HandleFunc("/models/", m.HandleGeminiMock)
	case "all", "":
		// Enable all APIs
		mux.HandleFunc("/chat/completions", m.HandleOpenAIMock)
		mux.HandleFunc("/v1/messages", m.HandleAnthropicMock)
		mux.HandleFunc("/v1beta/models/", m.HandleGeminiMock)
		mux.HandleFunc("/models/", m.HandleGeminiMock)
	default:
		return fmt.Errorf("unsupported provider: %s (supported: openai, anthropic, gemini, all)", config.Provider)
	}

	addr := ":" + strconv.Itoa(config.Port)
	fmt.Printf("Starting mock server on http://localhost%s\n", addr)
	if config.Provider != "" && config.Provider != "all" {
		fmt.Printf("Provider: %s\n", config.Provider)
	} else {
		fmt.Printf("Provider: all (OpenAI, Anthropic, Gemini)\n")
	}
	fmt.Printf("Test with: kode chat --base-url http://localhost%s \"Hello world\"\n", addr)

	return http.ListenAndServe(addr, mux)
}

// handleOpenAIMockTyped handles OpenAI API mock responses with typed request and response
func (m *MockServer) handleOpenAIMockTyped(ctx context.Context, request openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	rd := m.rand
	// Extract tools from request (OpenAI format)
	var availableTools []*tools.UnifiedTool
	if request.Tools != nil {
		for _, tool := range request.Tools {
			unifiedTool := &tools.UnifiedTool{
				Name:        tool.Function.Name,
				Description: tool.Function.Description.Value,
			}

			// Parse parameters if available
			if tool.Function.Parameters != nil {
				paramsJSON, _ := json.Marshal(tool.Function.Parameters)
				var params jsonschema.JsonSchema
				if err := json.Unmarshal(paramsJSON, &params); err == nil {
					unifiedTool.Parameters = &params
				}
			}

			availableTools = append(availableTools, unifiedTool)
		}
	}

	var lastIsUser bool
	var lastIsTool bool

	for n := len(request.Messages) - 1; n >= 0; n-- {
		msg := request.Messages[n]

		if msg.OfTool != nil {
			fmt.Fprintf(os.Stderr, "DEBUG msg: %s\n", "tool")
			lastIsTool = true
			break
		}
		if msg.OfUser != nil {
			fmt.Fprintf(os.Stderr, "DEBUG msg: %s\n", "user")
			lastIsUser = true
			break
		}
		fmt.Fprintf(os.Stderr, "DEBUG msg: %s\n", "unknown")
	}

	var callTool bool
	if len(availableTools) > 0 {
		if m.config.FirstMsgToolCall {
			if lastIsTool {
				callTool = false
			} else if lastIsUser {
				callTool = true
			}
		} else {
			callTool = randomCallTool(m, availableTools)
		}
	}

	if (len(request.Messages)+1)%6 == 0 {
		callTool = false
	}
	fmt.Fprintf(os.Stderr, "DEBUG shouldCallTool: %d, %v\n", len(request.Messages), callTool)

	// Generate response using OpenAI SDK types
	if callTool {
		toolName := GetRandomToolFromUserTools(availableTools)
		toolArgs := GetRandomToolArgsFromUserTools(availableTools)

		// Create proper OpenAI response using SDK types
		response := &openai.ChatCompletion{
			ID:      fmt.Sprintf("chatcmpl-mock-%d", rd.Int31()),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Role: "assistant",
						ToolCalls: []openai.ChatCompletionMessageToolCall{
							{
								ID:   fmt.Sprintf("call_mock_%d", rd.Int31()),
								Type: "function",
								Function: openai.ChatCompletionMessageToolCallFunction{
									Name:      toolName,
									Arguments: toolArgs,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     int64(rd.Intn(100) + 10),
				CompletionTokens: int64(rd.Intn(50) + 5),
				TotalTokens:      int64(rd.Intn(150) + 15),
			},
		}
		return response, nil
	} else {
		// Regular text response
		response := &openai.ChatCompletion{
			ID:      fmt.Sprintf("chatcmpl-mock-%d", rd.Int31()),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: GetRandomResponse(),
					},
					FinishReason: "stop",
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     int64(rd.Intn(100) + 10),
				CompletionTokens: int64(rd.Intn(50) + 5),
				TotalTokens:      int64(rd.Intn(150) + 15),
			},
		}
		return response, nil
	}
}

// handleAnthropicMockTyped handles Anthropic API mock responses with typed request and response
func (m *MockServer) handleAnthropicMockTyped(ctx context.Context, request anthropic.MessageNewParams) (*anthropic.Message, error) {
	rd := m.rand
	// Extract tools from request (Anthropic format)
	var availableTools []*tools.UnifiedTool
	if request.Tools != nil {
		for _, tool := range request.Tools {
			if tool.OfTool != nil {
				unifiedTool := &tools.UnifiedTool{
					Name:        tool.OfTool.Name,
					Description: tool.OfTool.Description.Value,
				}

				// Parse input_schema if available
				paramsJSON, _ := json.Marshal(tool.OfTool.InputSchema)
				var params jsonschema.JsonSchema
				if err := json.Unmarshal(paramsJSON, &params); err == nil {
					unifiedTool.Parameters = &params
				}

				availableTools = append(availableTools, unifiedTool)
			}
		}
	}

	// For mock purposes, use map format since SDK types are complex for response construction
	// This follows the same pattern as the original handleAnthropicMock
	if randomCallTool(m, availableTools) {
		toolName := GetRandomToolFromUserTools(availableTools)
		toolArgs := GetRandomToolArgsFromUserTools(availableTools)

		// Parse tool args to create proper input
		var toolInput map[string]interface{}
		if err := json.Unmarshal([]byte(toolArgs), &toolInput); err != nil {
			toolInput = map[string]interface{}{"query": "mock query"}
		}

		// Create response using map (since SDK types are complex for construction)
		// but this approach is simpler and more maintainable than duplicating SDK structs
		responseMap := map[string]interface{}{
			"id":    fmt.Sprintf("msg_mock_%d", rd.Int31()),
			"type":  "message",
			"role":  "assistant",
			"model": "claude-3-5-sonnet-20241022",
			"content": []map[string]interface{}{
				{
					"type":  "tool_use",
					"id":    fmt.Sprintf("toolu_mock_%d", rd.Int31()),
					"name":  toolName,
					"input": toolInput,
				},
			},
			"stop_reason": "tool_use",
			"usage": map[string]interface{}{
				"input_tokens":  rd.Intn(100) + 10,
				"output_tokens": rd.Intn(50) + 5,
			},
		}

		// Convert to SDK type
		responseBytes, _ := json.Marshal(responseMap)
		var response anthropic.Message
		if err := json.Unmarshal(responseBytes, &response); err != nil {
			return nil, fmt.Errorf("failed to create mock response: %w", err)
		}
		return &response, nil
	} else {
		// Regular text response
		responseMap := map[string]interface{}{
			"id":    fmt.Sprintf("msg_mock_%d", rd.Int31()),
			"type":  "message",
			"role":  "assistant",
			"model": "claude-3-5-sonnet-20241022",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": GetRandomResponse(),
				},
			},
			"stop_reason": "end_turn",
			"usage": map[string]interface{}{
				"input_tokens":  rd.Intn(100) + 10,
				"output_tokens": rd.Intn(50) + 5,
			},
		}

		// Convert to SDK type
		responseBytes, _ := json.Marshal(responseMap)
		var response anthropic.Message
		if err := json.Unmarshal(responseBytes, &response); err != nil {
			return nil, fmt.Errorf("failed to create mock response: %w", err)
		}
		return &response, nil
	}
}

func randomCallTool(m *MockServer, availableTools []*tools.UnifiedTool) bool {
	if len(availableTools) == 0 {
		return false
	}
	return m.rand.Float32() < 0.3
}

// handleGeminiMockTyped handles Gemini API mock responses with typed request and response
func (m *MockServer) handleGeminiMockTyped(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	rd := m.rand
	// Extract tools from request (Gemini format)
	var availableTools []*tools.UnifiedTool
	if config != nil && config.Tools != nil {
		for _, tool := range config.Tools {
			if tool.FunctionDeclarations != nil {
				for _, funcDecl := range tool.FunctionDeclarations {
					unifiedTool := &tools.UnifiedTool{
						Name:        funcDecl.Name,
						Description: funcDecl.Description,
					}

					// Parse parameters if available
					if funcDecl.Parameters != nil {
						paramsJSON, _ := json.Marshal(funcDecl.Parameters)
						var params jsonschema.JsonSchema
						if err := json.Unmarshal(paramsJSON, &params); err == nil {
							unifiedTool.Parameters = &params
						}
					}

					availableTools = append(availableTools, unifiedTool)
				}
			}
		}
	}

	// Generate response using Gemini SDK types
	if randomCallTool(m, availableTools) {
		toolName := GetRandomToolFromUserTools(availableTools)
		toolArgs := GetRandomToolArgsFromUserTools(availableTools)

		// Parse tool args to create proper args
		var toolArgsMap map[string]interface{}
		if err := json.Unmarshal([]byte(toolArgs), &toolArgsMap); err != nil {
			toolArgsMap = map[string]interface{}{"query": "mock query"}
		}

		response := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{
								FunctionCall: &genai.FunctionCall{
									Name: toolName,
									Args: toolArgsMap,
								},
							},
						},
						Role: "model",
					},
					FinishReason: genai.FinishReasonStop,
				},
			},
			UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
				PromptTokenCount:     int32(rd.Intn(100) + 10),
				CandidatesTokenCount: int32(rd.Intn(50) + 5),
				TotalTokenCount:      int32(rd.Intn(150) + 15),
			},
		}
		return response, nil
	} else {
		// Regular text response
		response := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{
								Text: GetRandomResponse(),
							},
						},
						Role: "model",
					},
					FinishReason: genai.FinishReasonStop,
				},
			},
			UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
				PromptTokenCount:     int32(rd.Intn(100) + 10),
				CandidatesTokenCount: int32(rd.Intn(50) + 5),
				TotalTokenCount:      int32(rd.Intn(150) + 15),
			},
		}
		return response, nil
	}
}

// HandleOpenAIMock handles OpenAI API mock responses
func (m *MockServer) HandleOpenAIMock(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse request body as openai.ChatCompletionNewParams
	var request openai.ChatCompletionNewParams
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Use the typed handler
	response, err := m.handleOpenAIMockTyped(r.Context(), request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleAnthropicMock handles Anthropic API mock responses
func (m *MockServer) HandleAnthropicMock(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse request body as anthropicsdk.MessageNewParams
	var request anthropic.MessageNewParams
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Use the typed handler
	response, err := m.handleAnthropicMockTyped(r.Context(), request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleGeminiMock handles Gemini API mock responses
func (m *MockServer) HandleGeminiMock(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse request body using SDK types directly
	var request GeminiAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract contents and tools directly (already in SDK format)
	contents := request.Contents
	config := &genai.GenerateContentConfig{
		Tools: request.Tools,
	}

	// Use the typed handler
	response, err := m.handleGeminiMockTyped(r.Context(), "gemini-pro", contents, config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return
	}

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}
