package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jordanistan/nightops/internal/geocoding"
)

var zipPattern = regexp.MustCompile(`^\d{5}(?:-\d{4})?$`)

// NominatimProvider resolves US ZIP codes through a Nominatim-compatible
// structured search endpoint. It does not store results; callers own caching.
type NominatimProvider struct {
	Endpoint string
	Client   *http.Client
	Timeout  time.Duration
}

// NewNominatimProvider creates a bounded ZIP geocoder.
func NewNominatimProvider(endpoint string, timeout time.Duration) *NominatimProvider {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &NominatimProvider{Endpoint: strings.TrimRight(strings.TrimSpace(endpoint), "?&"), Timeout: timeout, Client: &http.Client{Timeout: timeout}}
}

// Resolve looks up one US ZIP code using Nominatim's structured search API.
func (p *NominatimProvider) Resolve(ctx context.Context, zip string) (geocoding.Result, error) {
	zip = strings.TrimSpace(zip)
	if !zipPattern.MatchString(zip) {
		return geocoding.Result{}, fmt.Errorf("invalid ZIP %q", zip)
	}
	endpoint, err := url.Parse(p.Endpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return geocoding.Result{}, fmt.Errorf("invalid geocoding endpoint %q", p.Endpoint)
	}
	query := endpoint.Query()
	query.Set("postalcode", zip)
	query.Set("countrycodes", "us")
	query.Set("format", "jsonv2")
	query.Set("limit", "1")
	endpoint.RawQuery = query.Encode()
	requestContext, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return geocoding.Result{}, fmt.Errorf("create geocoding request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "NightOps/0.1 offline-first astronomy mission planner")
	response, err := p.client().Do(request)
	if err != nil {
		return geocoding.Result{}, fmt.Errorf("geocoding request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return geocoding.Result{}, fmt.Errorf("geocoding returned HTTP %d", response.StatusCode)
	}
	var results []nominatimResult
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&results); err != nil {
		return geocoding.Result{}, fmt.Errorf("decode geocoding response: %w", err)
	}
	if len(results) == 0 {
		return geocoding.Result{}, fmt.Errorf("no coordinate result found for ZIP %s", zip)
	}
	latitude, err := strconv.ParseFloat(results[0].Latitude, 64)
	if err != nil {
		return geocoding.Result{}, fmt.Errorf("decode geocoding latitude: %w", err)
	}
	longitude, err := strconv.ParseFloat(results[0].Longitude, 64)
	if err != nil {
		return geocoding.Result{}, fmt.Errorf("decode geocoding longitude: %w", err)
	}
	result := geocoding.Result{ZIP: zip, Label: results[0].DisplayName, Latitude: latitude, Longitude: longitude, RetrievedAt: time.Now().UTC()}
	if err := result.Validate(); err != nil {
		return geocoding.Result{}, err
	}
	return result, nil
}

func (p *NominatimProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return http.DefaultClient
}

type nominatimResult struct {
	DisplayName string `json:"display_name"`
	Latitude    string `json:"lat"`
	Longitude   string `json:"lon"`
}
