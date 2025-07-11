package providers

import (
	"fmt"
	"strings"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini    Provider = "gemini"
)

func GetModelProvider(model string) (Provider, error) {
	switch model {
	case ModelGPT4_1, ModelGPT4_1_Mini, ModelGPT4o, ModelGPT4oMini, ModelGPT4oNano, ModelGPTo4Mini, ModelGPTo3:
		return ProviderOpenAI, nil
	case ModelClaude3_7Sonnet, ModelClaude3_7Sonnet_20250219, ModelClaudeSonnet4, ModelClaudeSonnet4_20250514:
		return ProviderAnthropic, nil
	default:
		allModelsPrint := make([]string, 0, len(AllModels))
		for _, model := range AllModels {
			allModelsPrint = append(allModelsPrint, " - "+model)
		}
		return "", fmt.Errorf("unsupported model: %s\navailable:\n%s", model, strings.Join(allModelsPrint, "\n"))
	}
}
