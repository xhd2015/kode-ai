package providers

import (
	"github.com/shopspring/decimal"
	"github.com/xhd2015/kode-ai/types"
	"github.com/xhd2015/kode-ai/types/providers"
)

var _1M = decimal.NewFromInt(1e6)

func ComputeCost(apiShape APIShape, model string, usage types.TokenUsage) (types.TokenCost, bool) {
	return providers.ComputeCost(apiShape, model, usage)
}
