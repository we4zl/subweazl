package localstore

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/bprendie/subweazl/internal/radio"
)

func (s *Store) SaveRadioStations(stations []radio.Station) error {
	// Validate before replacing the encrypted snapshot.
	clean := make([]radio.Station, 0, len(stations))
	for _, st := range stations {
		var err error
		clean, err = radio.Upsert(clean, st)
		if err != nil {
			return err
		}
	}
	used := [5]bool{}
	for _, st := range clean {
		if st.HotSlot == 0 {
			continue
		}
		if used[st.HotSlot] {
			return errors.New("duplicate Hot 4 slot")
		}
		used[st.HotSlot] = true
	}
	blob, err := s.encryptedJSON(map[string]any{"stations": clean, "hot4_configured": true})
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`insert into radio_stations (id, payload) values (1, ?)
 on conflict(id) do update set payload = excluded.payload`, blob)
	return err
}

func (s *Store) RadioStations() ([]radio.Station, bool, error) {
	var blob string
	err := s.db.QueryRow(`select payload from radio_stations where id = 1`).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	payload, err := s.decryptPayload(blob)
	if err != nil {
		return nil, false, err
	}
	raw, err := json.Marshal(payload["stations"])
	if err != nil {
		return nil, false, err
	}
	var stations []radio.Station
	if err := json.Unmarshal(raw, &stations); err != nil {
		return nil, false, err
	}
	if payload["hot4_configured"] != true {
		radio.SeedHot4(stations)
	}
	return stations, true, nil
}
