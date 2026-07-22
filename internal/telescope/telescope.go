// Package telescope defines the narrow control port used by optional telescope adapters.
package telescope

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
)

// Status describes the last observed state of a telescope device.
type Status struct {
	Name      string
	Connected bool
}

// Provider controls a telescope without exposing a vendor-specific protocol to the application.
type Provider interface {
	Status(context.Context) (Status, error)
	SlewTo(context.Context, float64, float64) error
}

// CoordinateSlewProvider can slew using the observer's actual mission origin.
// Providers that require observer coordinates, such as Dwarf, implement this
// optional interface while retaining the narrow Provider contract.
type CoordinateSlewProvider interface {
	Provider
	SlewToAt(context.Context, float64, float64, float64, float64) error
}

// AlpacaProvider controls an ASCOM Alpaca-compatible telescope device.
// It never connects or moves a device until one of its methods is called.
type AlpacaProvider struct {
	Endpoint     string
	DeviceNumber int
	Client       *http.Client
	Timeout      time.Duration
}

// NewAlpacaProvider creates a bounded Alpaca adapter for one telescope device.
func NewAlpacaProvider(endpoint string, deviceNumber int, timeout time.Duration) *AlpacaProvider {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	if deviceNumber < 0 {
		deviceNumber = 0
	}
	return &AlpacaProvider{Endpoint: strings.TrimRight(endpoint, "/"), DeviceNumber: deviceNumber, Timeout: timeout, Client: &http.Client{Timeout: timeout}}
}

// Status reads the device name and connection state from the Alpaca endpoint.
func (p *AlpacaProvider) Status(ctx context.Context) (Status, error) {
	connected, err := p.getValue(ctx, "connected")
	if err != nil {
		return Status{}, err
	}
	var name string
	nameValue, err := p.getValue(ctx, "name")
	if err != nil {
		return Status{}, err
	}
	if err := json.Unmarshal(nameValue, &name); err != nil {
		return Status{}, fmt.Errorf("decode telescope name: %w", err)
	}
	var isConnected bool
	if err := json.Unmarshal(connected, &isConnected); err != nil {
		return Status{}, fmt.Errorf("decode telescope connection state: %w", err)
	}
	return Status{Name: name, Connected: isConnected}, nil
}

// SlewTo requests an asynchronous slew to right ascension in hours and declination in degrees.
func (p *AlpacaProvider) SlewTo(ctx context.Context, rightAscension, declination float64) error {
	if rightAscension < 0 || rightAscension >= 24 {
		return fmt.Errorf("right ascension must be in [0, 24), got %.4f", rightAscension)
	}
	if declination < -90 || declination > 90 {
		return fmt.Errorf("declination must be between -90 and 90, got %.4f", declination)
	}
	values := url.Values{}
	values.Set("RightAscension", strconv.FormatFloat(rightAscension, 'f', 8, 64))
	values.Set("Declination", strconv.FormatFloat(declination, 'f', 8, 64))
	_, err := p.request(ctx, http.MethodPut, "slewtocoordinatesasync", values)
	return err
}

func (p *AlpacaProvider) getValue(ctx context.Context, operation string) (json.RawMessage, error) {
	return p.request(ctx, http.MethodGet, operation, nil)
}

func (p *AlpacaProvider) request(ctx context.Context, method, operation string, values url.Values) (json.RawMessage, error) {
	base, err := url.Parse(p.Endpoint)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("invalid telescope endpoint %q", p.Endpoint)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/api/v1/telescope/" + strconv.Itoa(p.DeviceNumber) + "/" + operation
	var body io.Reader
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	requestContext, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, method, base.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create telescope request: %w", err)
	}
	if values != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	response, err := p.client().Do(request)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("telescope request: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read telescope response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("telescope returned HTTP %d", response.StatusCode)
	}
	var envelope alpacaResponse
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode telescope response: %w", err)
	}
	if envelope.ErrorNumber != 0 || envelope.ErrorMessage != "" {
		return nil, fmt.Errorf("telescope error %d: %s", envelope.ErrorNumber, envelope.ErrorMessage)
	}
	return envelope.Value, nil
}

func (p *AlpacaProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return http.DefaultClient
}

type alpacaResponse struct {
	ErrorNumber  int             `json:"ErrorNumber"`
	ErrorMessage string          `json:"ErrorMessage"`
	Value        json.RawMessage `json:"Value"`
}
