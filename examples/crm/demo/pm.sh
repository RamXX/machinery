#!/bin/sh
# The planner's loop on a throwaway copy of the repository with its own bd
# backlog: a design change becomes owed obligations and stories, the stories
# become waves under a frozen contract, and two stories whose ids are proven
# pass their tier gates with a builder and a separate verifier. Nothing here
# touches this repository, its git or any backlog outside the copy. Needs
# make up, bd and Java-free machinery; the copy is removed at the end.
set -eu

crm=$(cd "$(dirname "$0")/.." && pwd)
repo=$(cd "$crm/../.." && pwd)
MACHINERY=${MACHINERY:-$repo/.bin/machinery}
MACHINERY_TOOLS=${MACHINERY_TOOLS:-$repo/.bin}
export MACHINERY_TOOLS
PATH=$(dirname "$MACHINERY"):$MACHINERY_TOOLS:$PATH
export PATH
STACK="postgres 17 in docker, crm over http"

die() { echo "pm.sh: $*" >&2; exit 1; }
[ -x "$MACHINERY" ] || die "no machinery at $MACHINERY: run make build at the repository root"
command -v bd >/dev/null 2>&1 || die "bd is not on PATH"
health=$(docker inspect -f '{{.State.Health.Status}}' machinery-crm-pg 2>/dev/null || true)
[ "$health" = healthy ] || die "the database is not reachable (container machinery-crm-pg: ${health:-absent}): run make up first"

# Only the copy's own backlog may be found.
unset BEADS_DIR BEADS_DB BEADS_ACTOR
tmp=$(mktemp -d "${TMPDIR:-/tmp}"/crm-pm.XXXXXX)
tmp=$(cd "$tmp" && pwd -P)
trap 'rm -rf "$tmp"; echo; echo "pm.sh: removed the copy"' EXIT
trap 'exit 130' INT TERM

# show prints a command as it would be typed.
show() {
	line='$'
	for a in "$@"; do
		case $a in
		*[[:space:]]* | '') line="$line '$a'" ;;
		*) line="$line $a" ;;
		esac
	done
	printf '\n%s\n' "$line"
}
run() { show "$@"; "$@"; }
# may runs a command whose findings are the point, and goes on.
may() {
	show "$@"
	rc=0
	"$@" || rc=$?
	echo "(exit $rc)"
}
as() {
	actor=$1
	shift
	show "BEADS_ACTOR=$actor" "$@"
	BEADS_ACTOR=$actor "$@"
}
refused() {
	actor=$1
	shift
	show "BEADS_ACTOR=$actor" "$@"
	if BEADS_ACTOR=$actor "$@"; then die "expected a refusal: $*"; fi
	echo "(refused, as expected)"
}
commit() { run git commit -q -m "$1"; }

echo "== 1. a copy of the repository with its own git and backlog"
git -C "$repo" archive --format=tar HEAD | tar -x -C "$tmp"
cd "$tmp"
git init -q -b main
git config user.name planner
git config user.email planner@example.test
git add -A
git commit -q -m "copy of the repository"
echo "copy: $tmp, one commit"
show bd init --stealth --non-interactive -p crm
bd init --stealth --non-interactive -p crm >/dev/null
cd examples/crm
printf 'backlog: {tool: bd, base: main}\n' >>machinery.yaml
# The copy has a history of its own: its formal models were read at its
# first commit, which holds the same sources.
first=$(git rev-parse HEAD)
for f in design/formal/*.spec.yaml; do
	sed "s/^  read_at: .*/  read_at: $first/" "$f" >"$tmp/spec" && cat "$tmp/spec" >"$f"
done
run git add -A
commit "crm: plan in bd; the formal models were read at the first commit"

echo
echo "== 2. the planner: a design change, owed, planned, in waves"
cat >>design/model/crm.modelith.yaml <<'EOF'
  - name: Admin cancels a task assigned to a Rep
    actors: [Rep, Admin]
    steps:
      - "Rep creates a `Task` on a `Deal`; it is open."
      - "Admin cancels the `Task`; it is cancelled and the audit trail names the Admin."
    invariants_touched: [task-assignee-or-admin-writes]
EOF
cat >>design/failure-modes.yaml <<'EOF'
  - id: deal-qualify-without-user
    kind: abuse
    subject: act:Deal.qualify
    failure: "A caller without a known User (X-User-Id missing or naming no User) asks to qualify a deal."
    required: refuse
    behavior: "The request is refused with 401 UNAUTHENTICATED and the deal stays a Lead."
EOF
echo "design: a new scenario in the model, a new abuse failure mode"
run machinery gen
run git add -A
commit "crm: admin cancels a task; qualify without a user"
may machinery check --stage formal
may machinery formal sync record-ownership
run machinery formal sync record-ownership --class no-effect
run git add -A
commit "crm: the new scenario does not change who may write"
# The suite runs once, here: the owed list grows by what the change made
# new and unproven, and the evidence stays valid while only the owed file
# changes.
run machinery run --stack "$STACK" -- env GOMAXPROCS=2 go test ./e2e/... -count=1 -p 1
run machinery owed sync
run machinery owed list
run git add design/owed.yaml
commit "crm: owe the new scenario and the new abuse case"
run machinery plan --apply
run machinery waves
run machinery waves --freeze
# The stories plan --apply created, by the owed entries it linked.
story_of() { awk -v p="$1" '$1 == "-" && $2 == "id:" { id = $3 } $1 == "story:" && id ~ p { print $2; exit }' design/owed.yaml; }
fm=$(story_of '^fm:deal-qualify-without-user$')
sc=$(story_of '^sc:')
[ -n "$fm" ] && [ -n "$sc" ] && [ -n "$sc" ] || die "plan --apply did not create the fm and sc stories"
run machinery brief "$fm"
run machinery brief "$fm" --role verifier
run machinery brief "$sc" --role modeler

echo
echo "== 3. the gates, on stories whose ids are proven"
tables=design/gen/tables
task_ids=$(awk -F'|' '$4 ~ /on:/ { gsub(/ /, "", $2); printf " %s", $2 }' "$tables/task.table.md")
win=$(awk -F'|' '$4 ~ /on:win/ && $6 !~ /internal/ { gsub(/ /, "", $2); print $2 }' "$tables/deal.table.md")
win_ids=$(awk -F'|' -v t="$win" '{ c = $3; gsub(/ /, "", c) } $4 ~ /on:win/ || c == t { gsub(/ /, "", $2); printf " %s", $2 }' "$tables/deal.table.md")
printf 'machinery:ids act:Task.start act:Task.complete act:Task.cancel%s\nmachinery:scope internal/api/tasks.go internal/api/units_task.go\n' "$task_ids" >"$tmp/tasks.txt"
printf 'machinery:ids act:Deal.win%s\nmachinery:scope internal/api/units_deal.go internal/store/wins.go\n' "$win_ids" >"$tmp/win.txt"
run bd create --id crm-tasks --title "Serve the task writes" --body-file "$tmp/tasks.txt" --silent
run bd create --id crm-win --title "Serve the deal win" --body-file "$tmp/win.txt" --silent
run machinery status
for s in crm-tasks crm-win; do
	as builder machinery proof --story "$s" --post
done
may machinery story crm-tasks
refused builder machinery predict crm-tasks --observation "a Rep writes a task it is not assigned" --expect "403 and one refused audit row"
as verifier machinery predict crm-tasks --observation "a Rep writes a task it is not assigned" --expect "403 and one refused audit row"
as verifier machinery accept crm-tasks --verdict accept --observed "403 FORBIDDEN, the task unchanged, one refused audit row" --close
as verifier machinery predict crm-win --observation "two callers win one deal at once and both succeed" --expect "one 200, one 409, one win audit row"
as verifier machinery accept crm-win --verdict accept --observed "the replay was refuted in every round" --close
run machinery story crm-tasks
run machinery story crm-win
