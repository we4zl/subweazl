package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bprendie/subweazl/internal/directory"
	"github.com/bprendie/subweazl/internal/radio"
	"github.com/bprendie/subweazl/internal/subsonic"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type radioDirectoryMsg struct {
	request  uint64
	stations []directory.Station
	err      error
}

type radioResolvedMsg struct {
	request uint64
	station radio.Station
	url     string
	err     error
}

func (m *Model) showRadio(tab string) {
	m.clearNav()
	m.mode = modeRadio
	m.focus = focusMain
	m.radioTab = tab
	m.radioDirectoryRequest++
	m.resetInput()
	m.searching = false
	m.err = ""
	if err := m.loadRadioStations(); err != nil {
		m.err = err.Error()
		return
	}

	m.refreshRadioList()
}

func (m *Model) loadRadioStations() error {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		return errors.New("unlock the vault to use radio")
	}
	if m.radioLoaded {
		return nil
	}
	stations, found, err := m.vaultStore.RadioStations()
	if err != nil {
		return err
	}
	if !found {
		stations = radio.Defaults()
	}
	m.radioStations, m.radioLoaded = stations, true
	return nil
}

func (m Model) playRadioPreset(slot int) (Model, tea.Cmd) {
	if err := m.loadRadioStations(); err != nil {
		m.err = err.Error()
		return m, noop
	}
	if slot < 1 || slot > 4 {
		m.err = "radio preset must be between 1 and 4"
		return m, noop
	}
	for _, station := range m.radioStations {
		if station.HotSlot == slot {
			return m.startRadio(station)
		}
	}

	m.err = fmt.Sprintf("Hot 4 slot %d is empty; select a station and press H in Internet radio", slot)
	return m, noop
}

func (m *Model) refreshRadioList() {
	m.refreshTitle()
	var rows []list.Item
	for _, action := range []string{"Presets", "My stations", "SomaFM", "Icecast", "Add URL"} {
		rows = append(rows, item{kind: "radio_action", title: action, action: action})
	}
	if m.radioTab == "Presets" || m.radioTab == "My stations" {
		for _, st := range m.radioStations {
			if m.radioTab == "Presets" && !st.Preset {
				continue
			}
			rows = append(rows, radioItem(st, "saved station"))
		}
	}
	m.list.ResetFilter()
	m.list.SetItems(rows)
	if len(rows) > 5 {
		m.list.Select(5)
	} else {
		m.list.Select(0)
	}
	m.status = "enter: open/play · /: Icecast search · H: Hot 4 · ctrl+a: save · ctrl+p: preset · ctrl+r: rename · delete: remove · [/]: reorder"
}

func radioItem(st radio.Station, desc string) item {
	if st.HotSlot > 0 {
		desc = fmt.Sprintf("Hot 4 #%d · %s", st.HotSlot, desc)
	}
	if st.Preset {
		desc = "preset · " + desc
	}
	return item{kind: "radio", title: st.Name, desc: desc, radio: st}
}

func (m Model) radioAction(action string) (Model, tea.Cmd) {
	switch action {
	case "Presets", "My stations":
		m.showRadio(action)
		return m, noop
	case "Add URL":
		m.radioDraft = radio.Station{}
		m.promptRadio("url", "stream URL > ", "")
		return m, noop
	case "SomaFM", "Icecast":
		return m.loadRadioDirectory(action, "")
	}
	return m, noop
}

func (m Model) loadRadioDirectory(tab, query string) (Model, tea.Cmd) {
	m.showRadio(tab)
	if !m.radioLoaded {
		return m, noop
	}
	request := m.radioDirectoryRequest
	spin := m.beginSearch("loading " + tab + " radio directory")
	return m, tea.Batch(spin, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		client := directory.NewClient()
		var stations []directory.Station
		var err error
		if tab == "SomaFM" {
			stations, err = client.SomaFM(ctx)
		} else {
			stations, err = client.XiphSearch(ctx, query)
		}
		return radioDirectoryMsg{request: request, stations: stations, err: err}
	})
}

func (m *Model) applyRadioDirectory(msg radioDirectoryMsg) {
	if m.mode != modeRadio || msg.request != m.radioDirectoryRequest {
		return
	}
	m.searching = false
	if msg.err != nil {
		m.err = msg.err.Error()
		return
	}
	m.refreshRadioList()
	rows := append([]list.Item(nil), m.list.Items()...)
	for _, st := range msg.stations {
		station, err := radio.Normalize(radio.Station{Name: st.Name, URL: st.URL})
		if err != nil {
			continue
		}
		for _, saved := range m.radioStations {
			if saved.URL == station.URL {
				station = saved
				break
			}
		}
		rows = append(rows, radioItem(station, strings.TrimSpace(st.Source+" · "+st.Genre+" · "+st.Description)))
	}
	m.list.SetItems(rows)
	if len(rows) > 5 {
		m.list.Select(5)
	}
	m.status = fmt.Sprintf("%d stations · enter: play · H: Hot 4 · ctrl+a: save · ctrl+p: preset · /: Icecast search", len(rows)-5)
}

func (m *Model) promptRadio(action, prompt, value string) {
	m.radioInput = action
	m.input.Prompt = prompt
	m.input.CharLimit = 2048
	m.input.SetValue(value)
	m.input.Focus()
	m.err = ""
}

func (m Model) submitRadioInput() (Model, tea.Cmd) {
	value := strings.TrimSpace(m.input.Value())
	switch m.radioInput {
	case "hot4":
		slot, err := strconv.Atoi(value)
		if err != nil {
			m.err = "choose a Hot 4 slot from 1 to 4"
			return m, noop
		}
		next, err := radio.AssignHot4(m.radioStations, m.radioDraft, slot)
		if err != nil {
			m.err = err.Error()
			return m, noop
		}
		if !m.saveRadioStations(next) {
			return m, noop
		}
		name := m.radioDraft.Name
		m.showRadio("My stations")
		m.status = fmt.Sprintf("%s assigned to Hot 4 #%d", name, slot)
		for i, row := range m.list.Items() {
			if row.(item).radio.URL == m.radioDraft.URL {
				m.list.Select(i)
				break
			}
		}
		return m, noop
	case "search":
		return m.loadRadioDirectory("Icecast", value)
	case "url":
		if err := radio.ValidateURL(value); err != nil {
			m.err = err.Error()
			return m, noop
		}
		m.radioDraft.URL = value
		m.promptRadio("name", "station name > ", "")
		return m, noop
	case "name", "rename":
		m.radioDraft.Name = value
		next, err := radio.Upsert(m.radioStations, m.radioDraft)
		if err != nil {
			m.err = err.Error()
			return m, noop
		}
		if !m.saveRadioStations(next) {
			return m, noop
		}
	case "delete":
		if !strings.EqualFold(value, "y") {
			m.resetInput()
			m.status = "station deletion cancelled"
			return m, noop
		}
		next := make([]radio.Station, 0, len(m.radioStations))
		for _, st := range m.radioStations {
			if st.URL != m.radioDraft.URL {
				next = append(next, st)
			}
		}
		if !m.saveRadioStations(next) {
			return m, noop
		}
	}
	m.showRadio("My stations")
	return m, noop
}

func (m *Model) saveRadioStations(next []radio.Station) bool {
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		m.err = "unlock the vault to save stations"
		return false
	}
	if err := m.vaultStore.SaveRadioStations(next); err != nil {
		m.err = err.Error()
		return false
	}
	m.radioStations = next
	m.publishRemote(true)
	m.err = ""
	return true
}

func (m Model) handleRadioKey(key string) (Model, tea.Cmd, bool) {
	if key == "/" {
		m.promptRadio("search", "Icecast name/genre > ", "")
		return m, noop, true
	}
	switch key {
	case "H", "ctrl+a", "ctrl+p", "ctrl+r", "delete", "[", "]":
	default:
		return m, nil, false
	}
	it, ok := m.list.SelectedItem().(item)
	if !ok || it.kind != "radio" {
		m.err = "select a radio station first"
		return m, noop, true
	}
	st := it.radio
	index := -1
	for i, saved := range m.radioStations {
		if saved.URL == st.URL {
			index = i
			st = saved
			break
		}
	}
	switch key {
	case "H":
		m.radioDraft = st
		m.promptRadio("hot4", "Hot 4 slot [1–4, esc cancels] > ", "")
		names := [4]string{"Empty", "Empty", "Empty", "Empty"}
		for _, saved := range m.radioStations {
			if saved.HotSlot > 0 && saved.HotSlot <= 4 {
				names[saved.HotSlot-1] = saved.Name
			}
		}
		m.status = fmt.Sprintf("Assign %s · 1: %s · 2: %s · 3: %s · 4: %s", st.Name, names[0], names[1], names[2], names[3])
	case "ctrl+a", "ctrl+p":
		if key == "ctrl+p" {
			st.Preset = !st.Preset
		}
		next, err := radio.Upsert(m.radioStations, st)
		if err != nil {
			m.err = err.Error()
			return m, noop, true
		}
		if m.saveRadioStations(next) {
			m.showRadio("My stations")
		}
	case "ctrl+r", "delete":
		if index < 0 {
			m.err = "save the station first with ctrl+a"
			break
		}
		m.radioDraft = st
		if key == "ctrl+r" {
			m.promptRadio("rename", "station name > ", st.Name)
		} else {
			m.promptRadio("delete", "delete station? [y/N] > ", "")
			m.status = "delete saved station: " + st.Name
		}
	case "[", "]":
		if index < 0 || (m.radioTab != "Presets" && m.radioTab != "My stations") {
			m.err = "reorder from Presets or My stations"
			break
		}
		delta := 1
		if key == "[" {
			delta = -1
		}
		target := index + delta
		for target >= 0 && target < len(m.radioStations) && m.radioTab == "Presets" && !m.radioStations[target].Preset {
			target += delta
		}
		if target < 0 || target >= len(m.radioStations) {
			break
		}
		next := append([]radio.Station(nil), m.radioStations...)
		next[index], next[target] = next[target], next[index]
		if m.saveRadioStations(next) {
			m.refreshRadioList()
			for i, row := range m.list.Items() {
				if row.(item).radio.URL == st.URL {
					m.list.Select(i)
					break
				}
			}
		}
	}
	return m, noop, true
}

func (m Model) startRadio(st radio.Station) (Model, tea.Cmd) {
	st, err := radio.Normalize(st)
	if err != nil {
		m.err = err.Error()
		return m, noop
	}
	// Cancel unfinished curator playback handoffs before starting a radio stream.
	if m.curatorCancel != nil {
		m.curatorCancel()
		m.curatorCancel = nil
	}
	m.curatorID = ""
	m.curating = false
	m.radioRequest++
	m.radioPending = true
	m.searching = false
	request := m.radioRequest
	m.status = "connecting to " + st.Name
	m.err = ""
	return m, func() tea.Msg {
		stream, err := radio.ResolveStreamURL(context.Background(), st.URL)
		return radioResolvedMsg{request: request, station: st, url: stream, err: err}
	}
}

func (m *Model) applyRadioResolved(msg radioResolvedMsg) {
	if msg.request != m.radioRequest {
		return
	}
	m.radioPending = false
	if msg.err != nil {
		m.err = msg.err.Error()
		return
	}
	if err := m.player.Play(msg.url); err != nil {
		m.err = err.Error()
		return
	}
	m.radioRequest++
	m.radioPlaying = &msg.station
	// An empty library ID keeps radio outside Subsonic curation and persistence.
	m.playing = &subsonic.Track{Title: msg.station.Name, Artist: "Internet radio"}
	m.playSource = msg.url
	m.playerStarted, m.paused = true, false
	m.trackTitle = ""
	m.titlePoll = time.Time{}
	m.coverID, m.coverErr, m.coverArt = "", "", nil
	m.err = ""
	m.status = "live · " + msg.station.Name
	m.publishRemote(true)
}

// Media next/previous returns to the saved library position without skipping it.
func (m Model) resumeRadioQueue() (Model, tea.Cmd) {
	track, ok := m.queue.Current()
	if !ok {
		m.status = "radio is live; no library queue to resume"
		return m, noop
	}
	return m, m.play(track)
}

func (m *Model) moveHot4(from, to int) {
	if err := m.loadRadioStations(); err != nil {
		m.err = err.Error()
		return
	}
	next, err := radio.MoveHot4(m.radioStations, from, to)
	if err != nil {
		m.err = err.Error()
		return
	}
	if !m.saveRadioStations(next) {
		return
	}
	// Update visible badges without changing directory results, focus, or input.
	rows := append([]list.Item(nil), m.list.Items()...)
	for i, row := range rows {
		it, ok := row.(item)
		if !ok || it.kind != "radio" {
			continue
		}
		for _, saved := range next {
			if saved.URL == it.radio.URL {
				rows[i] = radioItem(saved, "saved station")
				break
			}
		}
	}
	cursor := m.list.Index()
	m.list.SetItems(rows)
	m.list.Select(cursor)
	m.status = "Hot 4 order saved"
}
