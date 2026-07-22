// Package ai provides an explicit, opt-in boundary for local mission assistance.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BriefInput contains only facts already available in NightOps planning state.
type BriefInput struct {
	Origin       string
	Target       string
	TargetWindow string
	Conditions   string
	Forecast     string
	Astronomy    string
	Route        string
	Equipment    string
}

// Provider generates a mission brief from supplied planning facts.
type Provider interface {
	GenerateBrief(context.Context, BriefInput) (string, error)
}

// OllamaProvider uses a local Ollama-compatible generate endpoint.
// It does not contact the network until GenerateBrief is called.
type OllamaProvider struct {
	Endpoint string
	Model    string
	Client   *http.Client
	Timeout  time.Duration
}

// NewOllamaProvider creates a bounded local AI provider.
func NewOllamaProvider(endpoint, model string, timeout time.Duration) *OllamaProvider {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &OllamaProvider{Endpoint: strings.TrimRight(endpoint, "/"), Model: strings.TrimSpace(model), Timeout: timeout, Client: &http.Client{Timeout: timeout}}
}

// GenerateBrief asks the local model for a concise plan grounded only in input facts.
func (p *OllamaProvider) GenerateBrief(ctx context.Context, input BriefInput) (string, error) {
	if strings.TrimSpace(p.Model) == "" {
		return "", fmt.Errorf("AI model is required")
	}
	if strings.TrimSpace(input.Origin) == "" {
		return "", fmt.Errorf("mission origin is required")
	}
	prompt := buildPrompt(input)
	body, err := json.Marshal(ollamaRequest{Model: p.Model, Prompt: prompt, Stream: false})
	if err != nil {
		return "", fmt.Errorf("encode AI request: %w", err)
	}
	endpoint, err := url.Parse(p.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return "", fmt.Errorf("invalid AI endpoint %q", p.Endpoint)
	}
	requestContext, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create AI request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := p.client().Do(request)
	if err != nil {
		return "", fmt.Errorf("AI request: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read AI response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("AI provider returned HTTP %d", response.StatusCode)
	}
	var payload ollamaResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("decode AI response: %w", err)
	}
	if strings.TrimSpace(payload.Error) != "" {
		return "", fmt.Errorf("AI provider: %s", payload.Error)
	}
	brief := strings.TrimSpace(payload.Response)
	if brief == "" {
		return "", fmt.Errorf("AI provider returned an empty brief")
	}
	return brief, nil
}

func buildPrompt(input BriefInput) string {
	return fmt.Sprintf("You are NightOps mission control. Produce a concise observing mission brief using only the facts below. Do not invent weather, coordinates, target visibility, equipment, recommendations, or times. If a fact is unknown, say unknown. Separate confirmed facts from suggested preparation.\n\nORIGIN: %s\nTARGET: %s\nTARGET WINDOW: %s\nCONDITIONS: %s\nFORECAST: %s\nASTRONOMY: %s\nROUTE: %s\nEQUIPMENT: %s", input.Origin, valueOrUnknown(input.Target), valueOrUnknown(input.TargetWindow), valueOrUnknown(input.Conditions), valueOrUnknown(input.Forecast), valueOrUnknown(input.Astronomy), valueOrUnknown(input.Route), valueOrUnknown(input.Equipment))
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func (p *OllamaProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return http.DefaultClient
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}
