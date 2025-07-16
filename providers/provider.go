package providers

import (
	"fmt"
	"strings"
)

type Provider string

const (
	ProviderAnthropic  Provider = "anthropic"
	ProviderGemini     Provider = "gemini"
	ProviderOpenAI     Provider = "openai"
	ProviderMoonshot   Provider = "moonshot"
	ProviderDeepSeek   Provider = "deepseek"
	ProviderQwen       Provider = "qwen"
	ProviderOpenRouter Provider = "openrouter"
)

type APIShape string

const (
	APIShapeOpenAI    APIShape = "openai"
	APIShapeAnthropic APIShape = "anthropic"
	APIShapeGemini    APIShape = "gemini"
)

func GetModelAPIShape(model string) (APIShape, error) {
	switch model {
	case ModelGPT4_1, ModelGPT4_1_Mini, ModelGPT4o, ModelGPT4oMini, ModelGPT4oNano, ModelGPTo4Mini, ModelGPTo3:
		return APIShapeOpenAI, nil
	case ModelClaude3_7Sonnet, ModelClaude3_7Sonnet_20250219, ModelClaudeSonnet4, ModelClaudeSonnet4_20250514:
		return APIShapeAnthropic, nil
	case ModelGemini2_0_Flash, ModelGemini2_0_Flash_001, ModelGemini2_5_Pro, ModelGemini2_5_Pro_0605, ModelGemini2_5_Flash, ModelGemini2_5_Flash_0520:
		return APIShapeGemini, nil
		// default fallback to open ai compatiable
	case ModelKimiK2, ModelKimiK2_0711_Preview, ModelOpenRouterKimiK2:
		return APIShapeOpenAI, nil
	default:
		allModelsPrint := make([]string, 0, len(AllModels))
		for _, model := range AllModels {
			allModelsPrint = append(allModelsPrint, " - "+model)
		}
		return "", fmt.Errorf("unsupported model: %s\navailable:\n%s", model, strings.Join(allModelsPrint, "\n"))
	}
}

func GetModelProvider(model string) (Provider, error) {
	switch model {
	case ModelGPT4_1, ModelGPT4_1_Mini, ModelGPT4o, ModelGPT4oMini, ModelGPT4oNano, ModelGPTo4Mini, ModelGPTo3:
		return ProviderOpenAI, nil
	case ModelClaude3_7Sonnet, ModelClaude3_7Sonnet_20250219, ModelClaudeSonnet4, ModelClaudeSonnet4_20250514:
		return ProviderAnthropic, nil
	case ModelGemini2_0_Flash, ModelGemini2_0_Flash_001, ModelGemini2_5_Pro, ModelGemini2_5_Pro_0605, ModelGemini2_5_Flash, ModelGemini2_5_Flash_0520:
		return ProviderGemini, nil
		// default fallback to open ai compatiable
	case ModelKimiK2, ModelKimiK2_0711_Preview:
		return ProviderMoonshot, nil
	case ModelOpenRouterKimiK2:
		return ProviderOpenRouter, nil
	default:
		allModelsPrint := make([]string, 0, len(AllModels))
		for _, model := range AllModels {
			allModelsPrint = append(allModelsPrint, " - "+model)
		}
		return "", fmt.Errorf("unsupported model: %s\navailable:\n%s", model, strings.Join(allModelsPrint, "\n"))
	}
}
