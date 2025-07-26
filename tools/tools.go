package tools

import (
	"encoding/json"
	"fmt"

	_ "embed"

	"github.com/anthropics/anthropic-sdk-go"
	anthropic_params "github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/openai/openai-go"
	openai_params "github.com/openai/openai-go/packages/param"
	"github.com/xhd2015/kode-ai/types"
	"github.com/xhd2015/llm-tools/jsonschema"
	"google.golang.org/genai"
)

//go:embed example_tool.json
var ExampleTool string

// Parse parses a unified tool definition from JSON bytes
func Parse(data []byte) (*UnifiedTool, error) {
	var tool types.UnifiedTool
	if err := json.Unmarshal(data, &tool); err != nil {
		return nil, fmt.Errorf("parse unified tool: %w", err)
	}
	if tool.Format == "" || tool.Format == "unified" {
		return ptrTool(&tool), nil
	}
	switch tool.Format {
	case "openai":
		openai, err := ParseToolForOpenAI(data)
		if err != nil {
			return nil, err
		}
		return OpenAIToUnified(openai)
	case "anthropic":
		ant, err := ParseToolForAnthropic(data)
		if err != nil {
			return nil, err
		}
		return AnthropicToUnified(ant)
	default:
		return nil, fmt.Errorf("unsupported format: %s", tool.Format)
	}
}

// ParseToolForOpenAI parses OpenAI tool format directly
func ParseToolForOpenAI(data []byte) (*openai.ChatCompletionToolParam, error) {
	var tool openai.ChatCompletionToolParam
	if err := json.Unmarshal(data, &tool); err != nil {
		return &openai.ChatCompletionToolParam{}, fmt.Errorf("parse OpenAI tool: %w", err)
	}
	return &tool, nil
}

// ParseToolForAnthropic parses Anthropic tool format directly
func ParseToolForAnthropic(data []byte) (*anthropic.ToolParam, error) {
	var tool anthropic.ToolParam
	if err := json.Unmarshal(data, &tool); err != nil {
		return &anthropic.ToolParam{}, fmt.Errorf("parse Anthropic tool: %w", err)
	}
	return &tool, nil
}

type UnifiedTool types.UnifiedTool

// ToOpenAI converts UnifiedTool to OpenAI ChatCompletionToolParam
func (t *UnifiedTool) ToOpenAI() (*openai.ChatCompletionToolParam, error) {
	return ConvertUnifiedToOpenAI(t)
}

// ToAnthropic converts UnifiedTool to Anthropic ToolParam
func (t *UnifiedTool) ToAnthropic() (*anthropic.ToolParam, error) {
	return ConvertUnifiedToAnthropic(t)
}

// ToGemini converts UnifiedTool to Gemini ToolParam
func (t *UnifiedTool) ToGemini() (*genai.FunctionDeclaration, error) {
	return ConvertUnifiedToGemini(t)
}

func ptrTool(t *types.UnifiedTool) *UnifiedTool {
	return (*UnifiedTool)(t)
}

// ConvertAnthropicToOpenAI converts Anthropic tool format to OpenAI format
func ConvertAnthropicToOpenAI(data []byte) (openai.ChatCompletionToolParam, error) {
	var anthropicTool struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		InputSchema map[string]interface{} `json:"input_schema"`
	}

	if err := json.Unmarshal(data, &anthropicTool); err != nil {
		return openai.ChatCompletionToolParam{}, fmt.Errorf("parse Anthropic tool: %w", err)
	}

	// Create OpenAI format
	openaiJSON := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        anthropicTool.Name,
			"description": anthropicTool.Description,
			"parameters":  anthropicTool.InputSchema,
		},
	}

	openaiData, err := json.Marshal(openaiJSON)
	if err != nil {
		return openai.ChatCompletionToolParam{}, fmt.Errorf("marshal to OpenAI format: %w", err)
	}

	var tool openai.ChatCompletionToolParam
	if err := json.Unmarshal(openaiData, &tool); err != nil {
		return openai.ChatCompletionToolParam{}, fmt.Errorf("unmarshal to OpenAI tool: %w", err)
	}

	return tool, nil
}

// ConvertOpenAIToAnthropic converts OpenAI tool format to Anthropic format
func ConvertOpenAIToAnthropic(data []byte) (anthropic.ToolParam, error) {
	var openaiTool struct {
		Function struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			Parameters  map[string]interface{} `json:"parameters"`
		} `json:"function"`
	}

	if err := json.Unmarshal(data, &openaiTool); err != nil {
		return anthropic.ToolParam{}, fmt.Errorf("parse OpenAI tool: %w", err)
	}

	// Create Anthropic format
	anthropicJSON := map[string]interface{}{
		"name":         openaiTool.Function.Name,
		"description":  openaiTool.Function.Description,
		"input_schema": openaiTool.Function.Parameters,
	}

	anthropicData, err := json.Marshal(anthropicJSON)
	if err != nil {
		return anthropic.ToolParam{}, fmt.Errorf("marshal to Anthropic format: %w", err)
	}

	var tool anthropic.ToolParam
	if err := json.Unmarshal(anthropicData, &tool); err != nil {
		return anthropic.ToolParam{}, fmt.Errorf("unmarshal to Anthropic tool: %w", err)
	}

	return tool, nil
}

// ConvertUnifiedToOpenAI converts UnifiedTool to OpenAI format
func ConvertUnifiedToOpenAI(unifiedTool *UnifiedTool) (*openai.ChatCompletionToolParam, error) {
	param := unifiedTool.Parameters

	return &openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        unifiedTool.Name,
			Description: openai_params.NewOpt(unifiedTool.Description),
			Parameters:  JsonschemaToMap(param),
		},
	}, nil
}

func JsonschemaToMap(schema *jsonschema.JsonSchema) map[string]any {
	if schema == nil {
		return nil
	}
	paramsMap := map[string]any{
		"type": schema.Type,
	}
	if schema.Properties != nil {
		paramsMap["properties"] = schema.Properties
	}
	if schema.Description != "" {
		paramsMap["description"] = schema.Description
	}
	if schema.Items != nil {
		paramsMap["items"] = schema.Items
	}
	if len(schema.Required) > 0 {
		paramsMap["required"] = schema.Required
	}
	if schema.Default != nil {
		paramsMap["default"] = schema.Default
	}
	return paramsMap
}

// ConvertUnifiedToAnthropic converts UnifiedTool to Anthropic format
func ConvertUnifiedToAnthropic(unifiedTool *UnifiedTool) (*anthropic.ToolParam, error) {
	params := convertParameter(unifiedTool.Parameters)
	if params == nil {
		params = &anthropic.ToolInputSchemaParam{}
	}
	return &anthropic.ToolParam{
		Name:        unifiedTool.Name,
		Description: anthropic_params.NewOpt(unifiedTool.Description),
		InputSchema: *params,
		Type:        "custom",
	}, nil
}

func ConvertUnifiedToGemini(unifiedTool *UnifiedTool) (*genai.FunctionDeclaration, error) {
	return &genai.FunctionDeclaration{
		// Behavior:    genai.BehaviorBlocking,
		Description: unifiedTool.Description,
		Name:        unifiedTool.Name,
		Parameters:  toGeminiSchema(unifiedTool.Parameters),
	}, nil
}

func toGeminiSchema(jschema *jsonschema.JsonSchema) *genai.Schema {
	if jschema == nil {
		return nil
	}
	return &genai.Schema{
		Type:        convertToGeminiType(jschema.Type),
		Description: jschema.Description,
		Properties:  toGeminiSchemaMap(jschema.Properties),
		Items:       toGeminiSchema(jschema.Items),
		Required:    jschema.Required,
		Default:     jschema.Default,
	}
}
func toGeminiSchemaMap(jschema map[string]*jsonschema.JsonSchema) map[string]*genai.Schema {
	if jschema == nil {
		return nil
	}
	schemaMap := make(map[string]*genai.Schema)
	for k, v := range jschema {
		schemaMap[k] = toGeminiSchema(v)
	}
	return schemaMap
}

func convertToGeminiType(t jsonschema.ParamType) genai.Type {
	switch t {
	case jsonschema.ParamTypeObject:
		return genai.TypeObject
	case jsonschema.ParamTypeString:
		return genai.TypeString
	case jsonschema.ParamTypeNumber:
		return genai.TypeNumber
	case jsonschema.ParamTypeBoolean:
		return genai.TypeBoolean
	case jsonschema.ParamTypeArray:
		return genai.TypeArray
	default:
		return genai.Type(t)
	}
}

func OpenAIToUnified(tool *openai.ChatCompletionToolParam) (*UnifiedTool, error) {
	params, err := convertAnyToSchemaProperies(tool.Function.Parameters)
	if err != nil {
		return nil, err
	}
	var required []string
	requiredParams, ok := tool.Function.Parameters["required"]
	if !ok {
		if list, ok := requiredParams.([]string); ok {
			required = list
		}
	}
	return &UnifiedTool{
		Name:        tool.Function.Name,
		Description: tool.Function.Description.Value,
		Parameters: &jsonschema.JsonSchema{
			Type:        jsonschema.ParamTypeObject,
			Properties:  params,
			Description: tool.Function.Description.Value,
			Required:    required,
		},
	}, nil
}

func AnthropicToUnified(tool *anthropic.ToolParam) (*UnifiedTool, error) {
	properties, err := convertAnyToSchemaProperies(tool.InputSchema.Properties)
	if err != nil {
		return nil, err
	}
	return &UnifiedTool{
		Name:        tool.Name,
		Description: tool.Description.Value,
		Parameters: &jsonschema.JsonSchema{
			Type:        jsonschema.ParamTypeObject,
			Properties:  properties,
			Description: tool.Description.Value,
			Required:    tool.InputSchema.Required,
		},
	}, nil
}

func convertAnyToSchemaProperies(v any) (map[string]*jsonschema.JsonSchema, error) {
	if v == nil {
		return nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var schema map[string]*jsonschema.JsonSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	return schema, nil
}

func convertParameter(parameters *jsonschema.JsonSchema) *anthropic.ToolInputSchemaParam {
	if parameters == nil {
		return nil
	}
	if parameters.Type != "object" {
		panic(fmt.Errorf("expect object type, but got %s", parameters.Type))
	}
	return &anthropic.ToolInputSchemaParam{
		Properties: parameters.Properties,
		Required:   parameters.Required,
	}
}
