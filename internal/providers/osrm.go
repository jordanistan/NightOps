package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jordanistan/nightops/internal/astronomy"
	"github.com/jordanistan/nightops/internal/routing"
)

// OSRMProvider retrieves driving routes from an OSRM-compatible HTTP service.
type OSRMProvider struct {
	Endpoint     string
	Client       *http.Client
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
	Sleep        func(context.Context, time.Duration) error
}

// NewOSRMProvider creates an OSRM adapter with a bounded request timeout.
func NewOSRMProvider(endpoint string) *OSRMProvider {
	return NewOSRMProviderWithPolicy(endpoint, 8*time.Second, 0, 0)
}

// NewOSRMProviderWithPolicy creates an OSRM adapter with bounded retries.
// Retries are reserved for transport failures and transient HTTP responses;
// malformed or semantically invalid provider responses fail immediately.
func NewOSRMProviderWithPolicy(endpoint string, timeout time.Duration, maxRetries int, backoff time.Duration) *OSRMProvider {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	if maxRetries < 0 {
		maxRetries = 0
	}
	if backoff < 0 {
		backoff = 0
	}
	return &OSRMProvider{Endpoint: strings.TrimRight(endpoint, "/"), Client: &http.Client{Timeout: timeout}, Timeout: timeout, MaxRetries: maxRetries, RetryBackoff: backoff, Sleep: waitForRetry}
}

// Route requests the fastest driving route between two coordinates.
func (p *OSRMProvider) Route(ctx context.Context, from, to astronomy.Coordinates) (routing.Plan, error) {
	if err := from.Validate(); err != nil {
		return routing.Plan{}, err
	}
	if err := to.Validate(); err != nil {
		return routing.Plan{}, err
	}
	base, err := url.Parse(p.Endpoint)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return routing.Plan{}, fmt.Errorf("invalid routing endpoint %q", p.Endpoint)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/route/v1/driving/" + coordinate(from.Longitude) + "," + coordinate(from.Latitude) + ";" + coordinate(to.Longitude) + "," + coordinate(to.Latitude)
	query := base.Query()
	query.Set("overview", "false")
	query.Set("steps", "false")
	base.RawQuery = query.Encode()
	var lastErr error
	for attempt := 0; attempt <= p.MaxRetries; attempt++ {
		plan, retry, err := p.fetch(ctx, base.String())
		if err == nil {
			return plan, nil
		}
		lastErr = err
		if !retry || attempt == p.MaxRetries {
			break
		}
		backoff := p.RetryBackoff * time.Duration(1<<attempt)
		if err := p.sleep(ctx, backoff); err != nil {
			return routing.Plan{}, err
		}
	}
	return routing.Plan{}, lastErr
}

func (p *OSRMProvider) fetch(ctx context.Context, endpoint string) (routing.Plan, bool, error) {
	requestContext := ctx
	cancel := func() {}
	if p.Timeout > 0 {
		requestContext, cancel = context.WithTimeout(ctx, p.Timeout)
	}
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint, nil)
	if err != nil {
		return routing.Plan{}, false, fmt.Errorf("create routing request: %w", err)
	}
	response, err := p.client().Do(request)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return routing.Plan{}, false, ctx.Err()
		}
		return routing.Plan{}, true, fmt.Errorf("fetch route: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return routing.Plan{}, true, fmt.Errorf("read route response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return routing.Plan{}, response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500, fmt.Errorf("routing provider returned HTTP %d", response.StatusCode)
	}
	var payload osrmResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return routing.Plan{}, false, fmt.Errorf("decode route response: %w", err)
	}
	if payload.Code != "Ok" || len(payload.Routes) == 0 {
		return routing.Plan{}, false, fmt.Errorf("routing provider returned %s", nonEmpty(payload.Code, "no route"))
	}
	selected := payload.Routes[0]
	plan := routing.Plan{FromLabel: "origin", ToLabel: "destination", DistanceKm: selected.Distance / 1000, DistanceMi: selected.Distance / 1609.344, TravelTime: formatDuration(selected.Duration), Provider: "osrm"}
	if err := plan.Validate(); err != nil {
		return routing.Plan{}, false, err
	}
	return plan, false, nil
}

func (p *OSRMProvider) sleep(ctx context.Context, duration time.Duration) error {
	if p.Sleep != nil {
		return p.Sleep(ctx, duration)
	}
	return waitForRetry(ctx, duration)
}

func waitForRetry(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *OSRMProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return http.DefaultClient
}

func nonEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func coordinate(value float64) string { return strconv.FormatFloat(value, 'f', 6, 64) }

func formatDuration(seconds float64) string {
	minutes := int(seconds / 60)
	if minutes < 1 {
		return "under 1 minute"
	}
	hours := minutes / 60
	minutes %= 60
	if hours == 0 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	if minutes == 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	return fmt.Sprintf("%d hours %d minutes", hours, minutes)
}

type osrmResponse struct {
	Code   string `json:"code"`
	Routes []struct {
		Distance float64 `json:"distance"`
		Duration float64 `json:"duration"`
	} `json:"routes"`
}
