package tui

import (
	"encoding/json"
	"github.com/bprendie/subweazl/internal/radio"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/remote"
)

type countingPublisher struct{ calls int }

func (p *countingPublisher) Publish(remote.Snapshot) { p.calls++ }

func TestRemoteSnapshotPublishesWeazlState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := newHomeTestModel(t)
	m.EnableRemote()
	snapshot, err := remote.ReadSnapshot()
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if !snapshot.Running || snapshot.State != "idle" || snapshot.PlaybackMode != m.playbackModeLabel() {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestUnchangedRemoteStateSkipsWritesAndMediaSignals(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := New(config.Config{})
	publisher := &countingPublisher{}
	m.SetMediaPublisher(publisher)
	m.EnableRemote()
	path, err := remote.StatePath()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	m.publishRemote()
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("unchanged status rewrote the snapshot")
	}
	if publisher.calls != 1 {
		t.Fatalf("media publish calls = %d, want 1", publisher.calls)
	}
	m.remotePublished = time.Now().Add(-31 * time.Second)
	m.publishRemote()
	heartbeat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !heartbeat.ModTime().After(after.ModTime()) {
		t.Fatal("heartbeat did not refresh the snapshot")
	}
	if publisher.calls != 1 {
		t.Fatalf("heartbeat emitted media update; calls = %d", publisher.calls)
	}
}

func TestRemoteQuitClosesWeazlVault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := newHomeTestModel(t)
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		t.Fatal("test vault is not unlocked")
	}
	next, cmd := m.Update(remote.Quit)
	got := next.(Model)
	if got.vaultStore != nil {
		t.Fatal("remote quit left the Weazl vault open")
	}
	if cmd == nil {
		t.Fatal("remote quit did not return a quit command")
	}
}

func TestWidgetHot4UsesExplicitSlotsWithoutStreamURLs(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := newHomeTestModel(t)
	stations := []radio.Station{
		{Name: "Not a preset", URL: "https://example.test/skip"},
		{Name: "First", URL: "https://example.test/private-token", Preset: true, HotSlot: 1},
		{Name: "Second", URL: "https://example.test/2", Preset: true, HotSlot: 2},
		{Name: "Third", URL: "https://example.test/3", Preset: true, HotSlot: 3},
		{Name: "Fourth", URL: "https://example.test/4", Preset: true, HotSlot: 4},
		{Name: "Fifth", URL: "https://example.test/5", Preset: true},
	}
	if err := m.vaultStore.SaveRadioStations(stations); err != nil {
		t.Fatal(err)
	}
	m.radioPlaying = &stations[2]
	m.EnableRemote()
	snapshot, err := remote.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.RadioPresets[0].Name != "First" || snapshot.RadioPresets[3].Name != "Fourth" || !snapshot.RadioPresets[1].Active {
		t.Fatalf("presets: %#v", snapshot.RadioPresets)
	}
	raw, _ := json.Marshal(snapshot)
	if strings.Contains(string(raw), "https://") || strings.Contains(string(raw), "private-token") {
		t.Fatal("snapshot leaked a stream URL")
	}
	stations[1].HotSlot, stations[4].HotSlot = stations[4].HotSlot, stations[1].HotSlot
	if !m.saveRadioStations(stations) {
		t.Fatal(m.err)
	}
	snapshot, err = remote.ReadSnapshot()
	if err != nil || snapshot.RadioPresets[0].Name != "Fourth" {
		t.Fatal("reorder was not published immediately", err)
	}
}

func TestWidgetRadioCommandResolvesPresetWithoutChangingView(t *testing.T) {
	m := newHomeTestModel(t)
	m.radioLoaded = true
	m.radioStations = []radio.Station{
		{Name: "Saved only", URL: "https://example.test/saved"},
		{Name: "One", URL: "https://example.test/1", Preset: true, HotSlot: 1},
		{Name: "Two", URL: "https://example.test/2", Preset: true, HotSlot: 2},
	}
	mode := m.mode
	next, cmd := m.Update(remote.Radio2)
	m = next.(Model)
	if m.mode != mode || !m.radioPending || m.status != "connecting to Two" || cmd == nil {
		t.Fatal("remote radio selection failed")
	}
	m.radioPending = false
	m, _ = m.playRadioPreset(4)
	if !strings.Contains(m.err, "empty") || m.radioPending {
		t.Fatal("empty slot started playback")
	}
	m.closeVaultStore()
	m, _ = m.playRadioPreset(1)
	if !strings.Contains(m.err, "unlock") || m.radioPending {
		t.Fatal("locked vault accepted a preset")
	}
}
