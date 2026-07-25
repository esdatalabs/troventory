---
description: Run one full red/green TDD cycle for a single service — prompt to Gherkin to failing test to implementation to passing test.
argument-hint: <feature> <service> <plain-language description of the behavior>
allowed-tools: Agent(gherkin-writer, step-writer, feature-implementer) Bash(scripts/tdd-gates/*.sh:*) Read
---

Run this pipeline for feature `$1`, service `$2`, given this behavior description:

> $3

Follow the stages in order below. Do not skip or reorder a stage. Each stage's gate script is the actual pass/fail check — trust it over any subagent's own self-report, and never advance past a failing gate.

## Stage A — Gherkin
1. Delegate to the `gherkin-writer` subagent: write scenarios in `internal/$1/acceptance/features/$2.feature` for the behavior above.
2. Run: `scripts/tdd-gates/stage-a-gate.sh internal/$1/acceptance/features/$2.feature`
3. On failure, re-delegate to `gherkin-writer` with the gate's exact output. Retry up to 3 times total, then stop and report to the user.

## Stage B — Failing test (compile-error red)
1. Delegate to the `step-writer` subagent: write step definitions, world wiring, fakes, and a symbol manifest for `internal/$1/acceptance/features/$2.feature`, targeting `internal/$1/services/$2`.
2. Run: `scripts/tdd-gates/stage-b-gate.sh $1 $2`
3. If it exits 2 (false red — an undefined symbol not in the manifest), re-delegate to `step-writer` with the gate's output; this is a bug in the test harness, not legitimate progress. Retry up to 3 times.
4. If it exits 0, proceed with the captured red-reason file it reports.
5. If it exits 1 for any other reason, stop and report — don't guess at a fix yourself.

## Stage C — Implementation
1. Delegate to the `feature-implementer` subagent, giving it the red-reason file's contents and the manifest, plus: make every symbol in the manifest exist per ARCHITECTURE.md, until the acceptance tests for `internal/$1/services/$2` pass.
2. Run: `scripts/tdd-gates/stage-c-gate.sh $1 $2`, looping up to 5 attempts, feeding the exact failing output back to `feature-implementer` each time.
3. If the gate exits 3 (scope violation — it touched acceptance/, a .feature file, or providers/), stop immediately. Do not retry automatically. Report the violation to the user; this is not a bug to route around.

## Stage D — Full regression
1. Run: `scripts/tdd-gates/stage-d-gate.sh`
2. Report the final pass/fail plus a short summary of what was written at each stage (files touched, scenarios covered, symbols implemented).
