//go:build !windows

package remote

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemoteCommandsAreValid(t *testing.T) {
	for _, command := range []Command{Toggle, Next, Previous, Stop, CycleMode, Quit, Visible, Hidden, Radio1, Radio2, Radio3, Radio4} {
		if !validCommand(command) {
			t.Fatalf("command %q should be accepted", command)
		}
	}
}

func TestSnapshotRoundTripUsesPrivateAtomicFile(t *testing.T) {
	stateHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", stateHome)
	want := Snapshot{Running: true, State: "playing", Title: "Weazl Signal", Artist: "Artist", Album: "Album", Duration: 240, PlaybackMode: "shuffle"}
	if err := WriteSnapshot(want); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	got, err := ReadSnapshot()
	if err != nil || got.Title != want.Title || got.PlaybackMode != want.PlaybackMode || got.UpdatedAt.IsZero() {
		t.Fatalf("snapshot = %#v, err = %v", got, err)
	}
	path, _ := StatePath()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, err = %v", info.Mode().Perm(), err)
	}
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".remote-*"))
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func TestSnapshotRejectsStaleRunningState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := WriteSnapshot(Snapshot{Running: true, State: "playing"}); err != nil {
		t.Fatal(err)
	}
	path, _ := StatePath()
	data := []byte(`{"running":true,"state":"playing","updated_at":"2000-01-01T00:00:00Z"}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSnapshot(); err == nil {
		t.Fatal("expected stale-state error")
	}
}

func TestUnixTransportDeliversCommandAndRejectsSecondServer(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	received := make(chan Command, 1)
	server, err := Listen(func(command Command) { received <- command })
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer server.Close()
	if second, err := Listen(func(Command) {}); err == nil {
		second.Close()
		t.Fatal("expected duplicate server error")
	}
	if err := Send(Next); err != nil {
		t.Fatalf("send: %v", err)
	}
	select {
	case got := <-received:
		if got != Next {
			t.Fatalf("command = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("command was not delivered")
	}
}

func TestRadioCommandsRejectOutOfRangeSlots(t *testing.T) {
	for _, cmd := range []Command{"radio-0", "radio-5", "radio--1", "radio-1 extra"} {
		if validCommand(cmd) || cmd.RadioSlot() != 0 {
			t.Fatalf("accepted %q", cmd)
		}
	}
	for i, cmd := range []Command{Radio1, Radio2, Radio3, Radio4} {
		if cmd.RadioSlot() != i+1 {
			t.Fatalf("wrong slot for %q", cmd)
		}
	}
}

func TestRadioMoveCommandValidation(t *testing.T) {
	for _, c := range []Command{"radio-move-1-4", "radio-move-4-2", "radio-move-2-2"} {
		from, to, ok := c.RadioMove()
		if !ok || from < 1 || to > 4 || !validCommand(c) {
			t.Fatal("valid move rejected", c)
		}
	}
	for _, c := range []Command{"radio-move-0-2", "radio-move-2-5", "radio-move-11-2", "radio-move-1-2-extra", "radio-move--1-2"} {
		if _, _, ok := c.RadioMove(); ok || validCommand(c) {
			t.Fatal("invalid move accepted", c)
		}
	}
}
