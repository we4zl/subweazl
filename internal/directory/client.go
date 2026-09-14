package directory

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	http *http.Client
}

func NewClient() Client {
	return Client{http: &http.Client{Timeout: 12 * time.Second}}
}

func (c Client) getString(ctx context.Context, rawurl string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawurl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "subweazl")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("%s: %s", rawurl, resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if err != nil {
		return "", err
	}
	if len(b) > 8<<20 {
		return "", fmt.Errorf("radio directory response is too large")
	}
	return string(b), nil
}
