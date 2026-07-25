#!/bin/bash
# PreToolUse hook for the step-writer subagent.
# Fires on Write|Edit. Allows:
#   - step definitions under acceptance/steps/
#   - the scenario world file (acceptance/world.go)
#   - the godog test runner (acceptance/main_test.go)
#   - hand-written fakes in an exported <pkg>test package (never a _test.go file,
#     since those aren't importable from the acceptance package)
#   - the symbol manifest under acceptance/.contract/
# Never entities/, services/<service>/ (non-test files), providers/, cmd/, or
# the .feature file itself.

INPUT=$(cat)
if command -v jq >/dev/null 2>&1; then
  FILE=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')
else
  FILE=$(echo "$INPUT" | python3 -c 'import json,sys
try:
    d = json.load(sys.stdin)
except Exception:
    print("")
else:
    print(d.get("tool_input", {}).get("file_path") or "")')
fi

[ -z "$FILE" ] && exit 0

if [[ "$FILE" =~ /acceptance/steps/ ]] \
   || [[ "$FILE" =~ /acceptance/world\.go$ ]] \
   || [[ "$FILE" =~ /acceptance/main_test\.go$ ]] \
   || [[ "$FILE" =~ /acceptance/\.contract/[^/]+\.ya?ml$ ]] \
   || [[ "$FILE" =~ /[a-zA-Z0-9_]+test/[a-zA-Z0-9_]+\.go$ ]]; then
  exit 0
fi

echo "Blocked: step-writer may only write step definitions, world.go, the godog runner, <pkg>test fakes, or the .contract manifest. Attempted: $FILE" >&2
exit 2
