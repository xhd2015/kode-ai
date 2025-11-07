package providers

import (
	"github.com/shopspring/decimal"
	"github.com/xhd2015/kode-ai/types"
)

var _1M = decimal.NewFromInt(1e6)

func ComputeCost(apiShape APIShape, model string, usage types.TokenUsage) (types.TokenCost, bool) {
	costDef, ok := GetModelCost(model)
	if !ok {
		return types.TokenCost{}, false
	}
	var inputUSD decimal.Decimal
	var inputBreakdown types.TokenCostInputBreakdown
	if apiShape == APIShapeAnthropic {
		inputCacheWriteUSD := requireFromString(costDef.InputCacheWriteUSDPer1M).Mul(decimal.NewFromInt(usage.InputBreakdown.CacheWrite)).Div(_1M)
		inputNonCacheReadUSD := requireFromString(costDef.InputUSDPer1M).Mul(decimal.NewFromInt(usage.InputBreakdown.NonCacheRead)).Div(_1M)
		inputCacheReadUSD := requireFromString(costDef.InputCacheReadUSDPer1M).Mul(decimal.NewFromInt(usage.InputBreakdown.CacheRead)).Div(_1M)

		inputUSD = inputCacheWriteUSD.Add(inputNonCacheReadUSD).Add(inputCacheReadUSD)
		inputBreakdown = types.TokenCostInputBreakdown{
			CacheWriteUSD:   inputCacheWriteUSD.String(),
			CacheReadUSD:    inputCacheReadUSD.String(),
			NonCacheReadUSD: inputNonCacheReadUSD.String(),
		}
	} else {
		inputCacheWriteUSD := decimal.Zero
		if costDef.InputCacheWriteUSDPer1M != "" {
			inputCacheWriteUSD = requireFromString(costDef.InputCacheWriteUSDPer1M).Mul(decimal.NewFromInt(usage.InputBreakdown.CacheWrite)).Div(_1M)
		}

		if costDef.InputCacheReadUSDPer1M != "" {
			inputCacheReadUSD := requireFromString(costDef.InputCacheReadUSDPer1M).Mul(decimal.NewFromInt(usage.InputBreakdown.CacheRead)).Div(_1M)
			nonCahceReadUSD := requireFromString(costDef.InputUSDPer1M).Mul(decimal.NewFromInt(usage.InputBreakdown.NonCacheRead)).Div(_1M)

			inputUSD = inputCacheReadUSD.Add(nonCahceReadUSD).Add(inputCacheWriteUSD)
		} else {
			inputUSD = requireFromString(costDef.InputUSDPer1M).Mul(decimal.NewFromInt(usage.Input)).Div(_1M)
		}
	}

	outputUSD := requireFromString(costDef.OutputUSDPer1M).Mul(decimal.NewFromInt(usage.Output)).Div(_1M)

	totalUSD := inputUSD.Add(outputUSD)
	return types.TokenCost{
		InputUSD:       inputUSD.String(),
		OutputUSD:      outputUSD.String(),
		TotalUSD:       totalUSD.String(),
		InputBreakdown: inputBreakdown,
	}, true
}

func requireFromString(s string) decimal.Decimal {
	if s == "" {
		return decimal.Zero
	}
	return decimal.RequireFromString(s)
}

func addDecimals(nums ...string) string {
	sum := decimal.Zero
	for _, num := range nums {
		sum = sum.Add(requireFromString(num))
	}
	return sum.String()
}
