package radio

import "errors"

// SeedHot4 preserves the former first-four-presets behavior when upgrading.
func SeedHot4(stations []Station) {
	slot := 1
	for i := range stations {
		stations[i].HotSlot = 0
		if stations[i].Preset && slot <= 4 {
			stations[i].HotSlot = slot
			slot++
		}
	}
}

// AssignHot4 swaps an existing Hot 4 station or replaces a slot with a saved or
// directory station. Displaced stations stay saved; the eight presets are independent.
func AssignHot4(stations []Station, station Station, slot int) ([]Station, error) {
	if slot < 1 || slot > 4 {
		return nil, errors.New("choose a Hot 4 slot from 1 to 4")
	}
	var err error
	station, err = Normalize(station)
	if err != nil {
		return nil, err
	}
	oldSlot := 0
	for _, saved := range stations {
		if saved.URL == station.URL {
			station = saved
			oldSlot = saved.HotSlot
			break
		}
	}
	station.HotSlot = 0
	next, err := Upsert(stations, station)
	if err != nil {
		return nil, err
	}
	for i := range next {
		if next[i].URL == station.URL {
			next[i].HotSlot = slot
		} else if next[i].HotSlot == slot {
			next[i].HotSlot = oldSlot
		}
	}
	return next, nil
}

// MoveHot4 inserts a dragged slot at its destination, shifting intervening slots.
func MoveHot4(stations []Station, from, to int) ([]Station, error) {
	if from < 1 || from > 4 || to < 1 || to > 4 {
		return nil, errors.New("Hot 4 slots must be between 1 and 4")
	}
	found := false
	for _, station := range stations {
		if station.HotSlot == from {
			found = true
		}
	}
	if !found {
		return nil, errors.New("the dragged Hot 4 slot is empty")
	}
	next := append([]Station(nil), stations...)
	for i := range next {
		slot := next[i].HotSlot
		switch {
		case slot == from:
			next[i].HotSlot = to
		case from < to && slot > from && slot <= to:
			next[i].HotSlot--
		case from > to && slot >= to && slot < from:
			next[i].HotSlot++
		}
	}
	return next, nil
}
