package providers

import "github.com/xhd2015/kode-ai/types"

// Re-export model constants from types package
const (
	ModelGPT4o                    = types.ModelGPT4o
	ModelGPT4oMini                = types.ModelGPT4oMini
	ModelGPT4oNano                = types.ModelGPT4oNano
	ModelGPTo4Mini                = types.ModelGPTo4Mini
	ModelGPTo3Mini                = types.ModelGPTo3Mini
	ModelGPT4_1                   = types.ModelGPT4_1
	ModelGPT4_1_Mini              = types.ModelGPT4_1_Mini
	ModelGPTo3                    = types.ModelGPTo3
	ModelClaude3_7Sonnet          = types.ModelClaude3_7Sonnet
	ModelClaude3_7Sonnet_20250219 = types.ModelClaude3_7Sonnet_20250219
	ModelClaudeSonnet4            = types.ModelClaudeSonnet4
	ModelClaudeSonnet4_20250514   = types.ModelClaudeSonnet4_20250514
	ModelGemini2_0_Flash          = types.ModelGemini2_0_Flash
	ModelGemini2_0_Flash_001      = types.ModelGemini2_0_Flash_001
	ModelGemini2_5_Pro            = types.ModelGemini2_5_Pro
	ModelGemini2_5_Pro_0605       = types.ModelGemini2_5_Pro_0605
	ModelGemini2_5_Flash          = types.ModelGemini2_5_Flash
	ModelGemini2_5_Flash_0520     = types.ModelGemini2_5_Flash_0520
	ModelKimiK2                   = types.ModelKimiK2
	ModelKimiK2_0711_Preview      = types.ModelKimiK2_0711_Preview
	ModelOpenRouterKimiK2         = types.ModelOpenRouterKimiK2
	ModelDeepSeekR1               = types.ModelDeepSeekR1
	ModelQwen25VL72BInstruct      = types.ModelQwen25VL72BInstruct
)

// Re-export types
type ModelCost = types.ModelCost

// AllModels re-exports from types package
var AllModels = types.AllModels

var modelAlias = map[string]string{
	ModelClaude3_7Sonnet: ModelClaude3_7Sonnet_20250219,
	ModelClaudeSonnet4:   ModelClaudeSonnet4_20250514,
	ModelGemini2_0_Flash: ModelGemini2_0_Flash_001,
	ModelGemini2_5_Pro:   ModelGemini2_5_Pro_0605,
	ModelGemini2_5_Flash: ModelGemini2_5_Flash_0520,
	ModelKimiK2:          ModelKimiK2_0711_Preview,
}

func GetUnderlyingModel(model string) string {
	underlyingModel, ok := modelAlias[model]
	if !ok {
		return model
	}
	return underlyingModel
}

var cluade3_7Cost = ModelCost{
	InputUSDPer1M:           "3.00",
	InputCacheWriteUSDPer1M: "3.75", // cache has 5minute duration
	InputCacheReadUSDPer1M:  "0.30",
	OutputUSDPer1M:          "15.00",
}

// see https://www.anthropic.com/news/claude-sonnet-4-5
var claudeSonnet4_5Cost = ModelCost{
	InputUSDPer1M:           "3.00",
	InputCacheWriteUSDPer1M: "3.75", // cache has 5minute duration
	InputCacheReadUSDPer1M:  "0.30",
	OutputUSDPer1M:          "15.00",
}

// for <200k input tokens
var gemini2_0_FlashCost = ModelCost{
	InputUSDPer1M:          "0.1",
	InputCacheReadUSDPer1M: "0.025",
	OutputUSDPer1M:         "0.4",
}
var gemini2_5_FlashCost = ModelCost{
	InputUSDPer1M:          "0.3",
	InputCacheReadUSDPer1M: "0.075",
	OutputUSDPer1M:         "2.5",
}
var gemini2_5_ProCost_Under200KInput = ModelCost{
	InputUSDPer1M:          "1.25",
	InputCacheReadUSDPer1M: "0.31",
	OutputUSDPer1M:         "10",
}

var modelCostMapping = map[string]ModelCost{
	ModelGPT4_1: ModelCost{
		InputUSDPer1M:          "2",
		InputCacheReadUSDPer1M: "0.50",
		OutputUSDPer1M:         "8",
	},
	ModelGPT4_1_Mini: ModelCost{
		InputUSDPer1M:          "0.4",
		InputCacheReadUSDPer1M: "0.10",
		OutputUSDPer1M:         "1.6",
	},
	ModelGPT4o: ModelCost{
		InputUSDPer1M:          "2.5",
		InputCacheReadUSDPer1M: "1.25",
		OutputUSDPer1M:         "10",
	},
	ModelGPT4oNano: ModelCost{
		InputUSDPer1M:          "0.1",
		InputCacheReadUSDPer1M: "0.025",
		OutputUSDPer1M:         "0.4",
	},
	ModelGPT4oMini: ModelCost{
		InputUSDPer1M:          "0.15",
		InputCacheReadUSDPer1M: "0.075",
		OutputUSDPer1M:         "0.6",
	},
	ModelGPTo4Mini: ModelCost{
		InputUSDPer1M:          "1.10",
		InputCacheReadUSDPer1M: "0.55",
		OutputUSDPer1M:         "4.40",
	},
	ModelGPTo3Mini: ModelCost{
		InputUSDPer1M:          "1.10",
		InputCacheReadUSDPer1M: "0.55",
		OutputUSDPer1M:         "4.40",
	},
	// NOTE: o3 price has dropped 5x, same with GPT-4.1
	ModelGPTo3: ModelCost{
		InputUSDPer1M:          "2",
		InputCacheReadUSDPer1M: "0.50",
		OutputUSDPer1M:         "8",
	},
	// see https://openai.com/api/pricing/
	ModelClaude3_7Sonnet:              cluade3_7Cost,
	ModelClaude3_7Sonnet_20250219:     cluade3_7Cost,
	ModelClaudeSonnet4:                cluade3_7Cost,
	ModelClaudeSonnet4_20250514:       cluade3_7Cost,
	types.ModelClaudeSonnet4_5:        claudeSonnet4_5Cost,
	types.ModelClaudeSonnet4_20250929: claudeSonnet4_5Cost,

	// see https://ai.google.dev/gemini-api/docs/pricing
	ModelGemini2_0_Flash:      gemini2_0_FlashCost,
	ModelGemini2_0_Flash_001:  gemini2_0_FlashCost,
	ModelGemini2_5_Pro:        gemini2_5_ProCost_Under200KInput,
	ModelGemini2_5_Pro_0605:   gemini2_5_ProCost_Under200KInput,
	ModelGemini2_5_Flash:      gemini2_5_FlashCost,
	ModelGemini2_5_Flash_0520: gemini2_5_FlashCost,

	// see https://platform.moonshot.cn/docs/pricing/chat
	ModelKimiK2_0711_Preview: ModelCost{
		InputUSDPer1M:          "0.56", // 4RMB
		InputCacheReadUSDPer1M: "0.14", // 1RMB
		OutputUSDPer1M:         "2.23", // 16RMB
	},
	ModelOpenRouterKimiK2: ModelCost{
		InputUSDPer1M:          "0.56", // 4RMB
		InputCacheReadUSDPer1M: "0.14", // 1RMB
		OutputUSDPer1M:         "2.23", // 16RMB
	},
}

func GetModelCost(model string) (ModelCost, bool) {
	modelCost, ok := modelCostMapping[model]
	if ok {
		return modelCost, true
	}
	underlyingModel := GetUnderlyingModel(model)
	if underlyingModel != "" {
		return ModelCost{}, false
	}
	underlyingModelCost, ok := modelCostMapping[underlyingModel]
	return underlyingModelCost, ok
}
