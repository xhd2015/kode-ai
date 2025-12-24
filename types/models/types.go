package models

// Provider represents the AI provider
type Provider string

// APIShape represents the API shape/format
type APIShape string

// Provider constants
const (
	ProviderAnthropic  Provider = "anthropic"
	ProviderGemini     Provider = "gemini"
	ProviderOpenAI     Provider = "openai"
	ProviderMoonshot   Provider = "moonshot"
	ProviderDeepSeek   Provider = "deepseek"
	ProviderQwen       Provider = "qwen"
	ProviderOpenRouter Provider = "openrouter"
)

// APIShape constants
const (
	APIShapeOpenAI    APIShape = "openai"
	APIShapeAnthropic APIShape = "anthropic"
	APIShapeGemini    APIShape = "gemini"
)

// ModelInfo contains all information about a model
type ModelInfo struct {
	Name     string
	Provider Provider
	Cost     ModelCost
	APIShape APIShape
}

// ModelCost represents the cost structure for a model
type ModelCost struct {
	InputUSDPer1M           string
	InputCacheWriteUSDPer1M string
	InputCacheReadUSDPer1M  string
	OutputUSDPer1M          string
}
