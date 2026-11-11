#!/bin/sh
# make acceptance-mutation: the APS example-mutation gate. For each feature
# with an Examples table, gherkin-mutator changes one example cell at a time
# and runs that feature's acceptance test against the changed IR through the
# persistent runner worker (acceptance/runner). A change no test notices
# survives and fails the gate; so does an error or no feature to mutate.
# Every run executes every mutation (APS level "full"). The mutator writes
# its manifest into the feature it tests, so it runs on a copy under build/
# and features/ never changes.
set -eu
cd "$(dirname "$0")/.."
export PATH="$PWD/.tools/bin:$PATH"
if ! command -v gherkin-mutator >/dev/null 2>&1; then
  echo "acceptance-mutation: gherkin-mutator missing; run scripts/install-tools.sh" >&2
  exit 1
fi
work=build/acceptance-mutation
rm -rf "$work"
mkdir -p "$work"
grep -rlE '^[[:space:]]*Examples:' features | sort > "$work/features.txt" || true
if [ ! -s "$work/features.txt" ]; then
  echo "acceptance-mutation: no feature has an Examples table; nothing to mutate" >&2
  exit 1
fi
# IR, generated tests and metadata for this run; acceptance.sh writes the
# feature list (test name, path) to build/acceptance/features.tsv.
sh scripts/acceptance.sh >/dev/null
go test -c -o "$work/generated.test" ./acceptance/generated
go build -o "$work/worker" ./acceptance/runner
tab=$(printf '\t')
status=0
while IFS= read -r f; do
  name=$(awk -F"$tab" -v p="$f" '$2 == p { print $1 }' build/acceptance/features.tsv)
  if [ -z "$name" ]; then
    echo "acceptance-mutation: no acceptance test for $f" >&2
    status=1
    continue
  fi
  echo "acceptance-mutation: $f" >&2
  mkdir -p "$work/$name"
  copy="$work/$name/$(basename "$f")"
  cp "$f" "$copy"
  if ! gherkin-mutator --feature "$copy" --generated-dir build/acceptance \
    --work-dir "$work/$name" --level full --timeout 10m \
    --runner-worker "$work/worker -bin $work/generated.test -test TestAcceptance_$name -dir acceptance/generated" </dev/null; then
    status=1
  fi
done < "$work/features.txt"
exit $status
