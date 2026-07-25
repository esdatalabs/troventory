#!/bin/bash
# Stage B gate: Gherkin -> failing (compile-error) test
# Usage: stage-b-gate.sh <feature> <service>
#   e.g. stage-b-gate.sh payments transfer
#
# "Failing test" is defined strictly as compile-error red here: the step defs
# reference symbols (types, constructors, methods) that don't exist yet, so
# `go build` fails pointing at exactly those symbols.
#
# This gate distinguishes two causes of the same non-zero exit code:
#   - legitimate red: undefined symbols are ones the manifest says are
#     supposed to be missing (they belong to Stage C).
#   - false red: an undefined symbol NOT in the manifest — a bug in the step
#     defs/fakes themselves, which must go back to step-writer, not forward
#     to feature-implementer.
#
# Exit 0 = legitimate compile-error red confirmed, red-reason captured.
# Exit 1 = not red, or a structural problem (missing manifest, etc).
# Exit 2 = false red — send back to step-writer.

set -uo pipefail
FEATURE="${1:?usage: stage-b-gate.sh <feature> <service>}"
SERVICE="${2:?usage: stage-b-gate.sh <feature> <service>}"

MANIFEST="internal/${FEATURE}/acceptance/.contract/${SERVICE}.yaml"
RED_REASON="/tmp/red-reason-${FEATURE}-${SERVICE}.txt"

if [ ! -f "$MANIFEST" ]; then
  echo "FAIL: manifest $MANIFEST not found — step-writer must emit one alongside the step defs"
  exit 1
fi

echo "-- build --"
BUILD_OUT=$(go build ./... 2>&1)
STATUS=$?

if [ $STATUS -eq 0 ]; then
  echo "FAIL: build is unexpectedly green — nothing is red yet. Either the service already exists, or the step defs aren't exercising anything real."
  exit 1
fi

echo "$BUILD_OUT"
echo "-- checking undefined symbols against manifest --"

UNDEFINED=$(echo "$BUILD_OUT" | grep -oE 'undefined: [A-Za-z0-9_.\*()]+' | sed 's/undefined: //' | sort -u)

if [ -z "$UNDEFINED" ]; then
  echo "FAIL: build failed, but not with an 'undefined: X' compile error. This might be a syntax error or bad import in the step defs themselves — that's a step-writer bug, not TDD red."
  exit 2
fi

if command -v yq >/dev/null 2>&1; then
  EXPECTED=$(yq -r '.symbols[]' "$MANIFEST" | sort -u)
else
  # Fallback for the manifest's simple `symbols:\n  - foo\n  - bar` shape —
  # not a general YAML parser, just enough to read this one list.
  EXPECTED=$(grep -E '^\s*-\s' "$MANIFEST" | sed -E 's/^\s*-\s*//' | sort -u)
fi
UNLISTED=0

for sym in $UNDEFINED; do
  if ! grep -qxF "$sym" <<< "$EXPECTED"; then
    echo "  '$sym' is undefined but NOT listed in $MANIFEST"
    UNLISTED=1
  fi
done

if [ "$UNLISTED" -eq 1 ]; then
  echo "FAIL: at least one compile error isn't accounted for in the manifest. This looks like a bug in the step defs or fakes, not legitimate TDD red — send back to step-writer, do not forward to feature-implementer."
  exit 2
fi

echo "$BUILD_OUT" > "$RED_REASON"
echo "PASS: compile-error red confirmed. Every undefined symbol is declared in $MANIFEST."
echo "Red reason captured at $RED_REASON — hand this to feature-implementer."
exit 0
