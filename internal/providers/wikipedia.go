package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jordanistan/nightops/internal/domain"
)

// WikipediaProvider retrieves a bounded page summary for a selected catalog
// target. The response is reference material, never a planning authority.
type WikipediaProvider struct {
	Endpoint string
	Timeout  time.Duration
	Client   *http.Client
}

// NewWikipediaProvider creates a provider for Wikipedia's REST page-summary
// endpoint or a compatible test/self-hosted endpoint.
func NewWikipediaProvider(endpoint string, timeout time.Duration) *WikipediaProvider {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "https://en.wikipedia.org/api/rest_v1/page/summary"
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &WikipediaProvider{Endpoint: strings.TrimRight(endpoint, "/"), Timeout: timeout, Client: &http.Client{Timeout: timeout}}
}

// Fetch retrieves a target's title, summary, page, and representative image.
func (p *WikipediaProvider) Fetch(ctx context.Context, target domain.MissionTarget) (domain.TargetKnowledge, error) {
	name := strings.TrimSpace(target.Name)
	if name == "" {
		return domain.TargetKnowledge{}, fmt.Errorf("target name is required")
	}
	endpoint, err := url.Parse(p.Endpoint + "/" + url.PathEscape(strings.ReplaceAll(name, " ", "_")))
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return domain.TargetKnowledge{}, fmt.Errorf("invalid Wikipedia endpoint %q", p.Endpoint)
	}
	requestContext, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return domain.TargetKnowledge{}, fmt.Errorf("create Wikipedia request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "NightOps/0.1 local astronomy mission planner")
	response, err := p.client().Do(request)
	if err != nil {
		return domain.TargetKnowledge{}, fmt.Errorf("Wikipedia request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return domain.TargetKnowledge{}, fmt.Errorf("Wikipedia returned HTTP %d", response.StatusCode)
	}
	var page wikipediaSummary
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&page); err != nil {
		return domain.TargetKnowledge{}, fmt.Errorf("decode Wikipedia response: %w", err)
	}
	knowledge := domain.TargetKnowledge{TargetID: target.ID, TargetName: target.Name, Title: page.Title, URL: page.ContentURLs.Desktop.Page, Summary: page.Extract, ImageURL: page.Thumbnail.Source, Source: "wikipedia", Status: "live", FetchedAt: time.Now().UTC()}
	if knowledge.Title == "" {
		knowledge.Title = target.Name
	}
	if err := knowledge.Validate(); err != nil {
		return domain.TargetKnowledge{}, err
	}
	return knowledge, nil
}

func (p *WikipediaProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return http.DefaultClient
}

type wikipediaSummary struct {
	Title     string `json:"title"`
	Extract   string `json:"extract"`
	Thumbnail struct {
		Source string `json:"source"`
	} `json:"thumbnail"`
	ContentURLs struct {
		Desktop struct {
			Page string `json:"page"`
		} `json:"desktop"`
	} `json:"content_urls"`
}
