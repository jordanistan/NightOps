package config

import "testing"

func TestDefaultsValidate(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestInvalidOrigin(t *testing.T) {
	cfg := Defaults()
	cfg.Origin.Mode = "unknown"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
