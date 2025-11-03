package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	anth_opt "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go"
	openai_opt "github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/xhd2015/kode-ai/internal/ioread"
	"github.com/xhd2015/kode-ai/internal/jsondecode"
	"github.com/xhd2015/kode-ai/providers"
	anthropic_helper "github.com/xhd2015/kode-ai/providers/anthropic"
	"github.com/xhd2015/kode-ai/tools"
	"github.com/xhd2015/kode-ai/types"
	"google.golang.org/genai"
)

// Client represents the chat client
type Client struct {
	config   Config
	apiShape providers.APIShape

	stdinReader types.StdinReader
	logger      types.Logger
}

// NewClient creates a new chat client
func NewClient(config Config) (*Client, error) {
	if config.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if config.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	// Auto-detect API shape from model if not provided
	apiShape, err := providers.GetModelAPIShape(config.Model)
	if err != nil {
		return nil, fmt.Errorf("determine API shape: %w", err)
	}

	logger := config.Logger
	if logger == nil {
		logger = types.LoggerFunc(func(ctx context.Context, logType types.LogType, format string, args ...interface{}) {
			// silent
		})
	}

	return &Client{
		config:   config,
		apiShape: apiShape,
		logger:   logger,
	}, nil
}

// Chat performs a chat conversation using functional options
func (c *Client) Chat(ctx context.Context, message string, opts ...types.ChatOption) (*types.Response, error) {
	req := types.Request{
		Message: message,
	}

	// Apply options
	for _, opt := range opts {
		opt(&req)
	}

	return c.ChatRequest(ctx, req)
}

// ChatRequest performs a chat conversation using a direct request
func (c *Client) ChatRequest(ctx context.Context, req types.Request) (*types.Response, error) {
	// Create clients
	clients, err := c.createClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("create clients: %w", err)
	}

	// Prepare tools
	toolInfoMapping, toolSchemas, err := c.prepareTools(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("prepare tools: %w", err)
	}

	// Convert tools to provider-specific formats
	var toolsOpenAI []openai.ChatCompletionToolParam
	var toolsAnthropic []anthropic.ToolUnionParam
	var toolsGemini []*genai.Tool

	switch c.apiShape {
	case providers.APIShapeOpenAI:
		toolsOpenAI, err = toolSchemas.ToOpenAI()
		if err != nil {
			return nil, fmt.Errorf("convert tools to OpenAI format: %w", err)
		}
	case providers.APIShapeAnthropic:
		toolsAnthropic, err = toolSchemas.ToAnthropic()
		if err != nil {
			return nil, fmt.Errorf("convert tools to Anthropic format: %w", err)
		}
	case providers.APIShapeGemini:
		toolsGemini, err = toolSchemas.ToGemini()
		if err != nil {
			return nil, fmt.Errorf("convert tools to Gemini format: %w", err)
		}
	}

	// Prepare system prompts and messages
	systemPrompts := GetSystemPrompts(req.History)
	var systemMessageOpenAI *openai.ChatCompletionMessageParamUnion
	var systemAnthropic []anthropic.TextBlockParam
	var systemMessageGemini *genai.Content

	if req.SystemPrompt != "" {
		content, err := ioread.ReadOrContent(req.SystemPrompt)
		if err != nil {
			return nil, fmt.Errorf("read system prompt: %w", err)
		}

		switch c.apiShape {
		case providers.APIShapeOpenAI:
			systemMessageOpenAI = &openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: param.NewOpt(content),
					},
				},
			}
		case providers.APIShapeAnthropic:
			systemMsg := anthropic.TextBlockParam{
				Text: content,
			}
			systemAnthropic = append(systemAnthropic, systemMsg)
		case providers.APIShapeGemini:
			systemMessageGemini = &genai.Content{
				Parts: []*genai.Part{
					{
						Text: content,
					},
				},
			}
		}
	} else if len(systemPrompts) > 0 {
		lastSystemPrompt := systemPrompts[len(systemPrompts)-1]
		switch c.apiShape {
		case providers.APIShapeOpenAI:
			systemMessageOpenAI = &openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: param.NewOpt(lastSystemPrompt),
					},
				},
			}
		case providers.APIShapeAnthropic:
			systemMsg := anthropic.TextBlockParam{
				Text: lastSystemPrompt,
			}
			systemAnthropic = append(systemAnthropic, systemMsg)
		case providers.APIShapeGemini:
			systemMessageGemini = &genai.Content{
				Parts: []*genai.Part{
					{
						Text: lastSystemPrompt,
					},
				},
			}
		}
	}

	// Convert history to provider-specific formats
	historyMsgs := Messages(req.History)

	var historicalMessagesOpenAI []openai.ChatCompletionMessageParamUnion
	var historicalMessagesAnthropic []anthropic.MessageParam
	var historicalMessagesGemini []*genai.Content

	switch c.apiShape {
	case providers.APIShapeOpenAI:
		historicalMessagesOpenAI, _, err = historyMsgs.ToOpenAI(false)
		if err != nil {
			return nil, fmt.Errorf("convert history to OpenAI format: %w", err)
		}
	case providers.APIShapeAnthropic:
		historicalMessagesAnthropic, _, err = historyMsgs.ToAnthropic()
		if err != nil {
			return nil, fmt.Errorf("convert history to Anthropic format: %w", err)
		}
	case providers.APIShapeGemini:
		historicalMessagesGemini, _, err = historyMsgs.ToGemini()
		if err != nil {
			return nil, fmt.Errorf("convert history to Gemini format: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.apiShape)
	}

	// Build messages
	msgsUnion, err := c.buildMessages(req.Message, systemMessageOpenAI, historicalMessagesOpenAI, historicalMessagesAnthropic, historicalMessagesGemini)
	if err != nil {
		return nil, fmt.Errorf("build messages: %w", err)
	}

	// Determine cache settings
	needCache := !req.NoCache
	if needCache && c.apiShape == providers.APIShapeAnthropic {
		systemAnthropic = anthropic_helper.MarkTextBlocksEphemeralCache(systemAnthropic)
		toolsAnthropic = anthropic_helper.MarkToolsEphemeralCache(toolsAnthropic)
	}

	var stream types.StreamContext
	if req.StreamPair != nil {
		stream = types.NewStreamContext(req.StreamPair.Output)
	}

	// Emit cache info event
	if req.EventCallback != nil {
		cacheStatus := "enabled"
		if !needCache {
			cacheStatus = "disabled"
		}
		req.EventCallback(types.Message{
			Type:    types.MsgType_CacheInfo,
			Model:   c.config.Model,
			Content: fmt.Sprintf("Prompt cache %s with %s", cacheStatus, c.config.Model),
			Metadata: types.Metadata{
				CacheInfo: &types.CacheInfoMetadata{
					CacheEnabled: needCache,
				},
			},
			Timestamp: time.Now().Unix(),
		})
	}

	// Execute conversation rounds
	maxRounds := req.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 1
	}

	var allMessages []types.Message
	var totalTokenUsage types.TokenUsage
	var allToolCalls []types.ToolCall
	hasMaxRound := req.MaxRounds > 1

	var toolUseNum int
	for _, msg := range req.History {
		if msg.Type == types.MsgType_ToolCall {
			toolUseNum++
		}
	}

	// Initialize stdin reader if streams are provided
	if req.StreamPair != nil {
		// Check if the input is already a StdinReader (e.g., WebSocket reader)
		if stdinReader, ok := req.StreamPair.Input.(types.StdinReader); ok {
			c.stdinReader = stdinReader
		} else {
			c.stdinReader = types.NewStdinReader(req.StreamPair.Input)
		}
	}

	for round := 0; round < maxRounds; round++ {
		// Make API call
		var tokenUsage types.TokenUsage
		var newToolUseNum int
		var stopped bool

		switch c.apiShape {
		case providers.APIShapeOpenAI:
			result, err := clients.OpenAI.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
				Model:    c.config.Model,
				Messages: msgsUnion.OpenAI,
				Tools:    toolsOpenAI,
				N:        param.NewOpt(int64(1)),
			})
			if err != nil {
				return nil, fmt.Errorf("OpenAI API call: %w", err)
			}

			res, err := c.processOpenAIResponse(ctx, stream, result, hasMaxRound, req, toolInfoMapping)
			if err != nil {
				return nil, fmt.Errorf("process OpenAI response: %w", err)
			}
			tokenUsage = res.TokenUsage
			allMessages = append(allMessages, res.Messages...)
			allToolCalls = append(allToolCalls, res.ToolCalls...)
			newToolUseNum = res.ToolUseNum
			stopped = res.Stopped

			msgsUnion.OpenAI = append(msgsUnion.OpenAI, res.RespMessages...)
			msgsUnion.OpenAI = append(msgsUnion.OpenAI, res.ToolResults...)

		case providers.APIShapeAnthropic:
			sendMessage := msgsUnion.Anthropic
			if needCache {
				sendMessage = anthropic_helper.MarkMsgsEphemeralCache(msgsUnion.Anthropic)
			}
			result, err := anthropic_helper.Chat(ctx, clients.Anthropic, anthropic.MessageNewParams{
				// if MaxTokens > 20K:  anthropic API call: streaming is strongly recommended for operations that may take longer than 10 minutes
				MaxTokens: 20 * 1024, // according to Anthropic, max for 4.5 is 64K, this effectively disables the limit
				Model:     anthropic.Model(c.config.Model),
				Messages:  sendMessage,
				System:    systemAnthropic,
				Tools:     toolsAnthropic,
			})
			if err != nil {
				return nil, fmt.Errorf("anthropic API call: %w", err)
			}

			res, err := c.processAnthropicResponse(ctx, stream, result, hasMaxRound, req, toolInfoMapping)
			if err != nil {
				return nil, fmt.Errorf("process Anthropic response: %w", err)
			}
			tokenUsage = res.TokenUsage
			allMessages = append(allMessages, res.Messages...)
			allToolCalls = append(allToolCalls, res.ToolCalls...)
			newToolUseNum = res.ToolUseNum
			stopped = res.Stopped

			if len(res.RespMessages) > 0 {
				msgsUnion.Anthropic = append(msgsUnion.Anthropic, anthropic.MessageParam{
					Role:    anthropic.MessageParamRole(result.Role),
					Content: res.RespMessages,
				})
			}
			if len(res.ToolResults) > 0 {
				msgsUnion.Anthropic = append(msgsUnion.Anthropic, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleUser,
					Content: res.ToolResults,
				})
			}

		case providers.APIShapeGemini:
			result, err := clients.Gemini.Models.GenerateContent(ctx, c.config.Model, msgsUnion.Gemini, &genai.GenerateContentConfig{
				HTTPOptions: &genai.HTTPOptions{
					APIVersion: "v1",
					Headers: http.Header{
						"Authorization": []string{fmt.Sprintf("Bearer %s", c.config.Token)},
					},
				},
				SystemInstruction: systemMessageGemini,
				Tools:             toolsGemini,
				CandidateCount:    1,
			})
			if err != nil {
				return nil, fmt.Errorf("Gemini API call: %w", err)
			}

			res, err := c.processGeminiResponse(ctx, stream, result, toolUseNum, hasMaxRound, req, toolInfoMapping)
			if err != nil {
				return nil, fmt.Errorf("process Gemini response: %w", err)
			}
			tokenUsage = res.TokenUsage
			allMessages = append(allMessages, res.Messages...)
			allToolCalls = append(allToolCalls, res.ToolCalls...)
			newToolUseNum = res.ToolUseNum
			stopped = res.Stopped

			if len(res.RespMessages) > 0 {
				msgsUnion.Gemini = append(msgsUnion.Gemini, res.RespMessages...)
			}
			if len(res.ToolResults) > 0 {
				msgsUnion.Gemini = append(msgsUnion.Gemini, res.ToolResults...)
			}

		default:
			return nil, fmt.Errorf("unsupported provider: %s", c.apiShape)
		}

		totalTokenUsage = totalTokenUsage.Add(tokenUsage)
		if req.EventCallback != nil {
			req.EventCallback(types.Message{
				Type:       types.MsgType_TokenUsage,
				TokenUsage: &tokenUsage,
			})
		}

		toolUseNum += newToolUseNum
		if stopped || newToolUseNum == 0 {
			// no more tool calls, stop
			// check if stream pair allow asking for user input
			if c.stdinReader != nil {
				msg, streamErr := types.StreamRequest(ctx, req.StreamPair.Output, c.stdinReader, types.Message{
					Type:     types.MsgType_StreamRequestUserMsg,
					StreamID: "user-input-" + uuid.New().String(),
				}, "")
				if streamErr != nil {
					if streamErr == types.ErrStreamEnd {
						break
					}
					return nil, fmt.Errorf("stream request: %w", streamErr)
				}
				if msg.Type == "" {
					msg.Type = types.MsgType_Msg
				}
				if msg.Role == "" {
					msg.Role = types.Role_User
				}

				streamErr = addToMsgUnion(c.apiShape, msgsUnion, msg)
				if streamErr != nil {
					return nil, fmt.Errorf("append messages: %w", streamErr)
				}

				allMessages = append(allMessages, msg)
				continue
			}
			break
		}
	}

	// Compute cost if possible
	var cost *types.TokenCost
	if costResult, ok := c.computeCost(totalTokenUsage); ok {
		cost = &costResult
	}

	return &types.Response{
		TokenUsage: totalTokenUsage,
		Cost:       cost,
		RoundsUsed: len(allMessages), // TODO: should be the number of rounds used
	}, nil
}

func addToMsgUnion(apiShape providers.APIShape, msgsUnion *MessagesUnion, msg types.Message) error {
	msgs := Messages{msg}
	switch apiShape {
	case providers.APIShapeOpenAI:
		providerMsgs, _, err := msgs.ToOpenAI(false)
		if err != nil {
			return fmt.Errorf("convert message to OpenAI format: %w", err)
		}
		if len(providerMsgs) == 0 {
			return fmt.Errorf("no messages to add")
		}
		msgsUnion.OpenAI = append(msgsUnion.OpenAI, providerMsgs...)
	case providers.APIShapeAnthropic:
		providerMsgs, _, err := msgs.ToAnthropic()
		if err != nil {
			return fmt.Errorf("convert message to Anthropic format: %w", err)
		}
		if len(providerMsgs) == 0 {
			return fmt.Errorf("no messages to add")
		}
		msgsUnion.Anthropic = append(msgsUnion.Anthropic, providerMsgs...)
	case providers.APIShapeGemini:
		providerMsgs, _, err := msgs.ToGemini()
		if err != nil {
			return fmt.Errorf("convert message to Gemini format: %w", err)
		}
		if len(providerMsgs) == 0 {
			return fmt.Errorf("no messages to add")
		}
		msgsUnion.Gemini = append(msgsUnion.Gemini, providerMsgs...)
	default:
		return fmt.Errorf("unsupported api shape: %s", apiShape)
	}
	return nil
}

// Response processing result types
type ResponseResult struct {
	Messages     []types.Message
	ToolCalls    []types.ToolCall
	TokenUsage   types.TokenUsage
	ToolUseNum   int
	Stopped      bool
	RespMessages []openai.ChatCompletionMessageParamUnion // For OpenAI
	ToolResults  []openai.ChatCompletionMessageParamUnion // For OpenAI
}

type AnthropicResponseResult struct {
	Messages     []types.Message
	ToolCalls    []types.ToolCall
	TokenUsage   types.TokenUsage
	ToolUseNum   int
	Stopped      bool
	RespMessages []anthropic.ContentBlockParamUnion
	ToolResults  []anthropic.ContentBlockParamUnion
}

type GeminiResponseResult struct {
	Messages     []types.Message
	ToolCalls    []types.ToolCall
	TokenUsage   types.TokenUsage
	ToolUseNum   int
	Stopped      bool
	RespMessages []*genai.Content
	ToolResults  []*genai.Content
}

const MAX_PRINT_LIMIT = 2048

func limitPrintLength(s string) string {
	if len(s) < MAX_PRINT_LIMIT+3 {
		return s
	}
	return s[:MAX_PRINT_LIMIT] + "..."
}

// processOpenAIResponse processes OpenAI API response
func (c *Client) processOpenAIResponse(ctx context.Context, stream types.StreamContext, result *openai.ChatCompletion, hasMaxRound bool, req types.Request, toolInfoMapping ToolInfoMapping) (*ResponseResult, error) {
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("response no choices")
	}
	firstChoice := result.Choices[0]
	var toolUseNum int
	var messages []types.Message
	var toolCalls []types.ToolCall
	var respMessages []openai.ChatCompletionMessageParamUnion
	var toolResults []openai.ChatCompletionMessageParamUnion

	// Handle main content
	if firstChoice.Message.Content != "" {
		// Emit message event
		if req.EventCallback != nil {
			req.EventCallback(types.Message{
				Type:      types.MsgType_Msg,
				Content:   firstChoice.Message.Content,
				Role:      types.Role_Assistant,
				Model:     c.config.Model,
				Timestamp: time.Now().Unix(),
			})
		}

		respMessages = append(respMessages, openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: param.NewOpt(firstChoice.Message.Content),
				},
			},
		})

		messages = append(messages, CreateMessage(types.MsgType_Msg, types.Role_Assistant, c.config.Model, firstChoice.Message.Content))
	}

	// Handle tool calls
	var recordToolCalls []openai.ChatCompletionMessageToolCallParam
	for _, toolCall := range firstChoice.Message.ToolCalls {
		toolUseNum++

		call, err := parseToolCall(toolCall.Function.Name, toolCall.ID, toolCall.Function.Arguments, req.DefaultToolCwd)
		if err != nil {
			return nil, fmt.Errorf("parse tool call: %w", err)
		}
		toolCalls = append(toolCalls, call)

		// Emit tool call event
		if req.EventCallback != nil {
			req.EventCallback(types.Message{
				Type:      types.MsgType_ToolCall,
				Content:   toolCall.Function.Arguments,
				ToolUseID: toolCall.ID,
				ToolName:  toolCall.Function.Name,
				Model:     c.config.Model,
				Role:      types.Role_Assistant,
				Timestamp: time.Now().Unix(),
			})
		}

		recordToolCalls = append(recordToolCalls, openai.ChatCompletionMessageToolCallParam{
			ID: toolCall.ID,
			Function: openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		})

		messages = append(messages, CreateToolCallMessage(types.Role_Assistant, c.config.Model, toolCall.Function.Name, toolCall.ID, toolCall.Function.Arguments))

		// Execute tool
		var stdout io.Writer
		if req.StreamPair != nil {
			stdout = req.StreamPair.Output
		}
		result, err := c.executeToolWithCallback(ctx, stream, call, req.ToolCallback, stdout, req.DefaultToolCwd, toolInfoMapping)
		if err != nil {
			return nil, fmt.Errorf("execute tool: %w", err)
		}

		var resultStr string
		if result.Error != "" {
			resultStr = fmt.Sprintf("Error: %v", result.Error)
		} else {
			resultJSON, err := json.Marshal(result.Content)
			if err != nil {
				resultStr = fmt.Sprintf("Error marshaling result: %v", err)
			} else {
				resultStr = string(resultJSON)
			}
		}

		// Emit tool result event
		if req.EventCallback != nil {
			req.EventCallback(types.Message{
				Type:      types.MsgType_ToolResult,
				Content:   limitPrintLength(resultStr),
				ToolUseID: toolCall.ID,
				ToolName:  toolCall.Function.Name,
				Model:     c.config.Model,
				Role:      types.Role_User,
				Timestamp: time.Now().Unix(),
			})
		}

		toolResults = append(toolResults, openai.ChatCompletionMessageParamUnion{
			OfTool: &openai.ChatCompletionToolMessageParam{
				ToolCallID: toolCall.ID,
				Content: openai.ChatCompletionToolMessageParamContentUnion{
					OfString: param.NewOpt(resultStr),
				},
			},
		})

		messages = append(messages, CreateToolResultMessage(types.Role_User, c.config.Model, toolCall.Function.Name, toolCall.ID, resultStr))
	}

	if len(recordToolCalls) > 0 {
		respMessages = append(respMessages, openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				ToolCalls: recordToolCalls,
			},
		})
	}

	return &ResponseResult{
		Messages:     messages,
		ToolCalls:    toolCalls,
		ToolUseNum:   toolUseNum,
		RespMessages: respMessages,
		ToolResults:  toolResults,
		Stopped:      firstChoice.FinishReason == "stop",
		TokenUsage: types.TokenUsage{
			Input:  result.Usage.PromptTokens,
			Output: result.Usage.CompletionTokens,
			Total:  result.Usage.TotalTokens,
			InputBreakdown: types.TokenUsageInputBreakdown{
				CacheRead:    result.Usage.PromptTokensDetails.CachedTokens,
				NonCacheRead: result.Usage.PromptTokens - result.Usage.PromptTokensDetails.CachedTokens,
			},
		},
	}, nil
}

// processAnthropicResponse processes Anthropic API response
func (c *Client) processAnthropicResponse(ctx context.Context, stream types.StreamContext, result *anthropic.Message, hasMaxRound bool, req types.Request, toolInfoMapping ToolInfoMapping) (*AnthropicResponseResult, error) {
	var toolUseNum int
	var messages []types.Message
	var toolCalls []types.ToolCall
	var respContents []anthropic.ContentBlockParamUnion
	var toolResults []anthropic.ContentBlockParamUnion

	for _, msg := range result.Content {
		switch msg.Type {
		case "text":
			txt := msg.AsText()

			// Emit message event
			if req.EventCallback != nil {
				req.EventCallback(types.Message{
					Type:      types.MsgType_Msg,
					Role:      types.Role_Assistant,
					Content:   txt.Text,
					Timestamp: time.Now().Unix(),
				})
			}

			respContents = append(respContents, anthropic.ContentBlockParamUnion{
				OfText: &anthropic.TextBlockParam{
					Text: txt.Text,
				},
			})

			messages = append(messages, CreateMessage(types.MsgType_Msg, types.Role_Assistant, c.config.Model, txt.Text))

		case "tool_use":
			toolUseNum++
			toolUse := msg.AsToolUse()

			call, err := parseToolCall(toolUse.Name, toolUse.ID, string(toolUse.Input), req.DefaultToolCwd)
			if err != nil {
				return nil, fmt.Errorf("parse tool call: %w", err)
			}
			toolCalls = append(toolCalls, call)

			// Emit tool call event
			if req.EventCallback != nil {
				req.EventCallback(types.Message{
					Type:      types.MsgType_ToolCall,
					Content:   string(toolUse.Input),
					Model:     c.config.Model,
					Role:      types.Role_Assistant,
					Timestamp: time.Now().Unix(),
					ToolUseID: toolUse.ID,
					ToolName:  toolUse.Name,
				})
			}

			input := json.RawMessage(toolUse.Input)
			respContents = append(respContents, anthropic.ContentBlockParamUnion{
				OfToolUse: &anthropic.ToolUseBlockParam{
					ID:    toolUse.ID,
					Input: input,
					Name:  toolUse.Name,
				},
			})

			messages = append(messages, CreateToolCallMessage(types.Role_Assistant, c.config.Model, toolUse.Name, toolUse.ID, string(toolUse.Input)))

			// Execute tool
			var stdout io.Writer
			if req.StreamPair != nil {
				stdout = req.StreamPair.Output
			}
			toolResult, err := c.executeToolWithCallback(ctx, stream, call, req.ToolCallback, stdout, req.DefaultToolCwd, toolInfoMapping)
			if err != nil {
				return nil, fmt.Errorf("execute tool: %w", err)
			}

			var resultStr string
			if toolResult.Error != "" {
				resultStr = fmt.Sprintf("Error: %v", toolResult.Error)
			} else {
				resultJSON, err := json.Marshal(toolResult.Content)
				if err != nil {
					resultStr = fmt.Sprintf("Error marshaling result: %v", err)
				} else {
					resultStr = string(resultJSON)
				}
			}

			// Emit tool result event
			if req.EventCallback != nil {
				req.EventCallback(types.Message{
					Type:      types.MsgType_ToolResult,
					Content:   limitPrintLength(resultStr),
					Model:     c.config.Model,
					Role:      types.Role_User,
					Timestamp: time.Now().Unix(),
					ToolUseID: toolUse.ID,
					ToolName:  toolUse.Name,
				})
			}

			toolResults = append(toolResults, anthropic.ContentBlockParamUnion{
				OfToolResult: &anthropic.ToolResultBlockParam{
					ToolUseID: toolUse.ID,
					Content: []anthropic.ToolResultBlockParamContentUnion{
						{
							OfText: &anthropic.TextBlockParam{
								Text: resultStr,
							},
						},
					},
				},
			})

			messages = append(messages, CreateToolResultMessage(types.Role_User, c.config.Model, toolUse.Name, toolUse.ID, resultStr))

		default:
			return nil, fmt.Errorf("unrecognized message type: %s", msg.Type)
		}
	}

	totalInput := result.Usage.InputTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens
	return &AnthropicResponseResult{
		Messages:     messages,
		ToolCalls:    toolCalls,
		ToolUseNum:   toolUseNum,
		RespMessages: respContents,
		ToolResults:  toolResults,
		Stopped:      result.StopReason == "end_turn",
		TokenUsage: types.TokenUsage{
			Input:  totalInput,
			Output: result.Usage.OutputTokens,
			Total:  totalInput + result.Usage.OutputTokens,
			InputBreakdown: types.TokenUsageInputBreakdown{
				CacheWrite:   result.Usage.CacheCreationInputTokens,
				CacheRead:    result.Usage.CacheReadInputTokens,
				NonCacheRead: result.Usage.InputTokens,
			},
		},
	}, nil
}

// processGeminiResponse processes Gemini API response
func (c *Client) processGeminiResponse(ctx context.Context, stream types.StreamContext, result *genai.GenerateContentResponse, toolUsedNum int, hasMaxRound bool, req types.Request, toolInfoMapping ToolInfoMapping) (*GeminiResponseResult, error) {
	var toolUseNum int
	var messages []types.Message
	var toolCalls []types.ToolCall
	var respContents []*genai.Content
	var toolResults []*genai.Content

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("empty result candidates")
	}
	choice := result.Candidates[0]

	for _, part := range choice.Content.Parts {
		if part.FunctionCall != nil {
			toolUseNum++
			toolUse := part.FunctionCall

			toolRecordID := toolUse.ID
			if toolUse.ID == "" {
				toolRecordID = strconv.FormatInt(int64(toolUsedNum+1), 10)
			}
			argsJSON, err := json.Marshal(toolUse.Args)
			if err != nil {
				return nil, fmt.Errorf("marshal args: %w", err)
			}
			argsJSONStr := string(argsJSON)

			call, err := parseToolCall(toolUse.Name, toolRecordID, argsJSONStr, req.DefaultToolCwd)
			if err != nil {
				return nil, fmt.Errorf("parse tool call: %w", err)
			}
			toolCalls = append(toolCalls, call)

			// Emit tool call event
			if req.EventCallback != nil {
				req.EventCallback(types.Message{
					Type:      types.MsgType_ToolCall,
					Content:   argsJSONStr,
					Model:     c.config.Model,
					Timestamp: time.Now().Unix(),
					Role:      types.Role_Assistant,
					ToolUseID: toolUse.ID,
					ToolName:  toolUse.Name,
				})
			}

			cloneToolUse := *toolUse
			respContents = append(respContents, &genai.Content{
				Parts: []*genai.Part{
					{
						FunctionCall: &cloneToolUse,
					},
				},
				Role: choice.Content.Role,
			})

			messages = append(messages, CreateToolCallMessage(types.Role_Assistant, c.config.Model, toolUse.Name, toolRecordID, argsJSONStr))

			// Execute tool
			var stdout io.Writer
			if req.StreamPair != nil {
				stdout = req.StreamPair.Output
			}
			toolResult, err := c.executeToolWithCallback(ctx, stream, call, req.ToolCallback, stdout, req.DefaultToolCwd, toolInfoMapping)
			if err != nil {
				return nil, fmt.Errorf("execute tool: %w", err)
			}

			var resultStr string
			if toolResult.Error != "" {
				resultStr = fmt.Sprintf("Error: %v", toolResult.Error)
			} else {
				resultJSON, err := json.Marshal(toolResult.Content)
				if err != nil {
					resultStr = fmt.Sprintf("Error marshaling result: %v", err)
				} else {
					resultStr = string(resultJSON)
				}
			}

			// Emit tool result event
			if req.EventCallback != nil {
				req.EventCallback(types.Message{
					Type:      types.MsgType_ToolResult,
					Content:   limitPrintLength(resultStr),
					Model:     c.config.Model,
					Role:      types.Role_User,
					Timestamp: time.Now().Unix(),
					ToolUseID: toolUse.ID,
					ToolName:  toolUse.Name,
				})
			}

			var response map[string]any
			err = jsondecode.UnmarshalSafe([]byte(resultStr), &response)
			if err != nil {
				return nil, fmt.Errorf("unmarshal tool result: %w", err)
			}

			toolResults = append(toolResults, &genai.Content{
				Role: genai.RoleUser,
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							ID:       toolUse.ID,
							Name:     toolUse.Name,
							Response: response,
						},
					},
				},
			})

			messages = append(messages, CreateToolResultMessage(types.Role_User, c.config.Model, toolUse.Name, toolRecordID, resultStr))

		} else if part.Text != "" {
			txt := part.Text

			// Emit message event
			if req.EventCallback != nil {
				req.EventCallback(types.Message{
					Type:      types.MsgType_Msg,
					Content:   txt,
					Role:      types.Role_Assistant,
					Timestamp: time.Now().Unix(),
				})
			}

			respContents = append(respContents, &genai.Content{
				Role: choice.Content.Role,
				Parts: []*genai.Part{
					{
						Text: txt,
					},
				},
			})

			messages = append(messages, CreateMessage(types.MsgType_Msg, types.Role_Assistant, c.config.Model, txt))
		}
	}

	var tokenUsage types.TokenUsage
	if result.UsageMetadata != nil {
		usage := result.UsageMetadata

		inputToken := usage.PromptTokenCount + usage.ToolUsePromptTokenCount
		outputToken := usage.CandidatesTokenCount
		cacheRead := usage.CachedContentTokenCount

		tokenUsage = types.TokenUsage{
			Input:  int64(inputToken),
			Output: int64(outputToken),
			Total:  int64(usage.TotalTokenCount),
			InputBreakdown: types.TokenUsageInputBreakdown{
				CacheRead:    int64(cacheRead),
				NonCacheRead: int64(inputToken - cacheRead),
			},
		}
	}

	return &GeminiResponseResult{
		Messages:     messages,
		ToolCalls:    toolCalls,
		ToolUseNum:   toolUseNum,
		RespMessages: respContents,
		ToolResults:  toolResults,
		Stopped:      choice.FinishReason == genai.FinishReasonStop,
		TokenUsage:   tokenUsage,
	}, nil
}

// createClients creates provider-specific clients
func (c *Client) createClients(ctx context.Context) (*ClientUnion, error) {
	var clientOpenAI *openai.Client
	var clientAnthropic *anthropic.Client
	var clientGemini *genai.Client

	switch c.apiShape {
	case providers.APIShapeOpenAI:
		var clientOptions []openai_opt.RequestOption
		if c.config.BaseURL != "" {
			clientOptions = append(clientOptions, openai_opt.WithBaseURL(c.config.BaseURL))
		}
		clientOptions = append(clientOptions, openai_opt.WithAPIKey(c.config.Token))
		if c.config.LogLevel >= types.LogLevelRequest {
			logger := log.New(os.Stderr, "", log.LstdFlags)
			clientOptions = append(clientOptions, openai_opt.WithDebugLog(logger))
		}
		client := openai.NewClient(clientOptions...)
		clientOpenAI = &client

	case providers.APIShapeAnthropic:
		var clientOpts []anth_opt.RequestOption
		if c.config.BaseURL != "" {
			clientOpts = append(clientOpts, anth_opt.WithBaseURL(c.config.BaseURL))
		}
		clientOpts = append(clientOpts, anth_opt.WithAPIKey(c.config.Token))
		if c.config.LogLevel >= types.LogLevelRequest {
			logger := log.New(os.Stderr, "", log.LstdFlags)
			clientOpts = append(clientOpts, anth_opt.WithDebugLog(logger))
		}
		clientAnthropic = anthropic_helper.NewClient(clientOpts...)

	case providers.APIShapeGemini:
		var err error
		clientGemini, err = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  c.config.Token,
			Backend: genai.BackendGeminiAPI,
			HTTPOptions: genai.HTTPOptions{
				BaseURL: c.config.BaseURL,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("create Gemini client: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.apiShape)
	}

	return &ClientUnion{
		OpenAI:    clientOpenAI,
		Anthropic: clientAnthropic,
		Gemini:    clientGemini,
	}, nil
}

// prepareTools prepares the tool schemas and mappings
func (c *Client) prepareTools(ctx context.Context, req types.Request) (ToolInfoMapping, tools.UnifiedTools, error) {
	toolInfoMapping := make(ToolInfoMapping)

	// Parse custom tool schemas
	toolSchemas, err := tools.ParseSchemas(req.ToolFiles, req.ToolJSONs, req.ToolDefinitions)
	if err != nil {
		return nil, nil, fmt.Errorf("parse tool schemas: %w", err)
	}
	for _, tool := range toolSchemas {
		if err := toolInfoMapping.AddTool(tool.Name, &ToolInfo{
			Name:           tool.Name,
			ToolDefinition: tool,
		}); err != nil {
			return nil, nil, err
		}
	}

	// Get builtin tools
	builtinTools, err := tools.GetBuiltinTools(req.Tools)
	if err != nil {
		return nil, nil, fmt.Errorf("get builtin tools: %w", err)
	}
	for _, tool := range builtinTools {
		if err := toolInfoMapping.AddTool(tool.Name, &ToolInfo{
			Name:           tool.Name,
			Builtin:        true,
			ToolDefinition: tool,
		}); err != nil {
			return nil, nil, err
		}
	}
	toolSchemas = append(toolSchemas, builtinTools...)

	// Setup MCP clients
	for _, mcpServer := range req.MCPServers {
		mcpClient, err := c.connectToMCPServer(mcpServer)
		if err != nil {
			return nil, nil, fmt.Errorf("connect to MCP server: %w", err)
		}
		res, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{})
		if err != nil {
			return nil, nil, fmt.Errorf("initialize MCP client: %w", err)
		}
		_ = res

		// Get MCP tools
		mcpTools, err := c.getMCPTools(ctx, mcpClient)
		if err != nil {
			return nil, nil, fmt.Errorf("list mcp tools: %w", err)
		}
		for _, tool := range mcpTools {
			if err := toolInfoMapping.AddTool(tool.Name, &ToolInfo{
				Name:           tool.Name,
				MCPServer:      mcpServer,
				MCPClient:      mcpClient,
				ToolDefinition: tool,
			}); err != nil {
				return nil, nil, err
			}
		}
		toolSchemas = append(toolSchemas, mcpTools...)
	}

	return toolInfoMapping, toolSchemas, nil
}

// buildMessages builds provider-specific message formats
func (c *Client) buildMessages(msg string, systemMessageOpenAI *openai.ChatCompletionMessageParamUnion, historicalMessagesOpenAI []openai.ChatCompletionMessageParamUnion, historicalMessagesAnthropic []anthropic.MessageParam, historicalMessagesGemini []*genai.Content) (*MessagesUnion, error) {
	var messagesOpenAI []openai.ChatCompletionMessageParamUnion
	var messagesAnthropic []anthropic.MessageParam
	var messagesGemini []*genai.Content

	switch c.apiShape {
	case providers.APIShapeOpenAI:
		if systemMessageOpenAI != nil {
			messagesOpenAI = append(messagesOpenAI, *systemMessageOpenAI)
		}
		messagesOpenAI = append(messagesOpenAI, historicalMessagesOpenAI...)
		if msg != "" {
			messagesOpenAI = append(messagesOpenAI, openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.NewOpt(msg),
					},
				},
			})
		}
		if len(messagesOpenAI) == 0 {
			return nil, fmt.Errorf("requires msg")
		}

	case providers.APIShapeAnthropic:
		messagesAnthropic = append(messagesAnthropic, historicalMessagesAnthropic...)
		if msg != "" {
			messagesAnthropic = append(messagesAnthropic, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg)),
			)
		}
		if len(messagesAnthropic) == 0 {
			return nil, fmt.Errorf("requires msg")
		}

	case providers.APIShapeGemini:
		messagesGemini = append(messagesGemini, historicalMessagesGemini...)
		if msg != "" {
			messagesGemini = append(messagesGemini, &genai.Content{
				Parts: []*genai.Part{
					{
						Text: msg,
					},
				},
				Role: genai.RoleUser,
			})
		}
		if len(messagesGemini) == 0 {
			return nil, fmt.Errorf("requires msg")
		}

	default:
		return nil, fmt.Errorf("unsupported provider: %s", c.apiShape)
	}

	return &MessagesUnion{
		OpenAI:    messagesOpenAI,
		Anthropic: messagesAnthropic,
		Gemini:    messagesGemini,
	}, nil
}

// computeCost computes the cost for the given token usage
func (c *Client) computeCost(usage types.TokenUsage) (types.TokenCost, bool) {
	// For now, return a simple implementation
	// This would need the full implementation from run/usage.go
	return types.TokenCost{
		TotalUSD: "0.00", // Placeholder
	}, false
}

// connectToMCPServer connects to an MCP server
func (c *Client) connectToMCPServer(mcpServerSpec string) (*client.Client, error) {
	// Reuse existing logic from run/tools.go
	if mcpServerSpec == "" {
		return nil, nil
	}

	if strings.Contains(mcpServerSpec, ":") {
		// Network connection (ip:port) - not supported by mark3labs/mcp-go directly
		return nil, fmt.Errorf("network MCP connections not yet supported by this client library")
	} else {
		// CLI connection - use client package
		mcpClient, err := client.NewStdioMCPClient(mcpServerSpec, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create MCP client: %w", err)
		}
		return mcpClient, nil
	}
}

// getMCPTools gets tools from an MCP server
func (c *Client) getMCPTools(ctx context.Context, mcpClient *client.Client) ([]*tools.UnifiedTool, error) {
	// Reuse existing logic from run/tools.go
	toolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}

	var unifiedTools []*tools.UnifiedTool
	for _, tool := range toolsResponse.Tools {
		// Convert MCP tool to unified tool format
		unifiedTool := &tools.UnifiedTool{
			Name:        tool.Name,
			Description: tool.Description,
			// Parameters would need proper conversion
		}
		unifiedTools = append(unifiedTools, unifiedTool)
	}

	return unifiedTools, nil
}
