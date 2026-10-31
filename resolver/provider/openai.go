package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/raskell-io/it-intelligence/resolver"
)

// openAIProvider talks to any endpoint that implements /chat/completions with JSON-schema response
// formats: Ollama, llama.cpp server, vLLM and most hosted gateways. It is the way to run inference on a
// local model. The key, if the endpoint needs one, comes from OPENAI_API_KEY.
type openAIProvider struct {
	base, model, key string
	http             *http.Client
}

func newOpenAI(c Config) (Provider, error) {
	if c.BaseURL == "" || c.Model == "" {
		return nil, fmt.Errorf("openai-compatible provider needs base_url and model")
	}
	return &openAIProvider{base: strings.TrimRight(c.BaseURL, "/"), model: c.Model, key: os.Getenv("OPENAI_API_KEY"),
		http: &http.Client{Timeout: 2 * time.Minute}}, nil
}

func (p *openAIProvider) Name() string  { return "openai-compatible" }
func (p *openAIProvider) Model() string { return p.model }

func (p *openAIProvider) Judge(ctx context.Context, c resolver.Candidate) (resolver.Judgement, error) {
	system, user := Prompt(c)
	body, _ := json.Marshal(map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"response_format": map[string]any{"type": "json_schema",
			"json_schema": map[string]any{"name": "judgement", "schema": Schema, "strict": true}},
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.base+"/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if p.key != "" {
		req.Header.Set("Authorization", "Bearer "+p.key)
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return resolver.Judgement{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resolver.Judgement{}, fmt.Errorf("openai-compatible: %s", resp.Status)
	}
	var out struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return resolver.Judgement{}, err
	}
	if len(out.Choices) == 0 {
		return resolver.Judgement{}, fmt.Errorf("openai-compatible: no choices")
	}
	return finish(p, c, out.Choices[0].Message.Content)
}
