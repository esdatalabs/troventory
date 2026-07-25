#!/bin/bash
# PreToolUse hook for the feature-implementer subagent.
# Fires on Write|Edit. This is the single most important boundary in the
# pipeline: the fastest way an LLM turns red green is editing the test
# instead of writing the feature. This hook makes that structurally
# unavailable rather than merely discouraged in a prompt.

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

if [[ "$FILE" =~ /acceptance/ ]]; then
  echo "Blocked: feature-implementer must never modify anything under acceptance/ — that tampers with the test instead of making it pass. Attempted: $FILE" >&2
  exit 2
fi

if [[ "$FILE" =~ \.feature$ ]]; then
  echo "Blocked: feature-implementer must never edit .feature files. Attempted: $FILE" >&2
  exit 2
fi

if [[ "$FILE" =~ /providers/ ]]; then
  echo "Blocked: feature-implementer builds against fakes only — real Providers are a separate, later stage. Attempted: $FILE" >&2
  exit 2
fi

exit 0
