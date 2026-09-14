package tui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bprendie/subweazl/internal/directory"
	"github.com/bprendie/subweazl/internal/playqueue"
	"github.com/bprendie/subweazl/internal/radio"
	"github.com/bprendie/subweazl/internal/remote"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
)

func fakeRadioPlayer(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell mpv fixture")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mpv"), []byte("#!/bin/sh\nexec sleep 60\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestRadioControlsPersistAndRestoreStations(t *testing.T) {
	m := newHomeTestModel(t)
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("6")})
	if m.mode != modeRadio || len(m.radioStations) != 7 {
		t.Fatal("radio entry/defaults missing")
	}
	m, _ = m.radioAction("Add URL")
	m.input.SetValue("https://example.test/live")
	m, _ = m.handleEnter()
	if m.radioInput != "name" {
		t.Fatal("URL should lead to name prompt")
	}
	m.input.SetValue("My station")
	m, _ = m.handleEnter()
	if m.input.Focused() || len(m.radioStations) != 8 {
		t.Fatal("station not saved")
	}
	m.list.Select(len(m.list.Items()) - 1)
	m, _, _ = m.handleRadioKey("ctrl+p")
	if !m.radioStations[7].Preset {
		t.Fatal("preset not promoted")
	}
	m.list.Select(len(m.list.Items()) - 1)
	m, _, _ = m.handleRadioKey("[")
	if m.radioStations[6].Name != "My station" {
		t.Fatal("reorder failed")
	}
	m, _, _ = m.handleRadioKey("ctrl+r")
	m.input.SetValue("Renamed")
	m, _ = m.handleEnter()
	if m.radioStations[6].Name != "Renamed" {
		t.Fatal("rename failed")
	}
	m.radioLoaded = false
	m.showRadio("My stations")
	if m.radioStations[6].Name != "Renamed" {
		t.Fatal("station not restored")
	}
	m.list.Select(11)
	m, _, _ = m.handleRadioKey("delete")
	if !strings.Contains(m.statusLine(), "Renamed") {
		t.Fatal("delete prompt must name target")
	}
	m.input.SetValue("n")
	m, _ = m.handleEnter()
	if len(m.radioStations) != 8 {
		t.Fatal("cancel deleted station")
	}
	m, _, _ = m.handleRadioKey("delete")
	m.input.SetValue("y")
	m, _ = m.handleEnter()
	if len(m.radioStations) != 7 {
		t.Fatal("confirmed deletion failed")
	}
}

func TestRadioDirectoryIgnoresStaleResults(t *testing.T) {
	m := newHomeTestModel(t)
	m.showRadio("SomaFM")
	request := m.radioDirectoryRequest
	m.showRadio("Presets")
	before := m.list.Items()
	m.applyRadioDirectory(radioDirectoryMsg{request: request, stations: []directory.Station{{Name: "stale", URL: "https://example.test/live"}}})
	if !reflect.DeepEqual(before, m.list.Items()) {
		t.Fatal("stale directory replaced preset view")
	}
	m.applyRadioDirectory(radioDirectoryMsg{request: m.radioDirectoryRequest, stations: []directory.Station{{Name: "bad", URL: "file:///tmp/audio"}, {Name: "valid", URL: "https://example.test/live"}}})
	if len(m.list.Items()) != len(before)+1 {
		t.Fatal("invalid directory URL accepted")
	}
}

func TestRadioPlaybackPreservesQueueAndRejectsLatePlayback(t *testing.T) {
	fakeRadioPlayer(t)
	m := newHomeTestModel(t)
	defer m.player.Stop()
	m.queue = playqueue.FromSnapshot(playqueue.Snapshot{Tracks: []subsonic.Track{testTrack("one"), testTrack("two")}, Current: 0})
	before := m.queue.Snapshot()
	station := radio.Station{Name: "Live station", URL: "https://example.test/live"}
	m, cmd := m.startRadio(station)
	resolved := cmd().(radioResolvedMsg)
	m.applyRadioResolved(resolved)
	if m.radioPlaying == nil || m.playing.ID != "" || !reflect.DeepEqual(before, m.queue.Snapshot()) {
		t.Fatal("radio changed library queue")
	}
	if _, ok := m.selectedOrPlayingTrack(); ok {
		t.Fatal("radio became a library track")
	}
	m, _ = m.generateMoodPlaylist()
	if !strings.Contains(m.err, "library track") {
		t.Fatal("Mood accepted radio")
	}
	m, _ = m.playNext()
	if m.radioPlaying != nil || m.playing.ID != "one" {
		t.Fatal("next skipped saved library position")
	}
	m.applyRadioResolved(resolved)
	if m.radioPlaying != nil {
		t.Fatal("old radio completion replaced library playback")
	}
	m, cmd = m.startRadio(station)
	resolved = cmd().(radioResolvedMsg)
	m.stop()
	m.applyRadioResolved(resolved)
	if m.isPlaying() {
		t.Fatal("late resolution restarted stopped playback")
	}
}

func TestRadioEndDoesNotAdvanceOrScrobble(t *testing.T) {
	m := newHomeTestModel(t)
	m.queue = playqueue.FromSnapshot(playqueue.Snapshot{Tracks: []subsonic.Track{testTrack("one"), testTrack("two")}, Current: 0})
	m.radioPlaying = &radio.Station{Name: "Live"}
	m.playing = &subsonic.Track{Title: "Live"}
	m.playerStarted = true
	next, _ := m.Update(tickMsg(time.Now()))
	m = next.(Model)
	if m.isPlaying() || m.queue.CurrentIndex() != 0 || !strings.Contains(m.err, "radio stream ended") {
		t.Fatal("ended radio advanced the queue")
	}
	if m.scrobble("", true)() != nil {
		t.Fatal("radio produced a scrobble")
	}
}

func TestRadioFailureKeepsExistingPlayback(t *testing.T) {
	m := newHomeTestModel(t)
	m.playing = &subsonic.Track{ID: "existing", Title: "Existing"}
	m.radioPending = true
	m.applyRadioResolved(radioResolvedMsg{request: m.radioRequest, err: errors.New("unavailable")})
	if m.playing.ID != "existing" || m.radioPending || m.err != "unavailable" {
		t.Fatal("lookup failure changed playback")
	}
}

func TestRadioInputEscapeAndStaleMetadata(t *testing.T) {
	m := newHomeTestModel(t)
	m.showRadio("Presets")
	m, _ = m.radioAction("Add URL")
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.radioInput != "" || m.input.Focused() {
		t.Fatal("escape did not cancel URL input")
	}
	m.playing = &subsonic.Track{Title: "Current"}
	m.radioRequest = 2
	next, _ := m.Update(titleMsg{title: "Old title", request: 1})
	if next.(Model).trackTitle != "" {
		t.Fatal("stale title leaked into new playback")
	}
}

func TestHot4AssignmentKeyAndRemoteReorder(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := newHomeTestModel(t)
	m.showRadio("My stations")
	st := radio.Station{Name: "Radio Prendie", URL: "https://radio.prendie.io/radio.mp3"}
	next, err := radio.Upsert(m.radioStations, st)
	if err != nil || !m.saveRadioStations(next) {
		t.Fatal("save station", err, m.err)
	}
	m.showRadio("My stations")
	m.list.Select(len(m.list.Items()) - 1)
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	if m.radioInput != "hot4" || !strings.Contains(m.status, "Radio Prendie") || !strings.Contains(m.status, "DEF CON") {
		t.Fatal("slot picker missing station or replacement names")
	}
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.radioInput != "" || m.radioStations[len(m.radioStations)-1].HotSlot != 0 {
		t.Fatal("escape changed assignment")
	}
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if m.input.Focused() || m.radioStations[len(m.radioStations)-1].HotSlot != 1 {
		t.Fatal("H then 1 failed")
	}
	m.EnableRemote()
	view := m.mode
	result, _ := m.Update(remote.Command("radio-move-1-4"))
	m = result.(Model)
	if m.mode != view || m.isPlaying() || m.radioStations[len(m.radioStations)-1].HotSlot != 4 {
		t.Fatal("drag command changed playback/view or failed")
	}
	snapshot, err := remote.ReadSnapshot()
	if err != nil || !snapshot.Hot4Ready || snapshot.RadioPresets[3].Name != "Radio Prendie" {
		t.Fatal("widget did not receive reorder", err)
	}
	m.radioLoaded = false
	if err := m.loadRadioStations(); err != nil {
		t.Fatal(err)
	}
	if m.radioStations[len(m.radioStations)-1].HotSlot != 4 {
		t.Fatal("Hot 4 did not persist")
	}
}
