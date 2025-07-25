package run

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/xhd2015/kode-ai/internal/jsondecode"
	"github.com/xhd2015/kode-ai/types"
	"google.golang.org/genai"
)

// Messages represents a slice of unified messages with conversion methods
type Messages []types.Message

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

// ToOpenAILegacy converts unified messages to OpenAI format (legacy method)
func (messages Messages) ToOpenAILegacy() []openai.ChatCompletionMessageParamUnion {
	var openaiMessages []openai.ChatCompletionMessageParamUnion

	for _, msg := range messages {
		switch msg.Role {
		case types.Role_User:
			userMsg := openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.NewOpt(msg.Content),
					},
				},
			}
			openaiMessages = append(openaiMessages, userMsg)
		case types.Role_Assistant:
			assistantMsg := openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.NewOpt(msg.Content),
					},
				},
			}
			openaiMessages = append(openaiMessages, assistantMsg)
		case types.Role_System:
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
			toolResult := anthropic.NewToolResultBlock(msg.ToolUseID)
			toolResult.OfToolResult.Content = append(toolResult.OfToolResult.Content, anthropic.ToolResultBlockParamContentUnion{
				OfText: &anthropic.TextBlockParam{
					Text: msg.Content,
				},
			})
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
		case types.MsgType_ToolResult:
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
