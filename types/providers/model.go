package providers

import "github.com/xhd2015/kode-ai/types"

var cluade3_7Cost = types.ModelCost{
	InputUSDPer1M:           "3.00",
	InputCacheWriteUSDPer1M: "3.75", // cache has 5minute duration
	InputCacheReadUSDPer1M:  "0.30",
	OutputUSDPer1M:          "15.00",
}

// see https://www.anthropic.com/news/claude-sonnet-4-5
var claudeSonnet4_5Cost = types.ModelCost{
	InputUSDPer1M:           "3.00",
	InputCacheWriteUSDPer1M: "3.75", // cache has 5minute duration
	InputCacheReadUSDPer1M:  "0.30",
	OutputUSDPer1M:          "15.00",
}

// for <200k input tokens
var gemini2_0_FlashCost = types.ModelCost{
	InputUSDPer1M:          "0.1",
	InputCacheReadUSDPer1M: "0.025",
	OutputUSDPer1M:         "0.4",
}
var gemini2_5_FlashCost = types.ModelCost{
	InputUSDPer1M:          "0.3",
	InputCacheReadUSDPer1M: "0.075",
	OutputUSDPer1M:         "2.5",
}
var gemini2_5_ProCost_Under200KInput = types.ModelCost{
	InputUSDPer1M:          "1.25",
	InputCacheReadUSDPer1M: "0.31",
	OutputUSDPer1M:         "10",
}

var modelCostMapping = map[string]types.ModelCost{
	types.ModelGPT4_1: types.ModelCost{
		InputUSDPer1M:          "2",
		InputCacheReadUSDPer1M: "0.50",
		OutputUSDPer1M:         "8",
	},
	types.ModelGPT4_1_Mini: types.ModelCost{
		InputUSDPer1M:          "0.4",
		InputCacheReadUSDPer1M: "0.10",
		OutputUSDPer1M:         "1.6",
	},
	types.ModelGPT4o: types.ModelCost{
		InputUSDPer1M:          "2.5",
		InputCacheReadUSDPer1M: "1.25",
		OutputUSDPer1M:         "10",
	},
	types.ModelGPT4oNano: types.ModelCost{
		InputUSDPer1M:          "0.1",
		InputCacheReadUSDPer1M: "0.025",
		OutputUSDPer1M:         "0.4",
	},
	types.ModelGPT4oMini: types.ModelCost{
		InputUSDPer1M:          "0.15",
		InputCacheReadUSDPer1M: "0.075",
		OutputUSDPer1M:         "0.6",
	},
	types.ModelGPTo4Mini: types.ModelCost{
		InputUSDPer1M:          "1.10",
		InputCacheReadUSDPer1M: "0.55",
		OutputUSDPer1M:         "4.40",
	},
	types.ModelGPTo3Mini: types.ModelCost{
		InputUSDPer1M:          "1.10",
		InputCacheReadUSDPer1M: "0.55",
		OutputUSDPer1M:         "4.40",
	},
	// NOTE: o3 price has dropped 5x, same with GPT-4.1
	types.ModelGPTo3: types.ModelCost{
		InputUSDPer1M:          "2",
		InputCacheReadUSDPer1M: "0.50",
		OutputUSDPer1M:         "8",
	},
	// see https://openai.com/api/pricing/
	types.ModelClaude3_7Sonnet:          cluade3_7Cost,
	types.ModelClaude3_7Sonnet_20250219: cluade3_7Cost,
	types.ModelClaudeSonnet4:            cluade3_7Cost,
	types.ModelClaudeSonnet4_20250514:   cluade3_7Cost,
	types.ModelClaudeSonnet4_5:          claudeSonnet4_5Cost,
	types.ModelClaudeSonnet4_20250929:   claudeSonnet4_5Cost,

	// see https://ai.google.dev/gemini-api/docs/pricing
	types.ModelGemini2_0_Flash:      gemini2_0_FlashCost,
	types.ModelGemini2_0_Flash_001:  gemini2_0_FlashCost,
	types.ModelGemini2_5_Pro:        gemini2_5_ProCost_Under200KInput,
	types.ModelGemini2_5_Pro_0605:   gemini2_5_ProCost_Under200KInput,
	types.ModelGemini2_5_Flash:      gemini2_5_FlashCost,
	types.ModelGemini2_5_Flash_0520: gemini2_5_FlashCost,

	// see https://platform.moonshot.cn/docs/pricing/chat
	types.ModelKimiK2_0711_Preview: types.ModelCost{
		InputUSDPer1M:          "0.56", // 4RMB
		InputCacheReadUSDPer1M: "0.14", // 1RMB
		OutputUSDPer1M:         "2.23", // 16RMB
	},
	types.ModelOpenRouterKimiK2: types.ModelCost{
		InputUSDPer1M:          "0.56", // 4RMB
		InputCacheReadUSDPer1M: "0.14", // 1RMB
		OutputUSDPer1M:         "2.23", // 16RMB
	},
}

func GetModelCost(model string) (types.ModelCost, bool) {
	modelCost, ok := modelCostMapping[model]
	if ok {
		return modelCost, true
	}
	underlyingModel := GetUnderlyingModel(model)
	if underlyingModel != "" {
		return types.ModelCost{}, false
	}
	underlyingModelCost, ok := modelCostMapping[underlyingModel]
	return underlyingModelCost, ok
}

var modelAlias = map[string]string{
	types.ModelClaude3_7Sonnet:  types.ModelClaude3_7Sonnet_20250219,
	types.ModelClaudeSonnet4:    types.ModelClaudeSonnet4_20250514,
	types.ModelGemini2_0_Flash:  types.ModelGemini2_0_Flash_001,
	types.ModelGemini2_5_Pro:    types.ModelGemini2_5_Pro_0605,
	types.ModelGemini2_5_Flash:  types.ModelGemini2_5_Flash_0520,
	types.ModelKimiK2:           types.ModelKimiK2_0711_Preview,
	types.ModelOpenRouterKimiK2: types.ModelOpenRouterKimiK2,
}

func GetUnderlyingModel(model string) string {
	underlyingModel, ok := modelAlias[model]
	if !ok {
		return model
	}
	return underlyingModel
}
