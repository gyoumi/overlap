// Package app is the overlap UI: a when2meet-style availability matcher
// built on grove. It is platform-agnostic (storage is injected), so the
// whole app is testable with grove/testdom.
package app

import (
	"time"

	g "github.com/gyoumi/grove"
	"github.com/gyoumi/overlap/schedule"
	"github.com/gyoumi/overlap/ui"
)

// Store persists the current event between sessions. The browser entry
// point backs it with localStorage; tests use an in-memory string.
type Store interface {
	Load() string // "" when nothing is stored
	Save(s string)
}

type Props struct {
	Store Store
}

// App switches between the event setup form and the availability grid.
func App(p Props) *g.Node {
	event, setEvent := g.UseStateLazy(func() *schedule.Event {
		if s := p.Store.Load(); s != "" {
			if r := schedule.Decode(s); r.IsOk() {
				e := r.Unwrap()
				return &e
			}
		}
		return nil
	})

	update := func(e schedule.Event) {
		ev := e
		setEvent(&ev)
		if r := schedule.Encode(e); r.IsOk() {
			p.Store.Save(r.Unwrap())
		}
	}
	reset := func() {
		setEvent(nil)
		p.Store.Save("")
	}

	return g.Div(g.Class("mx-auto flex min-h-svh max-w-3xl flex-col gap-6 p-6 text-foreground"),
		g.Header(g.Class("flex items-baseline justify-between"),
			g.H1(g.Class("text-2xl font-semibold tracking-tight"), "overlap"),
			g.P(g.Class("text-sm text-muted-foreground"), "find where schedules align"),
		),
		g.IfElse(event == nil,
			g.C(SetupForm, SetupProps{OnCreate: update}),
			g.C(EventView, EventProps{Event: deref(event), OnUpdate: update, OnReset: reset}),
		),
	)
}

func deref(e *schedule.Event) schedule.Event {
	if e == nil {
		return schedule.Event{}
	}
	return *e
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
