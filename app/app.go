// Package app is the overlap UI: a when2meet-style availability matcher
// built on grove. It is platform-agnostic (storage is injected), so the
// whole app is testable with grove/testdom.
package app

import (
	"fmt"
	"time"

	g "github.com/gyoumi/grove"
	"github.com/gyoumi/grove/router"
	"github.com/gyoumi/overlap/schedule"
	"github.com/gyoumi/overlap/ui"
)

// Store persists the workspace between sessions. The browser entry point
// backs it with localStorage; tests use an in-memory string.
type Store interface {
	Load() string // "" when nothing is stored
	Save(s string)
}

type Props struct {
	Store Store
}

// App owns the workspace (all events keyed by id) and routes between the
// home page and individual event grids.
func App(p Props) *g.Node {
	events, setEvents := g.UseStateLazy(func() map[string]schedule.Event {
		return loadWorkspace(p.Store)
	})

	persist := func(m map[string]schedule.Event) {
		setEvents(m)
		if r := schedule.EncodeAll(m); r.IsOk() {
			p.Store.Save(r.Unwrap())
		}
	}
	update := func(id string, e schedule.Event) {
		m := cloneEvents(events)
		m[id] = e
		persist(m)
	}
	remove := func(id string) {
		m := cloneEvents(events)
		delete(m, id)
		persist(m)
	}

	return g.Div(g.Class("mx-auto flex min-h-svh max-w-3xl flex-col gap-6 p-6 text-foreground"),
		g.Header(g.Class("flex items-baseline justify-between"),
			router.Link("/", g.Class("text-2xl font-semibold tracking-tight text-foreground no-underline"), "overlap"),
			g.P(g.Class("text-sm text-muted-foreground"), "find where schedules align"),
		),
		router.Routes(
			router.Route{Pattern: "/", Render: func(router.Params) *g.Node {
				return g.C(HomePage, HomeProps{
					Events: events,
					OnCreate: func(e schedule.Event) {
						id := newID()
						update(id, e)
						router.Navigate("/event/" + id)
					},
				})
			}},
			router.Route{Pattern: "/event/:id", Render: func(params router.Params) *g.Node {
				id := params["id"]
				e, ok := events[id]
				if !ok {
					return missingEvent()
				}
				return g.C(EventView, EventProps{
					Event:    e,
					OnUpdate: func(e schedule.Event) { update(id, e) },
					OnDelete: func() {
						remove(id)
						router.Navigate("/")
					},
				})
			}},
			router.Route{Pattern: "*", Render: func(router.Params) *g.Node {
				return missingEvent()
			}},
		),
	)
}

func missingEvent() *g.Node {
	return g.Div(g.Class("flex flex-col items-start gap-3"),
		g.P(g.Class("text-sm text-muted-foreground"), "That page doesn't exist (anymore)."),
		router.Link("/", g.Class("text-sm font-medium text-primary underline-offset-4 hover:underline"), "back to your events"),
	)
}

func loadWorkspace(store Store) map[string]schedule.Event {
	s := store.Load()
	if s == "" {
		return map[string]schedule.Event{}
	}
	if r := schedule.DecodeAll(s); r.IsOk() {
		return r.Unwrap()
	}
	// migrate a v1 single-event payload into the workspace format
	if r := schedule.Decode(s); r.IsOk() {
		return map[string]schedule.Event{newID(): r.Unwrap()}
	}
	return map[string]schedule.Event{}
}

func cloneEvents(m map[string]schedule.Event) map[string]schedule.Event {
	out := make(map[string]schedule.Event, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	return out
}

var idCounter int

func newID() string {
	idCounter++
	return fmt.Sprintf("%x-%d", time.Now().UnixNano()&0xFFFFFF, idCounter)
}

type HomeProps struct {
	Events   map[string]schedule.Event
	OnCreate func(schedule.Event)
}

// HomePage lists existing events and hosts the creation form.
func HomePage(p HomeProps) *g.Node {
	ids := sortedIDs(p.Events)
	return g.Div(g.Class("flex flex-col gap-6"),
		g.If(len(ids) > 0,
			g.Div(g.Class("flex flex-col gap-2"),
				g.H2(g.Class("text-lg font-medium"), "Your events"),
				g.Ul(g.Class("flex flex-col gap-1"),
					g.Map(ids, func(id string) *g.Node {
						e := p.Events[id]
						return g.Li(g.Key(id),
							router.Link("/event/"+id,
								g.Class("flex items-center justify-between rounded-md border px-4 py-3 text-sm no-underline transition-colors hover:bg-accent"),
								g.Data("event-link", id),
								g.Span(g.Class("font-medium"), e.Name),
								g.Span(g.Class("text-xs text-muted-foreground"),
									g.Textf("%d people · %d days", len(e.People), e.Days)),
							),
						)
					}),
				),
			),
		),
		g.C(SetupForm, SetupProps{OnCreate: p.OnCreate}),
	)
}

func sortedIDs(m map[string]schedule.Event) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	// newest first: ids embed a monotonic counter, but sorting by name is
	// friendlier for the list
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && m[ids[j]].Name < m[ids[j-1]].Name; j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	return ids
}

type SetupProps struct {
	OnCreate func(schedule.Event)
}

// SetupForm collects the event parameters and validates them through
// schedule.ParseForm; the typed ValidationError drives the inline message.
func SetupForm(p SetupProps) *g.Node {
	name, setName := g.UseState("")
	date, setDate := g.UseState(time.Now().Format("2006-01-02"))
	days, setDays := g.UseState("5")
	startHour, setStartHour := g.UseState("9")
	endHour, setEndHour := g.UseState("17")
	errMsg, setErrMsg := g.UseState("")

	create := func(*g.Event) {
		r := schedule.ParseForm(name, date, days, startHour, endHour)
		if r.IsErr() {
			setErrMsg(r.UnwrapErr().Error())
			return
		}
		p.OnCreate(r.Unwrap())
	}

	field := func(id, label, value string, set func(string)) *g.Node {
		return g.Div(g.Class("flex flex-col gap-1.5"),
			ui.Label(id, label),
			ui.Input(ui.InputProps{ID: id, Value: value, OnInput: func(e *g.Event) { set(e.Value()) }}),
		)
	}

	return ui.Card(g.Class("p-0"),
		ui.CardHeader(
			ui.CardTitle("New event"),
			ui.CardDescription("Pick the days and hours people should mark themselves available for."),
		),
		ui.CardContent(g.Class("flex flex-col gap-4"),
			field("name", "Event name", name, setName),
			g.Div(g.Class("grid grid-cols-2 gap-4 sm:grid-cols-4"),
				field("date", "First day (YYYY-MM-DD)", date, setDate),
				field("days", "Days (1-14)", days, setDays),
				field("start", "From hour (0-23)", startHour, setStartHour),
				field("end", "To hour (1-24)", endHour, setEndHour),
			),
			g.If(errMsg != "",
				g.P(g.Class("text-sm font-medium text-destructive"), g.Data("slot", "form-error"), errMsg),
			),
		),
		ui.CardFooter(
			ui.Button(ui.ButtonProps{OnClick: create}, "Create event"),
		),
	)
}
