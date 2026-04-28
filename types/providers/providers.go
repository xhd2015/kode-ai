package providers

import (
	"fmt"
	"strings"

	"github.com/xhd2015/kode-ai/types"
)

// Re-export provider and API shape types from types package
type Provider = types.Provider
type APIShape = types.APIShape

const (
	ProviderAnthropic  = types.ProviderAnthropic
	ProviderGemini     = types.ProviderGemini
	ProviderOpenAI     = types.ProviderOpenAI
	ProviderMoonshot   = types.ProviderMoonshot
	ProviderDeepSeek   = types.ProviderDeepSeek
	ProviderQwen       = types.ProviderQwen
	ProviderOpenRouter = types.ProviderOpenRouter
)

const (
	APIShapeOpenAI    = types.APIShapeOpenAI
	APIShapeAnthropic = types.APIShapeAnthropic
	APIShapeGemini    = types.APIShapeGemini
)

func GetModelAPIShape(model string) (APIShape, error) {
	// Try direct lookup
	modelInfo, ok := types.AllModelInfos[model]
	if ok {
		return modelInfo.APIShape, nil
	}

	// Try underlying model (alias)
	underlyingModel := GetUnderlyingModel(model)
	if underlyingModel != model {
		underlyingModelInfo, ok := types.AllModelInfos[underlyingModel]
		if ok {
			return underlyingModelInfo.APIShape, nil
		}
	}

	if IsLikelyClaudeModel(model) {
		return APIShapeAnthropic, nil
	}

	// Model not found
	allModels := types.GetAllModels()
	allModelsPrint := make([]string, 0, len(allModels))
	for _, model := range allModels {
		allModelsPrint = append(allModelsPrint, " - "+model)
	}
	return "", fmt.Errorf("unsupported model: %s\navailable:\n%s", model, strings.Join(allModelsPrint, "\n"))
}

func GetModelProvider(model string) (Provider, error) {
	// Try direct lookup
	modelInfo, ok := types.AllModelInfos[model]
	if ok {
		return modelInfo.Provider, nil
	}

	// Try underlying model (alias)
	underlyingModel := GetUnderlyingModel(model)
	if underlyingModel != model {
		underlyingModelInfo, ok := types.AllModelInfos[underlyingModel]
		if ok {
			return underlyingModelInfo.Provider, nil
		}
	}

	if IsLikelyClaudeModel(model) {
		return ProviderAnthropic, nil
	}

	// Model not found
	allModels := types.GetAllModels()
	allModelsPrint := make([]string, 0, len(allModels))
	for _, model := range allModels {
		allModelsPrint = append(allModelsPrint, " - "+model)
	}
	return "", fmt.Errorf("unsupported model: %s\navailable:\n%s", model, strings.Join(allModelsPrint, "\n"))
}

func IsLikelyClaudeModel(model string) bool {
	model = strings.ToLower(model)
	return strings.Contains(model, "claude-") || strings.Contains(model, "claude ")
}

func NormalizeAPIShape(apiShape string) (APIShape, error) {
	switch strings.ToLower(strings.TrimSpace(apiShape)) {
	case "":
		return "", nil
	case string(APIShapeOpenAI), "open-ai":
		return APIShapeOpenAI, nil
	case string(APIShapeAnthropic), "claude":
		return APIShapeAnthropic, nil
	case string(APIShapeGemini), "google":
		return APIShapeGemini, nil
	default:
		return "", fmt.Errorf("unsupported api shape: %s (available: openai, anthropic, gemini)", apiShape)
	}
}

func DefaultProviderForAPIShape(apiShape APIShape) (Provider, error) {
	switch apiShape {
	case APIShapeOpenAI:
		return ProviderOpenAI, nil
	case APIShapeAnthropic:
		return ProviderAnthropic, nil
	case APIShapeGemini:
		return ProviderGemini, nil
	default:
		return "", fmt.Errorf("unsupported api shape: %s", apiShape)
	}
}
