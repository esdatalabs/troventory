#!/bin/bash
# PreToolUse hook for the feature-implementer subagent, matcher: Bash.
# feature-implementer needs Bash for `go build`/`go test`/`golangci-lint`, which
# also means it could route around restrict-implementation.sh by shelling out
# (sed -i, redirection, rm, mv, cp, git checkout/restore) instead of using the
# Edit/Write tools directly. This is a heuristic backstop, not a full parser —
# it blocks the obvious cases and errs toward blocking when uncertain.

INPUT=$(cat)
if command -v jq >/dev/null 2>&1; then
  COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')
else
  COMMAND=$(echo "$INPUT" | python3 -c 'import json,sys
try:
    d = json.load(sys.stdin)
except Exception:
    print("")
else:
    print(d.get("tool_input", {}).get("command") or "")')
fi

[ -z "$COMMAND" ] && exit 0

if echo "$COMMAND" | grep -qE '(acceptance/|\.feature\b)'; then
  if echo "$COMMAND" | grep -qE '(>{1,2}|sed[[:space:]]+-i|rm[[:space:]]|mv[[:space:]]|cp[[:space:]]|git[[:space:]]+(checkout|restore|apply|rm))'; then
    echo "Blocked: this command appears to write to acceptance/ or a .feature file via the shell. feature-implementer must not modify test files by any means, direct or indirect." >&2
    exit 2
  fi
fi

exit 0
