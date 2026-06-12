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
	OnReset  func()
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

// EventView is the heart of the app: pick (or add) a participant, paint
// availability by dragging across the grid, and read the group heatmap.
func EventView(p EventProps) *g.Node {
	active, setActive := g.UseState("") // "" = group heatmap view
	newName, setNewName := g.UseState("")
	hovered, setHovered := g.UseState("")
	dragRef := g.UseRef(drag{})

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

	paint := func(key string) {
		if active == "" {
			return
		}
		p.OnUpdate(schedule.SetSlot(e, active, key, dragRef.Current.value))
	}

	cellOn := func(key string) bool {
		person := schedule.FindParticipant(e, active)
		return person.IsSome() && person.Unwrap().Slots[key]
	}

	dayLabels := schedule.DayLabels(e).UnwrapOrElse(func(error) []string {
		return make([]string, e.Days)
	})

	return g.Div(g.Class("flex flex-col gap-5 select-none"),
		// releasing the mouse anywhere ends the current paint stroke
		g.On("mouseup", func(*g.Event) { dragRef.Current = drag{} }),

		g.Div(g.Class("flex items-baseline justify-between"),
			g.H2(g.Class("text-lg font-medium"), e.Name),
			ui.Button(ui.ButtonProps{Variant: ui.ButtonGhost, Size: ui.ButtonSizeSm, OnClick: func(*g.Event) { p.OnReset() }},
				g.Data("slot", "reset"), "start over"),
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

		grid(e, active, hovered, dayLabels, cellOn, paint, setHovered, dragRef),

		footer(e, hovered),
	)
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

func grid(
	e schedule.Event,
	active, hovered string,
	dayLabels []string,
	cellOn func(string) bool,
	paint func(string),
	setHovered func(string),
	dragRef *g.Ref[drag],
) *g.Node {
	slots := e.SlotsPerDay()

	headerRow := []*g.Node{g.Div(g.Class("w-14"))}
	for _, label := range dayLabels {
		headerRow = append(headerRow,
			g.Div(g.Class("flex-1 pb-1 text-center text-xs font-medium text-muted-foreground"), label))
	}

	rows := []*g.Node{g.Div(g.Class("flex gap-px"), headerRow)}
	for slot := 0; slot < slots; slot++ {
		cells := []*g.Node{
			g.Div(g.Class("w-14 pr-2 text-right text-[11px] leading-5 text-muted-foreground"),
				g.If(slot%2 == 0, g.Text(e.SlotLabel(slot)))),
		}
		for day := 0; day < e.Days; day++ {
			key := schedule.SlotKey(day, slot)
			var cls string
			if active != "" {
				cls = "bg-muted/40"
				if cellOn(key) {
					cls = "bg-primary"
				}
			} else {
				cls = heatClass(e.CountAt(key), len(e.People))
			}
			k := key
			cells = append(cells, g.Div(
				g.Key(k),
				g.Class("h-5 flex-1 cursor-pointer rounded-[2px] transition-colors", cls),
				g.ClassIf(hovered == k && active == "", "ring-1 ring-ring"),
				g.Data("cell", k),
				g.AttrIf(active != "" && cellOn(k), "data-on", "1"),
				g.OnMouseDown(func(ev *g.Event) {
					ev.PreventDefault()
					if active == "" {
						return
					}
					dragRef.Current = drag{painting: true, value: !cellOn(k)}
					paint(k)
				}),
				g.OnMouseOver(func(*g.Event) {
					setHovered(k)
					if dragRef.Current.painting {
						paint(k)
					}
				}),
			))
		}
		rows = append(rows, g.Li(g.Key(schedule.SlotKey(-1, slot)), g.Class("flex gap-px"), cells))
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
