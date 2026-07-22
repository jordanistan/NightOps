package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestOllamaProviderGroundsPromptAndReturnsBrief(t *testing.T) {
	provider := NewOllamaProvider("http://ollama.test/api/generate", "nightops-test", time.Second)
	provider.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, expected := range []string{"Clarksville", "M 42", "unknown", "Do not invent"} {
			if !strings.Contains(text, expected) {
				t.Fatalf("AI prompt omitted %q: %s", expected, text)
			}
		}
		return jsonResponse(`{"response":"Confirmed origin: Clarksville. Target window: unknown.","done":true}`), nil
	})}
	brief, err := provider.GenerateBrief(context.Background(), BriefInput{Origin: "Clarksville", Target: "M 42"})
	if err != nil || brief == "" {
		t.Fatalf("brief=%q err=%v", brief, err)
	}
}

func TestOllamaProviderRejectsMissingModelBeforeRequest(t *testing.T) {
	requests := 0
	provider := NewOllamaProvider("http://ollama.test/api/generate", "", time.Second)
	provider.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { requests++; return jsonResponse(`{}`), nil })}
	if _, err := provider.GenerateBrief(context.Background(), BriefInput{Origin: "Home"}); err == nil {
		t.Fatal("missing model was accepted")
	}
	if requests != 0 {
		t.Fatalf("missing model made %d requests", requests)
	}
}

func TestOllamaProviderReportsProviderError(t *testing.T) {
	provider := NewOllamaProvider("http://ollama.test/api/generate", "model", time.Second)
	provider.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(`{"error":"model unavailable"}`), nil
	})}
	if _, err := provider.GenerateBrief(context.Background(), BriefInput{Origin: "Home"}); err == nil || !strings.Contains(err.Error(), "model unavailable") {
		t.Fatalf("provider error=%v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
