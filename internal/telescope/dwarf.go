package telescope

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	dwarfTelephotoCamera = 0
	dwarfStartGoto       = 11203
)

// DwarfProvider controls a Dwarf II telescope over its documented local
// WebSocket API. The adapter sends commands only when a method is called.
type DwarfProvider struct {
	Endpoint string
	Timeout  time.Duration
	dial     func(context.Context, string) (dwarfSocket, error)
}

type dwarfSocket interface {
	WriteMessage(int, []byte) error
	Close() error
}

type dwarfDialer func(context.Context, string) (dwarfSocket, error)

// NewDwarfProvider creates a bounded Dwarf II adapter for a device endpoint.
// Endpoint may be an IP address, host name, or complete ws:// URL. Bare hosts
// use the Dwarf II local API port 9900.
func NewDwarfProvider(endpoint string, timeout time.Duration) *DwarfProvider {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &DwarfProvider{Endpoint: dwarfWebSocketURL(endpoint), Timeout: timeout, dial: defaultDwarfDial}
}

// Status verifies that the Dwarf WebSocket endpoint accepts a connection.
// It does not claim that the telescope is tracking or ready for exposure.
func (p *DwarfProvider) Status(ctx context.Context) (Status, error) {
	socket, err := p.connect(ctx)
	if err != nil {
		return Status{}, err
	}
	if err := socket.Close(); err != nil {
		return Status{}, fmt.Errorf("close Dwarf connection: %w", err)
	}
	return Status{Name: "Dwarf II", Connected: true}, nil
}

// SlewTo requests a slew without observer coordinates. Dwarf requires those
// coordinates, so this method returns an actionable error instead of guessing.
func (p *DwarfProvider) SlewTo(context.Context, float64, float64) error {
	return fmt.Errorf("Dwarf slew requires mission-origin latitude and longitude")
}

// SlewToAt requests an asynchronous Dwarf goto using the supplied mission
// origin. Coordinates are never inferred or replaced with fabricated values.
func (p *DwarfProvider) SlewToAt(ctx context.Context, rightAscension, declination, latitude, longitude float64) error {
	if rightAscension < 0 || rightAscension >= 24 {
		return fmt.Errorf("right ascension must be in [0, 24), got %.4f", rightAscension)
	}
	if declination < -90 || declination > 90 {
		return fmt.Errorf("declination must be between -90 and 90, got %.4f", declination)
	}
	if latitude < -90 || latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90, got %.4f", latitude)
	}
	if longitude < -180 || longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180, got %.4f", longitude)
	}
	now := time.Now().Local()
	command := map[string]any{
		"interface": dwarfStartGoto,
		"camId":     dwarfTelephotoCamera,
		"lon":       longitude,
		"lat":       latitude,
		"date":      now.Format("2006-01-02 15:04:05"),
		"path":      "DWARF_GOTO_" + now.Format("20060102150405"),
		"ra":        strconv.FormatFloat(rightAscension, 'f', 8, 64),
		"dec":       strconv.FormatFloat(declination, 'f', 8, 64),
	}
	data, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("encode Dwarf goto: %w", err)
	}
	socket, err := p.connect(ctx)
	if err != nil {
		return err
	}
	defer socket.Close()
	if err := socket.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("send Dwarf goto: %w", err)
	}
	return nil
}

func (p *DwarfProvider) connect(ctx context.Context) (dwarfSocket, error) {
	if p == nil || p.dial == nil || strings.TrimSpace(p.Endpoint) == "" {
		return nil, fmt.Errorf("Dwarf endpoint is not configured")
	}
	requestContext, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	socket, err := p.dial(requestContext, p.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("connect to Dwarf: %w", err)
	}
	return socket, nil
}

func defaultDwarfDial(ctx context.Context, endpoint string) (dwarfSocket, error) {
	socket, _, err := websocket.DefaultDialer.DialContext(ctx, endpoint, nil)
	return socket, err
}

func dwarfWebSocketURL(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Scheme != "" {
		return endpoint
	}
	if _, _, err := net.SplitHostPort(endpoint); err == nil {
		return "ws://" + endpoint
	}
	return "ws://" + strings.TrimRight(endpoint, "/") + ":9900"
}
