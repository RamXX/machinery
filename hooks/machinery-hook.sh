#!/bin/sh
# machinery Claude Code plugin: the single hook shim. Forwards the hook event
# on stdin to `machinery hook`, which reads hook_event_name from the JSON and
# answers on stdout per the Claude Code hook contract.
#
# Deliberately boring, in this order:
#   1. Not a machinery-managed project (no .machinery.json, no
#      design/domain.modelith.yaml at the root): exit 0 with no output, so
#      the plugin never disturbs other repos or other plugins.
#   2. Managed but the machinery binary is missing: warn on stderr, exit 0.
#   3. Managed and the binary fails (e.g. older than the plugin): warn on
#      stderr, exit 0, so a version skew degrades to no governance instead
#      of breaking the session.
set -u

root="${CLAUDE_PROJECT_DIR:-$PWD}"
if [ ! -f "$root/.machinery.json" ] && [ ! -f "$root/design/domain.modelith.yaml" ]; then
  exit 0
fi

bin="$(command -v machinery 2>/dev/null || true)"
if [ -z "$bin" ] && [ -x "$HOME/.local/bin/machinery" ]; then
  bin="$HOME/.local/bin/machinery"
fi
if [ -z "$bin" ]; then
  echo "machinery plugin: this project is machinery-managed but the 'machinery' binary is not on PATH; governance hooks skipped. Install it: curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh" >&2
  exit 0
fi

if ! "$bin" hook --root "$root"; then
  echo "machinery plugin: 'machinery hook' failed; the binary may be older than the plugin. Re-run the installer to upgrade." >&2
fi
exit 0
