package targets

import (
	"strings"
	"testing"
)

func TestLoadEmbeddedCatalog(t *testing.T) {
	catalog, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Version != "targets-v1" || len(catalog.Targets) != 3 {
		t.Fatalf("unexpected target catalog: %+v", catalog)
	}
	if target, ok := catalog.Find("m31"); !ok || target.Name != "Andromeda Galaxy" || target.Source == "" {
		t.Fatalf("target lookup failed: %+v", target)
	}
}

func TestParseRejectsInvalidTarget(t *testing.T) {
	_, err := ParseCSV(strings.NewReader("id,name,kind,right_ascension_deg,declination_deg,source\ninvalid,Target,star,400,0,fixture\n"), "test")
	if err == nil {
		t.Fatal("expected invalid right ascension error")
	}
}
