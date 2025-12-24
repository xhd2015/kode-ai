package types

import "github.com/xhd2015/kode-ai/types/models"

// Provider represents the AI provider (re-exported from models package)
type Provider = models.Provider

// APIShape represents the API shape/format (re-exported from models package)
type APIShape = models.APIShape

// Model constants
const (
	// OpenAI
	ModelGPT4o           = "gpt-4o"      // $0.15 / 1M
	ModelGPT4oMini       = "gpt-4o-mini" // $0.15 / 1M
	ModelGPT4oNano       = "gpt-4o-nano" // $0.15 / 1M
	ModelGPTo4Mini       = "o4-mini"     // 10x gpt-4o-mini
	ModelGPTo3Mini       = "o3-mini"     // 10x gpt-4o-mini
	ModelGPT4_1          = "gpt-4.1"     // $2
	ModelGPT4_1_Mini     = "gpt-4.1-mini"
	ModelGPTo3           = "o3"               // $2
	ModelGPT5_20250807   = "gpt-5-2025-08-07" // $1.25
	ModelGPT5_2_20251211 = "gpt-5.2-2025-12-11"

	// Anthropic
	ModelClaude3_7Sonnet          = "claude-3-7-sonnet" // $3
	ModelClaude3_7Sonnet_20250219 = "claude-3-7-sonnet@20250219"

	ModelClaudeSonnet4          = "claude-sonnet-4" // $3
	ModelClaudeSonnet4_20250514 = "claude-sonnet-4@20250514"

	ModelClaudeSonnet4_5        = "claude-sonnet-4-5" // $3
	ModelClaudeSonnet4_20250929 = "claude-sonnet-4-5@20250929"

	// Gemini
	ModelGemini2_0_Flash      = "gemini-2.0-flash"
	ModelGemini2_0_Flash_001  = "gemini-2.0-flash-001"
	ModelGemini2_5_Pro        = "gemini-2.5-pro"
	ModelGemini2_5_Pro_0605   = "gemini-2.5-pro-preview-06-05"
	ModelGemini2_5_Flash      = "gemini-2.5-flash"
	ModelGemini2_5_Flash_0520 = "gemini-2.5-flash-preview-05-20"
	ModelGemini3ProPreview    = "gemini-3-pro-preview"

	// kimi
	ModelKimiK2              = "kimi-k2"
	ModelKimiK2_0711_Preview = "kimi-k2-0711-preview"
	ModelOpenRouterKimiK2    = "moonshotai/kimi-k2"

	ModelDeepSeekR1          = "DeepSeek-R1"
	ModelQwen25VL72BInstruct = "Qwen2.5-VL-72B-Instruct"
)

// GetAllModels returns all supported model names
func GetAllModels() []string {
	models := make([]string, 0, len(AllModelInfos))
	for modelName := range AllModelInfos {
		models = append(models, modelName)
	}
	return models
}

// Re-export Provider constants from models package
const (
	ProviderAnthropic  = models.ProviderAnthropic
	ProviderGemini     = models.ProviderGemini
	ProviderOpenAI     = models.ProviderOpenAI
	ProviderMoonshot   = models.ProviderMoonshot
	ProviderDeepSeek   = models.ProviderDeepSeek
	ProviderQwen       = models.ProviderQwen
	ProviderOpenRouter = models.ProviderOpenRouter
)

// Re-export APIShape constants from models package
const (
	APIShapeOpenAI    = models.APIShapeOpenAI
	APIShapeAnthropic = models.APIShapeAnthropic
	APIShapeGemini    = models.APIShapeGemini
)

// ModelCost represents the cost structure for a model (re-exported from models package)
type ModelCost = models.ModelCost

// ModelInfo contains all information about a model (re-exported from models package)
type ModelInfo = models.ModelInfo

// AllModelInfos contains all model information in a centralized map
// Model definitions are organized in types/models/ by provider
var AllModelInfos = initAllModelInfos()

func initAllModelInfos() map[string]ModelInfo {
	result := make(map[string]ModelInfo)

	// Merge all provider-specific model maps
	// Since ModelInfo is now a type alias, no conversion needed
	for k, v := range models.OpenAIModels {
		result[k] = v
	}
	for k, v := range models.AnthropicModels {
		result[k] = v
	}
	for k, v := range models.GeminiModels {
		result[k] = v
	}
	for k, v := range models.OtherModels {
		result[k] = v
	}

	return result
}
