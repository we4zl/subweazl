package directory

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/bprendie/subweazl/internal/security"
)

type somaPlaylist struct {
	URL     string `json:"url"`
	Format  string `json:"format"`
	Quality string `json:"quality"`
}

func (c Client) SomaFM(ctx context.Context) ([]Station, error) {
	body, err := c.getString(ctx, "https://api.somafm.com/channels.json")
	if err != nil {
		return nil, err
	}

	var payload struct {
		Channels []struct {
			Title       string         `json:"title"`
			Description string         `json:"description"`
			Genre       string         `json:"genre"`
			Listeners   int            `json:"listeners,string"`
			LastPlaying string         `json:"lastPlaying"`
			Playlists   []somaPlaylist `json:"playlists"`
		} `json:"channels"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, err
	}

	out := make([]Station, 0, len(payload.Channels))
	for _, ch := range payload.Channels {
		stream := firstPlaylist(ch.Playlists)
		if stream == "" {
			continue
		}
		out = append(out, Station{
			Name:        security.SanitizeStationName(ch.Title),
			Description: security.SanitizeDescription(ch.Description),
			Genre:       security.SanitizeGenre(ch.Genre),
			URL:         security.SanitizeURL(stream),
			Source:      "SomaFM",
			Listeners:   ch.Listeners,
			NowPlaying:  security.SanitizeNowPlaying(ch.LastPlaying),
		})
	}
	return out, nil
}

func firstPlaylist(playlists []somaPlaylist) string {
	for _, pref := range []string{"mp3", "aac"} {
		for _, p := range playlists {
			if strings.EqualFold(p.Format, pref) && p.URL != "" {
				return p.URL
			}
		}
	}
	if len(playlists) > 0 {
		return playlists[0].URL
	}
	return ""
}
