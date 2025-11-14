package types

// Provider represents the AI provider
type Provider string

// APIShape represents the API shape/format
type APIShape string

// Model constants
const (

	// OpenAI
	ModelGPT4o         = "gpt-4o"      // $0.15 / 1M
	ModelGPT4oMini     = "gpt-4o-mini" // $0.15 / 1M
	ModelGPT4oNano     = "gpt-4o-nano" // $0.15 / 1M
	ModelGPTo4Mini     = "o4-mini"     // 10x gpt-4o-mini
	ModelGPTo3Mini     = "o3-mini"     // 10x gpt-4o-mini
	ModelGPT4_1        = "gpt-4.1"     // $2
	ModelGPT4_1_Mini   = "gpt-4.1-mini"
	ModelGPTo3         = "o3"               // $2
	ModelGPT5_20250807 = "gpt-5-2025-08-07" // $1.25

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

	// kimi
	ModelKimiK2              = "kimi-k2"
	ModelKimiK2_0711_Preview = "kimi-k2-0711-preview"
	ModelOpenRouterKimiK2    = "moonshotai/kimi-k2"

	ModelDeepSeekR1          = "DeepSeek-R1"
	ModelQwen25VL72BInstruct = "Qwen2.5-VL-72B-Instruct"
)

// AllModels contains all supported models
var AllModels = []string{
	ModelGPT4_1,
	ModelGPT4_1_Mini,
	ModelGPTo3,
	ModelGPT4o,
	ModelGPT4oMini,
	ModelGPT4oNano,
	ModelGPTo4Mini,
	ModelGPTo3Mini,
	ModelGPT5_20250807,

	ModelClaudeSonnet4_5,
	ModelClaudeSonnet4_20250929,

	ModelClaudeSonnet4,
	ModelClaudeSonnet4_20250514,
	ModelClaude3_7Sonnet,
	ModelClaude3_7Sonnet_20250219,

	ModelGemini2_0_Flash,
	ModelGemini2_0_Flash_001,
	ModelGemini2_5_Pro,
	ModelGemini2_5_Pro_0605,
	ModelGemini2_5_Flash,
	ModelGemini2_5_Flash_0520,

	ModelKimiK2,
	ModelKimiK2_0711_Preview,

	ModelDeepSeekR1,
	ModelQwen25VL72BInstruct,
}

const (
	ProviderAnthropic  Provider = "anthropic"
	ProviderGemini     Provider = "gemini"
	ProviderOpenAI     Provider = "openai"
	ProviderMoonshot   Provider = "moonshot"
	ProviderDeepSeek   Provider = "deepseek"
	ProviderQwen       Provider = "qwen"
	ProviderOpenRouter Provider = "openrouter"
)

const (
	APIShapeOpenAI    APIShape = "openai"
	APIShapeAnthropic APIShape = "anthropic"
	APIShapeGemini    APIShape = "gemini"
)

// ModelCost represents the cost structure for a model
type ModelCost struct {
	InputUSDPer1M           string
	InputCacheWriteUSDPer1M string
	InputCacheReadUSDPer1M  string
	OutputUSDPer1M          string
}
