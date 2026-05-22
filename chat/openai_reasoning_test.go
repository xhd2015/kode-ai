package chat

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/openai/openai-go"
	"github.com/xhd2015/kode-ai/types"
)

func TestOpenAIAssistantMessageParamPreservesReasoningContentWithToolCalls(t *testing.T) {
	msg := openAIAssistantMessageParam("answer", []openai.ChatCompletionMessageToolCallParam{
		{
			ID: "call_123",
			Function: openai.ChatCompletionMessageToolCallFunctionParam{
				Name:      "lookup",
				Arguments: `{"q":"price"}`,
			},
		},
	}, json.RawMessage(`"thinking trace"`))

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	obj := unmarshalObject(t, data)
	if got := obj[openAIReasoningContentField]; got != "thinking trace" {
		t.Fatalf("reasoning_content = %#v, want %q; json=%s", got, "thinking trace", data)
	}
	if got := obj["content"]; got != "answer" {
		t.Fatalf("content = %#v, want %q; json=%s", got, "answer", data)
	}
	if _, ok := obj["tool_calls"].([]any); !ok {
		t.Fatalf("tool_calls missing or wrong type; json=%s", data)
	}
}

func TestMessagesToOpenAIPreservesReasoningContent(t *testing.T) {
	msgs, _, err := Messages{
		{
			Type:             types.MsgType_Msg,
			Role:             types.Role_Assistant,
			Content:          "answer",
			ReasoningContent: "thinking trace",
		},
		{
			Type:             types.MsgType_ToolCall,
			Role:             types.Role_Assistant,
			Content:          `{"q":"price"}`,
			ReasoningContent: "tool thinking",
			ToolUseID:        "call_123",
			ToolName:         "lookup",
		},
	}.ToOpenAI(false)
	if err != nil {
		t.Fatalf("convert messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("len(msgs) = %d, want 2", len(msgs))
	}

	contentJSON, err := json.Marshal(msgs[0])
	if err != nil {
		t.Fatalf("marshal content message: %v", err)
	}
	contentObj := unmarshalObject(t, contentJSON)
	if got := contentObj[openAIReasoningContentField]; got != "thinking trace" {
		t.Fatalf("content reasoning_content = %#v, want %q; json=%s", got, "thinking trace", contentJSON)
	}

	toolCallJSON, err := json.Marshal(msgs[1])
	if err != nil {
		t.Fatalf("marshal tool call message: %v", err)
	}
	toolCallObj := unmarshalObject(t, toolCallJSON)
	if got := toolCallObj[openAIReasoningContentField]; got != "tool thinking" {
		t.Fatalf("tool reasoning_content = %#v, want %q; json=%s", got, "tool thinking", toolCallJSON)
	}
}

func TestProcessOpenAIResponsePreservesReasoningContent(t *testing.T) {
	var result openai.ChatCompletion
	err := json.Unmarshal([]byte(`{
		"id": "chatcmpl_test",
		"object": "chat.completion",
		"created": 1,
		"model": "deepseek-reasoner",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "answer",
					"reasoning_content": "thinking trace"
				},
				"finish_reason": "stop"
			}
		],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 5,
			"total_tokens": 15,
			"prompt_tokens_details": {
				"cached_tokens": 2
			}
		}
	}`), &result)
	if err != nil {
		t.Fatalf("unmarshal completion: %v", err)
	}

	client := &Client{config: Config{Model: "deepseek-reasoner"}}
	res, err := client.processOpenAIResponse(context.Background(), nil, &result, false, types.Request{}, nil)
	if err != nil {
		t.Fatalf("process response: %v", err)
	}
	if len(res.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(res.Messages))
	}
	if got := res.Messages[0].ReasoningContent; got != "thinking trace" {
		t.Fatalf("message reasoning_content = %q, want %q", got, "thinking trace")
	}
	if len(res.RespMessages) != 1 {
		t.Fatalf("len(RespMessages) = %d, want 1", len(res.RespMessages))
	}

	data, err := json.Marshal(res.RespMessages[0])
	if err != nil {
		t.Fatalf("marshal response message: %v", err)
	}
	obj := unmarshalObject(t, data)
	if got := obj[openAIReasoningContentField]; got != "thinking trace" {
		t.Fatalf("resp reasoning_content = %#v, want %q; json=%s", got, "thinking trace", data)
	}
}

func unmarshalObject(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("unmarshal json object %s: %v", data, err)
	}
	return obj
}
