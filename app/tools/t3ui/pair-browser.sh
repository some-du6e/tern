#!/bin/sh
# pair-browser.sh SESSION LABEL opens a throwaway agent-browser session on
# the T3 web UI of this machine's server, paired with a short-lived token
# that can read threads and open terminals and preview tabs but not change
# threads. The token never reaches the output. The viewport is 1440x900 at
# 2x, the colour scheme light. Revoke the session afterwards with
#   t3 auth session revoke <id>   (the one listed as "client: LABEL").
set -eu
t3=${T3:-$(ls -d /root/.t3/runtime/versions/*/t3 | sort -V | tail -1)}
url=$("$t3" auth pairing create --scope orchestration:read --scope terminal:read --scope terminal:operate \
  --scope preview:operate --ttl 10m --label "$2" --base-url http://127.0.0.1:3773 --json 2>/dev/null |
  python3 -c 'import json,sys; print(json.load(sys.stdin)["pairUrl"])')
agent-browser --session "$1" set viewport 1440 900 2 >/dev/null 2>&1
agent-browser --session "$1" set media light >/dev/null 2>&1
agent-browser --session "$1" open "$url" >/dev/null 2>&1
agent-browser --session "$1" wait --load load >/dev/null 2>&1
agent-browser --session "$1" get title
