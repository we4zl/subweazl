package radio

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// ResolveStreamURL resolves PLS/M3U wrappers. HLS stays intact for mpv to decode.
func ResolveStreamURL(ctx context.Context, raw string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many radio redirects")
		}
		return ValidateURL(req.URL.String())
	}}
	for depth := 0; depth < 5; depth++ {
		if err := ValidateURL(raw); err != nil {
			return "", err
		}
		u, _ := url.Parse(raw)
		ext := strings.ToLower(path.Ext(u.Path))
		if ext != ".pls" && ext != ".m3u" && ext != ".m3u8" {
			return raw, nil
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
		if err != nil {
			return "", err
		}
		res, err := client.Do(req)
		if err != nil {
			return "", errors.New("could not reach radio playlist")
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
		res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return "", errors.New("radio playlist server is unavailable")
		}
		if readErr != nil || len(body) > 1<<20 {
			return "", errors.New("radio playlist is invalid or too large")
		}
		// Do not mistake an HLS segment or variant for a standalone stream.
		if strings.Contains(string(body), "#EXT-X-") {
			return res.Request.URL.String(), nil
		}
		raw, err = parsePlaylist(string(body), res.Request.URL)
		if err != nil {
			return "", err
		}
	}
	return "", errors.New("too many nested radio playlists")
}

func parsePlaylist(body string, base *url.URL) (string, error) {
	pls := strings.HasPrefix(strings.ToLower(strings.TrimSpace(body)), "[playlist]")
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok && strings.HasPrefix(strings.ToLower(key), "file") {
			line = strings.TrimSpace(value)
		} else if pls {
			continue
		}
		u, err := url.Parse(line)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(u).String()
		if ValidateURL(resolved) == nil {
			return resolved, nil
		}
	}
	return "", errors.New("radio playlist contains no HTTP/HTTPS stream")
}
