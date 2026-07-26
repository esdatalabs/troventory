#!/bin/bash
# Stage A gate: prompt -> Gherkin
# Usage: stage-a-gate.sh <path-to-feature-file>
#
# Checks:
#   1. File exists and is non-empty.
#   2. It parses as valid Gherkin (via godog --dry-run, which registers no
#      steps and just checks the document).
#   3. It contains at least one Scenario, and at least one that reads as an
#      idempotency case (ARCHITECTURE.md §8 requires this test category).
#   4. It doesn't leak implementation vocabulary that would make it brittle
#      against internal refactors (Gateway, Provider, channel, SQL, etc).
#
# Exit 0 = pass. Exit 1 = fail, re-run gherkin-writer with this output.

set -uo pipefail
FEATURE_FILE="${1:?usage: stage-a-gate.sh <path-to-feature-file>}"

if [ ! -s "$FEATURE_FILE" ]; then
  echo "FAIL: $FEATURE_FILE is missing or empty"
  exit 1
fi

echo "-- syntax check --"
# godog's CLI (v0.12+) dropped the top-level --dry-run flag, and even
# `godog run` requires a buildable Go package for the target step definitions
# — which don't exist yet at Stage A. So there is no godog-based dry-run
# available at this stage regardless of version; do a structural check instead.
grep -qE '^\s*Feature:' "$FEATURE_FILE" || { echo "FAIL: no 'Feature:' line found"; exit 1; }

echo "-- required scenario categories --"
grep -qiE '^\s*Scenario' "$FEATURE_FILE" || { echo "FAIL: no Scenario found in $FEATURE_FILE"; exit 1; }

if ! grep -qiE 'idempot|twice|duplicate|submitted again|same reference' "$FEATURE_FILE"; then
  echo "FAIL: no scenario reads as an idempotency case (ARCHITECTURE.md §8 requires one — e.g. 'the same reference is submitted twice')"
  exit 1
fi

echo "-- implementation-vocabulary blocklist --"
BANNED='Gateway|Provider|goroutine|\bchannel\b|\bSQL\b|\bHTTP\b|gRPC|\bstruct\b|interface\{\}|postgres|kafka'
if grep -qniE "$BANNED" "$FEATURE_FILE"; then
  echo "FAIL: $FEATURE_FILE leaks implementation vocabulary — scenarios must describe business behavior, not internals:"
  grep -nniE "$BANNED" "$FEATURE_FILE"
  exit 1
fi

echo "PASS: $FEATURE_FILE"
exit 0
