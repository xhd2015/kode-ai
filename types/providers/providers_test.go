package providers

import (
	"testing"

	"github.com/xhd2015/kode-ai/types"
)

func TestLikelyClaudeModelFallback(t *testing.T) {
	tests := []string{
		"claude-sonnet-4-6",
		"claude opus 4.1",
		"bedrock/claude-sonnet-4-6",
	}

	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			apiShape, err := GetModelAPIShape(model)
			if err != nil {
				t.Fatalf("GetModelAPIShape() error = %v", err)
			}
			if apiShape != APIShapeAnthropic {
				t.Fatalf("api shape = %q, want %q", apiShape, APIShapeAnthropic)
			}

			provider, err := GetModelProvider(model)
			if err != nil {
				t.Fatalf("GetModelProvider() error = %v", err)
			}
			if provider != ProviderAnthropic {
				t.Fatalf("provider = %q, want %q", provider, ProviderAnthropic)
			}
		})
	}
}

func TestNormalizeAPIShape(t *testing.T) {
	apiShape, err := NormalizeAPIShape("Anthropic")
	if err != nil {
		t.Fatalf("NormalizeAPIShape() error = %v", err)
	}
	if apiShape != APIShapeAnthropic {
		t.Fatalf("api shape = %q, want %q", apiShape, APIShapeAnthropic)
	}
}

func TestComputeCostWithModelCost(t *testing.T) {
	cost, ok := ComputeCostWithModelCost(APIShapeOpenAI, types.ModelCost{
		InputUSDPer1M:  "1",
		OutputUSDPer1M: "4",
	}, types.TokenUsage{
		Input:  1_000_000,
		Output: 500_000,
		Total:  1_500_000,
	})
	if !ok {
		t.Fatalf("expected cost to be computed")
	}
	if cost.InputUSD != "1" {
		t.Fatalf("input cost = %q, want %q", cost.InputUSD, "1")
	}
	if cost.OutputUSD != "2" {
		t.Fatalf("output cost = %q, want %q", cost.OutputUSD, "2")
	}
	if cost.TotalUSD != "3" {
		t.Fatalf("total cost = %q, want %q", cost.TotalUSD, "3")
	}
}
