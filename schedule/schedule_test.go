package schedule

import "testing"

func TestParseFormValidation(t *testing.T) {
	if r := ParseForm("  ", "2026-06-15", "5", "9", "17"); !r.IsErr() || r.UnwrapErr().Field != "name" {
		t.Fatalf("blank name should fail on name, got %v", r)
	}
	if r := ParseForm("standup", "junk", "5", "9", "17"); !r.IsErr() || r.UnwrapErr().Field != "start date" {
		t.Fatalf("bad date should fail on start date, got %v", r)
	}
	if r := ParseForm("standup", "2026-06-15", "99", "9", "17"); !r.IsErr() || r.UnwrapErr().Field != "days" {
		t.Fatalf("out-of-range days should fail, got %v", r)
	}
	if r := ParseForm("standup", "2026-06-15", "5", "17", "9"); !r.IsErr() || r.UnwrapErr().Field != "end hour" {
		t.Fatalf("inverted hours should fail, got %v", r)
	}

	e := ParseForm(" standup ", "2026-06-15", "5", "9", "17").Unwrap()
	if e.Name != "standup" || e.Days != 5 || e.SlotsPerDay() != 16 {
		t.Fatalf("parsed event wrong: %+v", e)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	e := ParseForm("standup", "2026-06-15", "3", "9", "12").Unwrap()
	e = AddParticipant(e, "ada")
	e = SetSlot(e, "ada", SlotKey(1, 2), true)

	s := Encode(e).Unwrap()
	got := Decode(s).Unwrap()
	if got.Name != "standup" || len(got.People) != 1 || !got.People[0].Slots["1:2"] {
		t.Fatalf("round trip lost data: %+v", got)
	}
	if Decode("{not json").IsOk() {
		t.Fatal("garbage should not decode")
	}
}

func TestDayLabelsAndSlotLabel(t *testing.T) {
	e := ParseForm("x", "2026-06-15", "3", "9", "12").Unwrap() // a Monday
	labels := DayLabels(e).Unwrap()
	if len(labels) != 3 || labels[0] != "Mon 6/15" || labels[2] != "Wed 6/17" {
		t.Fatalf("labels: %v", labels)
	}
	if got := e.SlotLabel(0); got != "9:00" {
		t.Fatalf("slot 0 = %s", got)
	}
	if got := e.SlotLabel(5); got != "11:30" {
		t.Fatalf("slot 5 = %s", got)
	}
}

func TestBestAndParseKey(t *testing.T) {
	e := ParseForm("x", "2026-06-15", "2", "9", "11").Unwrap()
	if Best(e).IsSome() {
		t.Fatal("no participants: Best should be None")
	}
	e = AddParticipant(e, "ada")
	e = AddParticipant(e, "lin")
	if Best(e).IsSome() {
		t.Fatal("nothing painted: Best should be None")
	}
	e = SetSlot(e, "ada", SlotKey(0, 1), true)
	e = SetSlot(e, "lin", SlotKey(0, 1), true)
	e = SetSlot(e, "lin", SlotKey(1, 0), true)

	best := Best(e).Unwrap()
	if best.Key != "0:1" || best.Count != 2 || best.Total != 2 {
		t.Fatalf("best: %+v", best)
	}
	ref := ParseKey(best.Key).Unwrap()
	if ref.Day != 0 || ref.Slot != 1 {
		t.Fatalf("ParseKey: %+v", ref)
	}
	if ParseKey("nope").IsSome() || ParseKey("1:x").IsSome() {
		t.Fatal("malformed keys should be None")
	}

	if names := e.NamesAt("0:1"); len(names) != 2 {
		t.Fatalf("NamesAt: %v", names)
	}
	if FindParticipant(e, "ada").IsNone() || FindParticipant(e, "ghost").IsSome() {
		t.Fatal("FindParticipant wrong")
	}
	if got := AddParticipant(e, "ada"); len(got.People) != 2 {
		t.Fatal("duplicate participant should be rejected")
	}
}
