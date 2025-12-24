package providers

import "github.com/xhd2015/kode-ai/types"

// GetModelCost returns the cost information for a model
func GetModelCost(model string) (types.ModelCost, bool) {
	// Try direct lookup
	modelInfo, ok := types.AllModelInfos[model]
	if ok {
		return modelInfo.Cost, true
	}

	// Try underlying model (alias)
	underlyingModel := GetUnderlyingModel(model)
	if underlyingModel == model {
		return types.ModelCost{}, false
	}

	underlyingModelInfo, ok := types.AllModelInfos[underlyingModel]
	if !ok {
		return types.ModelCost{}, false
	}
	return underlyingModelInfo.Cost, true
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
