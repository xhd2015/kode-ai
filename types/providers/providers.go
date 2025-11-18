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
	switch model {
	case types.ModelGPT4_1, types.ModelGPT4_1_Mini, types.ModelGPT4o, types.ModelGPT4oMini, types.ModelGPT4oNano, types.ModelGPTo4Mini, types.ModelGPTo3, types.ModelGPT5_20250807:
		return APIShapeOpenAI, nil
	case types.ModelClaude3_7Sonnet, types.ModelClaude3_7Sonnet_20250219, types.ModelClaudeSonnet4, types.ModelClaudeSonnet4_20250514, types.ModelClaudeSonnet4_5, types.ModelClaudeSonnet4_20250929:
		return APIShapeAnthropic, nil
	case types.ModelGemini2_0_Flash, types.ModelGemini2_0_Flash_001, types.ModelGemini2_5_Pro, types.ModelGemini2_5_Pro_0605, types.ModelGemini2_5_Flash, types.ModelGemini2_5_Flash_0520:
		return APIShapeGemini, nil
		// default fallback to open ai compatiable
	case types.ModelKimiK2, types.ModelKimiK2_0711_Preview, types.ModelOpenRouterKimiK2:
		return APIShapeOpenAI, nil
	default:
		allModelsPrint := make([]string, 0, len(types.AllModels))
		for _, model := range types.AllModels {
			allModelsPrint = append(allModelsPrint, " - "+model)
		}
		return "", fmt.Errorf("unsupported model: %s\navailable:\n%s", model, strings.Join(allModelsPrint, "\n"))
	}
}

func GetModelProvider(model string) (Provider, error) {
	switch model {
	case types.ModelGPT4_1, types.ModelGPT4_1_Mini, types.ModelGPT4o, types.ModelGPT4oMini, types.ModelGPT4oNano, types.ModelGPTo4Mini, types.ModelGPTo3, types.ModelGPT5_20250807:
		return ProviderOpenAI, nil
	case types.ModelClaude3_7Sonnet, types.ModelClaude3_7Sonnet_20250219, types.ModelClaudeSonnet4, types.ModelClaudeSonnet4_20250514, types.ModelClaudeSonnet4_5, types.ModelClaudeSonnet4_20250929:
		return ProviderAnthropic, nil
	case types.ModelGemini2_0_Flash, types.ModelGemini2_0_Flash_001, types.ModelGemini2_5_Pro, types.ModelGemini2_5_Pro_0605, types.ModelGemini2_5_Flash, types.ModelGemini2_5_Flash_0520:
		return ProviderGemini, nil
		// default fallback to open ai compatiable
	case types.ModelKimiK2, types.ModelKimiK2_0711_Preview:
		return ProviderMoonshot, nil
	case types.ModelOpenRouterKimiK2:
		return ProviderOpenRouter, nil
	default:
		allModelsPrint := make([]string, 0, len(types.AllModels))
		for _, model := range types.AllModels {
			allModelsPrint = append(allModelsPrint, " - "+model)
		}
		return "", fmt.Errorf("unsupported model: %s\navailable:\n%s", model, strings.Join(allModelsPrint, "\n"))
	}
}
