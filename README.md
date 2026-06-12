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
  code generation. The fallible domain logic lives in
  [`schedule/schedule.warchest`](schedule/schedule.warchest): form parsing
  returns `Result[Event, ValidationError]` (a typed error), storage
  round-trips through `Result`, and "best slot" is an `Option` because an
  empty grid has no answer.

Everything is client-side; the event is persisted to localStorage. Add the
people in your group, paint, and read the result off the heatmap — darker
cells mean more people are free, and the footer names the best slot found
so far.

## Run it

Requires local checkouts of `grove` and `warchest` as sibling directories
(see the `replace` directives in go.mod).

```sh
go install github.com/gyoumi/grove/cmd/grove@latest   # or build from ../grove
grove serve        # from this directory → http://localhost:8080
```

## Develop

```sh
go generate ./schedule/        # expand schedule.warchest → schedule.go
go test ./app/ ./schedule/     # full UI simulation + domain tests
grove build                    # production bundle into dist/
```

The `app` package takes its storage as an interface, so tests drive the
entire UI — event creation, drag-painting, heatmap buckets, persistence —
in plain `go test`.
