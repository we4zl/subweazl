package radio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolvePlaylists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start.pls":
			http.Redirect(w, r, "/nested/station.pls", http.StatusFound)
		case "/nested/station.pls":
			w.Write([]byte("[playlist]\nFile1=../list.m3u\n"))
		case "/list.m3u":
			w.Write([]byte("#EXTM3U\n/live?token=a=b\n"))
		case "/hls.m3u8":
			w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:5\npart.ts\n"))
		case "/loop.pls":
			w.Write([]byte("[playlist]\nFile1=/loop.pls\n"))
		case "/bad.pls":
			w.Write([]byte("[playlist]\nFile1=file:///etc/passwd\n"))
		case "/large.pls":
			w.Write([]byte(strings.Repeat("x", (1<<20)+1)))
		default:
			http.Error(w, "missing", 404)
		}
	}))
	defer server.Close()
	for _, tt := range []struct {
		path, want string
		fail       bool
	}{
		{"/start.pls", "/live?token=a=b", false},
		{"/hls.m3u8", "/hls.m3u8", false},
		{"/direct", "/direct", false},
		{"/loop.pls", "", true}, {"/bad.pls", "", true},
		{"/missing.pls", "", true}, {"/large.pls", "", true},
	} {
		t.Run(tt.path, func(t *testing.T) {
			got, err := ResolveStreamURL(context.Background(), server.URL+tt.path)
			if tt.fail {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil || got != server.URL+tt.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}

func TestResolveHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ResolveStreamURL(ctx, "https://example.test/station.pls")
	if err == nil {
		t.Fatal("cancelled lookup succeeded")
	}
}
