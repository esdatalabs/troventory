#!/bin/bash
# Stage D gate: full regression, run after any Stage C loop breaks clean.
# Not scoped to one feature — this is the check that a feature-by-feature
# pipeline didn't quietly break something outside its own package.

set -uo pipefail

echo "-- full test suite (race detector) --"
go test -race ./... || { echo "FAIL: full test suite"; exit 1; }

echo "-- full lint --"
if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run ./... || { echo "FAIL: full lint"; exit 1; }
else
  echo "NOTE: golangci-lint not found on PATH; skipping full lint pass."
fi

echo "PASS: full repo green"
exit 0
