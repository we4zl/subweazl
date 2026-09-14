# Weazltunes comparison and Internet radio integration

Comparison based on the sibling repositories `../web_weazltunes` and
`../weazltunes`, inspected on September 14, 2026.

| Capability | WeazlTunes web | Weazltunes terminal | Subweazl with this change |
| --- | --- | --- | --- |
| Subsonic/Navidrome library | Albums, tracks, search, favorites, server playlists | Radio only | Albums, tracks, search, server playlists, encrypted private playlists/cache |
| Audio | Browser audio through same-origin relay | mpv | Existing mpv player |
| Library queue and curation | Queue controls, random play, streamed Mood | No library queue | Queue controls, deterministic recommendations, AI Mix and Mood |
| Radio discovery | SomaFM, Icecast | SomaFM, Xiph/Icecast and Radio Browser fallback | Directory client adapted from terminal Weazltunes; directories load on demand |
| Saved radio | Seven default presets, custom URLs, eight preset limit, removal | Presets, custom stations, rename, reorder, removal | Same seven web defaults; encrypted saved stations, eight presets, four explicit Hot 4 slots, rename, reorder, confirmed deletion |
| Radio playlists | PLS/M3U relay resolution; HLS rejected | PLS/M3U resolution and mpv | Bounded PLS/M3U resolution, relative links, nested playlists, HLS passed intact to mpv |
| Live titles | ICY parsing in the radio relay | mpv title polling | Existing mpv IPC polling, sanitized and protected against stale replies |
| Private storage | Encrypted per-account server settings and radio/queue state | Local JSON station configuration | Existing encrypted vault, with an additive radio-stations table |
| Desktop controls | Browser media controls | Terminal controls | Existing MPRIS and remote controls also handle radio |

Subweazl's existing `r` action creates a Subsonic playlist from a library seed.
Internet radio is a separate section under `6`; it never passes stream URLs to
Subsonic APIs or stores synthetic songs in library playlists or history.

Radio retains the queue and its current track index. Next/previous resumes that
track from the beginning, and selecting a queue row explicitly switches back to
library playback. A radio disconnect does not advance, repeat, or scrobble the
library queue. Delayed playlist resolution and metadata responses cannot restart
stopped playback or replace a newer selection. Starting radio cancels an active
curator run so its later playback handoff cannot replace the station.

Directory and playlist requests have timeouts and response limits. Station names,
descriptions, and metadata use the existing terminal sanitizer. Only HTTP/HTTPS
station URLs are accepted. The resolver preserves HLS playlists for mpv and limits
nested PLS/M3U wrappers to five. There is no background directory polling or
second audio stream for metadata.

The existing server-setup and vault-unlock requirements still apply. Saved
stations from the sibling apps are not automatically imported. The My stations
view includes presets. Direct URL additions are saved first and played by
selecting the resulting row. Stream errors require an explicit reconnect.

## Rollback

The pre-change source and Linux executable are published in the
[checkpoint release](https://github.com/bprendie/subweazl/releases/tag/checkpoint-pre-internet-radio-2026-09-14),
commit `4c3a33255e4f9b38f3c8379bb39864f24e7133b2`.
The new database table is additive: the checkpoint binary ignores it. The release
contains source/executable rollback material, not personal configuration or vault
data.

## Validation

Automated tests cover encrypted station persistence, empty-list restoration,
preset limits, case-sensitive URL paths, URL validation, PLS/M3U redirects and
relative/query links, HLS preservation, response limits, cancellation, station
controls, stale directory/playback/title replies, queue preservation, and radio
end-of-stream behavior. The directory parser tests were carried over from
Weazltunes.

A local HTTP ICY fixture was also played with real mpv and its null audio output:
mpv exposed the supplied live song title, accepted pause via IPC, and made exactly
one audio connection. This verifies decoding and metadata without producing sound;
it does not guarantee availability of every public station.

Final checks passed: `go test ./...`, `go vet ./...`, and
`go build -o subweazl ./cmd/subweazl`. Live directory checks returned 46 SomaFM
channels and 15 Icecast ambient results; a SomaFM playlist resolved successfully.

## Hot 4

Hot 4 assignments are encrypted with saved stations and do not depend on the
regular eight-preset limit or order. `H` followed by `1`–`4` assigns the highlighted
radio station, retaining displaced stations. Existing Hot 4 assignments swap when
reassigned. The widget supports drag-to-insert ordering and keeps playback intact.
Old station snapshots migrate the former first four presets; later empty slots
stay empty. `radio-move-FROM-TO` commands accept slots 1–4 only.

Qt Quick interaction tests cover clicks, drag-to-reorder without playback,
empty drop targets, and disabled controls. Run them with:

```sh
QT_QPA_PLATFORM=offscreen /usr/lib/qt6/bin/qmltestrunner -input widget/subweazl/tests -o -,txt
```
