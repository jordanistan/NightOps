package telescope

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDwarfSlewUsesDocumentedJSONCommand(t *testing.T) {
	socket := &fakeDwarfSocket{}
	provider := NewDwarfProvider("192.168.88.1", time.Second)
	provider.dial = func(context.Context, string) (dwarfSocket, error) { return socket, nil }

	if err := provider.SlewToAt(context.Background(), 5.5, -12.25, 30.25, -97.75); err != nil {
		t.Fatalf("Dwarf slew failed: %v", err)
	}
	if socket.messageType != 1 {
		t.Fatalf("message type = %d, want text", socket.messageType)
	}
	var command map[string]any
	if err := json.Unmarshal(socket.payload, &command); err != nil {
		t.Fatalf("decode command: %v", err)
	}
	if command["interface"] != float64(dwarfStartGoto) || command["camId"] != float64(dwarfTelephotoCamera) {
		t.Fatalf("command identifiers = %+v", command)
	}
	if command["lat"] != 30.25 || command["lon"] != -97.75 || command["ra"] != "5.50000000" || command["dec"] != "-12.25000000" {
		t.Fatalf("command coordinates = %+v", command)
	}
	if command["date"] == "" || command["path"] == "" {
		t.Fatalf("command metadata missing: %+v", command)
	}
}

func TestDwarfSlewRejectsMissingOriginCoordinates(t *testing.T) {
	provider := NewDwarfProvider("192.168.88.1", time.Second)
	if err := provider.SlewTo(context.Background(), 5.5, -12.25); err == nil {
		t.Fatal("Dwarf accepted a slew without observer coordinates")
	}
}

func TestDwarfStatusUsesLocalWebSocketEndpoint(t *testing.T) {
	provider := NewDwarfProvider("192.168.88.1", time.Second)
	var endpoint string
	provider.dial = func(_ context.Context, address string) (dwarfSocket, error) {
		endpoint = address
		return &fakeDwarfSocket{}, nil
	}
	status, err := provider.Status(context.Background())
	if err != nil || !status.Connected || status.Name != "Dwarf II" {
		t.Fatalf("status = %+v err=%v", status, err)
	}
	if endpoint != "ws://192.168.88.1:9900" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

type fakeDwarfSocket struct {
	messageType int
	payload     []byte
}

func (s *fakeDwarfSocket) WriteMessage(messageType int, payload []byte) error {
	s.messageType = messageType
	s.payload = append([]byte(nil), payload...)
	return nil
}

func (*fakeDwarfSocket) Close() error { return nil }

func TestAlpacaStatusAndSlew(t *testing.T) {
	var slewValues url.Values
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var payload string
		switch r.URL.Path {
		case "/api/v1/telescope/2/connected":
			payload = `{"Value":true,"ErrorNumber":0,"ErrorMessage":""}`
		case "/api/v1/telescope/2/name":
			payload = `{"Value":"Dwarf Test","ErrorNumber":0,"ErrorMessage":""}`
		case "/api/v1/telescope/2/slewtocoordinatesasync":
			if r.Method != http.MethodPut {
				t.Fatalf("slew method = %s", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse slew form: %v", err)
			}
			slewValues = r.Form
			payload = `{"Value":true,"ErrorNumber":0,"ErrorMessage":""}`
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		return response(payload), nil
	})
	provider := NewAlpacaProvider("http://telescope.test", 2, time.Second)
	provider.Client = &http.Client{Transport: transport}

	status, err := provider.Status(context.Background())
	if err != nil || status.Name != "Dwarf Test" || !status.Connected {
		t.Fatalf("status = %+v err=%v", status, err)
	}
	if err := provider.SlewTo(context.Background(), 5.5, -12.25); err != nil {
		t.Fatalf("slew failed: %v", err)
	}
	if slewValues.Get("RightAscension") != "5.50000000" || slewValues.Get("Declination") != "-12.25000000" {
		t.Fatalf("slew coordinates = %v", slewValues)
	}
}

func TestAlpacaRejectsInvalidCoordinatesWithoutRequest(t *testing.T) {
	requests := 0
	provider := NewAlpacaProvider("http://telescope.test", 0, time.Second)
	provider.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { requests++; return response(`{"Value":true}`), nil })}
	if err := provider.SlewTo(context.Background(), 24, 0); err == nil {
		t.Fatal("invalid right ascension was accepted")
	}
	if err := provider.SlewTo(context.Background(), 1, 91); err == nil {
		t.Fatal("invalid declination was accepted")
	}
	if requests != 0 {
		t.Fatalf("invalid coordinates made %d requests", requests)
	}
}

func TestAlpacaReportsProviderErrors(t *testing.T) {
	provider := NewAlpacaProvider("http://telescope.test", 0, time.Second)
	provider.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/connected") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		return response(`{"Value":false,"ErrorNumber":1025,"ErrorMessage":"not connected"}`), nil
	})}
	_, err := provider.Status(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("provider error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func response(payload string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}
}
