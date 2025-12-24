package models

// OtherModels contains model definitions for Kimi, DeepSeek, Qwen, etc.
var OtherModels = map[string]ModelInfo{
	// Kimi Models
	"kimi-k2": {
		Name:     "kimi-k2",
		Provider: ProviderMoonshot,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "0.56",
			InputCacheReadUSDPer1M: "0.14",
			OutputUSDPer1M:         "2.23",
		},
	},
	"kimi-k2-0711-preview": {
		Name:     "kimi-k2-0711-preview",
		Provider: ProviderMoonshot,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "0.56",
			InputCacheReadUSDPer1M: "0.14",
			OutputUSDPer1M:         "2.23",
		},
	},
	"moonshotai/kimi-k2": {
		Name:     "moonshotai/kimi-k2",
		Provider: ProviderOpenRouter,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "0.56",
			InputCacheReadUSDPer1M: "0.14",
			OutputUSDPer1M:         "2.23",
		},
	},

	// DeepSeek Models
	"DeepSeek-R1": {
		Name:     "DeepSeek-R1",
		Provider: ProviderDeepSeek,
		APIShape: APIShapeOpenAI,
		Cost:     ModelCost{}, // Cost not specified yet
	},

	// Qwen Models
	"Qwen2.5-VL-72B-Instruct": {
		Name:     "Qwen2.5-VL-72B-Instruct",
		Provider: ProviderQwen,
		APIShape: APIShapeOpenAI,
		Cost:     ModelCost{}, // Cost not specified yet
	},
}
