package tools

import (
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go"
	"google.golang.org/genai"
)

type UnifiedTools []*UnifiedTool

func ParseSchemas(toolFiles []string, toolJSONs []string) (UnifiedTools, error) {
	toolsFromFiles, err := ParseSchemaFiles(toolFiles)
	if err != nil {
		return nil, err
	}
	toolsFromJSONs, err := ParseSchemaData(toolJSONs)
	if err != nil {
		return nil, err
	}
	tools := append(toolsFromFiles, toolsFromJSONs...)
	return tools, nil
}

func ParseSchemaFiles(toolFiles []string) (UnifiedTools, error) {
	var unifiedTools UnifiedTools
	for i, toolFile := range toolFiles {
		toolData, err := os.ReadFile(toolFile)
		if err != nil {
			return nil, err
		}

		tool, err := Parse(toolData)
		if err != nil {
			return nil, fmt.Errorf("parse tool schema at %d %s: %w", i, toolFile, err)
		}
		unifiedTools = append(unifiedTools, tool)
	}
	return unifiedTools, nil
}

func ParseSchemaData(toolJSONs []string) (UnifiedTools, error) {
	var unifiedTools UnifiedTools
	for i, toolData := range toolJSONs {
		tool, err := Parse([]byte(toolData))
		if err != nil {
			return nil, fmt.Errorf("parse tool schema at %d: %w", i, err)
		}
		unifiedTools = append(unifiedTools, tool)
	}
	return unifiedTools, nil
}

func (c UnifiedTools) ToOpenAI() ([]openai.ChatCompletionToolParam, error) {
	var openaiTools []openai.ChatCompletionToolParam
	for _, tool := range c {
		openaiTool, err := tool.ToOpenAI()
		if err != nil {
			return nil, err
		}
		openaiTools = append(openaiTools, *openaiTool)
	}
	return openaiTools, nil
}

func (c UnifiedTools) ToAnthropic() ([]anthropic.ToolUnionParam, error) {
	var anthropicTools []anthropic.ToolUnionParam
	for _, tool := range c {
		anthropicTool, err := tool.ToAnthropic()
		if err != nil {
			return nil, err
		}
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: anthropicTool,
		})
	}
	return anthropicTools, nil
}

func (c UnifiedTools) ToGemini() ([]*genai.Tool, error) {
	var genaiTools []*genai.FunctionDeclaration
	for _, tool := range c {
		genaiTool, err := tool.ToGemini()
		if err != nil {
			return nil, err
		}
		genaiTools = append(genaiTools, genaiTool)
	}
	if len(genaiTools) == 0 {
		return nil, nil
	}
	return []*genai.Tool{
		{
			FunctionDeclarations: genaiTools,
		},
	}, nil
}
