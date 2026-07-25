#!/bin/bash
# PreToolUse hook for the gherkin-writer subagent.
# Fires on Write|Edit. Allows only .feature files under an acceptance/features/
# directory. This is Stage A's file-scope boundary from the pipeline design:
# gherkin-writer produces scenarios, nothing else.

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

# No file_path on this tool call (shouldn't happen for Write/Edit) — let it through.
[ -z "$FILE" ] && exit 0

if [[ "$FILE" =~ /acceptance/features/[^/]+\.feature$ ]]; then
  exit 0
fi

echo "Blocked: gherkin-writer may only write *.feature files under an acceptance/features/ directory. Attempted: $FILE" >&2
exit 2
