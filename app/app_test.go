package app_test

import (
	"strings"
	"testing"

	g "github.com/gyoumi/grove"
	"github.com/gyoumi/grove/testdom"
	"github.com/gyoumi/overlap/app"
	"github.com/gyoumi/overlap/schedule"
)

type memStore struct{ s string }

func (m *memStore) Load() string  { return m.s }
func (m *memStore) Save(s string) { m.s = s }

func mount(store app.Store) *testdom.R {
	return testdom.Mount(g.C(app.App, app.Props{Store: store}))
}

func createEvent(t *testing.T, r *testdom.R, name string) {
	t.Helper()
	r.Input(r.FindByAttr("id", "name"), name)
	r.Input(r.FindByAttr("id", "date"), "2026-06-15")
	r.Click(r.FindText("Create event"))
	if r.FindByAttr("data-cell", "0:0") == nil {
		t.Fatalf("grid did not render after create: %s", r.HTML())
	}
}

func addPerson(t *testing.T, r *testdom.R, name string) {
	t.Helper()
	r.Input(r.FindByAttr("id", "new-person"), name)
	r.Click(r.FindByAttr("data-slot", "add-person"))
	if r.FindByAttr("data-person", name) == nil {
		t.Fatalf("participant %s not added: %s", name, r.HTML())
	}
}

func TestSetupValidationAndCreate(t *testing.T) {
	store := &memStore{}
	r := mount(store)

	if r.FindText("New event") == nil {
		t.Fatalf("setup form should show first: %s", r.HTML())
	}
	r.Click(r.FindText("Create event")) // empty name
	errEl := r.FindByAttr("data-slot", "form-error")
	if errEl == nil || !strings.Contains(errEl.TextContent(), "name") {
		t.Fatalf("expected a name validation error: %s", r.HTML())
	}

	createEvent(t, r, "standup")
	// default window 9-17 over 5 days → 16 slots/day
	if cells := r.FindAll("ul")[0]; cells == nil {
		t.Fatal("no grid list")
	}
	if got := len(r.Container.Children); got == 0 {
		t.Fatal("empty render")
	}
	if r.FindByAttr("data-cell", "4:15") == nil {
		t.Fatalf("expected 5x16 grid, missing last cell: %s", r.HTML())
	}
	if store.s == "" {
		t.Fatal("event should be persisted on create")
	}
}

func TestPaintingAndHeatmapAndPersistence(t *testing.T) {
	store := &memStore{}
	r := mount(store)
	createEvent(t, r, "standup")
	addPerson(t, r, "ada")

	// drag-paint ada across three slots: down on 0:0, over 0:1 and 0:2, up
	r.Fire(r.FindByAttr("data-cell", "0:0"), "mousedown", nil)
	r.Fire(r.FindByAttr("data-cell", "0:1"), "mouseover", nil)
	r.Fire(r.FindByAttr("data-cell", "0:2"), "mouseover", nil)
	r.Fire(r.FindByAttr("data-cell", "0:2"), "mouseup", nil)
	for _, key := range []string{"0:0", "0:1", "0:2"} {
		if cell := r.FindByAttr("data-cell", key); cell.Attrs["data-on"] != "1" {
			t.Fatalf("cell %s should be painted: %s", key, cell.HTML())
		}
	}

	// drag again starting on a painted cell → erases
	r.Fire(r.FindByAttr("data-cell", "0:1"), "mousedown", nil)
	r.Fire(r.FindByAttr("data-cell", "0:2"), "mouseover", nil)
	r.Fire(r.FindByAttr("data-cell", "0:2"), "mouseup", nil)
	if cell := r.FindByAttr("data-cell", "0:1"); cell.Attrs["data-on"] == "1" {
		t.Fatal("0:1 should be erased")
	}
	if cell := r.FindByAttr("data-cell", "0:0"); cell.Attrs["data-on"] != "1" {
		t.Fatal("0:0 should stay painted")
	}

	// second participant overlaps at 0:0
	addPerson(t, r, "lin")
	r.Fire(r.FindByAttr("data-cell", "0:0"), "mousedown", nil)
	r.Fire(r.FindByAttr("data-cell", "0:0"), "mouseup", nil)

	// group view: 0:0 has 2/2 → deepest bucket; best line names it
	r.Click(r.FindByAttr("data-person", "*"))
	if cls := r.FindByAttr("data-cell", "0:0").Attrs["class"]; !strings.Contains(cls, "bg-emerald-600") {
		t.Fatalf("0:0 should be deepest heat: %s", cls)
	}
	best := r.FindByAttr("data-slot", "best")
	if best == nil || !strings.Contains(best.TextContent(), "2 of 2") {
		t.Fatalf("best line wrong: %s", r.HTML())
	}
	if !strings.Contains(best.TextContent(), "Mon 6/15 at 9:00") {
		t.Fatalf("best should name Mon 9:00: %s", best.TextContent())
	}

	// hover details
	r.Fire(r.FindByAttr("data-cell", "0:0"), "mouseover", nil)
	hover := r.FindByAttr("data-slot", "hover-info")
	if hover == nil || !strings.Contains(hover.TextContent(), "ada, lin") {
		t.Fatalf("hover info wrong: %s", r.HTML())
	}

	// a fresh mount from the same store restores everything
	r2 := mount(store)
	if r2.FindByAttr("data-person", "ada") == nil || r2.FindByAttr("data-person", "lin") == nil {
		t.Fatalf("participants not restored: %s", r2.HTML())
	}
	if cls := r2.FindByAttr("data-cell", "0:0").Attrs["class"]; !strings.Contains(cls, "bg-emerald-600") {
		t.Fatalf("availability not restored: %s", cls)
	}

	// sanity: stored payload decodes to the same event
	ev := schedule.Decode(store.s).Unwrap()
	if len(ev.People) != 2 || !ev.People[0].Slots["0:0"] {
		t.Fatalf("stored event wrong: %+v", ev)
	}
}

func TestReset(t *testing.T) {
	store := &memStore{}
	r := mount(store)
	createEvent(t, r, "standup")
	r.Click(r.FindByAttr("data-slot", "reset"))
	if r.FindText("New event") == nil {
		t.Fatalf("reset should return to setup: %s", r.HTML())
	}
	if store.s != "" {
		t.Fatal("reset should clear the store")
	}
}
