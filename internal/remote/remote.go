package remote

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Command string

const (
	Radio1    Command = "radio-1"
	Radio2    Command = "radio-2"
	Radio3    Command = "radio-3"
	Radio4    Command = "radio-4"
	Toggle    Command = "toggle"
	Next      Command = "next"
	Previous  Command = "previous"
	Stop      Command = "stop"
	CycleMode Command = "mode"
	Quit      Command = "quit"
	Visible   Command = "visible"
	Hidden    Command = "hidden"
)

type RadioPreset struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func (c Command) RadioSlot() int {
	switch c {
	case Radio1:
		return 1
	case Radio2:
		return 2
	case Radio3:
		return 3
	case Radio4:
		return 4
	}
	return 0
}

func (c Command) RadioMove() (int, int, bool) {
	parts := strings.Split(string(c), "-")
	if len(parts) != 4 || parts[0] != "radio" || parts[1] != "move" || len(parts[2]) != 1 || len(parts[3]) != 1 {
		return 0, 0, false
	}
	from, to := int(parts[2][0]-'0'), int(parts[3][0]-'0')
	return from, to, from >= 1 && from <= 4 && to >= 1 && to <= 4
}

type Snapshot struct {
	Hot4Ready    bool           `json:"hot4_ready"`
	RadioPresets [4]RadioPreset `json:"radio_presets"`
	Running      bool           `json:"running"`
	State        string         `json:"state"`
	Title        string         `json:"title,omitempty"`
	Artist       string         `json:"artist,omitempty"`
	Album        string         `json:"album,omitempty"`
	Duration     int            `json:"duration,omitempty"`
	PlaybackMode string         `json:"playback_mode"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

const staleAfter = 75 * time.Second

func StatePath() (string, error) {
	root := os.Getenv("XDG_STATE_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(root, "subweazl", "remote.json"), nil
}

func SocketPath() (string, error) {
	root := os.Getenv("XDG_RUNTIME_DIR")
	if root == "" {
		root = filepath.Join(os.TempDir(), "subweazl-"+os.Getenv("USER"))
	}
	return filepath.Join(root, "subweazl", "control.sock"), nil
}

func WriteSnapshot(snapshot Snapshot) error {
	path, err := StatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	snapshot.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".remote-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func ReadSnapshot() (Snapshot, error) {
	path, err := StatePath()
	if err != nil {
		return Snapshot{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Running && time.Since(snapshot.UpdatedAt) > staleAfter {
		return snapshot, errors.New("Subweazl status is stale")
	}
	return snapshot, nil
}
