#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
export PATH="$PWD/.tools/bin:$PATH"
# APS mutates the values in Examples tables. No feature has one yet, so no
# mutation can run; exception EX-EC-002 in specs/001-m365-cli/plan.md records
# the gap. This fails once a feature gains an Examples table, because nothing
# here mutates it yet.
tables=$(grep -rlE '^[[:space:]]*Examples:' features | wc -l | tr -d ' ')
if [ "$tables" -gt 0 ]; then
  echo "acceptance-mutation: $tables feature(s) have Examples tables but no mutator is wired (EX-EC-002); wire gherkin-mutator first" >&2
  exit 1
fi
printf '%s\n' '{"ir":"build/acceptance/ir"}' | go run ./acceptance/runner >/dev/null
features=$(find features -name '*.feature' | wc -l | tr -d ' ')
echo "acceptance-mutation: no example mutations ran (no Examples tables in $features features; EX-EC-002)"
