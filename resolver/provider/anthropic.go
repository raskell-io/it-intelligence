//go:build !noanthropic

package provider

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go/v2"

	"github.com/raskell-io/it-intelligence/resolver"
)

// anthropicProvider calls the Claude API with structured outputs, so the answer is always the schema.
// Credentials come from the environment (ANTHROPIC_API_KEY or an `ant auth login` profile).
type anthropicProvider struct {
	client anthropic.Client
	model  string
}

func newAnthropic(c Config) (Provider, error) {
	model := c.Model
	if model == "" {
		model = "claude-opus-5-5"
	}
	return &anthropicProvider{client: anthropic.NewClient(), model: model}, nil
}

func (p *anthropicProvider) Name() string  { return "anthropic" }
func (p *anthropicProvider) Model() string { return p.model }

func (p *anthropicProvider) Judge(ctx context.Context, c resolver.Candidate) (resolver.Judgement, error) {
	system, user := Prompt(c)
	resp, err := p.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     p.model,
		MaxTokens: 16000,
		System:    []anthropic.BetaTextBlockParam{{Text: system}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(user))},
		OutputConfig: anthropic.BetaOutputConfigParam{
			// A small classification over metadata: low effort is enough.
			Effort: anthropic.BetaOutputConfigEffortLow,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: Schema},
		},
		// If a safety classifier declines, the API retries on a fallback model in the same call.
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
		Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: "default"},
	})
	if err != nil {
		return resolver.Judgement{}, fmt.Errorf("anthropic: %w", err)
	}
	if resp.StopReason == anthropic.BetaStopReasonRefusal {
		return resolver.Judgement{}, fmt.Errorf("anthropic: request declined (%s)", resp.StopDetails.Category)
	}
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			return finish(p, c, t.Text)
		}
	}
	return resolver.Judgement{}, fmt.Errorf("anthropic: no text in the response (stop reason %s)", resp.StopReason)
}
