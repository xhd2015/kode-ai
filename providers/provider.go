package providers

import (
	"github.com/xhd2015/kode-ai/types"
	"github.com/xhd2015/kode-ai/types/providers"
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
	return providers.GetModelAPIShape(model)
}

func GetModelProvider(model string) (Provider, error) {
	return providers.GetModelProvider(model)
}
