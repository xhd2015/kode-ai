package providers

const (
	ModelGPT4o                    = "gpt-4o"      // $0.15 / 1M
	ModelGPT4oMini                = "gpt-4o-mini" // $0.15 / 1M
	ModelGPT4oNano                = "gpt-4o-nano" // $0.15 / 1M
	ModelGPTo4Mini                = "o4-mini"     // 10x gpt-4o-mini
	ModelGPTo3Mini                = "o3-mini"     // 10x gpt-4o-mini
	ModelGPT4_1                   = "gpt-4.1"     // $2
	ModelGPT4_1_Mini              = "gpt-4.1-mini"
	ModelGPTo3                    = "o3"                // $2
	ModelClaude3_7Sonnet          = "claude-3-7-sonnet" // $3
	ModelClaude3_7Sonnet_20250219 = "claude-3-7-sonnet@20250219"
	ModelClaudeSonnet4            = "claude-sonnet-4" // $3
	ModelClaudeSonnet4_20250514   = "claude-sonnet-4@20250514"
	ModelDeepSeekR1               = "DeepSeek-R1"
	ModelQwen25VL72BInstruct      = "Qwen2.5-VL-72B-Instruct"
)

var AllModels = []string{
	ModelClaudeSonnet4,
	ModelClaudeSonnet4_20250514,
	ModelClaude3_7Sonnet,
	ModelClaude3_7Sonnet_20250219,
	ModelGPT4_1,
	ModelGPT4_1_Mini,
	ModelGPTo3,
	ModelGPT4o,
	ModelGPT4oMini,
	ModelGPT4oNano,
	ModelGPTo4Mini,
	ModelGPTo3Mini,
	ModelDeepSeekR1,
	ModelQwen25VL72BInstruct,
}

type ModelCost struct {
	InputUSDPer1M           string
	InputCacheWriteUSDPer1M string
	InputCacheReadUSDPer1M  string
	OutputUSDPer1M          string
}

var cluade3_7Cost = ModelCost{
	InputUSDPer1M:           "3.00",
	InputCacheWriteUSDPer1M: "3.75", // cache has 5minute duration
	InputCacheReadUSDPer1M:  "0.30",
	OutputUSDPer1M:          "15.00",
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
	ModelClaude3_7Sonnet:          cluade3_7Cost,
	ModelClaude3_7Sonnet_20250219: cluade3_7Cost,
	ModelClaudeSonnet4:            cluade3_7Cost,
	ModelClaudeSonnet4_20250514:   cluade3_7Cost,
}

func GetModelCost(model string) (ModelCost, bool) {
	modelCost, ok := modelCostMapping[model]
	if !ok {
		return ModelCost{}, false
	}
	return modelCost, true
}
