package app

import (
	"strings"

	g "github.com/gyoumi/grove"
	"github.com/gyoumi/overlap/schedule"
	"github.com/gyoumi/overlap/ui"
)

type EventProps struct {
	Event    schedule.Event
	OnUpdate func(schedule.Event)
	OnDelete func()
}

// heatClasses maps "fraction of the group available" to a background; index
// 0 is empty, buckets 1-5 deepen with attendance. (Literal class names so
// the Tailwind scanner sees them.)
var heatClasses = []string{
	"bg-muted/40",
	"bg-emerald-200 dark:bg-emerald-900",
	"bg-emerald-300 dark:bg-emerald-800",
	"bg-emerald-400 dark:bg-emerald-600",
	"bg-emerald-500",
	"bg-emerald-600",
}

func heatClass(count, total int) string {
	if count == 0 || total == 0 {
		return heatClasses[0]
	}
	bucket := (count*5 + total - 1) / total // ceil(count/total * 5)
	if bucket > 5 {
		bucket = 5
	}
	return heatClasses[bucket]
}

type drag struct {
	painting bool
	value    bool
}

// viewState is the per-render snapshot the grid handlers read through a
// ref: memoized rows keep handlers from their last render, so handlers
// must look up the latest state instead of closing over it.
type viewState struct {
	event    schedule.Event
	active   string
	onUpdate func(schedule.Event)
}

// EventView is the heart of the app: pick (or add) a participant, paint
// availability by dragging across the grid, and read the group heatmap.
func EventView(p EventProps) *g.Node {
	active, setActive := g.UseState("") // "" = group heatmap view
	newName, setNewName := g.UseState("")
	hovered, setHovered := g.UseState("")
	dragRef := g.UseRef(drag{})

	stateRef := g.UseRef(viewState{})
	stateRef.Current = viewState{event: p.Event, active: active, onUpdate: p.OnUpdate}

	e := p.Event

	addPerson := func() {
		name := strings.TrimSpace(newName)
		if name == "" {
			return
		}
		p.OnUpdate(schedule.AddParticipant(e, name))
		setActive(name)
		setNewName("")
	}

	onDown := func(key string, ev *g.Event) {
		ev.PreventDefault()
		s := stateRef.Current
		if s.active == "" {
			return
		}
		on := personHas(s.event, s.active, key)
		dragRef.Current = drag{painting: true, value: !on}
		s.onUpdate(schedule.SetSlot(s.event, s.active, key, !on))
	}
	onOver := func(key string) {
		setHovered(key)
		if !dragRef.Current.painting {
			return
		}
		s := stateRef.Current
		if s.active == "" {
			return
		}
		s.onUpdate(schedule.SetSlot(s.event, s.active, key, dragRef.Current.value))
	}

	dayLabels := schedule.DayLabels(e).UnwrapOrElse(func(error) []string {
		return make([]string, e.Days)
	})

	return g.Div(g.Class("flex flex-col gap-5 select-none"),
		// releasing the mouse anywhere ends the current paint stroke
		g.On("mouseup", func(*g.Event) { dragRef.Current = drag{} }),

		g.Div(g.Class("flex items-baseline justify-between"),
			g.H2(g.Class("text-lg font-medium"), e.Name),
			ui.Button(ui.ButtonProps{Variant: ui.ButtonGhost, Size: ui.ButtonSizeSm, OnClick: func(*g.Event) { p.OnDelete() }},
				g.Data("slot", "delete"), "delete event"),
		),

		// participant picker
		g.Div(g.Class("flex flex-wrap items-center gap-2"),
			pill("everyone", active == "", g.Data("person", "*"), func(*g.Event) { setActive("") }),
			g.Map(e.People, func(person schedule.Participant) *g.Node {
				name := person.Name
				return pill(name, active == name, g.Data("person", name), func(*g.Event) { setActive(name) }).
					WithKey(name)
			}),
			g.Div(g.Class("ml-auto flex items-center gap-2"),
				ui.Input(ui.InputProps{
					ID: "new-person", Placeholder: "add person…",
					Class: "h-8 w-36",
					Value: newName,
					OnInput: func(e *g.Event) { setNewName(e.Value()) },
					OnKeyDown: func(e *g.Event) {
						if e.Key() == "Enter" {
							addPerson()
						}
					},
				}),
				ui.Button(ui.ButtonProps{Size: ui.ButtonSizeSm, Variant: ui.ButtonSecondary, OnClick: func(*g.Event) { addPerson() }},
					g.Data("slot", "add-person"), "add"),
			),
		),

		g.IfElse(active == "",
			g.P(g.Class("text-sm text-muted-foreground"),
				"Group view — darker cells mean more people are free. Pick a person to paint their availability."),
			g.P(g.Class("text-sm text-muted-foreground"),
				g.Textf("Painting %s's availability — click or drag across the grid.", active)),
		),

		grid(e, active, hovered, dayLabels, onDown, onOver),

		footer(e, hovered),
	)
}

func personHas(e schedule.Event, name, key string) bool {
	p := schedule.FindParticipant(e, name)
	return p.IsSome() && p.Unwrap().Slots[key]
}

func pill(label string, selected bool, data g.Option, onClick func(*g.Event)) *g.Node {
	return g.Button(
		g.Class("cursor-pointer rounded-full border px-3 py-1 text-sm transition-colors"),
		g.ClassIf(selected, "border-transparent bg-primary text-primary-foreground"),
		g.ClassIf(!selected, "hover:bg-accent"),
		data,
		g.OnClick(onClick),
		label,
	)
}

// cellProps is the render-relevant data of one grid cell; rowProps of one
// row. Rows are memoized on this data, so a paint stroke re-renders only
// the rows it touches.
type cellProps struct {
	Key string
	Cls string
	On  bool
}

type rowProps struct {
	Label  string
	Cells  []cellProps
	OnDown func(string, *g.Event)
	OnOver func(string)
}

func rowEq(old, new rowProps) bool {
	if old.Label != new.Label || len(old.Cells) != len(new.Cells) {
		return false
	}
	for i := range old.Cells {
		if old.Cells[i] != new.Cells[i] {
			return false
		}
	}
	return true
}

func gridRow(p rowProps) *g.Node {
	cells := []*g.Node{
		g.Div(g.Class("w-14 pr-2 text-right text-[11px] leading-5 text-muted-foreground"), p.Label),
	}
	for _, c := range p.Cells {
		c := c
		cells = append(cells, g.Div(
			g.Key(c.Key),
			g.Class("h-5 flex-1 cursor-pointer rounded-[2px] transition-colors", c.Cls),
			g.Data("cell", c.Key),
			g.AttrIf(c.On, "data-on", "1"),
			g.OnMouseDown(func(ev *g.Event) { p.OnDown(c.Key, ev) }),
			g.OnMouseOver(func(*g.Event) { p.OnOver(c.Key) }),
		))
	}
	return g.Li(g.Class("flex gap-px"), cells)
}

func grid(
	e schedule.Event,
	active, hovered string,
	dayLabels []string,
	onDown func(string, *g.Event),
	onOver func(string),
) *g.Node {
	slots := e.SlotsPerDay()

	headerRow := []*g.Node{g.Div(g.Class("w-14"))}
	for _, label := range dayLabels {
		headerRow = append(headerRow,
			g.Div(g.Class("flex-1 pb-1 text-center text-xs font-medium text-muted-foreground"), label))
	}

	rows := []*g.Node{g.Li(g.Key("head"), g.Class("flex gap-px"), headerRow)}
	for slot := 0; slot < slots; slot++ {
		label := ""
		if slot%2 == 0 {
			label = e.SlotLabel(slot)
		}
		cells := make([]cellProps, 0, e.Days)
		for day := 0; day < e.Days; day++ {
			key := schedule.SlotKey(day, slot)
			var cls string
			on := false
			if active != "" {
				cls = "bg-muted/40"
				if personHas(e, active, key) {
					cls = "bg-primary"
					on = true
				}
			} else {
				cls = heatClass(e.CountAt(key), len(e.People))
				if hovered == key {
					cls += " ring-1 ring-ring"
				}
			}
			cells = append(cells, cellProps{Key: key, Cls: cls, On: on})
		}
		rows = append(rows,
			g.MemoEq(gridRow, rowProps{Label: label, Cells: cells, OnDown: onDown, OnOver: onOver}, rowEq).
				WithKey(schedule.SlotKey(-1, slot)))
	}

	return g.Div(g.Class("rounded-lg border p-3"),
		g.Ul(g.Class("flex flex-col gap-px"), rows),
	)
}

func footer(e schedule.Event, hovered string) *g.Node {
	best := schedule.Best(e)
	var bestLine *g.Node
	if best.IsSome() {
		b := best.Unwrap()
		if ref := schedule.ParseKey(b.Key); ref.IsSome() {
			r := ref.Unwrap()
			day := schedule.DayLabels(e).UnwrapOrElse(func(error) []string { return make([]string, e.Days) })[r.Day]
			bestLine = g.P(g.Class("text-sm"), g.Data("slot", "best"),
				g.Textf("Best so far: %s at %s — %d of %d available", day, e.SlotLabel(r.Slot), b.Count, b.Total))
		}
	} else {
		bestLine = g.P(g.Class("text-sm text-muted-foreground"), g.Data("slot", "best"),
			"No availability painted yet.")
	}

	var hoverLine *g.Node
	if hovered != "" {
		names := e.NamesAt(hovered)
		txt := "nobody yet"
		if len(names) > 0 {
			txt = strings.Join(names, ", ")
		}
		if ref := schedule.ParseKey(hovered); ref.IsSome() {
			r := ref.Unwrap()
			hoverLine = g.P(g.Class("text-xs text-muted-foreground"), g.Data("slot", "hover-info"),
				g.Textf("%s: %s", e.SlotLabel(r.Slot), txt))
		}
	}

	return g.Div(g.Class("flex flex-col gap-1"), bestLine, hoverLine)
}
