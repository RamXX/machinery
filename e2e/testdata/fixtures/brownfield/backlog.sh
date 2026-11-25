#!/bin/sh
# Creates the project's own bd backlog in the current directory: one epic, four
# stories, two blocking edges, one story closed. It only ever touches the
# database of the directory it runs in.
set -eu
[ ! -e .beads ] || { echo "backlog.sh: .beads already exists here" >&2; exit 1; }
unset BEADS_DIR BEADS_DB
export BD_NON_INTERACTIVE=1
bd init --stealth --non-interactive -p depot >/dev/null
epic=$(bd create "Depot shipping" -t epic --silent)
lifecycle=$(bd create "Shipment lifecycle" -t story --parent "$epic" --silent)
dispatch=$(bd create "Dispatch rules" -t story --parent "$epic" --silent)
bd create "Invoice rounding" -t story --parent "$epic" --silent >/dev/null
receipts=$(bd create "Delivery receipts" -t story --parent "$epic" --silent)
bd dep add "$dispatch" "$lifecycle" >/dev/null
bd dep add "$receipts" "$dispatch" >/dev/null
bd close "$lifecycle" --reason "shipped in the first release" >/dev/null
echo "backlog.sh: 1 epic, 4 stories"
