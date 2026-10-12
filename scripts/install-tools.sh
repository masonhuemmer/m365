#!/bin/sh
set -eu
# APS SHA recorded: accaa33d503340c56513ef387258f8da929ba902
# unclebob/Acceptance-Pipeline-Specification 2026-06-12
ROOT="$(CDPATH="" cd "$(dirname "$0")/.." && pwd)"
APS_SHA="accaa33d503340c56513ef387258f8da929ba902"
# Build tools with the toolchain CI uses (go.mod's toolchain line), so a
# newer local Go cannot produce tools that behave differently from CI.
GOTOOLCHAIN="${GOTOOLCHAIN:-$(sed -n 's/^toolchain //p' "$ROOT/go.mod")}"
export GOTOOLCHAIN
mkdir -p "$ROOT/.tools/bin"
GOBIN="$ROOT/.tools/bin"
export GOBIN
# gosec v2.29.0 (the latest tag) cannot read Go 1.27's compiler export data and
# reports "type errors" for every package. Upstream fixed it on main
# (securego/gosec#1724, #1772); this commit works on Go 1.26.9 and 1.27.2.
# Go back to a tag as soon as one newer than v2.29.0 is released.
go install github.com/securego/gosec/v2/cmd/gosec@71577268350e
go install golang.org/x/vuln/cmd/govulncheck@v1.6.0
go install github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0
if [ -d "$HOME/.local/bin" ]; then
  ln -sf "$GOBIN/gosec" "$HOME/.local/bin/gosec"
  ln -sf "$GOBIN/govulncheck" "$HOME/.local/bin/govulncheck"
  ln -sf "$GOBIN/gocyclo" "$HOME/.local/bin/gocyclo"
fi
if [ -n "${GITHUB_PATH:-}" ]; then
  echo "$GOBIN" >> "$GITHUB_PATH"
fi
if [ ! -d "$ROOT/.tools/aps/.git" ]; then
  git clone --filter=blob:none https://github.com/unclebob/Acceptance-Pipeline-Specification.git "$ROOT/.tools/aps"
fi
git -C "$ROOT/.tools/aps" fetch --depth 1 origin "$APS_SHA"
git -C "$ROOT/.tools/aps" checkout "$APS_SHA"
GOBIN="$ROOT/.tools/bin" go install -C "$ROOT/.tools/aps" ./cmd/gherkin-parser ./cmd/gherkin-ir-dry-checker ./cmd/gherkin-mutator
echo "tools installed APS=$APS_SHA"
