package types

import (
	"reflect"
	"testing"
)

func TestAddDecimals(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected string
	}{
		{
			name:     "simple addition",
			inputs:   []string{"0.10", "0.20"},
			expected: "0.30",
		},
		{
			name:     "multiple values",
			inputs:   []string{"1.50", "2.75", "0.25"},
			expected: "4.50",
		},
		{
			name:     "small decimals",
			inputs:   []string{"0.001", "0.002", "0.003"},
			expected: "0.006",
		},
		{
			name:     "zero values",
			inputs:   []string{"0.00", "0.00", "0.00"},
			expected: "0.00",
		},
		{
			name:     "empty strings",
			inputs:   []string{"", "0.50", ""},
			expected: "0.50",
		},
		{
			name:     "single value",
			inputs:   []string{"123.45"},
			expected: "123.45",
		},
		{
			name:     "no decimal point",
			inputs:   []string{"5", "10"},
			expected: "15.00",
		},
		{
			name:     "high precision",
			inputs:   []string{"0.123456789012345", "0.987654321098765"},
			expected: "1.11111111011111",
		},
		{
			name:     "large numbers",
			inputs:   []string{"999999.99", "0.01"},
			expected: "1000000.00",
		},
		{
			name:     "empty input",
			inputs:   []string{},
			expected: "0.00",
		},
		{
			name:     "invalid numbers mixed with valid",
			inputs:   []string{"0.50", "invalid", "0.25"},
			expected: "0.75",
		},
		{
			name:     "negative numbers",
			inputs:   []string{"-0.50", "1.00"},
			expected: "1.50", // Note: negative numbers are treated as invalid and skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addDecimals(tt.inputs...)
			if result != tt.expected {
				t.Errorf("addDecimals(%v) = %s, want %s", tt.inputs, result, tt.expected)
			}
		})
	}
}

func TestTokenUsage(t *testing.T) {
	usage1 := TokenUsage{
		Input:  100,
		Output: 50,
		Total:  150,
		InputBreakdown: TokenUsageInputBreakdown{
			CacheRead:    10,
			CacheWrite:   5,
			NonCacheRead: 85,
		},
		OutputBreakdown: TokenUsageOutputBreakdown{
			CacheOutput: 2,
		},
	}

	usage2 := TokenUsage{
		Input:  200,
		Output: 100,
		Total:  300,
		InputBreakdown: TokenUsageInputBreakdown{
			CacheRead:    20,
			CacheWrite:   10,
			NonCacheRead: 170,
		},
		OutputBreakdown: TokenUsageOutputBreakdown{
			CacheOutput: 3,
		},
	}

	combined := usage1.Add(usage2)

	if combined.Input != 300 {
		t.Errorf("Expected combined input 300, got %d", combined.Input)
	}
	if combined.Output != 150 {
		t.Errorf("Expected combined output 150, got %d", combined.Output)
	}
	if combined.Total != 450 {
		t.Errorf("Expected combined total 450, got %d", combined.Total)
	}
	if combined.InputBreakdown.CacheRead != 30 {
		t.Errorf("Expected combined cache read 30, got %d", combined.InputBreakdown.CacheRead)
	}
	if combined.InputBreakdown.CacheWrite != 15 {
		t.Errorf("Expected combined cache write 15, got %d", combined.InputBreakdown.CacheWrite)
	}
	if combined.InputBreakdown.NonCacheRead != 255 {
		t.Errorf("Expected combined non-cache read 255, got %d", combined.InputBreakdown.NonCacheRead)
	}
	if combined.OutputBreakdown.CacheOutput != 5 {
		t.Errorf("Expected combined cache output 5, got %d", combined.OutputBreakdown.CacheOutput)
	}
}

func TestTokenCost(t *testing.T) {
	cost1 := TokenCost{
		InputUSD:  "0.10",
		OutputUSD: "0.05",
		TotalUSD:  "0.15",
		InputBreakdown: TokenCostInputBreakdown{
			CacheWriteUSD:   "0.02",
			CacheReadUSD:    "0.01",
			NonCacheReadUSD: "0.07",
		},
	}

	cost2 := TokenCost{
		InputUSD:  "0.20",
		OutputUSD: "0.10",
		TotalUSD:  "0.30",
		InputBreakdown: TokenCostInputBreakdown{
			CacheWriteUSD:   "0.04",
			CacheReadUSD:    "0.02",
			NonCacheReadUSD: "0.14",
		},
	}

	combined := cost1.Add(cost2)

	expectedInputUSD := "0.30"
	if combined.InputUSD != expectedInputUSD {
		t.Errorf("Expected combined input USD %s, got %s", expectedInputUSD, combined.InputUSD)
	}

	expectedOutputUSD := "0.15"
	if combined.OutputUSD != expectedOutputUSD {
		t.Errorf("Expected combined output USD %s, got %s", expectedOutputUSD, combined.OutputUSD)
	}

	expectedTotalUSD := "0.45"
	if combined.TotalUSD != expectedTotalUSD {
		t.Errorf("Expected combined total USD %s, got %s", expectedTotalUSD, combined.TotalUSD)
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Type:    MsgType_Msg,
		Role:    Role_User,
		Content: "Hello, world!",
		Model:   "test-model",
	}

	if msg.Type != MsgType_Msg {
		t.Errorf("Expected message type %s, got %s", MsgType_Msg, msg.Type)
	}

	if msg.Role != Role_User {
		t.Errorf("Expected role %s, got %s", Role_User, msg.Role)
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got %s", msg.Content)
	}
}

func TestToolCall(t *testing.T) {
	toolCall := ToolCall{
		ID:   "test-id",
		Name: "test-tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
			"param2": 42,
		},
		RawArgs: `{"param1":"value1","param2":42}`,
	}

	if toolCall.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", toolCall.ID)
	}

	if toolCall.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got %s", toolCall.Name)
	}

	if len(toolCall.Arguments) != 2 {
		t.Errorf("Expected 2 arguments, got %d", len(toolCall.Arguments))
	}

	if toolCall.Arguments["param1"] != "value1" {
		t.Errorf("Expected param1 'value1', got %v", toolCall.Arguments["param1"])
	}

	if toolCall.Arguments["param2"] != 42 {
		t.Errorf("Expected param2 42, got %v", toolCall.Arguments["param2"])
	}
}

func TestParseMetadata(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		metadata  map[string]interface{}
		expected  interface{}
	}{
		{
			name:      "ToolCallRequest",
			eventType: EventTypeToolCallRequest,
			metadata: map[string]interface{}{
				"name":                "test_tool",
				"arguments":           map[string]interface{}{"param": "value"},
				"default_working_dir": "/tmp",
			},
			expected: ToolCallRequestMetadata{
				Name:              "test_tool",
				Arguments:         map[string]interface{}{"param": "value"},
				DefaultWorkingDir: "/tmp",
			},
		},
		{
			name:      "ToolCallResponse",
			eventType: EventTypeToolCallResponse,
			metadata: map[string]interface{}{
				"ok":     true,
				"result": "success",
			},
			expected: StreamResponseToolMetadata{
				OK:     true,
				Result: "success",
			},
		},
		{
			name:      "CacheInfo",
			eventType: EventTypeCacheInfo,
			metadata: map[string]interface{}{
				"cache_enabled": true,
				"model":         "gpt-4o",
			},
			expected: CacheInfoMetadata{
				CacheEnabled: true,
				Model:        "gpt-4o",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMetadata(tt.eventType, tt.metadata)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseMetadata() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEventParsedMetadata(t *testing.T) {
	event := Event{
		Type: EventTypeToolCallRequest,
		Metadata: map[string]interface{}{
			"name":      "test_tool",
			"arguments": map[string]interface{}{"param": "value"},
		},
	}

	parsed := event.ParsedMetadata()
	expected := ToolCallRequestMetadata{
		Name:      "test_tool",
		Arguments: map[string]interface{}{"param": "value"},
	}

	if !reflect.DeepEqual(parsed, expected) {
		t.Errorf("Event.ParsedMetadata() = %v, want %v", parsed, expected)
	}
}

func TestToMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata interface{ ToMetadata() map[string]interface{} }
		expected map[string]interface{}
	}{
		{
			name: "ToolCallRequestMetadata",
			metadata: ToolCallRequestMetadata{
				Name:              "test_tool",
				Arguments:         map[string]interface{}{"param": "value"},
				DefaultWorkingDir: "/tmp",
			},
			expected: map[string]interface{}{
				"name":                "test_tool",
				"arguments":           map[string]interface{}{"param": "value"},
				"default_working_dir": "/tmp",
			},
		},
		{
			name: "ToolCallResponseMetadata",
			metadata: StreamResponseToolMetadata{
				OK:     true,
				Result: "success",
			},
			expected: map[string]interface{}{
				"ok":     true,
				"result": "success",
			},
		},
		{
			name: "ToolCallMetadata",
			metadata: ToolCallMetadata{
				ToolName: "test_tool",
			},
			expected: map[string]interface{}{
				"tool_name": "test_tool",
			},
		},
		{
			name: "CacheInfoMetadata",
			metadata: CacheInfoMetadata{
				CacheEnabled: true,
				Model:        "gpt-4o",
			},
			expected: map[string]interface{}{
				"cache_enabled": true,
				"model":         "gpt-4o",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.ToMetadata()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ToMetadata() = %v, want %v", result, tt.expected)
			}
		})
	}
}
