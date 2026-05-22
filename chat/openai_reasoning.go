package chat

import (
	"encoding/json"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

const openAIReasoningContentField = "reasoning_content"

func openAIReasoningContentRaw(msg openai.ChatCompletionMessage) json.RawMessage {
	if msg.JSON.ExtraFields == nil {
		return nil
	}
	field, ok := msg.JSON.ExtraFields[openAIReasoningContentField]
	if !ok {
		return nil
	}
	raw := field.Raw()
	if raw == "" || raw == "null" || !json.Valid([]byte(raw)) {
		return nil
	}
	return json.RawMessage(raw)
}

func openAIReasoningContentString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func setOpenAIReasoningContentRaw(msg *openai.ChatCompletionAssistantMessageParam, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	msg.SetExtraFields(map[string]any{
		openAIReasoningContentField: raw,
	})
}

func setOpenAIReasoningContentString(msg *openai.ChatCompletionAssistantMessageParam, reasoningContent string) {
	if reasoningContent == "" {
		return
	}
	raw, err := json.Marshal(reasoningContent)
	if err != nil {
		return
	}
	setOpenAIReasoningContentRaw(msg, raw)
}

func openAIAssistantMessageParam(content string, toolCalls []openai.ChatCompletionMessageToolCallParam, reasoningContent json.RawMessage) openai.ChatCompletionMessageParamUnion {
	assistant := &openai.ChatCompletionAssistantMessageParam{}
	if content != "" {
		assistant.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
			OfString: param.NewOpt(content),
		}
	}
	if len(toolCalls) > 0 {
		assistant.ToolCalls = toolCalls
	}
	setOpenAIReasoningContentRaw(assistant, reasoningContent)
	return openai.ChatCompletionMessageParamUnion{
		OfAssistant: assistant,
	}
}
