package tui

import (
	"time"

	"github.com/bprendie/subweazl/internal/remote"
)

type mediaPublisher interface {
	Publish(remote.Snapshot)
}

func (m *Model) SetMediaPublisher(publisher mediaPublisher) {
	m.mediaPublisher = publisher
}

func (m *Model) EnableRemote() {
	m.remoteEnabled = true
	m.publishRemote(true)
}

func (m *Model) publishRemote(force ...bool) {
	if !m.remoteEnabled {
		return
	}
	snapshot := remote.Snapshot{Running: true, State: "idle", PlaybackMode: m.playbackModeLabel()}
	if m.loadRadioStations() == nil {
		snapshot.Hot4Ready = true
		for _, station := range m.radioStations {
			if station.HotSlot < 1 || station.HotSlot > 4 {
				continue
			}
			snapshot.RadioPresets[station.HotSlot-1] = remote.RadioPreset{Name: station.Name, Active: m.radioPlaying != nil && m.radioPlaying.URL == station.URL}
		}

	}
	if m.playing != nil {
		snapshot.State = "playing"
		if m.paused {
			snapshot.State = "paused"
		}
		snapshot.Title = m.playing.Title
		if m.trackTitle != "" {
			snapshot.Title = m.trackTitle
		}
		snapshot.Artist = m.playing.Artist
		snapshot.Album = m.playing.Album
		snapshot.Duration = m.playing.Duration
	}
	changed := snapshot != m.remoteLast
	if changed && m.mediaPublisher != nil {
		m.mediaPublisher.Publish(snapshot)
	}
	if !changed && time.Since(m.remotePublished) < 30*time.Second {
		return
	}
	if remote.WriteSnapshot(snapshot) == nil {
		m.remotePublished = time.Now()
		m.remoteLast = snapshot
	}
}
