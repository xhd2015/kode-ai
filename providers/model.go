package providers

import (
	"github.com/xhd2015/kode-ai/types"
	"github.com/xhd2015/kode-ai/types/providers"
)

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

func GetUnderlyingModel(model string) string {
	return providers.GetUnderlyingModel(model)
}

func GetModelCost(model string) (ModelCost, bool) {
	return providers.GetModelCost(model)
}
