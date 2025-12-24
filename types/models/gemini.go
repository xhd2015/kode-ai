package models

// GeminiModels contains all Gemini model definitions
var GeminiModels = map[string]ModelInfo{
	"gemini-2.0-flash": {
		Name:     "gemini-2.0-flash",
		Provider: ProviderGemini,
		APIShape: APIShapeGemini,
		Cost: ModelCost{
			InputUSDPer1M:          "0.1",
			InputCacheReadUSDPer1M: "0.025",
			OutputUSDPer1M:         "0.4",
		},
	},
	"gemini-2.0-flash-001": {
		Name:     "gemini-2.0-flash-001",
		Provider: ProviderGemini,
		APIShape: APIShapeGemini,
		Cost: ModelCost{
			InputUSDPer1M:          "0.1",
			InputCacheReadUSDPer1M: "0.025",
			OutputUSDPer1M:         "0.4",
		},
	},
	"gemini-2.5-pro": {
		Name:     "gemini-2.5-pro",
		Provider: ProviderGemini,
		APIShape: APIShapeGemini,
		Cost: ModelCost{
			InputUSDPer1M:          "1.25",
			InputCacheReadUSDPer1M: "0.31",
			OutputUSDPer1M:         "10",
		},
	},
	"gemini-2.5-pro-preview-06-05": {
		Name:     "gemini-2.5-pro-preview-06-05",
		Provider: ProviderGemini,
		APIShape: APIShapeGemini,
		Cost: ModelCost{
			InputUSDPer1M:          "1.25",
			InputCacheReadUSDPer1M: "0.31",
			OutputUSDPer1M:         "10",
		},
	},
	"gemini-2.5-flash": {
		Name:     "gemini-2.5-flash",
		Provider: ProviderGemini,
		APIShape: APIShapeGemini,
		Cost: ModelCost{
			InputUSDPer1M:          "0.3",
			InputCacheReadUSDPer1M: "0.075",
			OutputUSDPer1M:         "2.5",
		},
	},
	"gemini-2.5-flash-preview-05-20": {
		Name:     "gemini-2.5-flash-preview-05-20",
		Provider: ProviderGemini,
		APIShape: APIShapeGemini,
		Cost: ModelCost{
			InputUSDPer1M:          "0.3",
			InputCacheReadUSDPer1M: "0.075",
			OutputUSDPer1M:         "2.5",
		},
	},
	"gemini-3-pro-preview": {
		Name:     "gemini-3-pro-preview",
		Provider: ProviderGemini,
		APIShape: APIShapeGemini,
		Cost: ModelCost{
			InputUSDPer1M:          "2",
			InputCacheReadUSDPer1M: "0.2",
			OutputUSDPer1M:         "12",
		},
	},
}
