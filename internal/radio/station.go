// Package radio holds Internet radio stations independently of the music queue.
package radio

import (
	"embed"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"unicode"

	"github.com/bprendie/subweazl/internal/security"
)

//go:embed presets.json
var presets embed.FS

const MaxPresets = 8

type Station struct {
	HotSlot int    `json:"hot_slot,omitempty"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	Preset  bool   `json:"preset"`
}

func Defaults() []Station {
	data, _ := presets.ReadFile("presets.json")
	var stations []Station
	_ = json.Unmarshal(data, &stations)
	SeedHot4(stations)
	return stations
}

func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return errors.New("enter an HTTP or HTTPS stream/playlist URL")
	}
	return nil
}

func Normalize(st Station) (Station, error) {
	if st.HotSlot < 0 || st.HotSlot > 4 {
		return Station{}, errors.New("Hot 4 slot must be between 1 and 4")
	}
	st.URL = strings.TrimSpace(st.URL)
	if err := ValidateURL(st.URL); err != nil {
		return Station{}, err
	}
	st.Name = security.SanitizeStationName(st.Name)
	if st.Name == "" {
		u, _ := url.Parse(st.URL)
		st.Name = u.Hostname()
	}
	return st, nil
}

// Upsert keeps URL paths case-sensitive and updates a station in place.
func Upsert(stations []Station, st Station) ([]Station, error) {
	st, err := Normalize(st)
	if err != nil {
		return nil, err
	}
	next := append([]Station(nil), stations...)
	index, count := -1, 0
	for i, existing := range next {
		if existing.URL == st.URL {
			index = i
		} else if existing.Preset {
			count++
		}
	}
	if st.Preset && count >= MaxPresets {
		return nil, errors.New("eight presets full; remove a preset first with ctrl+p")
	}
	if index >= 0 {
		next[index] = st
	} else {
		next = append(next, st)
	}
	return next, nil
}
