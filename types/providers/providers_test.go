package providers

import "testing"

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
