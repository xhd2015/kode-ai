package models

// AnthropicModels contains all Anthropic model definitions
var AnthropicModels = map[string]ModelInfo{
	"claude-3-7-sonnet": {
		Name:     "claude-3-7-sonnet",
		Provider: ProviderAnthropic,
		APIShape: APIShapeAnthropic,
		Cost: ModelCost{
			InputUSDPer1M:           "3.00",
			InputCacheWriteUSDPer1M: "3.75",
			InputCacheReadUSDPer1M:  "0.30",
			OutputUSDPer1M:          "15.00",
		},
	},
	"claude-3-7-sonnet@20250219": {
		Name:     "claude-3-7-sonnet@20250219",
		Provider: ProviderAnthropic,
		APIShape: APIShapeAnthropic,
		Cost: ModelCost{
			InputUSDPer1M:           "3.00",
			InputCacheWriteUSDPer1M: "3.75",
			InputCacheReadUSDPer1M:  "0.30",
			OutputUSDPer1M:          "15.00",
		},
	},
	"claude-sonnet-4": {
		Name:     "claude-sonnet-4",
		Provider: ProviderAnthropic,
		APIShape: APIShapeAnthropic,
		Cost: ModelCost{
			InputUSDPer1M:           "3.00",
			InputCacheWriteUSDPer1M: "3.75",
			InputCacheReadUSDPer1M:  "0.30",
			OutputUSDPer1M:          "15.00",
		},
	},
	"claude-sonnet-4@20250514": {
		Name:     "claude-sonnet-4@20250514",
		Provider: ProviderAnthropic,
		APIShape: APIShapeAnthropic,
		Cost: ModelCost{
			InputUSDPer1M:           "3.00",
			InputCacheWriteUSDPer1M: "3.75",
			InputCacheReadUSDPer1M:  "0.30",
			OutputUSDPer1M:          "15.00",
		},
	},
	"claude-sonnet-4-5": {
		Name:     "claude-sonnet-4-5",
		Provider: ProviderAnthropic,
		APIShape: APIShapeAnthropic,
		Cost: ModelCost{
			InputUSDPer1M:           "3.00",
			InputCacheWriteUSDPer1M: "3.75",
			InputCacheReadUSDPer1M:  "0.30",
			OutputUSDPer1M:          "15.00",
		},
	},
	"claude-sonnet-4-5@20250929": {
		Name:     "claude-sonnet-4-5@20250929",
		Provider: ProviderAnthropic,
		APIShape: APIShapeAnthropic,
		Cost: ModelCost{
			InputUSDPer1M:           "3.00",
			InputCacheWriteUSDPer1M: "3.75",
			InputCacheReadUSDPer1M:  "0.30",
			OutputUSDPer1M:          "15.00",
		},
	},
}
