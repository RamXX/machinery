#!/bin/sh
# machinery Claude Code/Codex plugin: the single hook shim. Forwards the hook
# event on stdin to `machinery hook`, which reads hook_event_name from the JSON
# and answers on stdout using the hosts' compatible hook contract.
#
# Deliberately boring, in this order:
#   1. If the machinery binary exists, always dispatch. The Go hook owns the
#      managed/unmanaged decision and can recover a durable pre-shell route
#      after a shell command deletes or renames both project markers.
#   2. If the binary is absent, a truly unmanaged project stays a silent no-op.
#   3. Managed with no binary, or a failing binary, fails closed (exit 2)
#      with a diagnostic. A broken guard must never become no governance.
set -u

if [ -n "${CLAUDE_PROJECT_DIR:-}" ]; then
  root="$CLAUDE_PROJECT_DIR"
else
  root="$PWD"
fi
bin="$(command -v machinery 2>/dev/null || true)"
if [ -z "$bin" ] && [ -n "${HOME:-}" ] && [ -x "$HOME/.local/bin/machinery" ]; then
  bin="$HOME/.local/bin/machinery"
fi

# Marker probing is only the missing-binary diagnostic boundary. Never use it
# to skip an available Go hook: the markers are mutable by the shell event
# whose durable ledger the subsequent Post/Stop events must recover.
path_present() {
  [ -e "$1" ] || [ -L "$1" ]
}
if [ -z "$bin" ]; then
  if ! probe=$(unset CDPATH; cd -- "$root" 2>/dev/null && pwd -P); then
    echo "machinery plugin: BLOCKED because the project root cannot be inspected while the 'machinery' binary is unavailable." >&2
    exit 2
  fi
  managed=0
  while :; do
    if path_present "$probe/.machinery.json" || path_present "$probe/design/domain.modelith.yaml"; then
      managed=1
      break
    fi
    if [ "$probe" = / ]; then
      break
    fi
    parent=${probe%/*}
    if [ -z "$parent" ]; then
      parent=/
    fi
    if ! next=$(unset CDPATH; cd -- "$parent" 2>/dev/null && pwd -P); then
      echo "machinery plugin: BLOCKED because an ancestor of '$probe' cannot be inspected while the 'machinery' binary is unavailable." >&2
      exit 2
    fi
    if [ "$next" = "$probe" ]; then
      break
    fi
    probe=$next
  done
  if [ "$managed" -eq 0 ]; then
    exit 0
  fi
fi

if [ -z "$bin" ]; then
  echo "machinery plugin: BLOCKED because this project is machinery-managed but the 'machinery' binary is unavailable. Install it: curl -fsSL https://raw.githubusercontent.com/RamXX/machinery/main/install.sh | sh" >&2
  exit 2
fi

if ! "$bin" hook --root "$root"; then
  echo "machinery plugin: BLOCKED because 'machinery hook' could not complete; see the diagnostic above. Retry after any active machinery install, update, or uninstall finishes. If the failure persists, run 'machinery doctor'; reinstall only when doctor reports an installation or version mismatch." >&2
  exit 2
fi
exit 0
