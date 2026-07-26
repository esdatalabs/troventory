#!/bin/bash
# Stage C gate: implementation
# Usage: stage-c-gate.sh <feature> <service>
#
# This is a deterministic backstop, independent of the PreToolUse hooks — a
# subagent's own tool restrictions can only be as good as the harness
# enforcing them, so this gate re-checks the actual diff after the fact.
# It must be run from the repository root, with a clean starting tree (i.e.
# any diff present is exactly what feature-implementer produced).
#
# Exit 0 = green and in-scope.
# Exit 1 = build/test/lint failure — loop feature-implementer again.
# Exit 3 = scope violation (touched acceptance/ or a .feature file) — stop,
#          do not retry automatically, report to a human.

set -uo pipefail
FEATURE="${1:?usage: stage-c-gate.sh <feature> <service>}"
SERVICE="${2:?usage: stage-c-gate.sh <feature> <service>}"

echo "-- scope check (git diff) --"
CHANGED=$(git diff --name-only; git diff --cached --name-only; git status --porcelain | awk '{print $2}')
CHANGED=$(echo "$CHANGED" | sort -u)

if echo "$CHANGED" | grep -qE "internal/${FEATURE}/acceptance/"; then
  echo "FAIL: implementation stage touched acceptance/ files — scope violation, rejecting diff:"
  echo "$CHANGED" | grep -E "internal/${FEATURE}/acceptance/"
  exit 3
fi

if echo "$CHANGED" | grep -qE '\.feature$'; then
  echo "FAIL: a .feature file was modified during implementation — rejecting diff:"
  echo "$CHANGED" | grep -E '\.feature$'
  exit 3
fi

if echo "$CHANGED" | grep -qE "internal/${FEATURE}/providers/"; then
  echo "FAIL: implementation stage touched providers/ — that's a separate, later pipeline stage, rejecting diff:"
  echo "$CHANGED" | grep -E "internal/${FEATURE}/providers/"
  exit 3
fi

echo "-- build --"
go build ./... || { echo "FAIL: build"; exit 1; }

echo "-- test --"
go test "./internal/${FEATURE}/..." || { echo "FAIL: tests"; exit 1; }

echo "-- lint --"
if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run "./internal/${FEATURE}/..." || { echo "FAIL: lint (check depguard output for hexagon boundary violations)"; exit 1; }
else
  echo "NOTE: golangci-lint not found on PATH; skipping. Do not treat this stage as fully passed in CI without it — depguard is what mechanically catches services/ importing providers/."
fi

echo "PASS: ${FEATURE}/${SERVICE} implementation is green and in-scope"
exit 0
