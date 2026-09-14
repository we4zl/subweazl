package radio

import (
	"fmt"
	"reflect"
	"testing"
)

func TestAssignHot4WithEightFullPresets(t *testing.T) {
	var stations []Station
	for i := 0; i < 8; i++ {
		stations = append(stations, Station{Name: fmt.Sprint(i), URL: fmt.Sprintf("https://example.test/%d", i), Preset: true})
	}
	SeedHot4(stations)
	before := append([]Station(nil), stations...)
	prendie := Station{Name: "Radio Prendie", URL: "https://radio.prendie.io/radio.mp3"}
	next, err := AssignHot4(stations, prendie, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 9 || next[1].HotSlot != 0 || next[8].HotSlot != 2 || next[8].Preset {
		t.Fatalf("assignment: %#v", next)
	}
	if !reflect.DeepEqual(stations, before) {
		t.Fatal("assignment mutated original")
	}
	count := 0
	for _, st := range next {
		if st.Preset {
			count++
		}
	}
	if count != 8 {
		t.Fatal("Hot 4 changed regular presets")
	}
	next, err = AssignHot4(next, next[0], 2)
	if err != nil || next[0].HotSlot != 2 || next[8].HotSlot != 1 {
		t.Fatal("existing hot slots should swap", err)
	}
}

func TestMoveHot4ShiftsSlotsAndPreservesSavedOrder(t *testing.T) {
	stations := Defaults()
	original := append([]Station(nil), stations...)
	next, err := MoveHot4(stations, 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []int{4, 1, 2, 3} {
		if next[i].HotSlot != want {
			t.Fatal("wrong shift", next)
		}
	}
	back, err := MoveHot4(next, 4, 1)
	if err != nil || !reflect.DeepEqual(back, original) || !reflect.DeepEqual(stations, original) {
		t.Fatal("move changed station order or metadata", err)
	}
	stations[3].HotSlot = 0
	next, err = MoveHot4(stations, 1, 4)
	if err != nil || next[0].HotSlot != 4 {
		t.Fatal("move into empty slot failed", err)
	}
	for _, pair := range [][2]int{{0, 1}, {1, 5}, {4, 1}} {
		if _, err := MoveHot4(stations, pair[0], pair[1]); err == nil {
			t.Fatal("accepted invalid move", pair)
		}
	}
}
