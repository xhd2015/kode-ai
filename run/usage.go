package run

import (
	"fmt"
	"io"

	"github.com/shopspring/decimal"
	"github.com/xhd2015/kode-ai/internal/markdown"
	"github.com/xhd2015/kode-ai/providers"
	"github.com/xhd2015/kode-ai/types"
)

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
	var total types.TokenUsageCost
	var costs []types.TokenUsageCost
	for _, msg := range messages {
		if msg.Type != types.MsgType_TokenUsage {
			continue
		}
		if msg.TokenUsage == nil {
			continue
		}

		total.Usage = total.Usage.Add(*msg.TokenUsage)
		provider, err := providers.GetModelAPIShape(msg.Model)
		if err != nil {
			return err
		}

		modelCost, ok := providers.ComputeCost(provider, msg.Model, *msg.TokenUsage)
		if !ok {
			return fmt.Errorf("cannot compute cost for model: %s", msg.Model)
		}
		costs = append(costs, types.TokenUsageCost{
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

func printTokenUsage(w io.Writer, title string, tokenUsage types.TokenUsage, cost string) {
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
