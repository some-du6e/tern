#!/bin/sh
# inspect.sh SESSION 'QUERY-JSON' prints inspect.js's dump of the T3 web UI
# open in agent-browser session SESSION, e.g.
#   tools/t3ui/inspect.sh task-1234 '{"selector":"[data-sidebar=footer]","depth":4}'
set -eu
here=$(dirname "$0")
python3 -c 'import sys; print(open(sys.argv[1]).read().replace("__QUERY__", sys.argv[2]))' "$here/inspect.js" "$2" |
  agent-browser --session "$1" eval --stdin
