package models

// OpenAIModels contains all OpenAI model definitions
var OpenAIModels = map[string]ModelInfo{
	"gpt-4.1": {
		Name:     "gpt-4.1",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "2",
			InputCacheReadUSDPer1M: "0.50",
			OutputUSDPer1M:         "8",
		},
	},
	"gpt-4.1-mini": {
		Name:     "gpt-4.1-mini",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "0.4",
			InputCacheReadUSDPer1M: "0.10",
			OutputUSDPer1M:         "1.6",
		},
	},
	"gpt-4o": {
		Name:     "gpt-4o",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "2.5",
			InputCacheReadUSDPer1M: "1.25",
			OutputUSDPer1M:         "10",
		},
	},
	"gpt-4o-mini": {
		Name:     "gpt-4o-mini",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "0.15",
			InputCacheReadUSDPer1M: "0.075",
			OutputUSDPer1M:         "0.6",
		},
	},
	"gpt-4o-nano": {
		Name:     "gpt-4o-nano",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "0.1",
			InputCacheReadUSDPer1M: "0.025",
			OutputUSDPer1M:         "0.4",
		},
	},
	"o4-mini": {
		Name:     "o4-mini",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "1.10",
			InputCacheReadUSDPer1M: "0.55",
			OutputUSDPer1M:         "4.40",
		},
	},
	"o3-mini": {
		Name:     "o3-mini",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "1.10",
			InputCacheReadUSDPer1M: "0.55",
			OutputUSDPer1M:         "4.40",
		},
	},
	"o3": {
		Name:     "o3",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "2",
			InputCacheReadUSDPer1M: "0.50",
			OutputUSDPer1M:         "8",
		},
	},
	"gpt-5-2025-08-07": {
		Name:     "gpt-5-2025-08-07",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "1.25",
			InputCacheReadUSDPer1M: "0.125",
			OutputUSDPer1M:         "10",
		},
	},
	"gpt-5.2-2025-12-11": {
		Name:     "gpt-5.2-2025-12-11",
		Provider: ProviderOpenAI,
		APIShape: APIShapeOpenAI,
		Cost: ModelCost{
			InputUSDPer1M:          "1.75",
			InputCacheReadUSDPer1M: "0.175",
			OutputUSDPer1M:         "14",
		},
	},
}
