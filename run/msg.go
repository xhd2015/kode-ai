package run

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/xhd2015/kode-ai/internal/jsondecode"
	"google.golang.org/genai"
)

type MsgType string

const (
	MsgType_Msg        = "msg"
	MsgType_ToolCall   = "tool_call"
	MsgType_ToolResult = "tool_result"
	MsgType_TokenUsage = "token_usage"
	MsgType_StopReason = "stop_reason" // anthropic specific
)

type Role string

const (
	Role_User      = "user"
	Role_Assistant = "assistant"
	Role_System    = "system"
)

// Message represents a message in the chat record
type Message struct {
	Type    MsgType `json:"type"`
	Time    string  `json:"time"`
	Role    Role    `json:"role"`
	Model   string  `json:"model"`
	Content string  `json:"content"`

	ToolUseID  string      `json:"tool_use_id,omitempty"`
	ToolName   string      `json:"tool_name,omitempty"`
	TokenUsage *TokenUsage `json:"token_usage,omitempty"`
}

// Messages represents a slice of unified messages with conversion methods
type Messages []Message

// ToOpenAI converts unified messages to OpenAI format
func (messages Messages) ToOpenAI2() []openai.ChatCompletionMessageParamUnion {
	var openaiMessages []openai.ChatCompletionMessageParamUnion

	for _, msg := range messages {
		switch msg.Role {
		case Role_User:
			userMsg := openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.NewOpt(msg.Content),
					},
				},
			}
			openaiMessages = append(openaiMessages, userMsg)
		case Role_Assistant:
			assistantMsg := openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.NewOpt(msg.Content),
					},
				},
			}
			openaiMessages = append(openaiMessages, assistantMsg)
		case Role_System:
			systemMsg := openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: param.NewOpt(msg.Content),
					},
				},
			}
			openaiMessages = append(openaiMessages, systemMsg)
		}
	}

	return openaiMessages
}

// ToAnthropic converts unified messages to Anthropic format
func (messages Messages) ToOpenAI(keepSystemPrompts bool) (msgs []openai.ChatCompletionMessageParamUnion, systemPrompts []string, err error) {
	for _, msg := range messages {
		var msgUnion openai.ChatCompletionMessageParamUnion
		switch msg.Type {
		case MsgType_ToolCall:
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
		case MsgType_ToolResult:
			msgUnion.OfTool = &openai.ChatCompletionToolMessageParam{
				ToolCallID: msg.ToolUseID,
				Content: openai.ChatCompletionToolMessageParamContentUnion{
					OfString: param.NewOpt(msg.Content),
				},
			}
		case MsgType_Msg:
			switch msg.Role {
			case Role_User:
				msgUnion.OfUser = &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.NewOpt(msg.Content),
					},
				}
			case Role_Assistant:
				msgUnion.OfAssistant = &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.NewOpt(msg.Content),
					},
				}
			case Role_System:
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
		case MsgType_ToolCall:
			m := json.RawMessage(msg.Content)
			toolUse := anthropic.NewToolUseBlock(msg.ToolUseID, m, msg.ToolName)
			blocks = append(blocks, toolUse)
		case MsgType_ToolResult:
			toolResult := anthropic.NewToolResultBlock(msg.ToolUseID)
			toolResult.OfToolResult.Content = append(toolResult.OfToolResult.Content, anthropic.ToolResultBlockParamContentUnion{
				OfText: &anthropic.TextBlockParam{
					Text: msg.Content,
				},
			})
			blocks = append(blocks, toolResult)
		case MsgType_Msg:
			textBlock := anthropic.NewTextBlock(msg.Content)
			blocks = append(blocks, textBlock)
		default:
			continue
		}

		switch msg.Role {
		case Role_User:
			msgRole = anthropic.MessageParamRoleUser
		case Role_Assistant:
			msgRole = anthropic.MessageParamRoleAssistant
		case Role_System:
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

func (messages Messages) ToGemini() (msgs []*genai.Content, systemPrompts []string, err error) {
	for _, msg := range messages {
		if msg.Role == Role_System {
			systemPrompts = append(systemPrompts, msg.Content)
			continue
		}

		var parts []*genai.Part
		switch msg.Type {
		case MsgType_ToolCall:
			var args map[string]any
			err := jsondecode.UnmarshalSafe([]byte(msg.Content), &args)
			if err != nil {
				return nil, nil, err
			}

			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					// NOTE: Gemini tool use id is only client
					// side, don't put them back or the API will
					// report error saying
					// unrecognized "id" param
					//
					// ID:   msg.ToolUseID,
					Name: msg.ToolName,
					Args: args,
				},
			})
		case MsgType_ToolResult:
			var resp map[string]any
			err := jsondecode.UnmarshalSafe([]byte(msg.Content), &resp)
			if err != nil {
				return nil, nil, err
			}

			parts = append(parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					// ID:       msg.ToolUseID,
					Name:     msg.ToolName,
					Response: resp,
				},
			})
		case MsgType_Msg:
			parts = append(parts, &genai.Part{
				Text: msg.Content,
			})
		}

		var msgRole string
		switch msg.Role {
		case Role_User:
			msgRole = genai.RoleUser
		case Role_Assistant:
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
