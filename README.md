# overlap

A when2meet-style scheduling app: set up an event, add the people involved,
paint each person's availability on a grid, and watch the group heatmap
reveal where schedules align.

Built to exercise two of my other projects together:

- **[grove](https://github.com/gyoumi/grove)** — a React-style framework
  for Go compiled to WebAssembly (virtual DOM, hooks, Tailwind/shadcn
  styling). The whole UI is grove components; the interaction tests run
  against grove's `testdom` with no browser.
- **[warchest](https://github.com/gyoumi/warchest-errors)** — explicit
  error handling for Go (`Option[T]`, `Result[T, E]`, `?` propagation) via
  code generation. **The repo contains no `.go` files at all** — every Go
  source is authored as `.warchest` and the generated `.go` stays local
  (gitignored). Most files pass through the generator untouched; the
  fallible domain logic in
  [`schedule/schedule.warchest`](schedule/schedule.warchest) uses the
  extras: form parsing returns `Result[Event, ValidationError]` (a typed
  error), storage round-trips through `Result`, and "best slot" is an
  `Option` because an empty grid has no answer.

Everything is client-side; events are persisted to localStorage. Add the
people in your group, paint, and read the result off the heatmap — darker
cells mean more people are free, and the footer names the best slot found
so far.

The app keeps multiple events behind `grove/router` hash routes (`#/` for
the list, `#/event/<id>` per grid), and grid rows are memoized with
`g.MemoEq` so a paint stroke only re-renders the rows it touches — their
handlers read the latest state through a `UseRef`, the pattern grove's
Memo docs describe. Older single-event saves migrate into the workspace
format on load.

## Run it

Requires local checkouts of `grove` and `warchest` as sibling directories
(see the `replace` directives in go.mod).

```sh
go install github.com/gyoumi/grove/cmd/grove@latest   # or build from ../grove
go tool warchest generate ./...   # expand .warchest → .go (fresh checkouts)
grove serve                       # from this directory → http://localhost:8080
```

## Develop

Edit the `.warchest` files (never the generated `.go`), then:

```sh
go tool warchest generate ./...   # expand every *.warchest → *.go
go test ./app/ ./schedule/        # full UI simulation + domain tests
grove build                       # production bundle into dist/
```

(`go generate ./...` also works once the `.go` files exist locally — each
package's directive is carried in its generated output.)

The `app` package takes its storage and dark-mode hook as injected
dependencies, so tests drive the entire UI — event creation,
drag-painting, heatmap buckets, dark mode, persistence — in plain
`go test`.
