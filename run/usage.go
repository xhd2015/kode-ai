package run

import (
	"fmt"
	"io"

	"github.com/shopspring/decimal"
	"github.com/xhd2015/kode-ai/internal/markdown"
	"github.com/xhd2015/kode-ai/providers"
)

type TokenUsageCost struct {
	Usage TokenUsage
	Cost  TokenCost
}

// Anthropic:
//   - how to: https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching
//   - when: https://www.anthropic.com/news/prompt-caching
//   - summary:
//     . seems anthropic only caches for long enough texts
//     . The minimum cacheable prompt length is:
//     . 1024 tokens for Claude Opus 4, Claude Sonnet 4, Claude Sonnet 3.7, Claude Sonnet 3.5 and Claude Opus 3
//     . The cache is invalidated after 5 minutes
type TokenUsage struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
	Total  int64 `json:"total"`

	InputBreakdown  TokenUsageInputBreakdown  `json:"input_breakdown"`
	OutputBreakdown TokenUsageOutputBreakdown `json:"output_breakdown"`
}

type TokenUsageInputBreakdown struct {
	CacheWrite   int64 `json:"cache_write"` // anthropic specific
	CacheRead    int64 `json:"cache_read"`
	NonCacheRead int64 `json:"non_cache_read"`
}

type TokenUsageOutputBreakdown struct {
	CacheOutput int64 `json:"cache_output"`
}

func (c TokenUsage) Add(b TokenUsage) TokenUsage {
	return TokenUsage{
		Input:           c.Input + b.Input,
		Output:          c.Output + b.Output,
		Total:           c.Total + b.Total,
		InputBreakdown:  c.InputBreakdown.Add(b.InputBreakdown),
		OutputBreakdown: c.OutputBreakdown.Add(b.OutputBreakdown),
	}
}

func (c TokenUsageInputBreakdown) Add(b TokenUsageInputBreakdown) TokenUsageInputBreakdown {
	return TokenUsageInputBreakdown{
		CacheRead:  c.CacheRead + b.CacheRead,
		CacheWrite: c.CacheWrite + b.CacheWrite,
	}
}
func (c TokenUsageOutputBreakdown) Add(b TokenUsageOutputBreakdown) TokenUsageOutputBreakdown {
	return TokenUsageOutputBreakdown{
		CacheOutput: c.CacheOutput + b.CacheOutput,
	}
}

type TokenCost struct {
	// the three are available for all providers
	InputUSD  string
	OutputUSD string
	TotalUSD  string

	// Input breakdown
	// anthropic has this detail
	InputBreakdown TokenCostInputBreakdown
}

func (c TokenCost) Add(b TokenCost) TokenCost {
	return TokenCost{
		InputUSD:       addDecimals(c.InputUSD, b.InputUSD),
		OutputUSD:      addDecimals(c.OutputUSD, b.OutputUSD),
		TotalUSD:       addDecimals(c.TotalUSD, b.TotalUSD),
		InputBreakdown: c.InputBreakdown.Add(b.InputBreakdown),
	}
}

type TokenCostInputBreakdown struct {
	CacheWriteUSD   string
	CacheReadUSD    string
	NonCacheReadUSD string
}

func (c TokenCostInputBreakdown) Add(b TokenCostInputBreakdown) TokenCostInputBreakdown {
	return TokenCostInputBreakdown{
		CacheWriteUSD:   addDecimals(c.CacheWriteUSD, b.CacheWriteUSD),
		CacheReadUSD:    addDecimals(c.CacheReadUSD, b.CacheReadUSD),
		NonCacheReadUSD: addDecimals(c.NonCacheReadUSD, b.NonCacheReadUSD),
	}
}

func showUsageFromRecordFile(recordFile string) error {
	// read the record file
	messages, err := loadHistoricalMessages(recordFile)
	if err != nil {
		return fmt.Errorf("failed to load historical messages: %v", err)
	}

	return showUsageFromMessages(messages)
}

func showUsageFromMessages(messages Messages) error {
	// calculate the usage
	var total TokenUsageCost
	var costs []TokenUsageCost
	for _, msg := range messages {
		if msg.Type != MsgType_TokenUsage {
			continue
		}
		if msg.TokenUsage == nil {
			continue
		}

		total.Usage = total.Usage.Add(*msg.TokenUsage)
		provider, err := providers.GetModelProvider(msg.Model)
		if err != nil {
			return err
		}

		modelCost, ok := computeCost(provider, msg.Model, *msg.TokenUsage)
		if !ok {
			return fmt.Errorf("cannot compute cost for model: %s", msg.Model)
		}
		costs = append(costs, TokenUsageCost{
			Usage: *msg.TokenUsage,
			Cost:  modelCost,
		})
		total.Cost = total.Cost.Add(modelCost)
	}

	// show a markdown table

	usage := total.Usage
	markdown.PrintGenerate(func(w io.Writer) {
		totalCost := total.Cost

		cacheInputWriteUSD := totalCost.InputBreakdown.CacheWriteUSD
		cacheInputReadUSD := totalCost.InputBreakdown.CacheReadUSD

		fmt.Fprintf(w, "| No. | Input | Cached Input Read | Cache Input Creation | Output | Total|\n")
		fmt.Fprintf(w, "|-----|-------|-------------------|----------------------|--------|------|\n")

		if len(costs) > 1 {
			for i, cost := range costs {
				fmt.Fprintf(w, "| %d-Token | %d | %d | %d | %d | %d |\n", i+1, cost.Usage.Input, cost.Usage.InputBreakdown.CacheRead, cost.Usage.InputBreakdown.CacheWrite, cost.Usage.Output, cost.Usage.Total)
				fmt.Fprintf(w, "| %d-Cost|  %s | %s | %s | %s | $%s |\n", i+1, cost.Cost.InputUSD, cost.Cost.InputBreakdown.CacheReadUSD, cost.Cost.InputBreakdown.CacheWriteUSD, cost.Cost.OutputUSD, cost.Cost.TotalUSD)
			}
			fmt.Fprintf(w, "|-----|-------|-------------------|----------------------|--------|------|\n")
		}

		fmt.Fprintf(w, "| ALL-Token | %d | %d | %d | %d | %d |\n", usage.Input, usage.InputBreakdown.CacheRead, usage.InputBreakdown.CacheWrite, usage.Output, usage.Total)
		fmt.Fprintf(w, "| ALL-Cost | %s | %s | %s | %s | $%s |\n", totalCost.InputUSD, cacheInputReadUSD, cacheInputWriteUSD, totalCost.OutputUSD, totalCost.TotalUSD)
	})
	return nil
}

type Number string

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

var _1M = decimal.NewFromInt(1e6)

func computeCost(provider providers.Provider, model string, usage TokenUsage) (TokenCost, bool) {
	costDef, ok := providers.GetModelCost(model)
	if !ok {
		return TokenCost{}, false
	}
	var inputUSD decimal.Decimal
	var inputBreakdown TokenCostInputBreakdown
	if provider == providers.ProviderAnthropic {
		inputCacheWriteUSD := requireFromString(costDef.InputCacheWriteUSDPer1M).Mul(decimal.NewFromInt(usage.InputBreakdown.CacheWrite)).Div(_1M)
		inputNonCacheReadUSD := requireFromString(costDef.InputUSDPer1M).Mul(decimal.NewFromInt(usage.InputBreakdown.NonCacheRead)).Div(_1M)
		inputCacheReadUSD := requireFromString(costDef.InputCacheReadUSDPer1M).Mul(decimal.NewFromInt(usage.InputBreakdown.CacheRead)).Div(_1M)

		inputUSD = inputCacheWriteUSD.Add(inputNonCacheReadUSD).Add(inputCacheReadUSD)
		inputBreakdown = TokenCostInputBreakdown{
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
	return TokenCost{
		InputUSD:       inputUSD.String(),
		OutputUSD:      outputUSD.String(),
		TotalUSD:       totalUSD.String(),
		InputBreakdown: inputBreakdown,
	}, true
}

func printTokenUsage(w io.Writer, title string, tokenUsage TokenUsage, cost string) {
	fmt.Fprintf(w, "%s - Input: %d, Cache/R: %d, Cache/W: %d, NonCache/R: %d, Output: %d, Total: %d, Cost: %s\n",
		title,
		tokenUsage.Input,
		tokenUsage.InputBreakdown.CacheRead,
		tokenUsage.InputBreakdown.CacheWrite,
		tokenUsage.InputBreakdown.NonCacheRead,
		tokenUsage.Output,
		tokenUsage.Total,
		cost,
	)
}
