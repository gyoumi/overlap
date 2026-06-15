#!/usr/bin/env bash
# Bootstrap overlap from a fresh clone: fetch the warchest generator, install the
# grove dev-server CLI, and expand the .warchest sources into .go.
#
#     git clone <overlap> && cd overlap && ./setup.sh && grove serve
#
# grove is a public Go module, fetched automatically by `go`. The warchest
# generator lives in a separate (private) repo that overlap references via a
# `replace` directive as a sibling checkout — this script clones it for you.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 1. warchest generator: a sibling checkout that satisfies the go.mod `replace`.
warchest_dir="$(cd "$here/.." && pwd)/warchest-errors"
warchest_repo="git@github.com:gyoumi/warchest-errors.git"
if [ -d "$warchest_dir/.git" ]; then
  echo "==> warchest generator already present: $warchest_dir"
else
  echo "==> Cloning the warchest generator -> $warchest_dir"
  echo "    (private repo; needs SSH access to github.com/gyoumi/warchest-errors)"
  git clone "$warchest_repo" "$warchest_dir"
fi

# 2. grove dev-server CLI (`grove serve` / `grove build`). The grove *module* is
#    public and fetched by `go` on demand; this installs the CLI binary.
echo "==> Installing the grove CLI"
go install github.com/gyoumi/grove/cmd/grove@latest

# 3. PATH check so the freshly installed `grove` command is found.
bin="$(go env GOBIN)"; [ -n "$bin" ] || bin="$(go env GOPATH)/bin"
case ":$PATH:" in
  *":$bin:"*) ;;
  *) echo "!! $bin is not on your PATH — add it so the 'grove' command is found." ;;
esac

# 4. Expand every .warchest -> .go (gitignored; rerun whenever sources change).
echo "==> Generating Go sources (go tool warchest generate ./...)"
(cd "$here" && go tool warchest generate ./...)

echo
echo "Done. Start the app with:"
echo "    grove serve     # -> http://localhost:8080"
echo
echo "Editing .warchest in an editor? See ../warchest-errors/setup.sh for the"
echo "warchestls LSP (diagnostics, hover, completion)."
