# DEMO: the reference project, end to end

Follow it top to bottom. Each step is one command and what it prints (shortened: `...` marks a
cut, `<sha>` the commit you are on). Everything runs on this machine against a real Postgres in
Docker; nothing is mocked.

## 0. Prerequisites, and the database

You need Docker, Go, Java 21 (for the formal models), Node with npm (for the pinned XState),
Google Chrome or Chromium (for the browser tests; `CRM_CHROME` names the binary when it is not
at its usual place), `bd` (for the planner's demo), git and perl. From the repository root:

```
$ make build
go build -o .../.bin/machinery ./cmd/machinery
$ make tools
...
jars/tla2tools.jar: ok (v1.7.4)
jars/alloy.jar: ok (v6.2.0)
```

Then, in `examples/crm` (the pinned tools are found in `../../.bin`; set `MACHINERY_TOOLS` if
yours live elsewhere):

```
$ export PATH="$(cd ../.. && pwd)/.bin:$PATH"
$ make
  help     List the targets
  up       Start Postgres (empty, on 127.0.0.1:15432) and wait until it is healthy
  ...
$ make up
up: machinery-crm-pg healthy on 127.0.0.1:15432
```

## 1. `make check`: the design, the code and the tests agree

```
$ make check
go vet ./...
.../.bin/machinery check
== model ==
checked: 1 models, 4 entities, 9 actions, 11 invariants, 4 scenarios
passed
...
== formal ==
checked: 2 specs, 3 runs, 4 reach predicates, 10 mutations, 5 assumptions
  warn   finding: deal-win run race: WinAuditedOnce violated (assumption A1)
passed
...
machinery check: 13 passed, 0 failed, 0 not demonstrated, 3 not applicable
```

What each stage protects, in one line each:

```
$ machinery checks
model               The vocabulary every other artifact uses: entities, actions, invariants, scenarios.
machines            Every state answers every event: a transition, a refusal or a reasoned ignore.
failure-modes       Each declared failure and abuse case is well formed and has a required behavior.
risk                Each risk area and status entry names real design ids, so tier gates and statuses attach to real behavior.
xref-model-machine  The lifecycle a machine implements is the lifecycle the model promises.
xref-contract       Every action is reachable through a published operation.
xref-failure        Every side effect, dependency and tiered action has a declared failure behavior.
xstate              A generated table means what XState would execute for the same machine.
gen-drift           Tables and obligations in the tree are the ones the design produces.
formal              The formal model of a critical area is current, not vacuous and not weakened.
checkers            (not applicable here) Verdicts of external checkers are for this exact design.
arch                The layering the design declares holds in the code.
tests               Evidence comes from the real system: no unit tests, no test doubles.
trace               Every action is served by code and every obligation is claimed by a test or owed.
contract            (not applicable: no wave is frozen) Stories in parallel build against the same design.
intake              (not applicable: no backlog configured) A requirement reaches the backlog only through the design it changed.
ratchet             (machinery check --base <ref>) What was proven, covered or modeled stays so.
verify              (machinery verify) Each obligation is observed on the running system at this commit.
probe               (machinery probe) The tests would notice a wrong transition or a missing effect.
story               (machinery story <id>) A story is accepted with the formality its risk tier requires.
```

(The real output puts each `protects:` text on a second line under the stage name.)

## 2. `make evidence` and `machinery verify`: every obligation observed

```
$ make evidence
.../.bin/machinery run --stack "postgres 17 in docker, crm over http" -- env GOMAXPROCS=2 go test ./e2e/... -count=1 -p 1
ok  	github.com/RamXX/machinery/examples/crm/e2e	14.644s
machinery run: .machinery/receipts/<time>-<sha>.jsonl (exit 0)
$ machinery verify
== verify ==
checked: 60 obligations, 60 proven, 0 partial, 0 owed, 76 receipts
passed
```

The receipts count grows with each run at the same commit.

## 3. `machinery status`: nothing owed

```
$ machinery status
project: crm
obligations: 60 (tc 32, sc 4, inv 11, fm 10, fv 2, cx 1)
tiers: routine 47, consequential 0, critical 13
proven: 60  partial: 0  failed: 0  owed: 0  unproven and not owed: 0
owed without a story: 0
evidence: <sha> clean, 1 run(s), 76 receipt(s)
$ machinery owed list
owed: 0 entries
$ machinery proof
PROOF
commit:   <full sha>
tier:     critical (deal-win, record-ownership)
...
criteria:
  cx:deal-win:race -> TestReplayDealWinRace -> pass: 60 rounds: ... one 200, one 409, one win audit row, one deal_wins row each
  ...
  sc:rep-qualifies-and-wins-a-deal -> TestScenarioRepQualifiesAndWinsInBrowser -> pass: in Chrome: ... the page shows Lead, Qualified, Won and no simulated note; ...
  ...
not proven:
learnings:
$ grep -rn "Simulated: true" internal/
```

The last one prints nothing (exit 1): no effect is simulated any more.

## 4. See it work

In a second terminal, `make serve` (it builds and listens on 127.0.0.1:18080), then open
`http://127.0.0.1:18080/`. Create an Admin and two Reps, choose the first Rep and press Use,
create a deal with an amount, press qualify, then win: the stage reads Qualified, then Won, and
the audit trail at the bottom gains `qualify`, `win-attempt` and `win`. Choose the second Rep
and press qualify on a deal of the first: `FORBIDDEN: the caller may not qualify this record`
appears next to it and the stage does not change. The browser tests do exactly this in a
headless Chrome:

```
$ curl -s http://127.0.0.1:18080/healthz
{"status":"ok"}
$ go test ./e2e/... -count=1 -run 'InBrowser|TestUI' -v
--- PASS: TestScenarioRepQualifiesAndWinsInBrowser (0.97s)
--- PASS: TestUIRefusalShown (0.70s)
--- PASS: TestUIServesNoExternalResources (0.02s)
ok  	github.com/RamXX/machinery/examples/crm/e2e	2.436s
```

The database stops answering, then answers again; and the application role cannot move a deal
by SQL:

```
$ docker pause machinery-crm-pg
$ curl -s -m 5 http://127.0.0.1:18080/healthz
{"code":"STORE_UNAVAILABLE","message":"database not answering: context deadline exceeded"}
$ docker unpause machinery-crm-pg
$ curl -s http://127.0.0.1:18080/healthz
{"status":"ok"}
$ docker exec machinery-crm-pg psql -U crm_app -d crm -c "UPDATE deals SET stage = 'Won' WHERE stage = 'Lead'"
ERROR:  lifecycle: deal Lead -> Won outside the transition path
```

Stop `make serve` with Ctrl-C.

## 5. `make probe`: would the tests notice a wrong system?

```
$ make probe
.../.bin/machinery probe  -- env GOMAXPROCS=2 go test ./e2e/... -count=1 -p 1
...
--- FAIL: TestTransitions/tc:TASK-f7ab1e (0.01s)
    transitions_test.go:304: effect auditCancel on task:... not observed: 0 audit rows (cancel, ..., applied), want 1
...
== probe ==
checked: 33 mutations, 33 killed, 0 survived
passed
```

Each `FAIL` is the suite catching one mutated table. To see a survivor, weaken one test on a
throwaway branch (the probe needs a clean tree and evidence at its commit), probe its case, and
go back:

```
$ git switch -c try-survivor
$ perl -pi -e 's/auditRow\("cancel", "applied"\)/func(*testing.T, arranged) error { return nil }/' e2e/transitions_test.go
$ git commit -qam 'weaken one test: claim auditCancel without looking'
$ make evidence
...
$ machinery probe --ids tc:TASK-67d9f6 -- env GOMAXPROCS=2 go test ./e2e/... -count=1 -p 1
...
== probe ==
checked: 2 mutations, 1 killed, 1 survived
  ERROR  survived: tc:TASK-67d9f6 drop-effect (dropped auditCancel)
failed: 1 finding(s)
$ git switch -
$ git branch -D try-survivor
$ make evidence
...
machinery run: .machinery/receipts/<time>-<sha>.jsonl (exit 0)
```

The retarget is still killed (the test checks the state); the dropped audit row is not, because
the weakened test claims it without looking. `machinery run` keeps only the receipts that are
valid for the commit it runs at, so the run on the throwaway branch removed the evidence of
yours: the last `make evidence` puts it back.

## 6. `make formal`: the two critical areas, model checked

`deal-win` (TLA+, `design/formal/DealProps.tla`) claims: a deal is won at most once (its win is
audited at most once), a won deal carries its win audit, a won deal never leaves Won, and the
win write always ends. `record-ownership` (Alloy, `design/formal/crm_props.als`) claims: every
Deal has one owner, every Task one Deal and one assignee, only the owner or an Admin writes a
Deal, only the assignee or an Admin writes a Task, only an Admin reopens.

```
$ make formal
.../.bin/machinery formal gen --check
== formal-gen ==
checked: 2 skeletons
passed
.../.bin/machinery formal check --workers 2
== formal ==
checked: 2 specs, 3 runs, 4 reach predicates, 10 mutations, 5 assumptions
  warn   finding: deal-win run race: WinAuditedOnce violated (assumption A1)
passed
```

The `race` run takes the row lock away and TLC finds the double win: two callers begin the win
on one Qualified deal before either commits.

```
$ grep -E '"(caller|step)"' design/formal/traces/deal-win.race.trace.json | paste - -
      "caller": "c1",	      "step": "begin",
      "caller": "c1",	      "step": "commit",
      "caller": "c1",	      "step": "begin",
      "caller": "c2",	      "step": "begin",
      "caller": "c1",	      "step": "commit",
      "caller": "engine",	      "step": "do",
      "caller": "c2",	      "step": "commit",
      "caller": "engine",	      "step": "do",
```

`e2e/replay_test.go` replays that trace against the real service, 60 rounds, and the row lock
holds, so the counterexample is refuted:

```
$ machinery proof --ids cx:deal-win:race,fv:deal-win,fv:record-ownership
PROOF
...
criteria:
  cx:deal-win:race -> TestReplayDealWinRace -> pass: 60 rounds: win and win by owner and Admin at once, both held on the row lock before either wrote: one 200, one 409, one win audit row, one deal_wins row each
  fv:deal-win -> formal -> pass: 2 run(s), 4 mutation(s) detected
  fv:record-ownership -> formal -> pass: 1 run(s), 6 mutation(s) detected
not proven:
```

## 7. `make demo-pm`: the planner's loop, on a throwaway copy

```
$ make demo-pm
== 1. a copy of the repository with its own git and backlog
...
== 2. the planner: a design change, owed, planned, in waves
$ machinery owed list
fm:deal-qualify-without-user  unproven at <sha>
sc:admin-cancels-a-task-assigned-to-a-rep  unproven at <sha>
owed: 2 entries
$ machinery plan --apply
plan: 2 created, 0 existing, 0 edge(s)
$ machinery waves --freeze
...
wave: frozen <contract> at <sha>, 2 stories
$ machinery brief crm-win --role modeler
BRIEF crm-win (modeler)
title: Serve the deal win
tier: critical (deal-win: winning a deal is final and audited exactly once, even when callers act at the same time)
...
== 3. the gates, on stories whose ids are proven
$ machinery story crm-tasks
  ERROR  no prediction for <sha>: a verifier runs machinery predict before reading the proof
$ BEADS_ACTOR=builder machinery predict crm-tasks ...
machinery: refused: builder posted the proof for <sha>; a fresh verifier predicts
(refused, as expected)
...
$ machinery story crm-win
== story ==
checked: 5 ids, 1 proofs, 1 predictions, 1 verdicts
passed
pm.sh: removed the copy
```

## 8. `make verify`: the landing

```
$ make verify
.../.bin/machinery landing --base HEAD~1 --probe --workers 2 --stack "postgres 17 in docker, crm over http" -- env GOMAXPROCS=2 go test ./e2e/... -count=1 -p 1
...
== ratchet ==
checked: 60 obligations, 1 scope globs, 18 tiered ids, 0 baseline entries
passed
machinery check: 14 passed, 0 failed, 0 not demonstrated, 3 not applicable
== formal-verify ==
checked: 2 specs
passed
ok  	github.com/RamXX/machinery/examples/crm/e2e	14.917s
== verify ==
checked: 60 obligations, 60 proven, 0 partial, 0 owed, 152 receipts
passed
...
== probe ==
checked: 33 mutations, 33 killed, 0 survived
passed
landing: passed at <sha> (attempt 1, fix-forward 0 of 2)
```

## 9. `make down`: clean up

```
$ make down
down: machinery-crm-pg removed
$ docker ps -a --filter name=machinery-crm
CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES
$ docker volume ls --filter name=machinery-crm
DRIVER    VOLUME NAME
```
