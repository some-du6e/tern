#!/bin/sh
# unpair-browser.sh SESSION LABEL closes a browser that pair-browser.sh opened
# and revokes its T3 session, the one listed as "client: LABEL".
set -eu
t3=${T3:-$(ls -d /root/.t3/runtime/versions/*/t3 | sort -V | tail -1)}
agent-browser --session "$1" close >/dev/null 2>&1 || true
"$t3" auth session list 2>/dev/null | python3 -c '
import re, sys
cur = None
for line in sys.stdin:
    m = re.match(r"^([0-9a-f]{8}-[0-9a-f-]{27})\s*$", line)
    if m:
        cur = m.group(1)
    elif line.strip().startswith("client: " + sys.argv[1] + " ") and cur:
        print(cur)
' "$2" | while read -r id; do "$t3" auth session revoke "$id" >/dev/null 2>&1 && echo "revoked $id"; done
