// Package schedule is the domain logic for overlap: events, participants,
// availability slots, and the math that finds where schedules align. The
// fallible parts live in schedule.warchest and are expanded to Go by the
// warchest generator (go generate ./...).
package schedule

import "fmt"

// Event is a scheduling poll: a stretch of days, a window of hours, and the
// participants' painted availability.
type Event struct {
	Name      string        `json:"name"`
	StartDate string        `json:"startDate"` // YYYY-MM-DD
	Days      int           `json:"days"`
	StartHour int           `json:"startHour"` // inclusive, 0-23
	EndHour   int           `json:"endHour"`   // exclusive, 1-24
	People    []Participant `json:"people"`
}

// Participant is one person's availability. Slots is keyed by SlotKey.
type Participant struct {
	Name  string          `json:"name"`
	Slots map[string]bool `json:"slots"`
}

// ValidationError reports why an event form could not be turned into an
// Event; Field names the offending input.
type ValidationError struct {
	Field string
	Msg   string
}

func (e ValidationError) Error() string { return e.Field + ": " + e.Msg }

// BestSlot is the (or a) slot with the highest attendance.
type BestSlot struct {
	Key   string
	Count int
	Total int
}

// SlotRef is a SlotKey decomposed into grid coordinates.
type SlotRef struct {
	Day  int
	Slot int
}

// SetSlot returns a copy of the event with one participant's slot set;
// participants and slot maps are copied so renders can treat Events as
// immutable snapshots.
func SetSlot(e Event, name, key string, val bool) Event {
	people := make([]Participant, len(e.People))
	for i, p := range e.People {
		if p.Name != name {
			people[i] = p
			continue
		}
		slots := make(map[string]bool, len(p.Slots)+1)
		for k, v := range p.Slots {
			slots[k] = v
		}
		if val {
			slots[key] = true
		} else {
			delete(slots, key)
		}
		people[i] = Participant{Name: p.Name, Slots: slots}
	}
	e.People = people
	return e
}

// AddParticipant returns a copy of the event with a new (empty) participant
// appended; duplicates by name are rejected by returning the event as-is.
func AddParticipant(e Event, name string) Event {
	for _, p := range e.People {
		if p.Name == name {
			return e
		}
	}
	people := make([]Participant, len(e.People), len(e.People)+1)
	copy(people, e.People)
	e.People = append(people, Participant{Name: name, Slots: map[string]bool{}})
	return e
}

// SlotsPerDay returns how many half-hour slots one day spans.
func (e Event) SlotsPerDay() int { return (e.EndHour - e.StartHour) * 2 }

// SlotKey identifies a cell in the availability grid.
func SlotKey(day, slot int) string { return fmt.Sprintf("%d:%d", day, slot) }

// SlotLabel formats a slot index within the event's hour window, e.g. "9:30".
func (e Event) SlotLabel(slot int) string {
	h := e.StartHour + slot/2
	m := (slot % 2) * 30
	return fmt.Sprintf("%d:%02d", h, m)
}

// CountAt returns how many participants are available at the given key.
func (e Event) CountAt(key string) int {
	n := 0
	for _, p := range e.People {
		if p.Slots[key] {
			n++
		}
	}
	return n
}

// NamesAt lists who is available at the given key.
func (e Event) NamesAt(key string) []string {
	var names []string
	for _, p := range e.People {
		if p.Slots[key] {
			names = append(names, p.Name)
		}
	}
	return names
}
