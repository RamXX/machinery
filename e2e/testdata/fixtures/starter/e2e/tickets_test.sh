#!/bin/sh
# End-to-end test of the ticket desk: it runs the real bin/tickets, asserts
# what it does, and only then emits one receipt per obligation it observed.
#
# machinery:verifies tc:TICK-* sc:agent-closes-and-reopens-a-ticket inv:closed-has-resolution
set -eu
TICKETS_DIR="$(mktemp -d)"
export TICKETS_DIR
trap 'rm -rf "$TICKETS_DIR"' EXIT
t=bin/tickets

fail() { echo "FAIL: $1" >&2; exit 1; }
receipt() {
  # receipt <id> <effects-json> <observed>
  [ -n "${MACHINERY_RECEIPTS:-}" ] || return 0
  printf '{"v":1,"type":"receipt","id":"%s","outcome":"pass","test":"tickets_test.sh","effects":%s,"observed":"%s"}\n' "$1" "$2" "$3" >> "$MACHINERY_RECEIPTS"
}

# Open ignore:reopen: refused, nothing changes
[ "$($t open T1)" = Open ] || fail "open"
$t reopen T1 2>err.txt && fail "reopen of an open ticket was accepted"
grep -q TICKET_NOT_CLOSED err.txt || fail "reopen refusal code"
[ "$($t show T1)" = Open ] || fail "state changed on a refused reopen"
[ ! -f "$TICKETS_DIR/T1.log" ] || fail "an effect ran on a refused reopen"
receipt tc:TICK-e15af4 '[]' 'reopen refused with TICKET_NOT_CLOSED, still Open'

# Open on:close -> Closed, effect recordResolution
[ "$($t close T1)" = Closed ] || fail "close"
[ "$(cat "$TICKETS_DIR/T1.log")" = resolution ] || fail "no resolution recorded"
receipt tc:TICK-837662 '["recordResolution"]' 'Open -> Closed; resolution recorded'
receipt inv:closed-has-resolution '[]' 'the closed ticket has a resolution line'

# Closed ignore:close: refused, nothing changes
$t close T1 2>err.txt && fail "second close was accepted"
grep -q TICKET_ALREADY_CLOSED err.txt || fail "close refusal code"
[ "$(wc -l < "$TICKETS_DIR/T1.log" | tr -d ' ')" = 1 ] || fail "an effect ran on a refused close"
receipt tc:TICK-484ed2 '[]' 'close refused with TICKET_ALREADY_CLOSED, still Closed'

# Closed on:reopen -> Open, effect recordReopen
[ "$($t reopen T1)" = Open ] || fail "reopen"
[ "$(tail -1 "$TICKETS_DIR/T1.log")" = reopen ] || fail "no reopen recorded"
receipt tc:TICK-5c7bf0 '["recordReopen"]' 'Closed -> Open; reopen recorded'
receipt sc:agent-closes-and-reopens-a-ticket '[]' 'opened, closed with a resolution, reopened'
rm -f err.txt
echo "tickets: all checks passed"
