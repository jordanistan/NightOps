package console

import "testing"

func TestThemeForNameSelectsSupportedPalettes(t *testing.T) {
	mission := ThemeForName("mission-control")
	observatory := ThemeForName("observatory")
	if mission.Accent == observatory.Accent {
		t.Fatalf("supported themes share the same accent: %q", mission.Accent)
	}
	if fallback := ThemeForName("unknown"); fallback.Accent != mission.Accent {
		t.Fatal("unknown theme did not fall back to mission control")
	}
}
