package radio

import (
	"fmt"
	"testing"
)

func TestStationsEnforcePresetLimitAndPreserveURLCase(t *testing.T) {
	var stations []Station
	for i := 0; i < MaxPresets; i++ {
		var err error
		stations, err = Upsert(stations, Station{Name: "station", URL: fmt.Sprintf("https://example.test/%d", i), Preset: true})
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Upsert(stations, Station{URL: "https://example.test/extra", Preset: true}); err == nil {
		t.Fatal("ninth preset accepted")
	}
	updated, err := Upsert(stations, Station{Name: "renamed", URL: stations[0].URL, Preset: true})
	if err != nil || len(updated) != 8 || updated[0].Name != "renamed" || stations[0].Name != "station" {
		t.Fatal("upsert mutated input or failed", err)
	}
	for _, path := range []string{"Live", "live"} {
		updated, err = Upsert(updated, Station{URL: "https://example.test/" + path})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(updated) != 10 {
		t.Fatal("case-sensitive paths were merged")
	}
}

func TestStationValidation(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "--playlist=foo", "https:///no-host", "ftp://example.test/a", "https://example.test/\nstream"} {
		if _, err := Normalize(Station{URL: raw}); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	st, err := Normalize(Station{Name: "A\nB\x1b", URL: " https://example.test/live "})
	if err != nil || st.Name != "AB" {
		t.Fatalf("sanitize: %#v %v", st, err)
	}
	for _, st := range Defaults() {
		if _, err := Normalize(st); err != nil {
			t.Fatal(err)
		}
	}
}
