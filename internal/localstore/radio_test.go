package localstore

import (
	"errors"
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/radio"
)

func TestRadioStationsEncryptedAndEmptyPersists(t *testing.T) {
	store := newMigratedStore(t)
	stations := []radio.Station{{Name: "Private Station", URL: "https://example.test/secret", Preset: true}}
	if err := store.SaveRadioStations(stations); !errors.Is(err, errLocked) {
		t.Fatalf("locked write: %v", err)
	}
	if err := store.CreateVault("pw"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRadioStations(stations); err != nil {
		t.Fatal(err)
	}
	var blob string
	if err := store.db.QueryRow(`select payload from radio_stations where id = 1`).Scan(&blob); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(blob, "Private Station") || strings.Contains(blob, "secret") {
		t.Fatal("plaintext station stored")
	}
	got, found, err := store.RadioStations()
	if err != nil || !found || len(got) != 1 || got[0] != stations[0] {
		t.Fatalf("read: %#v %v %v", got, found, err)
	}
	if err := store.SaveRadioStations(nil); err != nil {
		t.Fatal(err)
	}
	got, found, err = store.RadioStations()
	if err != nil || !found || len(got) != 0 {
		t.Fatal("empty list must not reload defaults")
	}
}

func TestHot4MigratesLegacyAndPreservesEmptySlots(t *testing.T) {
	store := newMigratedStore(t)
	if err := store.CreateVault("pw"); err != nil {
		t.Fatal(err)
	}
	stations := radio.Defaults()
	for i := range stations {
		stations[i].HotSlot = 0
	}
	blob, err := store.encryptedJSON(map[string]any{"stations": stations})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`insert into radio_stations (id,payload) values (1,?)`, blob); err != nil {
		t.Fatal(err)
	}
	got, _, err := store.RadioStations()
	if err != nil || got[0].HotSlot != 1 || got[3].HotSlot != 4 || got[4].HotSlot != 0 {
		t.Fatal("legacy migration failed", err)
	}
	got[1].HotSlot = 0
	if err := store.SaveRadioStations(got); err != nil {
		t.Fatal(err)
	}
	restored, _, err := store.RadioStations()
	if err != nil || restored[1].HotSlot != 0 || restored[2].HotSlot != 3 {
		t.Fatal("empty slot was refilled", err)
	}
	restored[1].HotSlot = 1
	if err := store.SaveRadioStations(restored); err == nil {
		t.Fatal("duplicate slots accepted")
	}
	restored[1].HotSlot = 5
	if err := store.SaveRadioStations(restored); err == nil {
		t.Fatal("invalid slot accepted")
	}
}
