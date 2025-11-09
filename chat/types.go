package chat

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/xhd2015/kode-ai/internal/jsondecode"
	"github.com/xhd2015/kode-ai/providers"
	"github.com/xhd2015/kode-ai/types"
	"google.golang.org/genai"
)

// Messages is a local wrapper for conversion methods
type Messages []types.Message

// Config represents the client configuration with provider-specific fields
type Config struct {
	Model    string             // Required: Model name (e.g., "claude-3-7-sonnet")
	Token    string             // Required: API token
	BaseURL  string             // Optional: Custom API base URL
	Provider providers.Provider // Optional: Auto-detected from model if not specified
	LogLevel types.LogLevel     // Optional: None, Request, Response, Debug

	Logger types.Logger
}

// Provider-specific message unions for internal use
type ClientUnion struct {
	OpenAI    *openai.Client
	Anthropic *anthropic.Client
	Gemini    *genai.Client
}

type MessageHistoryUnion struct {
	FullHistory   Messages
	SystemPrompts []string

	OpenAI    []openai.ChatCompletionMessageParamUnion
	Anthropic []anthropic.MessageParam
	Gemini    []*genai.Content
}

type MessagesUnion struct {
	OpenAI    []openai.ChatCompletionMessageParamUnion
	Anthropic []anthropic.MessageParam
	Gemini    []*genai.Content
}

// Convert Messages to provider-specific formats (reuse existing logic)

// ToOpenAI converts unified messages to OpenAI format
func (messages Messages) ToOpenAI(keepSystemPrompts bool) (msgs []openai.ChatCompletionMessageParamUnion, systemPrompts []string, err error) {
	for _, msg := range messages {
		var msgUnion openai.ChatCompletionMessageParamUnion
		switch msg.Type {
		case types.MsgType_ToolCall:
			msgUnion.OfAssistant = &openai.ChatCompletionAssistantMessageParam{
				ToolCalls: []openai.ChatCompletionMessageToolCallParam{
					{
						ID: msg.ToolUseID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      msg.ToolName,
							Arguments: msg.Content,
						},
					},
				},
			}
		case types.MsgType_ToolResult:
			msgUnion.OfTool = &openai.ChatCompletionToolMessageParam{
				ToolCallID: msg.ToolUseID,
				Content: openai.ChatCompletionToolMessageParamContentUnion{
					OfString: param.NewOpt(msg.Content),
				},
			}
		case types.MsgType_Msg:
			switch msg.Role {
			case types.Role_User:
				msgUnion.OfUser = &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.NewOpt(msg.Content),
					},
				}
			case types.Role_Assistant:
				msgUnion.OfAssistant = &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.NewOpt(msg.Content),
					},
				}
			case types.Role_System:
				systemPrompts = append(systemPrompts, msg.Content)
				if keepSystemPrompts {
					msgUnion.OfSystem = &openai.ChatCompletionSystemMessageParam{
						Content: openai.ChatCompletionSystemMessageParamContentUnion{
							OfString: param.NewOpt(msg.Content),
						},
					}
				} else {
					continue
				}
			default:
				continue
			}
		default:
			continue
		}

		msgs = append(msgs, msgUnion)
	}

	return msgs, systemPrompts, nil
}

// ToAnthropic converts unified messages to Anthropic format
func (messages Messages) ToAnthropic() (msgs []anthropic.MessageParam, systemPrompts []string, err error) {
	for _, msg := range messages {
		var blocks []anthropic.ContentBlockParamUnion
		var msgRole anthropic.MessageParamRole
		switch msg.Type {
		case types.MsgType_ToolCall:
			m := json.RawMessage(msg.Content)
			toolUse := anthropic.NewToolUseBlock(msg.ToolUseID, m, msg.ToolName)
			blocks = append(blocks, toolUse)
		case types.MsgType_ToolResult:
			toolResult := anthropic.NewToolResultBlock(msg.ToolUseID, msg.Content, false)
			blocks = append(blocks, toolResult)
		case types.MsgType_Msg:
			textBlock := anthropic.NewTextBlock(msg.Content)
			blocks = append(blocks, textBlock)
		default:
			continue
		}

		switch msg.Role {
		case types.Role_User:
			msgRole = anthropic.MessageParamRoleUser
		case types.Role_Assistant:
			msgRole = anthropic.MessageParamRoleAssistant
		case types.Role_System:
			systemPrompts = append(systemPrompts, msg.Content)
			continue
		default:
			continue
		}

		msgs = append(msgs, anthropic.MessageParam{
			Role:    msgRole,
			Content: blocks,
		})
	}

	return msgs, systemPrompts, nil
}

// ToGemini converts unified messages to Gemini format
func (messages Messages) ToGemini() (msgs []*genai.Content, systemPrompts []string, err error) {
	for _, msg := range messages {
		if msg.Role == types.Role_System {
			systemPrompts = append(systemPrompts, msg.Content)
			continue
		}

		var parts []*genai.Part
		switch msg.Type {
		case types.MsgType_ToolCall:
			var args map[string]any
			err := jsondecode.UnmarshalSafe([]byte(msg.Content), &args)
			if err != nil {
				return nil, nil, err
			}

			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: msg.ToolName,
					Args: args,
				},
			})
		case types.MsgType_ToolResult:
			var resp map[string]any
			err := jsondecode.UnmarshalSafe([]byte(msg.Content), &resp)
			if err != nil {
				return nil, nil, err
			}

			parts = append(parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     msg.ToolName,
					Response: resp,
				},
			})
		case types.MsgType_Msg:
			parts = append(parts, &genai.Part{
				Text: msg.Content,
			})
		}

		var msgRole string
		switch msg.Role {
		case types.Role_User:
			msgRole = genai.RoleUser
		case types.Role_Assistant:
			msgRole = genai.RoleModel
		default:
			continue
		}

		if len(parts) == 0 {
			continue
		}

		msgs = append(msgs, &genai.Content{
			Parts: parts,
			Role:  msgRole,
		})
	}

	return msgs, systemPrompts, nil
}
